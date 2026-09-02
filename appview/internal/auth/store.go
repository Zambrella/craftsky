// Package auth contains OAuth storage (oauth.ClientAuthStore impl) and
// Craftsky bearer-token session management.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/federatedhttp"
	"social.craftsky/appview/internal/ownerlifecycle"
)

// ErrOAuthSessionNotFound is returned by GetSession / GetAuthRequestInfo
// when the requested row doesn't exist. Callers that need to distinguish
// not-found from other errors use errors.Is.
var ErrOAuthSessionNotFound = errors.New("oauth session/auth-request not found")

var (
	ErrAuthRequestMetadataInvalid = errors.New("auth request metadata invalid")
	ErrAuthRequestState           = errors.New("auth request state conflict")
	ErrAuthRequestCapacity        = errors.New("pending auth request capacity exhausted")
)

const (
	defaultPendingAuthRequestCapacity = 4096
	defaultAuthRequestRetention       = 24 * time.Hour

	// authRequestAdmissionLockKey serializes the process-independent pending
	// capacity check and insert. It is the signed int64 representation of the
	// stable ASCII domain "CSKY_AUT" and is deliberately unrelated to the
	// owner-lifecycle advisory-lock namespace.
	authRequestAdmissionLockKey int64 = 0x43534b595f415554
)

// StoreConfig carries TTLs and the logger used for lazy-cleanup errors.
type StoreConfig struct {
	SessionExpiry                time.Duration
	SessionInactivity            time.Duration
	SessionAbsoluteLifetime      time.Duration
	AuthRequestExpiry            time.Duration
	AuthRequestReservationExpiry time.Duration
	AuthRequestExchangeExpiry    time.Duration
	PendingAuthRequestCapacity   int
	AuthRequestTerminalRetention time.Duration
	Logger                       *slog.Logger
	OwnerLifecycles              *ownerlifecycle.Store
	EndpointValidator            OAuthSessionEndpointValidator
}

// OAuthSessionEndpointValidator returns federatedhttp.ErrDestinationRejected
// only for permanent URL/destination-policy violations. Cancellation, timeout,
// and resolver/dependency failures retain their typed causes so callers can
// retry without revoking otherwise valid credentials.
type OAuthSessionEndpointValidator interface {
	ValidateOrigin(context.Context, string) (*url.URL, error)
	ValidateOAuthEndpoint(context.Context, string, string) (*url.URL, error)
}

var ErrOAuthSessionEndpointInvalid = errors.New("OAuth session endpoint metadata invalid")

// PostgresAuthStore is a Postgres-backed implementation of
// oauth.ClientAuthStore. The ClientSessionData / AuthRequestData blobs
// are round-tripped as opaque JSONB; this code never inspects them
// beyond what indigo's serializer provides.
//
// Session cleanup is independent of OAuth auth-request cleanup. Auth requests
// use a bounded context-owned sweeper plus admission-time expiry reclamation.
type PostgresAuthStore struct {
	pool   *pgxpool.Pool
	cfg    StoreConfig
	owners *ownerlifecycle.Store
}

var _ oauth.ClientAuthStore = (*PostgresAuthStore)(nil)

// NewPostgresAuthStore returns a PostgresAuthStore backed by pool.
// If cfg.Logger is nil, slog.Default() is used.
func NewPostgresAuthStore(pool *pgxpool.Pool, cfg StoreConfig) *PostgresAuthStore {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.SessionAbsoluteLifetime <= 0 {
		cfg.SessionAbsoluteLifetime = cfg.SessionExpiry
	}
	if cfg.PendingAuthRequestCapacity <= 0 {
		cfg.PendingAuthRequestCapacity = defaultPendingAuthRequestCapacity
	}
	if cfg.AuthRequestReservationExpiry <= 0 {
		cfg.AuthRequestReservationExpiry = DefaultOAuthLoginStartTimeout + time.Minute
	}
	if cfg.AuthRequestExchangeExpiry <= 0 {
		cfg.AuthRequestExchangeExpiry = DefaultOAuthCallbackOperationTimeout + time.Minute
	}
	if cfg.AuthRequestTerminalRetention <= 0 {
		cfg.AuthRequestTerminalRetention = defaultAuthRequestRetention
	}
	return &PostgresAuthStore{pool: pool, cfg: cfg, owners: cfg.OwnerLifecycles}
}

type AuthRequestReservation struct {
	ID        uuid.UUID
	ExpiresAt time.Time
}

type RegistrationAuthorityBinding struct {
	State           string
	AttemptID       uuid.UUID
	Owner           syntax.DID
	OwnerGeneration int64
	AuthEpoch       int64
	Session         oauth.ClientSessionData
}

func (s *PostgresAuthStore) QuarantineRegistrationCredential(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
	session oauth.ClientSessionData,
	eligibleAt time.Time,
) error {
	if state == "" || attemptID == uuid.Nil || session.SessionID != state || eligibleAt.IsZero() {
		return ErrCallbackAttemptInvalid
	}
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal registration credential: %w", err)
	}
	expiresAt := eligibleAt.Add(s.cfg.AuthRequestTerminalRetention)
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var purpose OAuthPurpose
		var requestState AuthRequestState
		var requestAttempt uuid.UUID
		var issuer string
		if err := tx.QueryRow(ctx, `
			SELECT purpose,request_state,exchange_attempt_id,registration_issuer
			FROM oauth_auth_requests WHERE state=$1 FOR UPDATE
		`, state).Scan(&purpose, &requestState, &requestAttempt, &issuer); err != nil {
			return err
		}
		if purpose != RegistrationOAuthPurpose || requestState != AuthRequestExchangeStarted ||
			requestAttempt != attemptID || issuer != session.AuthServerURL {
			return ErrCallbackAttemptInvalid
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO oauth_unverified_credentials(request_state,data,status,eligible_at,expires_at)
			VALUES($1,$2,'held',$3,$4)
		`, state, data, eligibleAt, expiresAt)
		return err
	})
	if err != nil {
		return fmt.Errorf("quarantine registration credential: %w", err)
	}
	return nil
}

func (s *PostgresAuthStore) MarkRegistrationCredentialForCleanup(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
) error {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var credentialStatus string
		if err := tx.QueryRow(ctx, `
			SELECT credential.status
			FROM oauth_auth_requests request
			JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
			WHERE request.state=$1 AND request.purpose='registration'
			  AND request.request_state='exchange_started' AND request.exchange_attempt_id=$2
			FOR UPDATE OF request,credential
		`, state, attemptID).Scan(&credentialStatus); err != nil {
			return err
		}
		if credentialStatus != "held" {
			return ErrAuthRequestState
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_unverified_credentials
			SET status='pending',eligible_at=LEAST(eligible_at,now()),updated_at=now()
			WHERE request_state=$1 AND status='held'
		`, state); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='cleanup_pending',exchange_finished_at=now()
			WHERE state=$1 AND purpose='registration' AND request_state='exchange_started'
			  AND exchange_attempt_id=$2
		`, state, attemptID)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrAuthRequestState
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("mark registration credential for cleanup: %w", err)
	}
	return nil
}

var ErrSessionVersionChanged = errors.New("OAuth session version changed")

// SaveSession is Indigo's initial callback persistence hook. It is deliberately
// create-only and requires the attempt capability installed after the durable
// exchange_started transition. Refresh persistence uses SaveSessionVersion.
func (s *PostgresAuthStore) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
	attempt, ok := callbackAttemptFromContext(ctx)
	if !ok || !attempt.validFor(sess.AccountDID, sess.SessionID) {
		return ErrCallbackAttemptInvalid
	}
	if err := s.validateSessionEndpoints(ctx, sess); err != nil {
		return err
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	lifetime := s.cfg.SessionAbsoluteLifetime
	if lifetime <= 0 {
		return errors.New("OAuth session absolute lifetime is not configured")
	}
	return s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		var state string
		var generation, epoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,generation,auth_epoch
			FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
		`, sess.AccountDID).Scan(&state, &generation, &epoch); err != nil {
			return fmt.Errorf("load callback owner authority: %w", err)
		}
		if state == "terminal" || generation != attempt.OwnerGeneration || epoch != attempt.AuthEpoch {
			return ErrCallbackAttemptInvalid
		}
		var requestPurpose OAuthPurpose
		var requestState AuthRequestState
		var requestAttempt uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT purpose,request_state,exchange_attempt_id
			FROM oauth_auth_requests
			WHERE state=$1 FOR UPDATE
		`, attempt.State).Scan(&requestPurpose, &requestState, &requestAttempt); err != nil {
			return fmt.Errorf("lock callback attempt: %w", err)
		}
		if requestPurpose != attempt.Purpose || requestState != AuthRequestExchangeStarted || requestAttempt != attempt.AttemptID {
			return ErrCallbackAttemptInvalid
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO oauth_sessions(
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				row_version,absolute_expires_at,created_at,updated_at
			) VALUES($1,$2,$3,'pending_handoff',$4,$5,1,now()+make_interval(secs => $6::double precision),now(),now())
		`, sess.AccountDID, sess.SessionID, data, generation, epoch, lifetime.Seconds())
		if err != nil {
			return fmt.Errorf("insert pending OAuth session: %w", err)
		}
		return nil
	})
}

func (s *PostgresAuthStore) BindRegistrationAuthority(
	ctx context.Context,
	binding RegistrationAuthorityBinding,
) error {
	if binding.State == "" || binding.AttemptID == uuid.Nil || binding.Owner == "" ||
		binding.OwnerGeneration <= 0 || binding.AuthEpoch <= 0 ||
		binding.Session.AccountDID != binding.Owner || binding.Session.SessionID != binding.State {
		return ErrCallbackAttemptInvalid
	}
	if err := s.validateSessionEndpoints(ctx, binding.Session); err != nil {
		return err
	}
	data, err := json.Marshal(binding.Session)
	if err != nil {
		return fmt.Errorf("marshal bound registration session: %w", err)
	}
	if s.cfg.SessionAbsoluteLifetime <= 0 {
		return errors.New("OAuth session absolute lifetime is not configured")
	}
	err = s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		var lifecycleState string
		var generation, epoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,generation,auth_epoch
			FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
		`, binding.Owner).Scan(&lifecycleState, &generation, &epoch); err != nil {
			return fmt.Errorf("load registration owner authority: %w", err)
		}
		if (lifecycleState != "departed" && lifecycleState != "active") ||
			generation != binding.OwnerGeneration || epoch != binding.AuthEpoch {
			return ErrCallbackAttemptInvalid
		}

		var purpose OAuthPurpose
		var requestState AuthRequestState
		var attemptID uuid.UUID
		var issuer string
		var requestOwner *string
		var requestGeneration, requestEpoch *int64
		if err := tx.QueryRow(ctx, `
			SELECT purpose,request_state,exchange_attempt_id,registration_issuer,
			       owner_did,owner_generation,auth_epoch
			FROM oauth_auth_requests WHERE state=$1 FOR UPDATE
		`, binding.State).Scan(
			&purpose, &requestState, &attemptID, &issuer,
			&requestOwner, &requestGeneration, &requestEpoch,
		); err != nil {
			return fmt.Errorf("lock registration auth request: %w", err)
		}
		if purpose != RegistrationOAuthPurpose || requestState != AuthRequestExchangeStarted ||
			attemptID != binding.AttemptID || requestOwner != nil || requestGeneration != nil ||
			requestEpoch != nil || issuer != binding.Session.AuthServerURL {
			return ErrCallbackAttemptInvalid
		}

		var credentialData []byte
		var credentialStatus string
		if err := tx.QueryRow(ctx, `
			SELECT data,status FROM oauth_unverified_credentials
			WHERE request_state=$1 FOR UPDATE
		`, binding.State).Scan(&credentialData, &credentialStatus); err != nil {
			return fmt.Errorf("lock registration credential: %w", err)
		}
		var credential oauth.ClientSessionData
		if err := json.Unmarshal(credentialData, &credential); err != nil {
			return fmt.Errorf("decode registration credential: %w", err)
		}
		credential.HostURL = binding.Session.HostURL
		if credentialStatus != "held" || !reflect.DeepEqual(credential, binding.Session) {
			return ErrCallbackAttemptInvalid
		}

		command, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET owner_did=$2,owner_generation=$3,auth_epoch=$4
			WHERE state=$1 AND purpose='registration' AND request_state='exchange_started'
			  AND exchange_attempt_id=$5
			  AND owner_did IS NULL AND owner_generation IS NULL AND auth_epoch IS NULL
		`, binding.State, binding.Owner, generation, epoch, binding.AttemptID)
		if err != nil {
			return fmt.Errorf("attach registration authority: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrCallbackAttemptInvalid
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO oauth_sessions(
				account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
				row_version,absolute_expires_at,created_at,updated_at
			) VALUES($1,$2,$3,'pending_handoff',$4,$5,1,
			         now()+make_interval(secs => $6::double precision),now(),now())
		`, binding.Owner, binding.State, data, generation, epoch, s.cfg.SessionAbsoluteLifetime.Seconds()); err != nil {
			return fmt.Errorf("insert registration pending OAuth session: %w", err)
		}
		command, err = tx.Exec(ctx, `
			DELETE FROM oauth_unverified_credentials
			WHERE request_state=$1 AND status='held'
		`, binding.State)
		if err != nil {
			return fmt.Errorf("consume registration credential: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrCallbackAttemptInvalid
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("bind registration authority: %w", err)
	}
	return nil
}

func (s *PostgresAuthStore) SaveSessionVersion(
	ctx context.Context,
	sess oauth.ClientSessionData,
	expectedVersion int64,
) (int64, error) {
	if expectedVersion <= 0 {
		return 0, ErrSessionVersionChanged
	}
	if err := s.validateSessionEndpoints(ctx, sess); err != nil {
		return 0, err
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return 0, fmt.Errorf("marshal refreshed session: %w", err)
	}
	return s.saveSessionVersionData(ctx, s.pool, sess, data, expectedVersion)
}

type oauthSessionVersionQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *PostgresAuthStore) saveSessionVersionTx(
	ctx context.Context,
	tx pgx.Tx,
	sess oauth.ClientSessionData,
	expectedVersion int64,
) (int64, error) {
	data, err := json.Marshal(sess)
	if err != nil {
		return 0, fmt.Errorf("marshal refreshed session: %w", err)
	}
	return s.saveSessionVersionData(ctx, tx, sess, data, expectedVersion)
}

func (s *PostgresAuthStore) saveSessionVersionData(
	ctx context.Context,
	queryer oauthSessionVersionQueryer,
	sess oauth.ClientSessionData,
	data []byte,
	expectedVersion int64,
) (int64, error) {
	var version int64
	err := queryer.QueryRow(ctx, `
		UPDATE oauth_sessions
		SET data=$3,row_version=row_version+1,updated_at=now()
		WHERE account_did=$1 AND session_id=$2 AND row_version=$4
		  AND lifecycle_state IN ('active','deletion_only')
		RETURNING row_version
	`, sess.AccountDID, sess.SessionID, data, expectedVersion).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrSessionVersionChanged
	}
	if err != nil {
		return 0, fmt.Errorf("persist refreshed OAuth session: %w", err)
	}
	return version, nil
}

// GetSession is the ordinary Indigo resume boundary. Pending handoffs,
// deletion-only credentials, revocation-pending parents, old epochs, expired
// parents, and non-active owners are intentionally invisible here.
func (s *PostgresAuthStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	var data []byte
	err := s.pool.QueryRow(ctx,
		`SELECT oauth.data
		 FROM oauth_sessions oauth
		 JOIN owner_lifecycles owner ON owner.owner_did=oauth.account_did
		 WHERE oauth.account_did=$1 AND oauth.session_id=$2
		   AND oauth.lifecycle_state='active'
		   AND oauth.absolute_expires_at>now()
		   AND owner.state='active'
		   AND owner.auth_epoch=oauth.auth_epoch`,
		did.String(), sessionID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select session: %w", err)
	}
	var sess oauth.ClientSessionData
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	if err := s.validateSessionEndpoints(ctx, sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

type StoredOAuthSession struct {
	Data            oauth.ClientSessionData
	RowVersion      int64
	LifecycleState  string
	OwnerGeneration int64
	AuthEpoch       int64
	AbsoluteExpiry  time.Time
}

func (s *PostgresAuthStore) LoadActiveSession(
	ctx context.Context,
	did syntax.DID,
	sessionID string,
) (StoredOAuthSession, error) {
	record, err := s.loadActiveSessionRow(ctx, s.pool, did, sessionID)
	if err != nil {
		return StoredOAuthSession{}, err
	}
	if err := s.validateSessionEndpoints(ctx, record.Data); err != nil {
		return StoredOAuthSession{}, err
	}
	return record, nil
}

func (s *PostgresAuthStore) loadActiveSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	did syntax.DID,
	sessionID string,
) (StoredOAuthSession, error) {
	return s.loadActiveSessionRow(ctx, tx, did, sessionID)
}

func (s *PostgresAuthStore) loadActiveSessionRow(
	ctx context.Context,
	queryer oauthSessionVersionQueryer,
	did syntax.DID,
	sessionID string,
) (StoredOAuthSession, error) {
	var record StoredOAuthSession
	var data []byte
	err := queryer.QueryRow(ctx, `
		SELECT oauth.data,oauth.row_version,oauth.lifecycle_state,
		       oauth.owner_generation,oauth.auth_epoch,oauth.absolute_expires_at
		FROM oauth_sessions oauth
		JOIN owner_lifecycles owner ON owner.owner_did=oauth.account_did
		WHERE oauth.account_did=$1 AND oauth.session_id=$2
		  AND oauth.lifecycle_state='active' AND oauth.absolute_expires_at>now()
		  AND owner.state='active' AND owner.auth_epoch=oauth.auth_epoch
	`, did, sessionID).Scan(
		&data, &record.RowVersion, &record.LifecycleState,
		&record.OwnerGeneration, &record.AuthEpoch, &record.AbsoluteExpiry,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredOAuthSession{}, ErrOAuthSessionNotFound
	}
	if err != nil {
		return StoredOAuthSession{}, fmt.Errorf("load active OAuth session: %w", err)
	}
	if err := json.Unmarshal(data, &record.Data); err != nil {
		return StoredOAuthSession{}, fmt.Errorf("unmarshal active OAuth session: %w", err)
	}
	return record, nil
}

func (s *PostgresAuthStore) ResumePendingOnboardingSession(
	ctx context.Context,
	attempt CallbackAttempt,
) (StoredOAuthSession, error) {
	fromContext, ok := callbackAttemptFromContext(ctx)
	if !ok || fromContext != attempt || !attempt.validFor(attempt.Owner, attempt.State) ||
		!attempt.permitsOrdinaryOnboarding() {
		return StoredOAuthSession{}, ErrCallbackAttemptInvalid
	}
	var record StoredOAuthSession
	var data []byte
	err := s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			SELECT oauth.data,oauth.row_version,oauth.lifecycle_state,
			       oauth.owner_generation,oauth.auth_epoch,oauth.absolute_expires_at
			FROM oauth_sessions oauth
			JOIN oauth_auth_requests request ON request.state=oauth.session_id
			JOIN owner_lifecycles owner ON owner.owner_did=oauth.account_did
			WHERE oauth.account_did=$1 AND oauth.session_id=$2
			  AND oauth.lifecycle_state='pending_handoff'
			  AND oauth.owner_generation=$3 AND oauth.auth_epoch=$4
			  AND oauth.absolute_expires_at>now()
			  AND request.exchange_attempt_id=$5 AND request.request_state='exchange_started'
			  AND owner.state<>'terminal' AND owner.generation=$3 AND owner.auth_epoch=$4
		`, attempt.Owner, attempt.State, attempt.OwnerGeneration, attempt.AuthEpoch, attempt.AttemptID).Scan(
			&data, &record.RowVersion, &record.LifecycleState,
			&record.OwnerGeneration, &record.AuthEpoch, &record.AbsoluteExpiry,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredOAuthSession{}, ErrOAuthSessionNotFound
	}
	if err != nil {
		return StoredOAuthSession{}, fmt.Errorf("resume pending onboarding session: %w", err)
	}
	if err := json.Unmarshal(data, &record.Data); err != nil {
		return StoredOAuthSession{}, fmt.Errorf("unmarshal pending onboarding session: %w", err)
	}
	if err := s.validateSessionEndpoints(ctx, record.Data); err != nil {
		return StoredOAuthSession{}, err
	}
	return record, nil
}

func (s *PostgresAuthStore) validateSessionEndpoints(ctx context.Context, session oauth.ClientSessionData) error {
	if s == nil || s.cfg.EndpointValidator == nil || session.HostURL == "" || session.AuthServerURL == "" ||
		session.AuthServerTokenEndpoint == "" {
		return ErrOAuthSessionEndpointInvalid
	}
	host, err := s.cfg.EndpointValidator.ValidateOrigin(ctx, session.HostURL)
	if err != nil {
		return classifySessionEndpointValidationError(err)
	}
	if host == nil || host.String() != session.HostURL {
		return ErrOAuthSessionEndpointInvalid
	}
	issuer, err := s.cfg.EndpointValidator.ValidateOrigin(ctx, session.AuthServerURL)
	if err != nil {
		return classifySessionEndpointValidationError(err)
	}
	if issuer == nil || issuer.String() != session.AuthServerURL {
		return ErrOAuthSessionEndpointInvalid
	}
	tokenEndpoint, err := s.cfg.EndpointValidator.ValidateOAuthEndpoint(
		ctx, session.AuthServerURL, session.AuthServerTokenEndpoint,
	)
	if err != nil {
		return classifySessionEndpointValidationError(err)
	}
	if tokenEndpoint == nil || tokenEndpoint.String() != session.AuthServerTokenEndpoint {
		return ErrOAuthSessionEndpointInvalid
	}
	if session.AuthServerRevocationEndpoint != "" {
		revocationEndpoint, err := s.cfg.EndpointValidator.ValidateOAuthEndpoint(
			ctx, session.AuthServerURL, session.AuthServerRevocationEndpoint,
		)
		if err != nil {
			return classifySessionEndpointValidationError(err)
		}
		if revocationEndpoint == nil || revocationEndpoint.String() != session.AuthServerRevocationEndpoint {
			return ErrOAuthSessionEndpointInvalid
		}
	}
	return nil
}

func classifySessionEndpointValidationError(err error) error {
	if errors.Is(err, federatedhttp.ErrDestinationRejected) {
		return ErrOAuthSessionEndpointInvalid
	}
	return err
}

// DeletionCredentialOperationBinder locks and updates the exact account
// deletion operation between the authorization-request and parent-session
// lock classes. Returning an error rolls back the entire credential binding.
type DeletionCredentialOperationBinder func(context.Context, pgx.Tx) error

// BindDeletionCredential converts the callback's pending parent into the
// narrow deletion-only capability. It never creates a CraftSky child and it
// binds all authority dimensions: owner, operation, session, auth epoch, and
// positive credential generation.
func (s *PostgresAuthStore) BindDeletionCredential(
	ctx context.Context,
	attempt CallbackAttempt,
	operationID uuid.UUID,
	credentialGeneration int64,
	operationBinder ...DeletionCredentialOperationBinder,
) error {
	boundAttempt, ok := callbackAttemptFromContext(ctx)
	if !ok || boundAttempt != attempt || attempt.Purpose != AccountDeletionOAuthPurpose ||
		!attempt.validFor(attempt.Owner, attempt.State) || operationID == uuid.Nil ||
		credentialGeneration <= 0 || len(operationBinder) > 1 {
		return ErrCallbackAttemptInvalid
	}
	return s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		var ownerState string
		var generation, authEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,generation,auth_epoch FROM owner_lifecycles
			WHERE owner_did=$1 FOR UPDATE
		`, attempt.Owner).Scan(&ownerState, &generation, &authEpoch); err != nil {
			return err
		}
		if (ownerState != "deletion_pending" && ownerState != "deleting") ||
			generation != attempt.OwnerGeneration || authEpoch != attempt.AuthEpoch {
			return ErrCallbackAttemptInvalid
		}
		var purpose OAuthPurpose
		var jobID *uuid.UUID
		var requestState AuthRequestState
		var attemptID *uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT purpose,account_deletion_job_id,request_state,exchange_attempt_id
			FROM oauth_auth_requests WHERE state=$1 FOR UPDATE
		`, attempt.State).Scan(&purpose, &jobID, &requestState, &attemptID); err != nil {
			return err
		}
		if purpose != AccountDeletionOAuthPurpose || jobID == nil || *jobID != operationID ||
			requestState != AuthRequestExchangeStarted || attemptID == nil || *attemptID != attempt.AttemptID {
			return ErrCallbackAttemptInvalid
		}
		if len(operationBinder) == 1 && operationBinder[0] != nil {
			if err := operationBinder[0](ctx, tx); err != nil {
				return err
			}
		}
		var parentState string
		var parentGeneration, parentEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT lifecycle_state,owner_generation,auth_epoch
			FROM oauth_sessions WHERE account_did=$1 AND session_id=$2 FOR UPDATE
		`, attempt.Owner, attempt.State).Scan(&parentState, &parentGeneration, &parentEpoch); err != nil {
			return err
		}
		if parentState != "pending_handoff" || parentGeneration != generation || parentEpoch != authEpoch {
			return ErrCallbackAttemptInvalid
		}
		now := time.Now().UTC()
		command, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='deletion_only',deletion_operation_id=$3,
			    deletion_credential_generation=$4,row_version=row_version+1,updated_at=$5
			WHERE account_did=$1 AND session_id=$2 AND lifecycle_state='pending_handoff'
		`, attempt.Owner, attempt.State, operationID, credentialGeneration, now)
		if err != nil {
			return fmt.Errorf("bind deletion-only OAuth parent: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrCallbackAttemptInvalid
		}
		command, err = tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='consumed',exchange_finished_at=$3,consumed_at=COALESCE(consumed_at,$3)
			WHERE state=$1 AND exchange_attempt_id=$2 AND request_state='exchange_started'
		`, attempt.State, attempt.AttemptID, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrCallbackAttemptInvalid
		}
		return nil
	})
}

// AbandonPendingSession is the durable compensation path after the initial
// parent exists but callback finalization fails. Local authority is removed in
// one transaction and upstream revocation is delegated to the bounded worker.
func (s *PostgresAuthStore) AbandonPendingSession(ctx context.Context, attempt CallbackAttempt) error {
	boundAttempt, ok := callbackAttemptFromContext(ctx)
	if !ok || boundAttempt != attempt || !attempt.validFor(attempt.Owner, attempt.State) {
		return ErrCallbackAttemptInvalid
	}
	return s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		var ownerEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT auth_epoch FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
		`, attempt.Owner).Scan(&ownerEpoch); err != nil {
			return err
		}
		if ownerEpoch != attempt.AuthEpoch {
			return ErrCallbackAttemptInvalid
		}
		var requestAttempt *uuid.UUID
		if err := tx.QueryRow(ctx, `
			SELECT exchange_attempt_id FROM oauth_auth_requests WHERE state=$1 FOR UPDATE
		`, attempt.State).Scan(&requestAttempt); err != nil {
			return err
		}
		if requestAttempt == nil || *requestAttempt != attempt.AttemptID {
			return ErrCallbackAttemptInvalid
		}
		var parentState string
		if err := tx.QueryRow(ctx, `
			SELECT lifecycle_state FROM oauth_sessions
			WHERE account_did=$1 AND session_id=$2 FOR UPDATE
		`, attempt.Owner, attempt.State).Scan(&parentState); err != nil {
			return err
		}
		if parentState != "pending_handoff" && parentState != "deletion_only" && parentState != "revocation_pending" {
			return ErrCallbackAttemptInvalid
		}
		now := time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,$3)
			WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
		`, attempt.Owner, attempt.State, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='revocation_pending',
			    deletion_operation_id=NULL,deletion_credential_generation=NULL,
			    revocation_requested_at=COALESCE(revocation_requested_at,$3),
			    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,$3),
			    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
			    row_version=row_version+1,updated_at=$3
			WHERE account_did=$1 AND session_id=$2
		`, attempt.Owner, attempt.State, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_handoff_exchanges
			SET state='revoked',code_hash=NULL,consumed_at=COALESCE(consumed_at,$3),updated_at=$3
			WHERE owner_did=$1 AND oauth_session_id=$2 AND state IN ('ready','redeemed')
		`, attempt.Owner, attempt.State, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='consumed',exchange_finished_at=COALESCE(exchange_finished_at,$3),
			    consumed_at=COALESCE(consumed_at,$3)
			WHERE state=$1 AND exchange_attempt_id=$2
		`, attempt.State, attempt.AttemptID, now)
		return err
	})
}

// DeleteSession is local-first revocation for Indigo compatibility. The row is
// retained only for the bounded upstream cleanup worker.
func (s *PostgresAuthStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		now := time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='revocation_pending',revocation_requested_at=COALESCE(revocation_requested_at,$3),
			    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,$3),
			    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
			    row_version=row_version+1,updated_at=$3
			WHERE account_did=$1 AND session_id=$2 AND lifecycle_state<>'revocation_pending'
		`, did, sessionID, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,$3)
			WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
		`, did, sessionID, now)
		return err
	})
}

func (s *PostgresAuthStore) DeleteRevokedSession(ctx context.Context, did syntax.DID, sessionID string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM oauth_sessions
		WHERE account_did=$1 AND session_id=$2 AND lifecycle_state='revocation_pending'
	`, did, sessionID)
	return err
}

func (s *PostgresAuthStore) ReserveAuthRequestCapacity(ctx context.Context) (AuthRequestReservation, error) {
	reservation := AuthRequestReservation{ID: uuid.New()}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockAuthRequestAdmission(ctx, tx); err != nil {
			return err
		}
		now := time.Now().UTC()
		pending, err := s.reclaimAndCountAuthRequestCapacity(ctx, tx, now)
		if err != nil {
			return err
		}
		if pending >= s.cfg.PendingAuthRequestCapacity {
			return ErrAuthRequestCapacity
		}
		reservation.ExpiresAt = now.Add(s.cfg.AuthRequestReservationExpiry)
		_, err = tx.Exec(ctx, `
			INSERT INTO oauth_auth_request_reservations(id,expires_at)
			VALUES($1,$2)
		`, reservation.ID, reservation.ExpiresAt)
		return err
	})
	if err != nil {
		return AuthRequestReservation{}, fmt.Errorf("reserve auth request capacity: %w", err)
	}
	return reservation, nil
}

func (s *PostgresAuthStore) ReleaseAuthRequestCapacity(ctx context.Context, reservationID uuid.UUID) error {
	if reservationID == uuid.Nil {
		return ErrAuthRequestMetadataInvalid
	}
	_, err := s.pool.Exec(ctx, `DELETE FROM oauth_auth_request_reservations WHERE id=$1`, reservationID)
	if err != nil {
		return fmt.Errorf("release auth request capacity: %w", err)
	}
	return nil
}

// SaveAuthRequestInfo inserts an owner-bound login or deletion request. State
// is the primary key, so duplicate states retain create-only semantics.
func (s *PostgresAuthStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal auth request: %w", err)
	}
	metadata, ok := AuthRequestMetadataFromContext(ctx)
	if !ok || !metadata.valid() || metadata.Purpose == RegistrationOAuthPurpose || info.State == "" || info.RequestURI == "" {
		return ErrAuthRequestMetadataInvalid
	}
	err = s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		var state string
		var generation, epoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,generation,auth_epoch
			FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
		`, metadata.Owner).Scan(&state, &generation, &epoch); err != nil {
			return err
		}
		if state == "terminal" || generation != metadata.OwnerGeneration || epoch != metadata.AuthEpoch {
			return ErrAuthRequestMetadataInvalid
		}
		if err := lockAuthRequestAdmission(ctx, tx); err != nil {
			return err
		}
		pending, err := s.reclaimAndCountAuthRequestCapacity(ctx, tx, time.Now().UTC())
		if err != nil {
			return err
		}
		if pending >= s.cfg.PendingAuthRequestCapacity {
			return ErrAuthRequestCapacity
		}
		return insertAuthRequest(ctx, tx, info, data, metadata)
	})
	if err != nil {
		return fmt.Errorf("insert auth request: %w", err)
	}
	return nil
}

func (s *PostgresAuthStore) SaveRegistrationAuthRequest(
	ctx context.Context,
	reservationID uuid.UUID,
	info oauth.AuthRequestData,
) error {
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal registration auth request: %w", err)
	}
	metadata, ok := AuthRequestMetadataFromContext(ctx)
	if !ok || !metadata.valid() || metadata.Purpose != RegistrationOAuthPurpose ||
		reservationID == uuid.Nil || info.State == "" || info.RequestURI == "" {
		return ErrAuthRequestMetadataInvalid
	}
	err = pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockAuthRequestAdmission(ctx, tx); err != nil {
			return err
		}
		now := time.Now().UTC()
		if _, err := s.reclaimAndCountAuthRequestCapacity(ctx, tx, now); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `
			DELETE FROM oauth_auth_request_reservations
			WHERE id=$1 AND expires_at>$2
		`, reservationID, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrAuthRequestCapacity
		}
		return insertAuthRequest(ctx, tx, info, data, metadata)
	})
	if err != nil {
		return fmt.Errorf("insert registration auth request: %w", err)
	}
	return nil
}

func lockAuthRequestAdmission(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, authRequestAdmissionLockKey); err != nil {
		return fmt.Errorf("lock auth request admission: %w", err)
	}
	return nil
}

func (s *PostgresAuthStore) reclaimAndCountAuthRequestCapacity(
	ctx context.Context,
	tx pgx.Tx,
	now time.Time,
) (int, error) {
	if _, err := tx.Exec(ctx, `DELETE FROM oauth_auth_request_reservations WHERE expires_at<=$1`, now); err != nil {
		return 0, fmt.Errorf("delete expired auth request reservations: %w", err)
	}
	readyCutoff := now.Add(-s.cfg.AuthRequestExpiry)
	if _, err := tx.Exec(ctx, `
		WITH expired AS (
			SELECT state
			FROM oauth_auth_requests
			WHERE request_state='ready' AND created_at<$1
			ORDER BY created_at,state
			LIMIT $2
			FOR UPDATE
		)
		DELETE FROM oauth_auth_requests AS requests
		USING expired
		WHERE requests.state=expired.state
	`, readyCutoff, s.cfg.PendingAuthRequestCapacity); err != nil {
		return 0, fmt.Errorf("delete expired ready auth requests: %w", err)
	}
	var pending int
	if err := tx.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM oauth_auth_request_reservations WHERE expires_at>$2)
			+
			(SELECT count(*) FROM oauth_auth_requests
			 WHERE (request_state='ready' AND created_at>=$1)
			    OR request_state IN ('exchange_started','cleanup_pending')
			    OR (request_state='exchange_ambiguous'
			        AND (purpose<>'registration' OR exchange_finished_at>=$1)))
	`, readyCutoff, now).Scan(&pending); err != nil {
		return 0, fmt.Errorf("count pending auth requests and reservations: %w", err)
	}
	return pending, nil
}

func insertAuthRequest(
	ctx context.Context,
	tx pgx.Tx,
	info oauth.AuthRequestData,
	data []byte,
	metadata AuthRequestMetadata,
) error {
	var deletionOwner, deletionJob, owner, generation, epoch, provider, issuer any
	if metadata.Purpose == RegistrationOAuthPurpose {
		provider = metadata.RegistrationProviderOrigin
		issuer = metadata.RegistrationIssuer
	} else {
		owner = metadata.Owner
		generation = metadata.OwnerGeneration
		epoch = metadata.AuthEpoch
		if metadata.Purpose == AccountDeletionOAuthPurpose {
			deletionOwner = metadata.Owner
			deletionJob = metadata.JobID
		}
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO oauth_auth_requests (
			state,data,purpose,account_deletion_owner_did,account_deletion_job_id,
			owner_did,owner_generation,auth_epoch,request_uri,request_state,
			handoff_mode,loopback_redirect_uri,device_id,
			registration_provider_origin,registration_issuer
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'ready',$10,$11,$12,$13,$14)
	`, info.State, data, metadata.Purpose, deletionOwner, deletionJob,
		owner, generation, epoch, info.RequestURI,
		metadata.HandoffMode, nullableString(metadata.LoopbackURI), metadata.DeviceID,
		provider, issuer)
	return err
}

// GetAuthRequestInfo returns the auth request for state, or ErrOAuthSessionNotFound
// if no matching usable row exists. Expiry is reclaimed by admission and the
// independent sweeper rather than callback traffic.
func (s *PostgresAuthStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	var data []byte
	err := s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT data FROM oauth_auth_requests
			 WHERE state = $1
			   AND ((request_state='ready' AND created_at>=$2) OR request_state='exchange_started')`,
			state, time.Now().Add(-s.cfg.AuthRequestExpiry)).Scan(&data)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select auth request: %w", err)
	}
	var info oauth.AuthRequestData
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal auth request: %w", err)
	}
	return &info, nil
}

func (s *PostgresAuthStore) GetRegistrationAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	var data []byte
	err := s.pool.QueryRow(ctx, `
		SELECT data FROM oauth_auth_requests
		WHERE state=$1 AND purpose='registration' AND request_state='exchange_started'
	`, state).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOAuthSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("select registration auth request: %w", err)
	}
	var info oauth.AuthRequestData
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("unmarshal registration auth request: %w", err)
	}
	return &info, nil
}

// DeleteAuthRequestInfo implements Indigo's destructive-sounding callback as
// a logical consume. Non-secret request and attempt evidence remains durable
// for compensation, ambiguity alerts, and cleanup.
func (s *PostgresAuthStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	return s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET consumed_at=COALESCE(consumed_at,now()),
			    request_state=CASE WHEN request_state='ready' THEN 'consumed' ELSE request_state END
			WHERE state=$1 AND request_state IN ('ready','exchange_started','consumed')
		`, state)
		return err
	})
}

func (s *PostgresAuthStore) LoadAuthRequestMetadata(ctx context.Context, state string) (AuthRequestMetadata, error) {
	var metadata AuthRequestMetadata
	var createdAt time.Time
	var loopback *string
	var jobID *uuid.UUID
	var attemptID *uuid.UUID
	var owner *string
	var ownerGeneration *int64
	var authEpoch *int64
	var registrationProvider *string
	var registrationIssuer *string
	err := s.pool.QueryRow(ctx, `
		SELECT purpose,owner_did,owner_generation,auth_epoch,account_deletion_job_id,
		       handoff_mode,loopback_redirect_uri,device_id,request_state,exchange_attempt_id,
		       registration_provider_origin,registration_issuer,created_at
		FROM oauth_auth_requests WHERE state=$1
	`, state).Scan(
		&metadata.Purpose, &owner, &ownerGeneration, &authEpoch,
		&jobID, &metadata.HandoffMode, &loopback, &metadata.DeviceID,
		&metadata.RequestState, &attemptID, &registrationProvider, &registrationIssuer, &createdAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthRequestMetadata{}, ErrOAuthSessionNotFound
	}
	if err != nil {
		return AuthRequestMetadata{}, fmt.Errorf("load auth request metadata: %w", err)
	}
	if loopback != nil {
		metadata.LoopbackURI = *loopback
	}
	if jobID != nil {
		metadata.JobID = *jobID
	}
	if attemptID != nil {
		metadata.ExchangeAttemptID = *attemptID
	}
	if owner != nil {
		metadata.Owner = syntax.DID(*owner)
	}
	if ownerGeneration != nil {
		metadata.OwnerGeneration = *ownerGeneration
	}
	if authEpoch != nil {
		metadata.AuthEpoch = *authEpoch
	}
	if registrationProvider != nil {
		metadata.RegistrationProviderOrigin = *registrationProvider
	}
	if registrationIssuer != nil {
		metadata.RegistrationIssuer = *registrationIssuer
	}
	metadata.ExpiresAt = createdAt.Add(s.cfg.AuthRequestExpiry)
	return metadata, nil
}

func (s *PostgresAuthStore) BeginExchange(ctx context.Context, state string) (uuid.UUID, error) {
	attemptID := uuid.New()
	var stored uuid.UUID
	err := s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='exchange_started',exchange_attempt_id=$2,
			    exchange_started_at=now(),consumed_at=now()
			WHERE state=$1 AND request_state='ready'
			RETURNING exchange_attempt_id
		`, state, attemptID).Scan(&stored)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAuthRequestState
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin OAuth exchange: %w", err)
	}
	return stored, nil
}

func (s *PostgresAuthStore) BeginRegistrationExchange(ctx context.Context, state string) (uuid.UUID, error) {
	attemptID := uuid.New()
	var stored uuid.UUID
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='exchange_started',exchange_attempt_id=$2,
			    exchange_started_at=now(),consumed_at=now()
			WHERE state=$1 AND purpose='registration' AND request_state='ready'
			  AND created_at>=$3
			RETURNING exchange_attempt_id
		`, state, attemptID, time.Now().Add(-s.cfg.AuthRequestExpiry)).Scan(&stored)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAuthRequestState
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("begin registration OAuth exchange: %w", err)
	}
	return stored, nil
}

type RegistrationExchangeReconciliationStats struct {
	CleanupPending int64
	Ambiguous      int64
}

func (s *PostgresAuthStore) ReconcileStaleRegistrationExchanges(
	ctx context.Context,
	batch int,
) (RegistrationExchangeReconciliationStats, error) {
	if batch <= 0 {
		return RegistrationExchangeReconciliationStats{}, errors.New("registration exchange reconciliation batch must be positive")
	}
	cutoff := time.Now().Add(-s.cfg.AuthRequestExchangeExpiry)
	var stats RegistrationExchangeReconciliationStats
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT state
			FROM oauth_auth_requests
			WHERE purpose='registration' AND request_state='exchange_started'
			  AND exchange_started_at<$1
			ORDER BY exchange_started_at,state
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		`, cutoff, batch)
		if err != nil {
			return fmt.Errorf("select stale registration exchanges: %w", err)
		}
		var states []string
		for rows.Next() {
			var state string
			if err := rows.Scan(&state); err != nil {
				rows.Close()
				return err
			}
			states = append(states, state)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()

		for _, state := range states {
			credential, err := tx.Exec(ctx, `
				UPDATE oauth_unverified_credentials
				SET status='pending',eligible_at=LEAST(eligible_at,now()),updated_at=now()
				WHERE request_state=$1 AND status='held'
			`, state)
			if err != nil {
				return fmt.Errorf("release stale registration credential: %w", err)
			}
			finalState := AuthRequestExchangeAmbiguous
			if credential.RowsAffected() == 1 {
				finalState = AuthRequestCleanupPending
				stats.CleanupPending++
			} else {
				stats.Ambiguous++
			}
			command, err := tx.Exec(ctx, `
				UPDATE oauth_auth_requests
				SET request_state=$2,exchange_finished_at=now()
				WHERE state=$1 AND purpose='registration' AND request_state='exchange_started'
			`, state, finalState)
			if err != nil {
				return fmt.Errorf("finish stale registration exchange: %w", err)
			}
			if command.RowsAffected() != 1 {
				return ErrAuthRequestState
			}
		}
		return nil
	})
	if err != nil {
		return RegistrationExchangeReconciliationStats{}, err
	}
	return stats, nil
}

func (s *PostgresAuthStore) MarkExchangeAmbiguous(ctx context.Context, state string, attemptID uuid.UUID) error {
	return s.finishExchangeAttempt(ctx, state, attemptID, AuthRequestExchangeAmbiguous)
}

func (s *PostgresAuthStore) MarkExchangeFailed(ctx context.Context, state string, attemptID uuid.UUID) error {
	return s.finishExchangeAttempt(ctx, state, attemptID, AuthRequestExchangeFailed)
}

func (s *PostgresAuthStore) FinishRegistrationExchange(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
	finalState AuthRequestState,
) error {
	if finalState != AuthRequestExchangeFailed && finalState != AuthRequestExchangeAmbiguous {
		return ErrAuthRequestState
	}
	command, err := s.pool.Exec(ctx, `
		UPDATE oauth_auth_requests
		SET request_state=$3,exchange_finished_at=now()
		WHERE state=$1 AND purpose='registration' AND exchange_attempt_id=$2
		  AND request_state='exchange_started'
	`, state, attemptID, finalState)
	if err != nil {
		return fmt.Errorf("finish registration OAuth exchange attempt: %w", err)
	}
	if command.RowsAffected() != 1 {
		return ErrAuthRequestState
	}
	return nil
}

func (s *PostgresAuthStore) finishExchangeAttempt(
	ctx context.Context,
	state string,
	attemptID uuid.UUID,
	finalState AuthRequestState,
) error {
	if finalState != AuthRequestExchangeFailed && finalState != AuthRequestExchangeAmbiguous {
		return ErrAuthRequestState
	}
	var affected int64
	err := s.withAuthTransaction(ctx, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET request_state=$3,exchange_finished_at=now()
			WHERE state=$1 AND exchange_attempt_id=$2 AND request_state='exchange_started'
		`, state, attemptID, finalState)
		affected = command.RowsAffected()
		return err
	})
	if err != nil {
		return fmt.Errorf("finish OAuth exchange attempt: %w", err)
	}
	if affected != 1 {
		return ErrAuthRequestState
	}
	return nil
}

// AuthRequestSweepStats supplies the bounded queue signals needed by the
// context-owned sweeper loop without exposing request state, URI, owner, or
// device identifiers.
type AuthRequestSweepStats struct {
	Deleted                int64
	Pending                int64
	OldestPendingCreatedAt *time.Time
}

// SweepAuthRequests deletes at most batch safely disposable rows. Ready rows
// expire from creation time. Terminal rows are retained independently from
// their terminal transition. Login/deletion ambiguity remains durable;
// registration ambiguity is terminal after its separate capacity interval.
func (s *PostgresAuthStore) SweepAuthRequests(ctx context.Context, batch int) (AuthRequestSweepStats, error) {
	if batch <= 0 {
		return AuthRequestSweepStats{}, errors.New("auth request sweep batch must be positive")
	}
	readyCutoff := time.Now().Add(-s.cfg.AuthRequestExpiry)
	terminalCutoff := time.Now().Add(-s.cfg.AuthRequestTerminalRetention)
	var stats AuthRequestSweepStats
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM oauth_auth_request_reservations WHERE expires_at<=now()`); err != nil {
			return fmt.Errorf("delete expired auth request reservations: %w", err)
		}
		command, err := tx.Exec(ctx, `
			WITH candidates AS (
				SELECT state
				FROM oauth_auth_requests
				WHERE (request_state='ready' AND created_at<$1)
				   OR (request_state='exchange_failed' AND exchange_finished_at<$2)
				   OR (purpose='registration' AND request_state='exchange_ambiguous' AND exchange_finished_at<$2)
				   OR (request_state IN ('consumed','revoked') AND consumed_at<$2)
				ORDER BY created_at,state
				LIMIT $3
				FOR UPDATE SKIP LOCKED
			)
			DELETE FROM oauth_auth_requests AS requests
			USING candidates
			WHERE requests.state=candidates.state
		`, readyCutoff, terminalCutoff, batch)
		if err != nil {
			return fmt.Errorf("delete expired auth requests: %w", err)
		}
		stats.Deleted = command.RowsAffected()
		if err := tx.QueryRow(ctx, `
			SELECT count(*),min(created_at)
			FROM (
				SELECT created_at
				FROM oauth_auth_requests
				WHERE (request_state='ready' AND created_at>=$1)
				   OR request_state IN ('exchange_started','cleanup_pending')
				   OR (request_state='exchange_ambiguous'
				       AND (purpose<>'registration' OR exchange_finished_at>=$1))
				UNION ALL
				SELECT created_at
				FROM oauth_auth_request_reservations
				WHERE expires_at>now()
			) pending
		`, readyCutoff).Scan(&stats.Pending, &stats.OldestPendingCreatedAt); err != nil {
			return fmt.Errorf("inspect pending auth request backlog: %w", err)
		}
		return nil
	})
	if err != nil {
		return AuthRequestSweepStats{}, err
	}
	return stats, nil
}

func (s *PostgresAuthStore) withAuthTransaction(ctx context.Context, callback func(pgx.Tx) error) error {
	if s.owners != nil {
		return s.owners.WithAuthTransaction(ctx, callback)
	}
	return pgx.BeginFunc(ctx, s.pool, callback)
}
