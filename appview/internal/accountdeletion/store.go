package accountdeletion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrOperationNotFound = errors.New("account deletion operation not found")

type Store struct {
	pool      *pgxpool.Pool
	now       func() time.Time
	telemetry *DeletionTelemetry
}

func (store *Store) SetTelemetry(telemetry *DeletionTelemetry) {
	if store != nil {
		store.telemetry = telemetry
	}
}

func NewStore(pool *pgxpool.Pool, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{pool: pool, now: now}
}

func HashSecret(value string) []byte {
	hash := sha256.Sum256([]byte(value))
	return append([]byte(nil), hash[:]...)
}

type IntentRecord struct {
	JobID                  uuid.UUID
	Owner                  syntax.DID
	DeviceID               string
	StatusCapabilityHash   []byte
	ConfirmationHandleHash []byte
	ExpiresAt              time.Time
}

type AcceptanceRequest struct {
	JobID              uuid.UUID
	Owner              syntax.DID
	StatusCapability   string
	ReauthProof        string
	ConfirmationHandle string
}

type Operation struct {
	JobID                  uuid.UUID
	Owner                  syntax.DID
	Status                 Status
	Phase                  Phase
	AcceptedAt             time.Time
	DeletionOAuthSessionID string
}

func (store *Store) CreateIntent(ctx context.Context, intent IntentRecord) error {
	if store == nil || store.pool == nil || intent.JobID == uuid.Nil || intent.Owner == "" || intent.DeviceID == "" ||
		len(intent.StatusCapabilityHash) == 0 || len(intent.ConfirmationHandleHash) == 0 {
		return errors.New("invalid account deletion intent")
	}
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var (
			expiredJobID        uuid.UUID
			expiredOAuthSession *string
		)
		err := tx.QueryRow(ctx, `
			SELECT id,reauth_oauth_session_id
			FROM account_deletion_operations
			WHERE owner_did=$1 AND state='intent' AND intent_expires_at<=$2
			FOR UPDATE
		`, intent.Owner, store.now().UTC()).Scan(&expiredJobID, &expiredOAuthSession)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock expired account deletion intent: %w", err)
		}
		if err == nil {
			if _, err := tx.Exec(ctx, `
				DELETE FROM oauth_auth_requests
				WHERE purpose='accountDeletion'
				  AND account_deletion_owner_did=$1
				  AND account_deletion_job_id=$2
			`, intent.Owner, expiredJobID); err != nil {
				return fmt.Errorf("delete expired account deletion OAuth request: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				DELETE FROM account_deletion_operations
				WHERE id=$1 AND owner_did=$2 AND state='intent'
			`, expiredJobID, intent.Owner); err != nil {
				return fmt.Errorf("delete expired account deletion intent: %w", err)
			}
			if expiredOAuthSession != nil {
				if _, err := tx.Exec(ctx, `
					DELETE FROM oauth_sessions
					WHERE account_did=$1 AND session_id=$2
				`, intent.Owner, *expiredOAuthSession); err != nil {
					return fmt.Errorf("delete expired account deletion OAuth session: %w", err)
				}
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_operations(
				id,owner_did,state,confirmation_handle_hash,intent_expires_at
			) VALUES($1,$2,'intent',$3,$4)
		`, intent.JobID, intent.Owner, intent.ConfirmationHandleHash, intent.ExpiresAt); err != nil {
			return fmt.Errorf("insert account deletion intent: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_status_credentials(
				token_hash,job_id,owner_did,device_id,expires_at
			) VALUES($1,$2,$3,$4,$5)
		`, intent.StatusCapabilityHash, intent.JobID, intent.Owner, intent.DeviceID, intent.ExpiresAt.Add(30*24*time.Hour)); err != nil {
			return fmt.Errorf("insert account deletion status credential: %w", err)
		}
		return nil
	})
}

func (store *Store) CancelIntent(ctx context.Context, jobID uuid.UUID, owner syntax.DID, statusCapability string) error {
	if store == nil || store.pool == nil || jobID == uuid.Nil || owner == "" || statusCapability == "" {
		return ErrStatusUnauthorized
	}
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var (
			status        Status
			statusExists  bool
			reauthSession *string
		)
		err := tx.QueryRow(ctx, `
			SELECT operation.state,operation.reauth_oauth_session_id,
			       EXISTS(
				   SELECT 1 FROM account_deletion_status_credentials credential
				   WHERE credential.job_id=operation.id
				     AND credential.owner_did=operation.owner_did
				     AND credential.token_hash=$3
				     AND credential.revoked_at IS NULL
				     AND credential.expires_at>$4
			   )
			FROM account_deletion_operations operation
			WHERE operation.id=$1 AND operation.owner_did=$2
			FOR UPDATE
		`, jobID, owner, HashSecret(statusCapability), store.now().UTC()).Scan(&status, &reauthSession, &statusExists)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if !statusExists {
			return ErrStatusUnauthorized
		}
		if status != StatusIntent && status != StatusCanceled {
			return ErrPointOfNoReturn
		}
		if _, err := tx.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1 AND owner_did=$2`, jobID, owner); err != nil {
			return fmt.Errorf("delete account deletion intent: %w", err)
		}
		if reauthSession != nil {
			if _, err := tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, *reauthSession); err != nil {
				return fmt.Errorf("delete account deletion reauthentication session: %w", err)
			}
		}
		return nil
	})
}

func (store *Store) CompleteReauthentication(ctx context.Context, jobID uuid.UUID, owner syntax.DID, oauthSessionID string, proofHash []byte) error {
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		now := store.now().UTC()
		var (
			status       Status
			intentExpiry *time.Time
			boundSession *string
			category     *ErrorCategory
		)
		if err := tx.QueryRow(ctx, `
			SELECT state,intent_expires_at,deletion_oauth_session_id,error_category
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2
			FOR UPDATE
		`, jobID, owner).Scan(&status, &intentExpiry, &boundSession, &category); err != nil {
			return ErrReauthenticationRequired
		}
		switch {
		case status == StatusIntent && intentExpiry != nil && now.Before(*intentExpiry):
			_, err := tx.Exec(ctx, `
				UPDATE account_deletion_operations
				SET reauth_oauth_session_id=$3,intent_proof_hash=$4,updated_at=$5
				WHERE id=$1 AND owner_did=$2
			`, jobID, owner, oauthSessionID, proofHash, now)
			return err
		case status == StatusNeedsAttention && category != nil && *category == ErrorCategoryReauthentication && boundSession != nil:
			oldSession := *boundSession
			if _, err := tx.Exec(ctx, `
				UPDATE account_deletion_operations
				SET deletion_oauth_session_id=$3,reauth_oauth_session_id=NULL,
				    intent_proof_hash=NULL,intent_expires_at=NULL,
				    state='active',attempt_count=0,error_category=NULL,
				    next_attempt_at=$4,updated_at=$4
				WHERE id=$1 AND owner_did=$2
			`, jobID, owner, oauthSessionID, now); err != nil {
				return err
			}
			if oldSession != oauthSessionID {
				if _, err := tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, oldSession); err != nil {
					return err
				}
			}
			return nil
		default:
			return ErrReauthenticationRequired
		}
	})
}

func (store *Store) RegisterExpected(
	ctx context.Context,
	jobID string,
	owner syntax.DID,
	uri syntax.ATURI,
	collection syntax.NSID,
) error {
	parsedJobID, err := uuid.Parse(jobID)
	if err != nil || uri.Authority().String() != owner.String() || uri.Collection() != collection || !registeredCraftskyCollection(collection) {
		return errors.New("invalid expected CraftSky record")
	}
	_, err = store.pool.Exec(ctx, `
		INSERT INTO account_deletion_expected_records(
			job_id,uri,collection,registered_at
		) VALUES($1,$2,$3,$4)
		ON CONFLICT(job_id,uri) DO NOTHING
	`, parsedJobID, uri, collection, store.now().UTC())
	if err != nil {
		return fmt.Errorf("persist expected CraftSky record: %w", err)
	}
	return nil
}

func registeredCraftskyCollection(collection syntax.NSID) bool {
	for _, registered := range CraftskyRecordCollections() {
		if registered == collection {
			return true
		}
	}
	return false
}

func (store *Store) MarkDeleteRequested(ctx context.Context, jobID string, owner syntax.DID, uri syntax.ATURI) error {
	parsedJobID, err := uuid.Parse(jobID)
	if err != nil || uri.Authority().String() != owner.String() {
		return errors.New("invalid expected CraftSky delete request")
	}
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_expected_records expected
		SET delete_requested_at=COALESCE(delete_requested_at,$4)
		FROM account_deletion_operations operation
		WHERE expected.job_id=$1 AND expected.uri=$2
		  AND operation.id=expected.job_id AND operation.owner_did=$3
	`, parsedJobID, uri, owner, store.now().UTC())
	if err != nil {
		return fmt.Errorf("persist expected CraftSky delete request: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrOperationNotFound
	}
	return nil
}

func (store *Store) CleanupStepComplete(ctx context.Context, jobID uuid.UUID, owner syntax.DID, component string) (bool, error) {
	if store == nil || store.pool == nil || jobID == uuid.Nil || owner == "" || component == "" {
		return false, errors.New("invalid private cleanup checkpoint")
	}
	var complete bool
	err := store.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM account_deletion_cleanup_steps step
			JOIN account_deletion_operations operation ON operation.id=step.job_id
			WHERE step.job_id=$1 AND operation.owner_did=$2 AND step.component=$3
		)
	`, jobID, owner, component).Scan(&complete)
	if err != nil {
		return false, fmt.Errorf("read private cleanup checkpoint: %w", err)
	}
	return complete, nil
}

func (store *Store) CompleteCleanupStep(ctx context.Context, jobID uuid.UUID, owner syntax.DID, component string) error {
	if store == nil || store.pool == nil || jobID == uuid.Nil || owner == "" || component == "" {
		return errors.New("invalid private cleanup checkpoint")
	}
	result, err := store.pool.Exec(ctx, `
		INSERT INTO account_deletion_cleanup_steps(job_id,component,completed_at)
		SELECT id,$3,$4
		FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2
		ON CONFLICT(job_id,component) DO NOTHING
	`, jobID, owner, component, store.now().UTC())
	if err != nil {
		return fmt.Errorf("persist private cleanup checkpoint: %w", err)
	}
	if result.RowsAffected() == 0 {
		complete, readErr := store.CleanupStepComplete(ctx, jobID, owner, component)
		if readErr != nil {
			return readErr
		}
		if !complete {
			return ErrOperationNotFound
		}
	}
	return nil
}

func (store *Store) ClaimDue(ctx context.Context, workerID string, leaseDuration time.Duration) (ClaimedOperation, bool, error) {
	if store == nil || store.pool == nil || workerID == "" || leaseDuration <= 0 {
		return ClaimedOperation{}, false, errors.New("invalid account deletion lease")
	}
	now := store.now().UTC()
	leaseToken := uuid.New()
	var operation ClaimedOperation
	err := store.pool.QueryRow(ctx, `
		WITH candidate AS (
			SELECT id
			FROM account_deletion_operations
			WHERE state IN ('active','retrying')
			  AND next_attempt_at IS NOT NULL AND next_attempt_at<=$1
			  AND (lease_expires_at IS NULL OR lease_expires_at<=$1)
			ORDER BY next_attempt_at,id
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE account_deletion_operations operation
		SET lease_owner=$2,lease_token=$3,lease_expires_at=$4,updated_at=$1
		FROM candidate
		WHERE operation.id=candidate.id
		RETURNING operation.id,operation.owner_did,operation.state,
		          operation.phase,operation.attempt_count,operation.lease_token
	`, now, workerID, leaseToken, now.Add(leaseDuration)).Scan(
		&operation.JobID, &operation.Owner, &operation.Status,
		&operation.Phase, &operation.AttemptCount, &operation.LeaseToken,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClaimedOperation{}, false, nil
	}
	if err != nil {
		return ClaimedOperation{}, false, fmt.Errorf("claim account deletion operation: %w", err)
	}
	return operation, true, nil
}

func (store *Store) RecordFailure(
	ctx context.Context,
	operation ClaimedOperation,
	decision RetryDecision,
	category ErrorCategory,
	nextAttemptCount int,
) error {
	if nextAttemptCount <= operation.AttemptCount || operation.LeaseToken == uuid.Nil {
		return errors.New("invalid account deletion failure update")
	}
	status := StatusNeedsAttention
	var nextAttempt *time.Time
	if decision.Action == RetrySchedule {
		status = StatusRetrying
		at := decision.At.UTC()
		nextAttempt = &at
	}
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET state=$4,attempt_count=$5,next_attempt_at=$6,error_category=$7,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$8
		WHERE id=$1 AND owner_did=$2 AND lease_token=$3
	`, operation.JobID, operation.Owner, operation.LeaseToken, status,
		nextAttemptCount, nextAttempt, category, store.now().UTC())
	if err != nil {
		return fmt.Errorf("record account deletion failure: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrOperationNotFound
	}
	return nil
}

func (store *Store) CompleteAttempt(ctx context.Context, operation ClaimedOperation) error {
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET state='active',attempt_count=0,error_category=NULL,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$4
		WHERE id=$1 AND owner_did=$2 AND lease_token=$3
	`, operation.JobID, operation.Owner, operation.LeaseToken, store.now().UTC())
	if err != nil {
		return fmt.Errorf("complete account deletion attempt: %w", err)
	}
	if result.RowsAffected() != 1 {
		var finalized bool
		if err := store.pool.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM account_deletion_audits WHERE job_id=$1 AND did=$2)
		`, operation.JobID, operation.Owner).Scan(&finalized); err != nil {
			return err
		}
		if !finalized {
			return ErrOperationNotFound
		}
	}
	return nil
}

func (store *Store) DeferAttempt(ctx context.Context, operation ClaimedOperation, at time.Time) error {
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET state='active',next_attempt_at=$4,error_category=NULL,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$5
		WHERE id=$1 AND owner_did=$2 AND lease_token=$3
	`, operation.JobID, operation.Owner, operation.LeaseToken, at.UTC(), store.now().UTC())
	if err != nil {
		return fmt.Errorf("defer account deletion attempt: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrOperationNotFound
	}
	return nil
}

func (store *Store) AdvancePhase(ctx context.Context, operation ClaimedOperation, phase Phase) error {
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET state='active',phase=$4,next_attempt_at=$5,error_category=NULL,updated_at=$5
		WHERE id=$1 AND owner_did=$2 AND lease_token=$3
	`, operation.JobID, operation.Owner, operation.LeaseToken, phase, store.now().UTC())
	if err != nil {
		return fmt.Errorf("advance account deletion phase: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrOperationNotFound
	}
	return nil
}

func (store *Store) BoundOAuthSession(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (string, error) {
	var sessionID string
	err := store.pool.QueryRow(ctx, `
		SELECT deletion_oauth_session_id
		FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2 AND deletion_oauth_session_id IS NOT NULL
	`, jobID, owner).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrBoundOAuthUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("read bound deletion OAuth session: %w", err)
	}
	return sessionID, nil
}

func (store *Store) PrivateFinalizationGates(ctx context.Context, jobID uuid.UUID, owner syntax.DID, components []string) (TerminalGates, error) {
	var gates TerminalGates
	err := store.pool.QueryRow(ctx, `
		SELECT
			NOT EXISTS(
				SELECT 1 FROM unnest($3::text[]) required(component)
				WHERE NOT EXISTS(
					SELECT 1 FROM account_deletion_cleanup_steps step
					WHERE step.job_id=$1 AND step.component=required.component
				)
			),
			NOT EXISTS(SELECT 1 FROM craftsky_sessions WHERE account_did=$2),
			NOT EXISTS(SELECT 1 FROM push_account_subscriptions WHERE account_did=$2),
			NOT EXISTS(
				SELECT 1
				FROM account_deletion_cleanup_artifacts artifact
				JOIN scheduled_post_cleanup_jobs cleanup ON cleanup.id=artifact.artifact_id
				WHERE artifact.job_id=$1 AND artifact.component='scheduledPosts'
			)
	`, jobID, owner, components).Scan(
		&gates.PrivateCleanupComplete,
		&gates.OrdinarySessionsAbsent,
		&gates.SubscriptionsAbsent,
		&gates.ScheduledObjectCleanupComplete,
	)
	if err != nil {
		return TerminalGates{}, fmt.Errorf("read private finalization gates: %w", err)
	}
	return gates, nil
}

func (store *Store) ManualRetry(ctx context.Context, jobID uuid.UUID, owner syntax.DID) error {
	if store == nil || store.pool == nil || jobID == uuid.Nil || owner == "" {
		return errors.New("invalid account deletion retry")
	}
	now := store.now().UTC()
	var phase Phase
	err := store.pool.QueryRow(ctx, `
		UPDATE account_deletion_operations
		SET state='active',attempt_count=0,next_attempt_at=$3,error_category=NULL,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$3
		WHERE id=$1 AND owner_did=$2 AND state='needsAttention'
		RETURNING phase
	`, jobID, owner, now).Scan(&phase)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidTransition
	}
	if err != nil {
		return fmt.Errorf("manual retry account deletion: %w", err)
	}
	if store.telemetry != nil {
		store.telemetry.ManualRetry(ctx, phase)
	}
	return nil
}

func (store *Store) Accept(ctx context.Context, request AcceptanceRequest) (Operation, error) {
	var accepted Operation
	newlyAccepted := false
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var (
			status          Status
			phase           *Phase
			acceptedAt      *time.Time
			boundSession    *string
			reauthSession   *string
			proofHash       []byte
			handleHash      []byte
			intentExpiresAt *time.Time
		)
		if err := tx.QueryRow(ctx, `
			SELECT state,phase,accepted_at,deletion_oauth_session_id,
			       reauth_oauth_session_id,intent_proof_hash,
			       confirmation_handle_hash,intent_expires_at
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2
			FOR UPDATE
		`, request.JobID, request.Owner).Scan(
			&status, &phase, &acceptedAt, &boundSession, &reauthSession,
			&proofHash, &handleHash, &intentExpiresAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return err
		}
		if status != StatusIntent {
			if boundSession == nil || acceptedAt == nil || phase == nil {
				return ErrDeletionAlreadyPending
			}
			accepted = Operation{
				JobID: request.JobID, Owner: request.Owner, Status: status, Phase: *phase,
				AcceptedAt: *acceptedAt, DeletionOAuthSessionID: *boundSession,
			}
			return nil
		}

		now := store.now().UTC()
		if intentExpiresAt == nil || !now.Before(*intentExpiresAt) || reauthSession == nil ||
			!bytes.Equal(proofHash, HashSecret(request.ReauthProof)) {
			return ErrReauthenticationRequired
		}
		if !bytes.Equal(handleHash, HashSecret(request.ConfirmationHandle)) {
			return ErrConfirmationHandleMismatch
		}
		var statusCredentialExists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM account_deletion_status_credentials
				WHERE token_hash=$1 AND job_id=$2 AND owner_did=$3
				  AND revoked_at IS NULL AND expires_at>$4
			)
		`, HashSecret(request.StatusCapability), request.JobID, request.Owner, now).Scan(&statusCredentialExists); err != nil {
			return err
		}
		if !statusCredentialExists {
			return ErrStatusUnauthorized
		}
		var oauthExists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM oauth_sessions WHERE account_did=$1 AND session_id=$2
			)
		`, request.Owner, *reauthSession).Scan(&oauthExists); err != nil {
			return err
		}
		if !oauthExists {
			return ErrReauthenticationRequired
		}

		// Bind first. The operation-to-OAuth foreign key protects this session
		// from every bearer and unbound OAuth deletion that follows.
		if _, err := tx.Exec(ctx, `
			UPDATE account_deletion_operations
			SET state='active',phase='queued',accepted_at=$3,
			    deletion_oauth_session_id=reauth_oauth_session_id,
			    reauth_oauth_session_id=NULL,intent_proof_hash=NULL,
			    confirmation_handle_hash=NULL,intent_expires_at=NULL,
			    next_attempt_at=$3,updated_at=$3
			WHERE id=$1 AND owner_did=$2
		`, request.JobID, request.Owner, now); err != nil {
			return fmt.Errorf("bind deletion OAuth session: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_recovery_credentials(
				token_hash,job_id,owner_did,device_id
			)
			SELECT token_hash,$1,account_did,last_device_id
			FROM craftsky_sessions WHERE account_did=$2
			ON CONFLICT(token_hash) DO NOTHING
		`, request.JobID, request.Owner); err != nil {
			return fmt.Errorf("capture status recovery credentials: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			WITH removed AS (
				DELETE FROM push_account_subscriptions
				WHERE account_did=$1 RETURNING installation_id
			)
			DELETE FROM push_installations installation
			WHERE installation.id IN (SELECT installation_id FROM removed)
			  AND NOT EXISTS(
				SELECT 1 FROM push_account_subscriptions remaining
				WHERE remaining.installation_id=installation.id
				  AND remaining.account_did<>$1
			  )
		`, request.Owner); err != nil {
			return fmt.Errorf("remove account push subscriptions: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_sessions WHERE account_did=$1`, request.Owner); err != nil {
			return fmt.Errorf("remove ordinary CraftSky sessions: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM oauth_sessions
			WHERE account_did=$1 AND session_id<>$2
		`, request.Owner, *reauthSession); err != nil {
			return fmt.Errorf("remove unbound OAuth sessions: %w", err)
		}
		accepted = Operation{
			JobID: request.JobID, Owner: request.Owner, Status: StatusActive,
			Phase: PhaseQueued, AcceptedAt: now, DeletionOAuthSessionID: *reauthSession,
		}
		newlyAccepted = true
		return nil
	})
	if err == nil && newlyAccepted && store.telemetry != nil {
		store.telemetry.Accepted(ctx)
	}
	return accepted, err
}

func (store *Store) GetOperation(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (Operation, error) {
	var operation Operation
	err := store.pool.QueryRow(ctx, `
		SELECT id,owner_did,state,COALESCE(phase,''),accepted_at,
		       COALESCE(deletion_oauth_session_id,'')
		FROM account_deletion_operations WHERE id=$1 AND owner_did=$2
	`, jobID, owner).Scan(
		&operation.JobID, &operation.Owner, &operation.Status, &operation.Phase,
		&operation.AcceptedAt, &operation.DeletionOAuthSessionID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, ErrOperationNotFound
	}
	return operation, err
}

func (store *Store) FinalizeSuccess(ctx context.Context, jobID uuid.UUID, owner syntax.DID, terminalAt time.Time) error {
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var acceptedAt time.Time
		var boundSession string
		if err := tx.QueryRow(ctx, `
			SELECT accepted_at,deletion_oauth_session_id
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND accepted_at IS NOT NULL
			FOR UPDATE
		`, jobID, owner).Scan(&acceptedAt, &boundSession); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_audits(
				job_id,did,accepted_at,terminal_at,outcome,expires_at
			) VALUES($1,$2,$3,$4,'deleted',$5)
			ON CONFLICT(job_id) DO NOTHING
		`, jobID, owner, acceptedAt, terminalAt, terminalAt.Add(30*24*time.Hour)); err != nil {
			return fmt.Errorf("insert account deletion audit: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1 AND owner_did=$2`, jobID, owner); err != nil {
			return fmt.Errorf("remove operational deletion state: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, boundSession); err != nil {
			return fmt.Errorf("remove bound deletion OAuth session: %w", err)
		}
		return nil
	})
	if err == nil && store.telemetry != nil {
		store.telemetry.TerminalSuccess(ctx)
	}
	return err
}
