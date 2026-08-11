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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrOperationNotFound = errors.New("account deletion operation not found")

type Store struct {
	pool *pgxpool.Pool
	now  func() time.Time
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
	ConfirmationHandleHash []byte
	ExpiresAt              time.Time
}

type AcceptanceRequest struct {
	JobID              uuid.UUID
	Owner              syntax.DID
	ReauthProof        string
	ConfirmationHandle string
}

type Operation struct {
	JobID                  uuid.UUID
	Owner                  syntax.DID
	Status                 Status
	AcceptedAt             time.Time
	DeletionOAuthSessionID string
}

func (store *Store) CreateIntent(ctx context.Context, intent IntentRecord) error {
	if store == nil || store.pool == nil || intent.JobID == uuid.Nil || intent.Owner == "" ||
		len(intent.ConfirmationHandleHash) == 0 || intent.ExpiresAt.IsZero() {
		return errors.New("invalid account deletion intent")
	}
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
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
				WHERE purpose='accountDeletion' AND account_deletion_owner_did=$1
				  AND account_deletion_job_id=$2
			`, intent.Owner, expiredJobID); err != nil {
				return fmt.Errorf("delete expired account deletion OAuth request: %w", err)
			}
			if _, err := tx.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1`, expiredJobID); err != nil {
				return fmt.Errorf("delete expired account deletion intent: %w", err)
			}
			if expiredOAuthSession != nil {
				if _, err := tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, intent.Owner, *expiredOAuthSession); err != nil {
					return fmt.Errorf("delete expired account deletion OAuth session: %w", err)
				}
			}
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO account_deletion_operations(
				id,owner_did,state,confirmation_handle_hash,intent_expires_at
			) VALUES($1,$2,'intent',$3,$4)
		`, intent.JobID, intent.Owner, intent.ConfirmationHandleHash, intent.ExpiresAt.UTC())
		return err
	})
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrDeletionAlreadyPending
	}
	if err != nil {
		return fmt.Errorf("create account deletion intent: %w", err)
	}
	return nil
}

func (store *Store) CancelIntent(ctx context.Context, jobID uuid.UUID, owner syntax.DID) error {
	if store == nil || store.pool == nil || jobID == uuid.Nil || owner == "" {
		return ErrOperationNotFound
	}
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var reauthSession *string
		err := tx.QueryRow(ctx, `
			SELECT reauth_oauth_session_id FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND state='intent' FOR UPDATE
		`, jobID, owner).Scan(&reauthSession)
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if readErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM account_deletion_operations WHERE id=$1 AND owner_did=$2)`, jobID, owner).Scan(&exists); readErr != nil {
				return readErr
			}
			if exists {
				return ErrPointOfNoReturn
			}
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM oauth_auth_requests
			WHERE purpose='accountDeletion' AND account_deletion_owner_did=$1
			  AND account_deletion_job_id=$2
		`, owner, jobID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1 AND owner_did=$2`, jobID, owner); err != nil {
			return err
		}
		if reauthSession != nil {
			_, err = tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, *reauthSession)
		}
		return err
	})
}

func (store *Store) CompleteReauthentication(ctx context.Context, jobID uuid.UUID, owner syntax.DID, oauthSessionID string, proofHash []byte) error {
	if store == nil || store.pool == nil || jobID == uuid.Nil || owner == "" || oauthSessionID == "" || len(proofHash) == 0 {
		return ErrReauthenticationRequired
	}
	now := store.now().UTC()
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET reauth_oauth_session_id=$3,intent_proof_hash=$4,updated_at=$5
		WHERE id=$1 AND owner_did=$2 AND state='intent' AND intent_expires_at>$5
	`, jobID, owner, oauthSessionID, proofHash, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrReauthenticationRequired
	}
	return nil
}

func (store *Store) Accept(ctx context.Context, request AcceptanceRequest) (Operation, error) {
	var accepted Operation
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var (
			status          Status
			acceptedAt      *time.Time
			boundSession    *string
			reauthSession   *string
			proofHash       []byte
			handleHash      []byte
			intentExpiresAt *time.Time
		)
		if err := tx.QueryRow(ctx, `
			SELECT state,accepted_at,deletion_oauth_session_id,reauth_oauth_session_id,
			       intent_proof_hash,confirmation_handle_hash,intent_expires_at
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 FOR UPDATE
		`, request.JobID, request.Owner).Scan(
			&status, &acceptedAt, &boundSession, &reauthSession,
			&proofHash, &handleHash, &intentExpiresAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return err
		}
		if status != StatusIntent {
			if boundSession == nil || acceptedAt == nil {
				return ErrDeletionAlreadyPending
			}
			accepted = Operation{
				JobID: request.JobID, Owner: request.Owner, Status: status,
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
		var oauthExists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM oauth_sessions WHERE account_did=$1 AND session_id=$2)
		`, request.Owner, *reauthSession).Scan(&oauthExists); err != nil {
			return err
		}
		if !oauthExists {
			return ErrReauthenticationRequired
		}

		// The durable operation binds its exact fresh OAuth authority before
		// ordinary sessions or other OAuth sessions are removed.
		if _, err := tx.Exec(ctx, `
			UPDATE account_deletion_operations
			SET state='active',accepted_at=$3,
			    deletion_oauth_session_id=reauth_oauth_session_id,
			    reauth_oauth_session_id=NULL,intent_proof_hash=NULL,
			    confirmation_handle_hash=NULL,intent_expires_at=NULL,
			    attempt_count=0,next_attempt_at=$3,error_category=NULL,updated_at=$3
			WHERE id=$1 AND owner_did=$2
		`, request.JobID, request.Owner, now); err != nil {
			return fmt.Errorf("bind deletion OAuth session: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM craftsky_sessions WHERE account_did=$1`, request.Owner); err != nil {
			return fmt.Errorf("remove ordinary CraftSky sessions: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id<>$2
		`, request.Owner, *reauthSession); err != nil {
			return fmt.Errorf("remove unbound OAuth sessions: %w", err)
		}
		accepted = Operation{
			JobID: request.JobID, Owner: request.Owner, Status: StatusActive,
			AcceptedAt: now, DeletionOAuthSessionID: *reauthSession,
		}
		return nil
	})
	return accepted, err
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
			SELECT id FROM account_deletion_operations
			WHERE state IN ('active','retrying') AND next_attempt_at<=$1
			  AND (lease_expires_at IS NULL OR lease_expires_at<=$1)
			ORDER BY next_attempt_at,id LIMIT 1 FOR UPDATE SKIP LOCKED
		)
		UPDATE account_deletion_operations operation
		SET lease_owner=$2,lease_token=$3,lease_expires_at=$4,updated_at=$1
		FROM candidate WHERE operation.id=candidate.id
		RETURNING operation.id,operation.owner_did,operation.attempt_count,operation.lease_token
	`, now, workerID, leaseToken, now.Add(leaseDuration)).Scan(
		&operation.JobID, &operation.Owner, &operation.AttemptCount, &operation.LeaseToken,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClaimedOperation{}, false, nil
	}
	if err != nil {
		return ClaimedOperation{}, false, fmt.Errorf("claim account deletion operation: %w", err)
	}
	return operation, true, nil
}

func (store *Store) RecordFailure(ctx context.Context, operation ClaimedOperation, nextAt time.Time, category ErrorCategory, nextAttemptCount int) error {
	if nextAttemptCount <= operation.AttemptCount || operation.LeaseToken == uuid.Nil || nextAt.IsZero() {
		return errors.New("invalid account deletion failure update")
	}
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_operations
		SET state='retrying',attempt_count=$4,next_attempt_at=$5,error_category=$6,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$7
		WHERE id=$1 AND owner_did=$2 AND lease_token=$3
	`, operation.JobID, operation.Owner, operation.LeaseToken, nextAttemptCount,
		nextAt.UTC(), category, store.now().UTC())
	if err != nil {
		return fmt.Errorf("record account deletion failure: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrOperationNotFound
	}
	return nil
}

func (store *Store) BoundOAuthSession(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (string, error) {
	var sessionID string
	err := store.pool.QueryRow(ctx, `
		SELECT deletion_oauth_session_id FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2 AND state IN ('active','retrying')
		  AND deletion_oauth_session_id IS NOT NULL
	`, jobID, owner).Scan(&sessionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrBoundOAuthUnauthorized
	}
	if err != nil {
		return "", fmt.Errorf("read bound deletion OAuth session: %w", err)
	}
	return sessionID, nil
}

// CompleteAttempt removes the only retained deletion authority and operation
// atomically. A successful processor run has already completed private cleanup
// and the PDS deleter's final empty scan.
func (store *Store) CompleteAttempt(ctx context.Context, operation ClaimedOperation) error {
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var sessionID string
		if err := tx.QueryRow(ctx, `
			SELECT deletion_oauth_session_id FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND lease_token=$3 FOR UPDATE
		`, operation.JobID, operation.Owner, operation.LeaseToken).Scan(&sessionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1`, operation.JobID); err != nil {
			return fmt.Errorf("remove account deletion operation: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, operation.Owner, sessionID); err != nil {
			return fmt.Errorf("remove bound deletion OAuth session: %w", err)
		}
		return nil
	})
}

// RefreshBoundOAuthFromLogin lets a later fresh OAuth login replace unusable
// worker authority without restoring membership or minting a CraftSky bearer.
func (store *Store) RefreshBoundOAuthFromLogin(ctx context.Context, owner syntax.DID, newSessionID string) (bool, error) {
	if store == nil || store.pool == nil || owner == "" || newSessionID == "" {
		return false, errors.New("invalid account deletion OAuth refresh")
	}
	refreshed := false
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var oldSessionID string
		err := tx.QueryRow(ctx, `
			SELECT deletion_oauth_session_id FROM account_deletion_operations
			WHERE owner_did=$1 AND state IN ('active','retrying') FOR UPDATE
		`, owner).Scan(&oldSessionID)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		now := store.now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE account_deletion_operations
			SET deletion_oauth_session_id=$2,state='active',attempt_count=0,
			    next_attempt_at=$3,error_category=NULL,
			    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,updated_at=$3
			WHERE owner_did=$1
		`, owner, newSessionID, now); err != nil {
			return err
		}
		if oldSessionID != newSessionID {
			if _, err := tx.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1 AND session_id=$2`, owner, oldSessionID); err != nil {
				return err
			}
		}
		refreshed = true
		return nil
	})
	return refreshed, err
}
