package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

const (
	DefaultOAuthSessionOperationTimeout = 30 * time.Second
	MaximumOAuthSessionOperationTimeout = 90 * time.Second
)

var ErrOAuthSessionPersistenceIndeterminate = errors.New("OAuth session credential persistence is indeterminate")

type OAuthSessionCoordinatorOptions struct {
	App              *oauth.ClientApp
	Store            *PostgresAuthStore
	Owners           *ownerlifecycle.Store
	OperationTimeout time.Duration
}

// OAuthSessionCoordinator is the only ordinary parent-resume boundary. It
// holds the shared owner fence and exclusive parent fence across one complete
// authenticated PDS operation, and turns Indigo's error-blind persistence
// callback into a result the caller cannot accidentally ignore.
type OAuthSessionCoordinator struct {
	app              *oauth.ClientApp
	store            *PostgresAuthStore
	owners           *ownerlifecycle.Store
	operationTimeout time.Duration
}

type OAuthSessionOperation func(context.Context, *oauth.ClientSession) error

func NewOAuthSessionCoordinator(options OAuthSessionCoordinatorOptions) (*OAuthSessionCoordinator, error) {
	if options.App == nil || options.App.Client == nil || options.App.Config == nil ||
		options.Store == nil || options.Owners == nil {
		return nil, errors.New("OAuth session coordinator dependencies are unavailable")
	}
	timeout, err := normalizeOAuthOperationTimeout(
		options.OperationTimeout,
		DefaultOAuthSessionOperationTimeout,
		MaximumOAuthSessionOperationTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("OAuth session operation timeout: %w", err)
	}
	return &OAuthSessionCoordinator{
		app: options.App, store: options.Store, owners: options.Owners, operationTimeout: timeout,
	}, nil
}

func (coordinator *OAuthSessionCoordinator) WithActiveSession(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	operation OAuthSessionOperation,
) error {
	if coordinator == nil || owner == "" || sessionID == "" || operation == nil {
		return ErrOAuthSessionNotFound
	}
	operationCtx, cancel := oauthOperationContext(ctx, coordinator.operationTimeout)
	defer cancel()
	return coordinator.owners.WithActiveSessionAuth(
		operationCtx,
		owner,
		sessionID,
		func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
			return coordinator.withFencedSession(authCtx, authority, sessionID, operation)
		},
	)
}

// WithActiveEffectSession resumes one ordinary OAuth parent inside the same
// dedicated PostgreSQL connection that already holds every participant owner
// fence. This is the authenticated remote-effect entry point: callers must
// not compose WithActiveEffects with WithActiveSession, because that would
// recursively acquire the owner fence and violate the global lock order.
func (coordinator *OAuthSessionCoordinator) WithActiveEffectSession(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	owner syntax.DID,
	sessionID string,
	operation OAuthSessionOperation,
) error {
	if coordinator == nil || owner == "" || sessionID == "" || operation == nil {
		return ErrOAuthSessionNotFound
	}
	operationCtx, cancel := oauthOperationContext(ctx, coordinator.operationTimeout)
	defer cancel()
	return coordinator.owners.WithActiveEffectSessionAuth(
		operationCtx,
		expected,
		owner,
		sessionID,
		func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
			return coordinator.withFencedSession(authCtx, authority, sessionID, operation)
		},
	)
}

// WithDeletionSession resumes only the exact accepted deletion credential.
// Operation ID, session ID, credential generation, owner lifecycle generation,
// and auth epoch are revalidated under the owner and parent fences before
// every read or rotating-token persistence write.
func (coordinator *OAuthSessionCoordinator) WithDeletionSession(
	ctx context.Context,
	owner syntax.DID,
	deletionAuthority DeletionSessionAuthority,
	operation OAuthSessionOperation,
) error {
	if coordinator == nil || owner == "" || !deletionAuthority.valid() || operation == nil {
		return ErrOAuthSessionNotFound
	}
	operationCtx, cancel := oauthOperationContext(ctx, coordinator.operationTimeout)
	defer cancel()
	err := coordinator.owners.WithDeletionSessionAuth(
		operationCtx,
		owner,
		deletionAuthority.SessionID,
		func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
			return coordinator.withFencedDeletionSession(authCtx, authority, deletionAuthority, operation)
		},
	)
	if errors.Is(err, ownerlifecycle.ErrOwnerNotDeleting) ||
		errors.Is(err, ownerlifecycle.ErrTerminalOwner) ||
		errors.Is(err, ownerlifecycle.ErrInvalidOwner) {
		return ErrOAuthSessionNotFound
	}
	return err
}

func (coordinator *OAuthSessionCoordinator) withFencedSession(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	sessionID string,
	operation OAuthSessionOperation,
) error {
	for attempt := 0; attempt < 2; attempt++ {
		record, err := coordinator.loadFencedSession(ctx, authority, sessionID)
		if err != nil {
			return err
		}
		if err := coordinator.store.validateSessionEndpoints(ctx, record.Data); err != nil {
			if errors.Is(err, ErrOAuthSessionEndpointInvalid) {
				terminalErr := coordinator.markTerminalVersion(
					ctx, authority, sessionID, record.RowVersion,
				)
				if errors.Is(terminalErr, ErrSessionVersionChanged) && attempt == 0 {
					continue
				}
				return errors.Join(err, terminalErr)
			}
			return err
		}
		session, persistence, err := coordinator.clientSession(
			authority,
			record,
			func(callbackCtx context.Context, updated oauth.ClientSessionData, expectedVersion int64) (int64, error) {
				return coordinator.persistActiveSession(callbackCtx, authority, updated, expectedVersion)
			},
		)
		if err != nil {
			return err
		}
		operationErr := operation(ctx, session)
		version, persistErr := persistence.result()
		if persistErr != nil {
			session.PersistSessionCallback = nil
			revocationErr := session.RevokeSession(ctx)
			queueErr := coordinator.markTerminalVersion(ctx, authority, sessionID, version)
			return errors.Join(ErrOAuthSessionPersistenceIndeterminate, persistErr, revocationErr, queueErr)
		}
		if operationErr == nil {
			return nil
		}
		translated := TranslatePDSError(operationErr)
		if !errors.Is(translated, ErrPDSSessionExpired) {
			return operationErr
		}
		if err := coordinator.markTerminalVersion(ctx, authority, sessionID, version); err != nil {
			if errors.Is(err, ErrSessionVersionChanged) && attempt == 0 {
				continue
			}
			return errors.Join(translated, err)
		}
		return translated
	}
	return ErrSessionVersionChanged
}

func (coordinator *OAuthSessionCoordinator) withFencedDeletionSession(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	deletionAuthority DeletionSessionAuthority,
	operation OAuthSessionOperation,
) error {
	for attempt := 0; attempt < 2; attempt++ {
		record, err := coordinator.loadFencedDeletionSession(ctx, authority, deletionAuthority)
		if err != nil {
			return err
		}
		if err := coordinator.store.validateSessionEndpoints(ctx, record.Data); err != nil {
			if errors.Is(err, ErrOAuthSessionEndpointInvalid) {
				terminalErr := coordinator.markDeletionTerminalVersion(
					ctx, authority, deletionAuthority, record.RowVersion,
				)
				if errors.Is(terminalErr, ErrSessionVersionChanged) && attempt == 0 {
					continue
				}
				if terminalErr != nil {
					return errors.Join(err, terminalErr)
				}
				return errors.Join(err, ErrDeletionReauthenticationRequired)
			}
			return err
		}
		session, persistence, err := coordinator.clientSession(
			authority,
			record,
			func(callbackCtx context.Context, updated oauth.ClientSessionData, expectedVersion int64) (int64, error) {
				return coordinator.persistDeletionSession(
					callbackCtx, authority, deletionAuthority, updated, expectedVersion,
				)
			},
		)
		if err != nil {
			return err
		}
		operationErr := operation(ctx, session)
		version, persistErr := persistence.result()
		if persistErr != nil {
			session.PersistSessionCallback = nil
			revocationErr := session.RevokeSession(ctx)
			queueErr := coordinator.markDeletionTerminalVersion(ctx, authority, deletionAuthority, version)
			return errors.Join(
				ErrOAuthSessionPersistenceIndeterminate,
				ErrDeletionReauthenticationRequired,
				persistErr,
				revocationErr,
				queueErr,
			)
		}
		if operationErr == nil {
			return nil
		}
		translated := TranslatePDSError(operationErr)
		if !errors.Is(translated, ErrPDSSessionExpired) {
			return operationErr
		}
		if err := coordinator.markDeletionTerminalVersion(ctx, authority, deletionAuthority, version); err != nil {
			if errors.Is(err, ErrSessionVersionChanged) && attempt == 0 {
				continue
			}
			return errors.Join(translated, err)
		}
		return errors.Join(translated, ErrDeletionReauthenticationRequired)
	}
	return ErrSessionVersionChanged
}

func (coordinator *OAuthSessionCoordinator) loadFencedSession(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	sessionID string,
) (StoredOAuthSession, error) {
	var record StoredOAuthSession
	err := coordinator.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		var ownerState ownerlifecycle.State
		var authEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,auth_epoch FROM owner_lifecycles
			WHERE owner_did=$1 FOR SHARE
		`, authority.Owner).Scan(&ownerState, &authEpoch); err != nil {
			return err
		}
		if ownerState != ownerlifecycle.StateActive || authEpoch != authority.AuthEpoch {
			return ErrOAuthSessionNotFound
		}
		var err error
		record, err = coordinator.store.loadActiveSessionTx(ctx, tx, authority.Owner, sessionID)
		return err
	})
	return record, err
}

type sessionPersistenceState struct {
	mu      sync.Mutex
	version int64
	err     error
}

type sessionVersionPersister func(
	context.Context,
	oauth.ClientSessionData,
	int64,
) (int64, error)

func (state *sessionPersistenceState) result() (int64, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.version, state.err
}

func (coordinator *OAuthSessionCoordinator) clientSession(
	authority ownerlifecycle.Lifecycle,
	record StoredOAuthSession,
	persist sessionVersionPersister,
) (*oauth.ClientSession, *sessionPersistenceState, error) {
	if persist == nil {
		return nil, nil, errors.New("OAuth session persistence callback is unavailable")
	}
	privateKey, err := atcrypto.ParsePrivateMultibase(record.Data.DPoPPrivateKeyMultibase)
	if err != nil {
		return nil, nil, fmt.Errorf("parse OAuth session DPoP key: %w", err)
	}
	data := record.Data
	session := &oauth.ClientSession{
		Client: coordinator.app.Client, Config: coordinator.app.Config,
		Data: &data, DPoPPrivateKey: privateKey,
	}
	persistence := &sessionPersistenceState{version: record.RowVersion}
	session.PersistSessionCallback = func(callbackCtx context.Context, updated *oauth.ClientSessionData) {
		persistence.mu.Lock()
		defer persistence.mu.Unlock()
		if persistence.err != nil {
			return
		}
		if updated == nil || updated.AccountDID != authority.Owner || updated.SessionID != data.SessionID ||
			updated.HostURL != data.HostURL || updated.AuthServerURL != data.AuthServerURL ||
			updated.AuthServerTokenEndpoint != data.AuthServerTokenEndpoint ||
			updated.AuthServerRevocationEndpoint != data.AuthServerRevocationEndpoint {
			persistence.err = ErrOAuthSessionEndpointInvalid
			return
		}
		if err := coordinator.store.validateSessionEndpoints(callbackCtx, *updated); err != nil {
			persistence.err = err
			return
		}
		nextVersion, err := persist(callbackCtx, *updated, persistence.version)
		if err != nil {
			persistence.err = err
			return
		}
		persistence.version = nextVersion
	}
	return session, persistence, nil
}

func (coordinator *OAuthSessionCoordinator) persistActiveSession(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	updated oauth.ClientSessionData,
	expectedVersion int64,
) (int64, error) {
	var nextVersion int64
	err := coordinator.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		var ownerState ownerlifecycle.State
		var authEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,auth_epoch FROM owner_lifecycles
			WHERE owner_did=$1 FOR SHARE
		`, authority.Owner).Scan(&ownerState, &authEpoch); err != nil {
			return err
		}
		if ownerState != ownerlifecycle.StateActive || authEpoch != authority.AuthEpoch {
			return ErrSessionVersionChanged
		}
		var err error
		nextVersion, err = coordinator.store.saveSessionVersionTx(ctx, tx, updated, expectedVersion)
		return err
	})
	return nextVersion, err
}

func (coordinator *OAuthSessionCoordinator) loadFencedDeletionSession(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	deletionAuthority DeletionSessionAuthority,
) (StoredOAuthSession, error) {
	var record StoredOAuthSession
	err := coordinator.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		var err error
		record, err = coordinator.lockDeletionSessionTx(ctx, tx, authority, deletionAuthority, 0)
		return err
	})
	return record, err
}

func (coordinator *OAuthSessionCoordinator) persistDeletionSession(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	deletionAuthority DeletionSessionAuthority,
	updated oauth.ClientSessionData,
	expectedVersion int64,
) (int64, error) {
	var nextVersion int64
	err := coordinator.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		if _, err := coordinator.lockDeletionSessionTx(
			ctx, tx, authority, deletionAuthority, expectedVersion,
		); err != nil {
			return err
		}
		var err error
		nextVersion, err = coordinator.store.saveSessionVersionTx(ctx, tx, updated, expectedVersion)
		return err
	})
	return nextVersion, err
}

// lockDeletionSessionTx follows lifecycle -> operation -> parent row order.
// expectedVersion zero means load the current version; a positive value is a
// compare-and-set precondition for refresh or terminal classification.
func (coordinator *OAuthSessionCoordinator) lockDeletionSessionTx(
	ctx context.Context,
	tx pgx.Tx,
	authority ownerlifecycle.Lifecycle,
	deletionAuthority DeletionSessionAuthority,
	expectedVersion int64,
) (StoredOAuthSession, error) {
	var state ownerlifecycle.State
	var generation, authEpoch int64
	if err := tx.QueryRow(ctx, `
		SELECT state,generation,auth_epoch FROM owner_lifecycles
		WHERE owner_did=$1 FOR SHARE
	`, authority.Owner).Scan(&state, &generation, &authEpoch); err != nil {
		return StoredOAuthSession{}, err
	}
	if state != ownerlifecycle.StateDeleting || generation != authority.Generation ||
		authEpoch != authority.AuthEpoch || generation != deletionAuthority.OwnerGeneration+2 {
		return StoredOAuthSession{}, ErrOAuthSessionNotFound
	}
	var operationSession string
	var operationCredentialGeneration int64
	if err := tx.QueryRow(ctx, `
		SELECT deletion_oauth_session_id,deletion_credential_generation
		FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2 AND owner_generation=$3
		  AND state IN ('active','retrying') AND accepted_at IS NOT NULL
		  AND lease_token=$4 AND lease_expires_at>now()
		FOR SHARE
	`, deletionAuthority.OperationID, authority.Owner, deletionAuthority.OwnerGeneration,
		deletionAuthority.LeaseToken).Scan(
		&operationSession, &operationCredentialGeneration,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredOAuthSession{}, ErrOAuthSessionNotFound
		}
		return StoredOAuthSession{}, err
	}
	if operationSession != deletionAuthority.SessionID ||
		operationCredentialGeneration != deletionAuthority.CredentialGeneration {
		return StoredOAuthSession{}, ErrOAuthSessionNotFound
	}
	var record StoredOAuthSession
	var data []byte
	err := tx.QueryRow(ctx, `
		SELECT data,row_version,lifecycle_state,owner_generation,auth_epoch,absolute_expires_at
		FROM oauth_sessions
		WHERE account_did=$1 AND session_id=$2
		  AND lifecycle_state='deletion_only'
		  AND owner_generation=$3 AND auth_epoch=$4
		  AND deletion_operation_id=$5 AND deletion_credential_generation=$6
		  AND absolute_expires_at>now()
		FOR UPDATE
	`, authority.Owner, deletionAuthority.SessionID, generation-1, authEpoch,
		deletionAuthority.OperationID, deletionAuthority.CredentialGeneration).Scan(
		&data, &record.RowVersion, &record.LifecycleState,
		&record.OwnerGeneration, &record.AuthEpoch, &record.AbsoluteExpiry,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return StoredOAuthSession{}, ErrOAuthSessionNotFound
	}
	if err != nil {
		return StoredOAuthSession{}, err
	}
	if expectedVersion > 0 && record.RowVersion != expectedVersion {
		return StoredOAuthSession{}, ErrSessionVersionChanged
	}
	if err := json.Unmarshal(data, &record.Data); err != nil {
		return StoredOAuthSession{}, fmt.Errorf("unmarshal deletion OAuth session: %w", err)
	}
	return record, nil
}

func (coordinator *OAuthSessionCoordinator) markDeletionTerminalVersion(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	deletionAuthority DeletionSessionAuthority,
	expectedVersion int64,
) error {
	return coordinator.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		if _, err := coordinator.lockDeletionSessionTx(
			ctx, tx, authority, deletionAuthority, expectedVersion,
		); err != nil {
			return err
		}
		now := time.Now().UTC()
		operation, err := tx.Exec(ctx, `
			UPDATE account_deletion_operations
			SET state='reauth_required',deletion_oauth_session_id=NULL,
			    deletion_credential_generation=NULL,error_category='reauthentication',
			    next_attempt_at=NULL,lease_owner=NULL,lease_token=NULL,
			    lease_expires_at=NULL,updated_at=$3
			WHERE id=$1 AND owner_did=$2 AND state IN ('active','retrying')
		`, deletionAuthority.OperationID, authority.Owner, now)
		if err != nil {
			return err
		}
		if operation.RowsAffected() != 1 {
			return ErrSessionVersionChanged
		}
		parent, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
			    deletion_credential_generation=NULL,
			    revocation_requested_at=COALESCE(revocation_requested_at,$4),
			    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,$4),
			    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
			    row_version=row_version+1,updated_at=$4
			WHERE account_did=$1 AND session_id=$2 AND row_version=$3
		`, authority.Owner, deletionAuthority.SessionID, expectedVersion, now)
		if err != nil {
			return err
		}
		if parent.RowsAffected() != 1 {
			return ErrSessionVersionChanged
		}
		return nil
	})
}

func (coordinator *OAuthSessionCoordinator) markTerminalVersion(
	ctx context.Context,
	authority ownerlifecycle.Lifecycle,
	sessionID string,
	expectedVersion int64,
) error {
	return coordinator.owners.WithAuthTransaction(ctx, func(tx pgx.Tx) error {
		var ownerState ownerlifecycle.State
		var authEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT state,auth_epoch FROM owner_lifecycles
			WHERE owner_did=$1 FOR SHARE
		`, authority.Owner).Scan(&ownerState, &authEpoch); err != nil {
			return err
		}
		if ownerState != ownerlifecycle.StateActive || authEpoch != authority.AuthEpoch {
			return ErrSessionVersionChanged
		}
		var currentVersion int64
		if err := tx.QueryRow(ctx, `
			SELECT row_version FROM oauth_sessions
			WHERE account_did=$1 AND session_id=$2 AND lifecycle_state='active'
			FOR UPDATE
		`, authority.Owner, sessionID).Scan(&currentVersion); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSessionVersionChanged
			}
			return err
		}
		if currentVersion != expectedVersion {
			return ErrSessionVersionChanged
		}
		rows, err := tx.Query(ctx, `
			SELECT token_hash FROM craftsky_sessions
			WHERE account_did=$1 AND oauth_session_id=$2
			ORDER BY token_hash FOR UPDATE
		`, authority.Owner, sessionID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var tokenHash []byte
			if err := rows.Scan(&tokenHash); err != nil {
				rows.Close()
				return err
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='revocation_pending',revocation_requested_at=now(),
			    cleanup_next_attempt_at=now(),row_version=row_version+1,updated_at=now()
			WHERE account_did=$1 AND session_id=$2 AND row_version=$3
		`, authority.Owner, sessionID, expectedVersion); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,now())
			WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
		`, authority.Owner, sessionID)
		return err
	})
}
