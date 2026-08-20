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

type OAuthRevocationProcessor struct {
	pool                   *pgxpool.Pool
	revoker                OAuthCredentialRevoker
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
		pool: options.Pool, revoker: options.Revoker, now: options.Now,
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
	return len(claims), errors.Join(processingErrors...)
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
	retentionExpired := !claim.RequestedAt.Add(processor.maxCredentialRetention).After(claimedAt)
	if retentionExpired || claim.Attempts >= processor.maxAttempts {
		finalCtx, cancel := finalizationContext(ctx, processor.operationTimeout)
		defer cancel()
		_, err := processor.deleteClaim(finalCtx, claim)
		return err
	}
	var data oauth.ClientSessionData
	if err := json.Unmarshal(claim.Data, &data); err != nil {
		return processor.retryClaim(ctx, claim, claimedAt, "invalid_credential")
	}
	operationCtx, cancel := context.WithTimeout(ctx, processor.operationTimeout)
	err := processor.revoker.RevokeSession(operationCtx, data)
	cancel()
	if err == nil {
		finalCtx, finalCancel := finalizationContext(ctx, processor.operationTimeout)
		defer finalCancel()
		_, deleteErr := processor.deleteClaim(finalCtx, claim)
		return deleteErr
	}
	return processor.retryClaim(ctx, claim, claimedAt, cleanupFailureCategory(err))
}

func (processor *OAuthRevocationProcessor) retryClaim(
	ctx context.Context,
	claim oauthRevocationClaim,
	now time.Time,
	category string,
) error {
	nextAttempts := claim.Attempts + 1
	finalCtx, cancel := finalizationContext(ctx, processor.operationTimeout)
	defer cancel()
	if nextAttempts >= processor.maxAttempts ||
		!claim.RequestedAt.Add(processor.maxCredentialRetention).After(now) {
		_, err := processor.deleteClaim(finalCtx, claim)
		return err
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
		return fmt.Errorf("schedule OAuth revocation retry: %w", err)
	}
	_ = command.RowsAffected() // Zero means a newer lease/state fenced this worker.
	return nil
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
