package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maximumAuthCleanupBatchSize = 1000

// OAuthCredentialRevoker is the narrow remote effect used by the durable
// revocation processor. Implementations must be idempotent: a process may
// crash after the authorization server accepts revocation but before the
// lease-token CAS deletes the local row.
type OAuthCredentialRevoker interface {
	RevokeSession(context.Context, oauth.ClientSessionData) error
}

// IndigoOAuthCredentialRevoker reconstructs only an already-revoked local
// parent. It validates every stored destination again and uses the OAuth app's
// hardened client; it never calls the process-default HTTP client.
type IndigoOAuthCredentialRevoker struct {
	app   *oauth.ClientApp
	store *PostgresAuthStore
}

func NewIndigoOAuthCredentialRevoker(
	app *oauth.ClientApp,
	store *PostgresAuthStore,
) (*IndigoOAuthCredentialRevoker, error) {
	if app == nil || app.Client == nil || app.Config == nil || store == nil {
		return nil, errors.New("credential revoker dependencies are required")
	}
	return &IndigoOAuthCredentialRevoker{app: app, store: store}, nil
}

func (revoker *IndigoOAuthCredentialRevoker) RevokeSession(
	ctx context.Context,
	data oauth.ClientSessionData,
) error {
	if revoker == nil || revoker.app == nil || revoker.store == nil {
		return errors.New("credential revoker is unavailable")
	}
	if err := revoker.store.validateSessionEndpoints(ctx, data); err != nil {
		return err
	}
	privateKey, err := atcrypto.ParsePrivateMultibase(data.DPoPPrivateKeyMultibase)
	if err != nil {
		return fmt.Errorf("parse revocation DPoP key: %w", err)
	}
	copyOfData := data
	session := oauth.ClientSession{
		Client: revoker.app.Client, Config: revoker.app.Config,
		Data: &copyOfData, DPoPPrivateKey: privateKey,
	}
	return session.RevokeSession(ctx)
}

type OAuthRevocationProcessorOptions struct {
	Pool                   *pgxpool.Pool
	Revoker                OAuthCredentialRevoker
	Observer               OAuthCleanupObserver
	Now                    func() time.Time
	NewLeaseToken          func() uuid.UUID
	BatchSize              int
	LeaseDuration          time.Duration
	OperationTimeout       time.Duration
	MaxAttempts            int
	BaseBackoff            time.Duration
	MaxBackoff             time.Duration
	MaxCredentialRetention time.Duration
}

type OAuthCleanupObserver interface {
	ObserveOAuthCleanup(ctx context.Context, credentialKind, result, reason string, duration time.Duration, attempt int)
}

type OAuthRevocationProcessor struct {
	pool                   *pgxpool.Pool
	revoker                OAuthCredentialRevoker
	observer               OAuthCleanupObserver
	now                    func() time.Time
	newLeaseToken          func() uuid.UUID
	batchSize              int
	leaseDuration          time.Duration
	operationTimeout       time.Duration
	maxAttempts            int
	baseBackoff            time.Duration
	maxBackoff             time.Duration
	maxCredentialRetention time.Duration
}

type oauthRevocationClaim struct {
	Owner       syntax.DID
	SessionID   string
	Data        []byte
	RowVersion  int64
	Attempts    int
	RequestedAt time.Time
	LeaseToken  uuid.UUID
}

func NewOAuthRevocationProcessor(
	options OAuthRevocationProcessorOptions,
) (*OAuthRevocationProcessor, error) {
	if options.Pool == nil || options.Revoker == nil || options.BatchSize < 1 ||
		options.BatchSize > maximumAuthCleanupBatchSize || options.LeaseDuration <= 0 ||
		options.OperationTimeout <= 0 || options.OperationTimeout >= options.LeaseDuration ||
		options.MaxAttempts < 1 || options.BaseBackoff <= 0 ||
		options.MaxBackoff < options.BaseBackoff || options.MaxCredentialRetention <= 0 {
		return nil, errors.New("invalid OAuth revocation processor options")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewLeaseToken == nil {
		options.NewLeaseToken = uuid.New
	}
	return &OAuthRevocationProcessor{
		pool: options.Pool, revoker: options.Revoker, observer: options.Observer, now: options.Now,
		newLeaseToken: options.NewLeaseToken, batchSize: options.BatchSize,
		leaseDuration: options.LeaseDuration, operationTimeout: options.OperationTimeout,
		maxAttempts: options.MaxAttempts, baseBackoff: options.BaseBackoff,
		maxBackoff: options.MaxBackoff, maxCredentialRetention: options.MaxCredentialRetention,
	}, nil
}

func (processor *OAuthRevocationProcessor) ProcessBatch(ctx context.Context) (int, error) {
	if processor == nil || processor.pool == nil || processor.revoker == nil {
		return 0, errors.New("OAuth revocation processor is unavailable")
	}
	now := processor.now().UTC()
	claims, err := processor.claim(ctx, now)
	if err != nil {
		return 0, err
	}
	var processingErrors []error
	for _, claim := range claims {
		if err := processor.processClaim(ctx, claim, now); err != nil {
			processingErrors = append(processingErrors, err)
		}
	}
	callbackCount, callbackErr := processor.processCallbackCredentials(ctx, now)
	if callbackErr != nil {
		processingErrors = append(processingErrors, callbackErr)
	}
	return len(claims) + callbackCount, errors.Join(processingErrors...)
}

type callbackCredentialClaim struct {
	State      string
	Data       []byte
	Attempts   int
	CreatedAt  time.Time
	LeaseToken uuid.UUID
}

func (processor *OAuthRevocationProcessor) processCallbackCredentials(
	ctx context.Context,
	now time.Time,
) (int, error) {
	claims, err := processor.claimCallbackCredentials(ctx, now)
	if err != nil {
		return 0, err
	}
	var processingErrors []error
	for _, claim := range claims {
		started := time.Now()
		var data oauth.ClientSessionData
		if err := json.Unmarshal(claim.Data, &data); err != nil {
			applied, finishErr := processor.finishCallbackCredential(
				ctx, claim, now, false, "invalid_credential",
			)
			exhausted := claim.Attempts+1 >= processor.maxAttempts ||
				!claim.CreatedAt.Add(processor.maxCredentialRetention).After(now)
			reason := "invalid_credential"
			if finishErr == nil && exhausted {
				if !claim.CreatedAt.Add(processor.maxCredentialRetention).After(now) {
					reason = "retention_expired"
				} else {
					reason = "attempts_exhausted"
				}
			}
			if applied || finishErr != nil {
				processor.observeCleanup(ctx, "callback", cleanupResult(finishErr, false, exhausted), cleanupReason(finishErr, reason), started, claim.Attempts+1)
			}
			processingErrors = append(processingErrors, finishErr)
			continue
		}
		operationCtx, cancel := context.WithTimeout(ctx, processor.operationTimeout)
		revokeErr := processor.revoker.RevokeSession(operationCtx, data)
		cancel()
		revokeReason := "none"
		if revokeErr != nil {
			revokeReason = cleanupFailureCategory(revokeErr)
		}
		applied, finishErr := processor.finishCallbackCredential(
			ctx, claim, now, revokeErr == nil, revokeReason,
		)
		exhausted := claim.Attempts+1 >= processor.maxAttempts ||
			!claim.CreatedAt.Add(processor.maxCredentialRetention).After(now)
		result := cleanupResult(finishErr, revokeErr == nil, exhausted)
		reason := cleanupReason(finishErr, revokeReason)
		if finishErr == nil && revokeErr != nil && exhausted {
			if !claim.CreatedAt.Add(processor.maxCredentialRetention).After(now) {
				reason = "retention_expired"
			} else {
				reason = "attempts_exhausted"
			}
		}
		if applied || finishErr != nil {
			processor.observeCleanup(ctx, "callback", result, reason, started, claim.Attempts+1)
		}
		processingErrors = append(processingErrors, finishErr)
	}
	return len(claims), errors.Join(processingErrors...)
}

func (processor *OAuthRevocationProcessor) claimCallbackCredentials(
	ctx context.Context,
	now time.Time,
) ([]callbackCredentialClaim, error) {
	tx, err := processor.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT request.state,credential.data,credential.cleanup_attempts,credential.created_at
		FROM oauth_auth_requests request
		JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
		WHERE request.purpose IN ('login','registration')
		  AND request.request_state IN ('exchange_started','cleanup_pending')
		  AND credential.status IN ('held','pending')
		  AND credential.eligible_at<=$1
		  AND COALESCE(credential.cleanup_next_attempt_at,credential.eligible_at)<=$1
		  AND (credential.cleanup_lease_token IS NULL OR credential.cleanup_lease_expires_at<=$1)
		ORDER BY credential.eligible_at,request.state
		LIMIT $2
		FOR UPDATE OF request,credential SKIP LOCKED
	`, now, processor.batchSize)
	if err != nil {
		return nil, fmt.Errorf("claim callback credentials: %w", err)
	}
	var claims []callbackCredentialClaim
	for rows.Next() {
		var claim callbackCredentialClaim
		if err := rows.Scan(&claim.State, &claim.Data, &claim.Attempts, &claim.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		claim.LeaseToken = processor.newLeaseToken()
		if claim.LeaseToken == uuid.Nil {
			rows.Close()
			return nil, errors.New("callback credential lease token is invalid")
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for _, claim := range claims {
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_auth_requests
			SET request_state='cleanup_pending',exchange_finished_at=COALESCE(exchange_finished_at,$2)
			WHERE state=$1 AND purpose IN ('login','registration')
			  AND request_state IN ('exchange_started','cleanup_pending')
		`, claim.State, now); err != nil {
			return nil, err
		}
		command, err := tx.Exec(ctx, `
			UPDATE oauth_unverified_credentials
			SET status='pending',cleanup_lease_token=$2,cleanup_lease_expires_at=$3,updated_at=$4
			WHERE request_state=$1 AND status IN ('held','pending')
		`, claim.State, claim.LeaseToken, now.Add(processor.leaseDuration), now)
		if err != nil {
			return nil, err
		}
		if command.RowsAffected() != 1 {
			return nil, ErrAuthRequestState
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claims, nil
}

func (processor *OAuthRevocationProcessor) finishCallbackCredential(
	ctx context.Context,
	claim callbackCredentialClaim,
	now time.Time,
	revoked bool,
	category string,
) (bool, error) {
	finalCtx, cancel := finalizationContext(ctx, processor.operationTimeout)
	defer cancel()
	applied := false
	err := pgx.BeginFunc(finalCtx, processor.pool, func(tx pgx.Tx) error {
		var requestState string
		if err := tx.QueryRow(finalCtx, `
				SELECT request_state FROM oauth_auth_requests WHERE state=$1 FOR UPDATE
			`, claim.State).Scan(&requestState); err != nil {
			return err
		}
		if requestState != string(AuthRequestCleanupPending) {
			return nil
		}
		var leaseToken *uuid.UUID
		if err := tx.QueryRow(finalCtx, `
			SELECT cleanup_lease_token FROM oauth_unverified_credentials
			WHERE request_state=$1 FOR UPDATE
		`, claim.State).Scan(&leaseToken); err != nil {
			return err
		}
		if leaseToken == nil || *leaseToken != claim.LeaseToken {
			return nil
		}
		exhausted := claim.Attempts+1 >= processor.maxAttempts ||
			!claim.CreatedAt.Add(processor.maxCredentialRetention).After(now)
		if revoked || exhausted {
			if _, err := tx.Exec(finalCtx, `
				DELETE FROM oauth_unverified_credentials
				WHERE request_state=$1 AND cleanup_lease_token=$2
			`, claim.State, claim.LeaseToken); err != nil {
				return err
			}
			finalState := AuthRequestRevoked
			if !revoked {
				finalState = AuthRequestExchangeAmbiguous
			}
			command, err := tx.Exec(finalCtx, `
				UPDATE oauth_auth_requests
				SET request_state=$2,exchange_finished_at=COALESCE(exchange_finished_at,$3)
				WHERE state=$1 AND request_state='cleanup_pending'
			`, claim.State, finalState, now)
			if err == nil {
				applied = command.RowsAffected() == 1
			}
			return err
		}
		command, err := tx.Exec(finalCtx, `
			UPDATE oauth_unverified_credentials
			SET cleanup_attempts=$2,cleanup_next_attempt_at=$3,
			    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
			    cleanup_last_category=$4,updated_at=$5
			WHERE request_state=$1 AND cleanup_lease_token=$6
		`, claim.State, claim.Attempts+1,
			now.Add(cleanupBackoff(processor.baseBackoff, processor.maxBackoff, claim.Attempts+1)),
			category, now, claim.LeaseToken)
		if err == nil {
			applied = command.RowsAffected() == 1
		}
		return err
	})
	return applied, err
}

func (processor *OAuthRevocationProcessor) claim(
	ctx context.Context,
	now time.Time,
) ([]oauthRevocationClaim, error) {
	tx, err := processor.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT account_did,session_id,data,row_version,cleanup_attempts,revocation_requested_at
		FROM oauth_sessions
		WHERE lifecycle_state='revocation_pending'
		  AND (
		    cleanup_attempts >= $3
		    OR revocation_requested_at <= $4
		    OR COALESCE(cleanup_next_attempt_at,revocation_requested_at) <= $1
		  )
		  AND (cleanup_lease_token IS NULL OR cleanup_lease_expires_at <= $1)
		ORDER BY COALESCE(cleanup_next_attempt_at,revocation_requested_at),account_did,session_id
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, now, processor.batchSize, processor.maxAttempts, now.Add(-processor.maxCredentialRetention))
	if err != nil {
		return nil, fmt.Errorf("claim OAuth revocations: %w", err)
	}
	claims := make([]oauthRevocationClaim, 0, processor.batchSize)
	for rows.Next() {
		var claim oauthRevocationClaim
		if err := rows.Scan(
			&claim.Owner, &claim.SessionID, &claim.Data, &claim.RowVersion,
			&claim.Attempts, &claim.RequestedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}
		claim.LeaseToken = processor.newLeaseToken()
		if claim.LeaseToken == uuid.Nil {
			rows.Close()
			return nil, errors.New("OAuth revocation lease token is invalid")
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for _, claim := range claims {
		command, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET cleanup_lease_token=$4,cleanup_lease_expires_at=$5,updated_at=$6
			WHERE account_did=$1 AND session_id=$2 AND row_version=$3
			  AND lifecycle_state='revocation_pending'
		`, claim.Owner, claim.SessionID, claim.RowVersion, claim.LeaseToken,
			now.Add(processor.leaseDuration), now)
		if err != nil {
			return nil, fmt.Errorf("lease OAuth revocation: %w", err)
		}
		if command.RowsAffected() != 1 {
			return nil, errors.New("OAuth revocation changed while claiming")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claims, nil
}

func (processor *OAuthRevocationProcessor) processClaim(
	ctx context.Context,
	claim oauthRevocationClaim,
	claimedAt time.Time,
) error {
	started := time.Now()
	retentionExpired := !claim.RequestedAt.Add(processor.maxCredentialRetention).After(claimedAt)
	if retentionExpired || claim.Attempts >= processor.maxAttempts {
		finalCtx, cancel := finalizationContext(ctx, processor.operationTimeout)
		defer cancel()
		applied, err := processor.deleteClaim(finalCtx, claim)
		reason := "attempts_exhausted"
		if retentionExpired {
			reason = "retention_expired"
		}
		if applied || err != nil {
			processor.observeCleanup(ctx, "parent", cleanupResult(err, false, true), cleanupReason(err, reason), started, claim.Attempts)
		}
		return err
	}
	var data oauth.ClientSessionData
	if err := json.Unmarshal(claim.Data, &data); err != nil {
		applied, retryErr := processor.retryClaim(ctx, claim, claimedAt, "invalid_credential")
		exhausted := claim.Attempts+1 >= processor.maxAttempts
		reason := "invalid_credential"
		if retryErr == nil && exhausted {
			reason = "attempts_exhausted"
		}
		if applied || retryErr != nil {
			processor.observeCleanup(ctx, "parent", cleanupResult(retryErr, false, exhausted), cleanupReason(retryErr, reason), started, claim.Attempts+1)
		}
		return retryErr
	}
	operationCtx, cancel := context.WithTimeout(ctx, processor.operationTimeout)
	err := processor.revoker.RevokeSession(operationCtx, data)
	cancel()
	if err == nil {
		finalCtx, finalCancel := finalizationContext(ctx, processor.operationTimeout)
		defer finalCancel()
		applied, deleteErr := processor.deleteClaim(finalCtx, claim)
		if applied || deleteErr != nil {
			processor.observeCleanup(ctx, "parent", cleanupResult(deleteErr, true, false), cleanupReason(deleteErr, "none"), started, claim.Attempts+1)
		}
		return deleteErr
	}
	reason := cleanupFailureCategory(err)
	applied, retryErr := processor.retryClaim(ctx, claim, claimedAt, reason)
	exhausted := claim.Attempts+1 >= processor.maxAttempts
	if retryErr == nil && exhausted {
		reason = "attempts_exhausted"
	}
	if applied || retryErr != nil {
		processor.observeCleanup(ctx, "parent", cleanupResult(retryErr, false, exhausted), cleanupReason(retryErr, reason), started, claim.Attempts+1)
	}
	return retryErr
}

func (processor *OAuthRevocationProcessor) observeCleanup(
	ctx context.Context,
	credentialKind string,
	result string,
	reason string,
	started time.Time,
	attempt int,
) {
	if processor.observer != nil {
		processor.observer.ObserveOAuthCleanup(ctx, credentialKind, result, reason, time.Since(started), attempt)
	}
}

func cleanupResult(err error, success, exhausted bool) string {
	if err != nil {
		return "error"
	}
	if success {
		return "success"
	}
	if exhausted {
		return "discarded"
	}
	return "retry"
}

func cleanupReason(err error, reason string) string {
	if err != nil {
		return "store_failed"
	}
	return reason
}

func (processor *OAuthRevocationProcessor) retryClaim(
	ctx context.Context,
	claim oauthRevocationClaim,
	now time.Time,
	category string,
) (bool, error) {
	nextAttempts := claim.Attempts + 1
	finalCtx, cancel := finalizationContext(ctx, processor.operationTimeout)
	defer cancel()
	if nextAttempts >= processor.maxAttempts ||
		!claim.RequestedAt.Add(processor.maxCredentialRetention).After(now) {
		return processor.deleteClaim(finalCtx, claim)
	}
	command, err := processor.pool.Exec(finalCtx, `
		UPDATE oauth_sessions
		SET cleanup_attempts=$5,cleanup_next_attempt_at=$6,
		    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
		    cleanup_last_category=$7,updated_at=$8
		WHERE account_did=$1 AND session_id=$2 AND row_version=$3
		  AND lifecycle_state='revocation_pending' AND cleanup_lease_token=$4
	`, claim.Owner, claim.SessionID, claim.RowVersion, claim.LeaseToken,
		nextAttempts, now.Add(cleanupBackoff(processor.baseBackoff, processor.maxBackoff, nextAttempts)),
		category, now)
	if err != nil {
		return false, fmt.Errorf("schedule OAuth revocation retry: %w", err)
	}
	return command.RowsAffected() == 1, nil
}

func (processor *OAuthRevocationProcessor) deleteClaim(
	ctx context.Context,
	claim oauthRevocationClaim,
) (bool, error) {
	command, err := processor.pool.Exec(ctx, `
		DELETE FROM oauth_sessions
		WHERE account_did=$1 AND session_id=$2 AND row_version=$3
		  AND lifecycle_state='revocation_pending' AND cleanup_lease_token=$4
	`, claim.Owner, claim.SessionID, claim.RowVersion, claim.LeaseToken)
	if err != nil {
		return false, fmt.Errorf("delete revoked OAuth session: %w", err)
	}
	return command.RowsAffected() == 1, nil
}

type AuxiliaryCleanupProcessorOptions struct {
	Pool             *pgxpool.Pool
	Cleaner          FencedNotificationSubscriptionCleaner
	Now              func() time.Time
	NewLeaseToken    func() uuid.UUID
	BatchSize        int
	LeaseDuration    time.Duration
	OperationTimeout time.Duration
	MaxAttempts      int
	BaseBackoff      time.Duration
	MaxBackoff       time.Duration
}

type AuxiliaryCleanupProcessor struct {
	pool             *pgxpool.Pool
	cleaner          FencedNotificationSubscriptionCleaner
	now              func() time.Time
	newLeaseToken    func() uuid.UUID
	batchSize        int
	leaseDuration    time.Duration
	operationTimeout time.Duration
	maxAttempts      int
	baseBackoff      time.Duration
	maxBackoff       time.Duration
}

type auxiliaryCleanupClaim struct {
	ID             uuid.UUID
	Owner          syntax.DID
	AuthEpoch      int64
	Kind           string
	InstallationID *string
	Attempts       int
	CreatedAt      time.Time
	LeaseToken     uuid.UUID
}

// FencedNotificationSubscriptionCleaner applies a durable cleanup job only to
// subscriptions last activated no later than the job's creation time. A
// delayed or reclaimed job must not deactivate a newer registration.
type FencedNotificationSubscriptionCleaner interface {
	DeactivateForInstallationBefore(context.Context, string, string, time.Time) error
	DeactivateForAccountBefore(context.Context, string, time.Time) error
}

func NewAuxiliaryCleanupProcessor(
	options AuxiliaryCleanupProcessorOptions,
) (*AuxiliaryCleanupProcessor, error) {
	if options.Pool == nil || options.Cleaner == nil || options.BatchSize < 1 ||
		options.BatchSize > maximumAuthCleanupBatchSize || options.LeaseDuration <= 0 ||
		options.OperationTimeout <= 0 || options.OperationTimeout >= options.LeaseDuration ||
		options.MaxAttempts < 1 || options.BaseBackoff <= 0 || options.MaxBackoff < options.BaseBackoff {
		return nil, errors.New("invalid auxiliary cleanup processor options")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.NewLeaseToken == nil {
		options.NewLeaseToken = uuid.New
	}
	return &AuxiliaryCleanupProcessor{
		pool: options.Pool, cleaner: options.Cleaner, now: options.Now,
		newLeaseToken: options.NewLeaseToken, batchSize: options.BatchSize,
		leaseDuration: options.LeaseDuration, operationTimeout: options.OperationTimeout,
		maxAttempts: options.MaxAttempts, baseBackoff: options.BaseBackoff, maxBackoff: options.MaxBackoff,
	}, nil
}

func (processor *AuxiliaryCleanupProcessor) ProcessBatch(ctx context.Context) (int, error) {
	if processor == nil || processor.pool == nil || processor.cleaner == nil {
		return 0, errors.New("auxiliary cleanup processor is unavailable")
	}
	now := processor.now().UTC()
	claims, err := processor.claim(ctx, now)
	if err != nil {
		return 0, err
	}
	var processingErrors []error
	for _, claim := range claims {
		if err := processor.processClaim(ctx, claim, now); err != nil {
			processingErrors = append(processingErrors, err)
		}
	}
	return len(claims), errors.Join(processingErrors...)
}

func (processor *AuxiliaryCleanupProcessor) claim(
	ctx context.Context,
	now time.Time,
) ([]auxiliaryCleanupClaim, error) {
	tx, err := processor.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT id,owner_did,auth_epoch,kind,installation_id,attempt_count,created_at
		FROM auth_auxiliary_cleanup_jobs
		WHERE (
		    (state='pending' AND next_attempt_at <= $1)
		    OR (state='leased' AND lease_expires_at <= $1)
		  )
		ORDER BY next_attempt_at,id
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`, now, processor.batchSize)
	if err != nil {
		return nil, fmt.Errorf("claim auxiliary cleanup: %w", err)
	}
	claims := make([]auxiliaryCleanupClaim, 0, processor.batchSize)
	for rows.Next() {
		var claim auxiliaryCleanupClaim
		if err := rows.Scan(
			&claim.ID, &claim.Owner, &claim.AuthEpoch, &claim.Kind,
			&claim.InstallationID, &claim.Attempts, &claim.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, err
		}
		claim.LeaseToken = processor.newLeaseToken()
		if claim.LeaseToken == uuid.Nil {
			rows.Close()
			return nil, errors.New("auxiliary cleanup lease token is invalid")
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for _, claim := range claims {
		command, err := tx.Exec(ctx, `
			UPDATE auth_auxiliary_cleanup_jobs
			SET state='leased',lease_token=$2,lease_expires_at=$3,updated_at=$4
			WHERE id=$1 AND state IN ('pending','leased')
		`, claim.ID, claim.LeaseToken, now.Add(processor.leaseDuration), now)
		if err != nil {
			return nil, fmt.Errorf("lease auxiliary cleanup: %w", err)
		}
		if command.RowsAffected() != 1 {
			return nil, errors.New("auxiliary cleanup changed while claiming")
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claims, nil
}

func (processor *AuxiliaryCleanupProcessor) processClaim(
	ctx context.Context,
	claim auxiliaryCleanupClaim,
	claimedAt time.Time,
) error {
	if claim.Attempts >= processor.maxAttempts {
		return processor.finishFailure(ctx, claim, claimedAt, "attempts_exhausted")
	}
	operationCtx, cancel := context.WithTimeout(ctx, processor.operationTimeout)
	var err error
	switch claim.Kind {
	case "installation_push":
		if claim.InstallationID == nil || *claim.InstallationID == "" {
			err = errors.New("installation cleanup target is invalid")
		} else {
			err = processor.cleaner.DeactivateForInstallationBefore(
				operationCtx, claim.Owner.String(), *claim.InstallationID, claim.CreatedAt,
			)
		}
	case "account_push":
		err = processor.cleaner.DeactivateForAccountBefore(
			operationCtx, claim.Owner.String(), claim.CreatedAt,
		)
	default:
		err = errors.New("auxiliary cleanup kind is invalid")
	}
	cancel()
	if err != nil {
		return processor.finishFailure(ctx, claim, claimedAt, cleanupFailureCategory(err))
	}
	finalCtx, finalCancel := finalizationContext(ctx, processor.operationTimeout)
	defer finalCancel()
	command, err := processor.pool.Exec(finalCtx, `
		UPDATE auth_auxiliary_cleanup_jobs
		SET state='complete',lease_token=NULL,lease_expires_at=NULL,
		    last_category=NULL,updated_at=$3
		WHERE id=$1 AND state='leased' AND lease_token=$2
	`, claim.ID, claim.LeaseToken, claimedAt)
	if err != nil {
		return fmt.Errorf("complete auxiliary cleanup: %w", err)
	}
	_ = command.RowsAffected()
	return nil
}

func (processor *AuxiliaryCleanupProcessor) finishFailure(
	ctx context.Context,
	claim auxiliaryCleanupClaim,
	now time.Time,
	category string,
) error {
	nextAttempts := claim.Attempts + 1
	state := "pending"
	if nextAttempts >= processor.maxAttempts {
		state = "exhausted"
	}
	nextAt := now.Add(cleanupBackoff(processor.baseBackoff, processor.maxBackoff, nextAttempts))
	finalCtx, cancel := finalizationContext(ctx, processor.operationTimeout)
	defer cancel()
	command, err := processor.pool.Exec(finalCtx, `
		UPDATE auth_auxiliary_cleanup_jobs
		SET state=$3,attempt_count=$4,next_attempt_at=$5,
		    lease_token=NULL,lease_expires_at=NULL,last_category=$6,updated_at=$7
		WHERE id=$1 AND state='leased' AND lease_token=$2
	`, claim.ID, claim.LeaseToken, state, nextAttempts, nextAt, category, now)
	if err != nil {
		return fmt.Errorf("record auxiliary cleanup failure: %w", err)
	}
	_ = command.RowsAffected()
	return nil
}

func cleanupBackoff(base, maximum time.Duration, attempt int) time.Duration {
	delay := base
	for current := 1; current < attempt && delay < maximum; current++ {
		if delay > maximum/2 {
			return maximum
		}
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
}

func cleanupFailureCategory(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, ErrOAuthSessionEndpointInvalid):
		return "invalid_credential"
	default:
		return "dependency_unavailable"
	}
}

func finalizationContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(parent), timeout)
}
