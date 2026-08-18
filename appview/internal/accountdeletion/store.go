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

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
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

// CreateIntentParticipant inserts the operation at the account-deletion row
// lock position in the same transaction as active -> deletion_pending and
// auth/session invalidation. owner_generation deliberately records the
// originating active generation used by remote-effect safety reconciliation.
func (store *Store) CreateIntentParticipant(intent IntentRecord) ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if store == nil || intent.JobID == uuid.Nil || intent.Owner == "" ||
			len(intent.ConfirmationHandleHash) == 0 || intent.ExpiresAt.IsZero() ||
			before.Owner != intent.Owner || after.Owner != intent.Owner ||
			before.State != ownerlifecycle.StateActive || after.State != ownerlifecycle.StateDeletionPending {
			return errors.New("invalid account deletion intent transition")
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_operations(
				id,owner_did,owner_generation,state,
				confirmation_handle_hash,intent_expires_at
			) VALUES($1,$2,$3,'intent',$4,$5)
		`, intent.JobID, intent.Owner, before.Generation,
			intent.ConfirmationHandleHash, intent.ExpiresAt.UTC())
		if err != nil {
			var postgresError *pgconn.PgError
			if errors.As(err, &postgresError) && postgresError.Code == "23505" {
				return ErrDeletionAlreadyPending
			}
		}
		return err
	}
}

func (store *Store) VerifyDeletionOAuthRequest(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	jobID uuid.UUID,
	authority ownerlifecycle.Lifecycle,
) error {
	if owner == "" || jobID == uuid.Nil || authority.Owner != owner ||
		authority.State != ownerlifecycle.StateDeletionPending || authority.Generation <= 1 {
		return ErrOperationNotFound
	}
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT true FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2 AND owner_generation=$3
		  AND state='intent' AND intent_expires_at>$4
		FOR UPDATE
	`, jobID, owner, authority.Generation-1, store.now().UTC()).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOperationNotFound
		}
		return err
	}
	return nil
}

// AuthorizeRecoverySession permits an existing CraftSky child to reach only
// the recovery route class while its exact deletion intent is still current.
// The caller already holds the lifecycle row lock; locking the operation here
// preserves the global owner -> operation -> OAuth parent -> child order.
func (store *Store) AuthorizeRecoverySession(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	authority ownerlifecycle.Lifecycle,
) error {
	if store == nil || tx == nil || owner == "" || authority.Owner != owner ||
		authority.State != ownerlifecycle.StateDeletionPending || authority.Generation <= 1 {
		return auth.ErrRecoverySessionNotAuthorized
	}
	var current bool
	err := tx.QueryRow(ctx, `
		SELECT true
		FROM account_deletion_operations
		WHERE owner_did=$1 AND owner_generation=$2
		  AND state='intent' AND intent_expires_at>$3
		FOR SHARE
	`, owner, authority.Generation-1, store.now().UTC()).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrRecoverySessionNotAuthorized
	}
	if err != nil {
		return fmt.Errorf("authorize account deletion recovery session: %w", err)
	}
	return nil
}

func (store *Store) CancelIntentParticipant(
	jobID uuid.UUID,
	owner syntax.DID,
) ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if jobID == uuid.Nil || owner == "" || before.Owner != owner || after.Owner != owner ||
			before.State != ownerlifecycle.StateDeletionPending ||
			(after.State != ownerlifecycle.StateActive && after.State != ownerlifecycle.StateDeparted) {
			return ErrOperationNotFound
		}
		var profileExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1)`, owner).Scan(&profileExists); err != nil {
			return err
		}
		if profileExists != (after.State == ownerlifecycle.StateActive) {
			return ownerlifecycle.ErrGenerationChanged
		}
		requestRows, err := tx.Query(ctx, `
			SELECT state FROM oauth_auth_requests
			WHERE purpose='accountDeletion' AND account_deletion_owner_did=$1
			  AND account_deletion_job_id=$2
			ORDER BY state FOR UPDATE
		`, owner, jobID)
		if err != nil {
			return err
		}
		for requestRows.Next() {
			var state string
			if err := requestRows.Scan(&state); err != nil {
				requestRows.Close()
				return err
			}
		}
		if err := requestRows.Err(); err != nil {
			requestRows.Close()
			return err
		}
		requestRows.Close()
		if _, err := tx.Exec(ctx, `
			DELETE FROM oauth_auth_requests
			WHERE purpose='accountDeletion' AND account_deletion_owner_did=$1
			  AND account_deletion_job_id=$2 AND request_state<>'exchange_ambiguous'
		`, owner, jobID); err != nil {
			return err
		}
		var status Status
		var reauthSession *string
		if err := tx.QueryRow(ctx, `
			SELECT state,reauth_oauth_session_id FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 FOR UPDATE
		`, jobID, owner).Scan(&status, &reauthSession); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return err
		}
		if status != StatusIntent {
			return ErrPointOfNoReturn
		}
		if reauthSession != nil {
			command, err := tx.Exec(ctx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
				    deletion_credential_generation=NULL,
				    revocation_requested_at=COALESCE(revocation_requested_at,now()),
				    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,now()),
				    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
				    row_version=row_version+1,updated_at=now()
				WHERE account_did=$1 AND session_id=$2
				  AND lifecycle_state='deletion_only' AND deletion_operation_id=$3
			`, owner, *reauthSession, jobID)
			if err != nil {
				return err
			}
			if command.RowsAffected() != 1 {
				return ErrReauthenticationRequired
			}
		}
		command, err := tx.Exec(ctx, `
			DELETE FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND state='intent'
		`, jobID, owner)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrOperationNotFound
		}
		return nil
	}
}

func (store *Store) CancellationTarget(ctx context.Context, owner syntax.DID) (ownerlifecycle.State, error) {
	if store == nil || store.pool == nil || owner == "" {
		return "", ErrOperationNotFound
	}
	var profileExists bool
	if err := store.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM craftsky_profiles WHERE did=$1)
	`, owner).Scan(&profileExists); err != nil {
		return "", err
	}
	if profileExists {
		return ownerlifecycle.StateActive, nil
	}
	return ownerlifecycle.StateDeparted, nil
}

func (store *Store) ListExpiredIntents(ctx context.Context, limit int) ([]ExpiredIntent, error) {
	if store == nil || store.pool == nil || limit <= 0 || limit > 1000 {
		return nil, errors.New("invalid expired account deletion intent query")
	}
	rows, err := store.pool.Query(ctx, `
		SELECT id,owner_did
		FROM account_deletion_operations
		WHERE state='intent' AND intent_expires_at<=$1
		ORDER BY intent_expires_at,id
		LIMIT $2
	`, store.now().UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	intents := make([]ExpiredIntent, 0, limit)
	for rows.Next() {
		var intent ExpiredIntent
		if err := rows.Scan(&intent.JobID, &intent.Owner); err != nil {
			return nil, err
		}
		intents = append(intents, intent)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return intents, nil
}

// ProfileDepartureParticipant removes a still-reversible deletion intent when
// the membership profile is deleted first. It is composed inside the session
// lifecycle participant, which has already locked owner auth-request rows;
// the operation and its optional deletion-only parent are then handled in the
// repository-wide order. Accepted deletion is deliberately not handled here.
func (store *Store) ProfileDepartureParticipant() ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if store == nil || before.Owner == "" || before.Owner != after.Owner {
			return errors.New("invalid profile departure account deletion participant")
		}
		if before.State != ownerlifecycle.StateDeletionPending || after.State != ownerlifecycle.StateDeparted {
			return nil
		}
		var jobID uuid.UUID
		var reauthSession *string
		err := tx.QueryRow(ctx, `
			SELECT id,reauth_oauth_session_id
			FROM account_deletion_operations
			WHERE owner_did=$1 AND state='intent'
			FOR UPDATE
		`, before.Owner).Scan(&jobID, &reauthSession)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOperationNotFound
		}
		if err != nil {
			return err
		}
		if reauthSession != nil {
			command, err := tx.Exec(ctx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
				    deletion_credential_generation=NULL,
				    revocation_requested_at=COALESCE(revocation_requested_at,now()),
				    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,now()),
				    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
				    row_version=row_version+1,updated_at=now()
				WHERE account_did=$1 AND session_id=$2
				  AND lifecycle_state='deletion_only' AND deletion_operation_id=$3
			`, before.Owner, *reauthSession, jobID)
			if err != nil {
				return err
			}
			if command.RowsAffected() != 1 {
				return ErrReauthenticationRequired
			}
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM oauth_auth_requests
			WHERE purpose='accountDeletion' AND account_deletion_owner_did=$1
			  AND account_deletion_job_id=$2 AND request_state<>'exchange_ambiguous'
		`, before.Owner, jobID); err != nil {
			return err
		}
		command, err := tx.Exec(ctx, `
			DELETE FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND state='intent'
		`, jobID, before.Owner)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrOperationNotFound
		}
		return nil
	}
}

func (store *Store) ReauthenticationGeneration(
	ctx context.Context,
	jobID uuid.UUID,
	owner syntax.DID,
) (int64, error) {
	var generation int64
	err := store.pool.QueryRow(ctx, `
		SELECT COALESCE(deletion_credential_generation,0)+1
		FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2 AND state='intent'
		  AND intent_expires_at>$3 AND reauth_oauth_session_id IS NULL
	`, jobID, owner, store.now().UTC()).Scan(&generation)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrReauthenticationRequired
	}
	return generation, err
}

func (store *Store) BindReauthentication(
	jobID uuid.UUID,
	owner syntax.DID,
	sessionID string,
	credentialGeneration int64,
	proofHash []byte,
) auth.DeletionCredentialOperationBinder {
	return func(ctx context.Context, tx pgx.Tx) error {
		if jobID == uuid.Nil || owner == "" || sessionID == "" ||
			credentialGeneration <= 0 || len(proofHash) == 0 {
			return ErrReauthenticationRequired
		}
		var status Status
		var existingGeneration *int64
		var existingReauth *string
		var expiresAt *time.Time
		if err := tx.QueryRow(ctx, `
			SELECT state,deletion_credential_generation,reauth_oauth_session_id,intent_expires_at
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 FOR UPDATE
		`, jobID, owner).Scan(&status, &existingGeneration, &existingReauth, &expiresAt); err != nil {
			return err
		}
		now := store.now().UTC()
		if status != StatusIntent || expiresAt == nil || !now.Before(*expiresAt) ||
			existingGeneration != nil || existingReauth != nil {
			return ErrReauthenticationRequired
		}
		command, err := tx.Exec(ctx, `
			UPDATE account_deletion_operations
			SET reauth_oauth_session_id=$3,deletion_credential_generation=$4,
			    intent_proof_hash=$5,updated_at=$6
			WHERE id=$1 AND owner_did=$2 AND state='intent'
		`, jobID, owner, sessionID, credentialGeneration, proofHash, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrReauthenticationRequired
		}
		return nil
	}
}

func (store *Store) AcceptanceBinding(
	ctx context.Context,
	request AcceptanceRequest,
) (auth.DeletionCredentialBinding, error) {
	var binding auth.DeletionCredentialBinding
	var status Status
	var proofHash, handleHash []byte
	var expiresAt *time.Time
	err := store.pool.QueryRow(ctx, `
		SELECT state,reauth_oauth_session_id,deletion_credential_generation,
		       intent_proof_hash,confirmation_handle_hash,intent_expires_at
		FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2
	`, request.JobID, request.Owner).Scan(
		&status, &binding.SessionID, &binding.CredentialGeneration,
		&proofHash, &handleHash, &expiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return binding, ErrOperationNotFound
	}
	if err != nil {
		return binding, err
	}
	binding.OperationID = request.JobID
	now := store.now().UTC()
	if status != StatusIntent || expiresAt == nil || !now.Before(*expiresAt) ||
		!bytes.Equal(proofHash, HashSecret(request.ReauthProof)) {
		return auth.DeletionCredentialBinding{}, ErrReauthenticationRequired
	}
	if !bytes.Equal(handleHash, HashSecret(request.ConfirmationHandle)) {
		return auth.DeletionCredentialBinding{}, ErrConfirmationHandleMismatch
	}
	return binding, nil
}

func (store *Store) AcceptParticipant(
	request AcceptanceRequest,
	binding auth.DeletionCredentialBinding,
) ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if before.Owner != request.Owner || after.Owner != request.Owner ||
			before.State != ownerlifecycle.StateDeletionPending || after.State != ownerlifecycle.StateDeleting ||
			binding.OperationID != request.JobID || binding.SessionID == "" || binding.CredentialGeneration <= 0 {
			return ErrReauthenticationRequired
		}
		var status Status
		var ownerGeneration int64
		var reauthSession string
		var credentialGeneration int64
		var proofHash, handleHash []byte
		var expiresAt time.Time
		if err := tx.QueryRow(ctx, `
			SELECT state,owner_generation,reauth_oauth_session_id,
			       deletion_credential_generation,intent_proof_hash,
			       confirmation_handle_hash,intent_expires_at
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 FOR UPDATE
		`, request.JobID, request.Owner).Scan(
			&status, &ownerGeneration, &reauthSession, &credentialGeneration,
			&proofHash, &handleHash, &expiresAt,
		); err != nil {
			return err
		}
		now := store.now().UTC()
		if status != StatusIntent || reauthSession != binding.SessionID ||
			credentialGeneration != binding.CredentialGeneration || !now.Before(expiresAt) ||
			!bytes.Equal(proofHash, HashSecret(request.ReauthProof)) {
			return ErrReauthenticationRequired
		}
		if !bytes.Equal(handleHash, HashSecret(request.ConfirmationHandle)) {
			return ErrConfirmationHandleMismatch
		}
		command, err := tx.Exec(ctx, `
			UPDATE account_deletion_operations
			SET state='active',accepted_at=$3,
			    deletion_oauth_session_id=reauth_oauth_session_id,
			    reauth_oauth_session_id=NULL,intent_proof_hash=NULL,
			    confirmation_handle_hash=NULL,intent_expires_at=NULL,
			    attempt_count=0,next_attempt_at=$3,error_category=NULL,updated_at=$3
			WHERE id=$1 AND owner_did=$2 AND state='intent'
		`, request.JobID, request.Owner, now)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return ErrOperationNotFound
		}
		return store.adoptUncertainPDSAttemptsTx(ctx, tx, ClaimedOperation{
			JobID: request.JobID, Owner: request.Owner, OwnerGeneration: ownerGeneration,
		})
	}
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
				if _, err := tx.Exec(ctx, `
					UPDATE oauth_sessions
					SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
					    deletion_credential_generation=NULL,
					    revocation_requested_at=COALESCE(revocation_requested_at,now()),
					    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,now()),
					    row_version=row_version+1,updated_at=now()
					WHERE account_did=$1 AND session_id=$2
				`, intent.Owner, *expiredOAuthSession); err != nil {
					return fmt.Errorf("queue expired account deletion OAuth session: %w", err)
				}
			}
		}
		result, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_operations(
				id,owner_did,owner_generation,state,
				confirmation_handle_hash,intent_expires_at
			)
			SELECT $1,$2,lifecycle.generation,'intent',$3,$4
			FROM owner_lifecycles lifecycle
			WHERE lifecycle.owner_did=$2 AND lifecycle.state='active'
		`, intent.JobID, intent.Owner, intent.ConfirmationHandleHash, intent.ExpiresAt.UTC())
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("account deletion owner is not active")
		}
		return nil
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
			_, err = tx.Exec(ctx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
				    deletion_credential_generation=NULL,
				    revocation_requested_at=COALESCE(revocation_requested_at,now()),
				    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,now()),
				    row_version=row_version+1,updated_at=now()
				WHERE account_did=$1 AND session_id=$2
			`, owner, *reauthSession)
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
			ownerGeneration int64
			acceptedAt      *time.Time
			boundSession    *string
			reauthSession   *string
			proofHash       []byte
			handleHash      []byte
			intentExpiresAt *time.Time
		)
		if err := tx.QueryRow(ctx, `
			SELECT state,owner_generation,accepted_at,deletion_oauth_session_id,reauth_oauth_session_id,
			       intent_proof_hash,confirmation_handle_hash,intent_expires_at
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 FOR UPDATE
		`, request.JobID, request.Owner).Scan(
			&status, &ownerGeneration, &acceptedAt, &boundSession, &reauthSession,
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
		if _, err := tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,now())
			WHERE account_did=$1 AND lifecycle_state<>'revoked'
		`, request.Owner); err != nil {
			return fmt.Errorf("revoke ordinary CraftSky sessions: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
			    deletion_credential_generation=NULL,
			    revocation_requested_at=COALESCE(revocation_requested_at,now()),
			    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,now()),
			    row_version=row_version+1,updated_at=now()
			WHERE account_did=$1 AND session_id<>$2 AND lifecycle_state<>'revocation_pending'
		`, request.Owner, *reauthSession); err != nil {
			return fmt.Errorf("queue unbound OAuth sessions: %w", err)
		}
		if err := store.adoptUncertainPDSAttemptsTx(ctx, tx, ClaimedOperation{
			JobID: request.JobID, Owner: request.Owner, OwnerGeneration: ownerGeneration,
		}); err != nil {
			return err
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
		RETURNING operation.id,operation.owner_did,operation.owner_generation,
		          operation.attempt_count,operation.lease_token,operation.lease_expires_at
	`, now, workerID, leaseToken, now.Add(leaseDuration)).Scan(
		&operation.JobID, &operation.Owner, &operation.OwnerGeneration,
		&operation.AttemptCount, &operation.LeaseToken, &operation.LeaseExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ClaimedOperation{}, false, nil
	}
	if err != nil {
		return ClaimedOperation{}, false, fmt.Errorf("claim account deletion operation: %w", err)
	}
	return operation, true, nil
}

// AdoptUncertainPDSAttempts inventories only outcome-uncertain ordinary PDS
// writes from the deletion operation's originating owner generation. The
// deterministic key is parsed and scope-checked before it becomes deletion
// authority; malformed, cross-owner, and non-CraftSky keys fail closed.
func (store *Store) AdoptUncertainPDSAttempts(ctx context.Context, operation ClaimedOperation) error {
	if store == nil || store.pool == nil || operation.JobID == uuid.Nil ||
		operation.Owner == "" || operation.OwnerGeneration <= 0 {
		return errors.New("invalid account deletion PDS safety scope")
	}
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT true
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND owner_generation=$3
			  AND state IN ('active','retrying')
			FOR UPDATE
		`, operation.JobID, operation.Owner, operation.OwnerGeneration).Scan(&exists); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return fmt.Errorf("lock account deletion PDS safety scope: %w", err)
		}
		return store.adoptUncertainPDSAttemptsTx(ctx, tx, operation)
	})
}

func (store *Store) adoptUncertainPDSAttemptsTx(
	ctx context.Context,
	tx pgx.Tx,
	operation ClaimedOperation,
) error {
	rows, err := tx.Query(ctx, `
		SELECT operation_id,deterministic_key,remote_deadline
		FROM owner_effect_attempts
		WHERE owner_did=$1 AND owner_generation=$2
		  AND effect_kind='pds_record'
		  AND effect_action='put_record'
		  AND remote_outcome IN ('dispatched','outcome_unknown_pre_transition')
		ORDER BY operation_id
		FOR UPDATE
	`, operation.Owner, operation.OwnerGeneration)
	if err != nil {
		return fmt.Errorf("select uncertain PDS attempts: %w", err)
	}
	type uncertainAttempt struct {
		operationID string
		exactKey    string
		deadline    time.Time
	}
	attempts := make([]uncertainAttempt, 0)
	for rows.Next() {
		var attempt uncertainAttempt
		if err := rows.Scan(&attempt.operationID, &attempt.exactKey, &attempt.deadline); err != nil {
			rows.Close()
			return fmt.Errorf("scan uncertain PDS attempt: %w", err)
		}
		adopt, err := shouldAdoptPDSSafetyURI(operation.Owner, attempt.exactKey)
		if err != nil {
			rows.Close()
			return err
		}
		if !adopt {
			continue
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read uncertain PDS attempts: %w", err)
	}
	rows.Close()

	now := store.now().UTC()
	for _, attempt := range attempts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_safety_tombstones(
				id,operation_id,owner_did,owner_generation,kind,exact_key,
				source_attempt_id,state,remote_deadline,next_attempt_at,
				created_at,updated_at
			) VALUES($1,$2,$3,$4,'pds_record',$5,$6,'pending',$7,$8,$8,$8)
			ON CONFLICT DO NOTHING
		`, uuid.New(), operation.JobID, operation.Owner, operation.OwnerGeneration,
			attempt.exactKey, attempt.operationID, attempt.deadline, now); err != nil {
			return fmt.Errorf("adopt uncertain PDS attempt: %w", err)
		}
	}
	return nil
}

// shouldAdoptPDSSafetyURI separates validation from deletion authority. Every
// recorded PDS effect must still be a well-formed exact key owned by the
// deleting DID, but only registered social.craftsky.* collections may be
// promoted into the deletion worker's narrow delete capability. Valid
// app.bsky.* attempts remain in the lifecycle effect ledger for visibility and
// reconciliation; they cannot block or broaden CraftSky account deletion.
func shouldAdoptPDSSafetyURI(owner syntax.DID, raw string) (bool, error) {
	uri, err := syntax.ParseATURI(raw)
	if err != nil || uri.Authority().String() != owner.String() || uri.RecordKey().String() == "" {
		return false, errors.New("PDS safety key is malformed or belongs to another owner")
	}
	return isDeletionCollection(uri.Collection()), nil
}

func parsePDSSafetyURI(owner syntax.DID, raw string) (syntax.ATURI, error) {
	adopt, err := shouldAdoptPDSSafetyURI(owner, raw)
	if err != nil || !adopt {
		return "", errors.New("PDS safety key is outside the exact deletion scope")
	}
	uri, _ := syntax.ParseATURI(raw)
	return uri, nil
}

// ClaimDuePDSSafety leases at most one exact URI from the already-leased
// deletion operation. Limiting each operation attempt to one safety key keeps
// reconciliation bounded even for a large repository.
func (store *Store) ClaimDuePDSSafety(
	ctx context.Context,
	operation ClaimedOperation,
) (PDSSafetyClaim, bool, error) {
	if store == nil || store.pool == nil || operation.JobID == uuid.Nil ||
		operation.Owner == "" || operation.OwnerGeneration <= 0 ||
		operation.LeaseToken == uuid.Nil || operation.LeaseExpiresAt.IsZero() {
		return PDSSafetyClaim{}, false, errors.New("invalid account deletion PDS safety lease")
	}
	now := store.now().UTC()
	var (
		claim  PDSSafetyClaim
		rawURI string
	)
	err := store.pool.QueryRow(ctx, `
		WITH operation AS (
			SELECT id
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND owner_generation=$3
			  AND lease_token=$4 AND lease_expires_at=$5
		), candidate AS (
			SELECT safety.id
			FROM account_deletion_safety_tombstones safety, operation
			WHERE safety.operation_id=operation.id
			  AND safety.owner_did=$2 AND safety.owner_generation=$3
			  AND safety.kind='pds_record'
			  AND (
				(safety.state='pending' AND safety.next_attempt_at<=$6)
				OR (safety.state='reconciling' AND safety.lease_expires_at<=$6)
			  )
			ORDER BY safety.next_attempt_at NULLS FIRST,safety.id
			LIMIT 1 FOR UPDATE SKIP LOCKED
		)
		UPDATE account_deletion_safety_tombstones safety
		SET state='reconciling',attempts=attempts+1,next_attempt_at=NULL,
		    lease_token=$4,lease_expires_at=$5,updated_at=$6
		FROM candidate
		WHERE safety.id=candidate.id
		RETURNING safety.id,safety.operation_id,safety.owner_did,
		          safety.owner_generation,safety.exact_key,safety.source_attempt_id,
		          safety.attempts,safety.lease_token,safety.lease_expires_at
	`, operation.JobID, operation.Owner, operation.OwnerGeneration,
		operation.LeaseToken, operation.LeaseExpiresAt, now).Scan(
		&claim.ID, &claim.OperationID, &claim.Owner, &claim.OwnerGeneration,
		&rawURI, &claim.SourceAttemptID, &claim.Attempts,
		&claim.LeaseToken, &claim.LeaseExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PDSSafetyClaim{}, false, nil
	}
	if err != nil {
		return PDSSafetyClaim{}, false, fmt.Errorf("claim account deletion PDS safety: %w", err)
	}
	claim.URI, err = parsePDSSafetyURI(claim.Owner, rawURI)
	if err != nil {
		return PDSSafetyClaim{}, false, err
	}
	return claim, true, nil
}

// RecordPDSSafetyPending releases a successful exact-key delete/absence check
// back to the bounded reconciler. Without a tested finite PDS settlement
// guarantee, that observation is deliberately not terminal proof.
func (store *Store) RecordPDSSafetyPending(
	ctx context.Context,
	claim PDSSafetyClaim,
	nextAttemptAt time.Time,
	category string,
) error {
	if store == nil || store.pool == nil || claim.ID == uuid.Nil ||
		claim.LeaseToken == uuid.Nil || nextAttemptAt.IsZero() {
		return errors.New("invalid account deletion PDS safety retry")
	}
	result, err := store.pool.Exec(ctx, `
		UPDATE account_deletion_safety_tombstones
		SET state='pending',next_attempt_at=$3,lease_token=NULL,
		    lease_expires_at=NULL,last_result_category=$4,updated_at=$5
		WHERE id=$1 AND lease_token=$2 AND state='reconciling'
	`, claim.ID, claim.LeaseToken, nextAttemptAt.UTC(), category, store.now().UTC())
	if err != nil {
		return fmt.Errorf("record account deletion PDS safety retry: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrSafetyLeaseLost
	}
	return nil
}

func (store *Store) SafetyConverged(ctx context.Context, operation ClaimedOperation) (bool, error) {
	if store == nil || store.pool == nil || operation.JobID == uuid.Nil ||
		operation.Owner == "" || operation.OwnerGeneration <= 0 {
		return false, errors.New("invalid account deletion safety convergence scope")
	}
	var converged bool
	err := store.pool.QueryRow(ctx, `
		SELECT NOT EXISTS(
			SELECT 1 FROM account_deletion_safety_tombstones
			WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=$3
			  AND state<>'settled'
		)
	`, operation.JobID, operation.Owner, operation.OwnerGeneration).Scan(&converged)
	if err != nil {
		return false, fmt.Errorf("read account deletion safety convergence: %w", err)
	}
	return converged, nil
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

// EligibleDeletionCredential returns the one accepted, childless-capable
// deletion authority that logout-all may preserve. The operation row is
// locked before the auth layer locks parent rows, preserving the repository's
// lifecycle -> operation -> parent lock order.
func (store *Store) EligibleDeletionCredential(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	authority ownerlifecycle.Lifecycle,
) (string, int64, bool, error) {
	if store == nil || tx == nil || owner == "" || authority.Owner != owner ||
		authority.State != ownerlifecycle.StateDeleting || authority.Generation <= 2 {
		return "", 0, false, nil
	}
	var sessionID string
	var credentialGeneration int64
	err := tx.QueryRow(ctx, `
		SELECT operation.deletion_oauth_session_id,operation.deletion_credential_generation
		FROM account_deletion_operations operation
		JOIN oauth_sessions parent
		  ON parent.account_did=operation.owner_did
		 AND parent.session_id=operation.deletion_oauth_session_id
		WHERE operation.owner_did=$1
		  AND operation.owner_generation=$2
		  AND operation.state IN ('active','retrying')
		  AND operation.accepted_at IS NOT NULL
		  AND operation.deletion_oauth_session_id IS NOT NULL
		  AND operation.deletion_credential_generation IS NOT NULL
		  AND parent.lifecycle_state='deletion_only'
		  AND parent.deletion_operation_id=operation.id
		  AND parent.deletion_credential_generation=operation.deletion_credential_generation
		  AND parent.owner_generation=$3
		  AND parent.auth_epoch=$4
		FOR UPDATE OF operation
	`, owner, authority.Generation-2, authority.Generation-1, authority.AuthEpoch).Scan(
		&sessionID, &credentialGeneration,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, fmt.Errorf("lock eligible deletion credential: %w", err)
	}
	if sessionID == "" || credentialGeneration <= 0 {
		return "", 0, false, nil
	}
	return sessionID, credentialGeneration, true, nil
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

// BoundDeletionCredential returns the exact accepted OAuth authority owned by
// the caller's current worker lease. The lease token and originating owner
// generation prevent a stale worker from retaining destructive PDS access.
func (store *Store) BoundDeletionCredential(
	ctx context.Context,
	operation ClaimedOperation,
) (auth.DeletionSessionAuthority, error) {
	var authority auth.DeletionSessionAuthority
	if store == nil || store.pool == nil || operation.JobID == uuid.Nil ||
		operation.Owner == "" || operation.OwnerGeneration <= 0 || operation.LeaseToken == uuid.Nil {
		return authority, ErrBoundOAuthUnauthorized
	}
	authority.OperationID = operation.JobID
	authority.OwnerGeneration = operation.OwnerGeneration
	authority.LeaseToken = operation.LeaseToken
	err := store.pool.QueryRow(ctx, `
		SELECT deletion_oauth_session_id,deletion_credential_generation
		FROM account_deletion_operations
		WHERE id=$1 AND owner_did=$2 AND owner_generation=$3
		  AND state IN ('active','retrying') AND accepted_at IS NOT NULL
		  AND deletion_oauth_session_id IS NOT NULL
		  AND deletion_credential_generation IS NOT NULL
		  AND lease_token=$4 AND lease_expires_at>$5
	`, operation.JobID, operation.Owner, operation.OwnerGeneration,
		operation.LeaseToken, store.now().UTC()).Scan(
		&authority.SessionID, &authority.CredentialGeneration,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.DeletionSessionAuthority{}, ErrBoundOAuthUnauthorized
	}
	if err != nil {
		return auth.DeletionSessionAuthority{}, fmt.Errorf("read bound deletion credential: %w", err)
	}
	return authority, nil
}

var _ auth.DeletionCredentialExemption = (*Store)(nil)

// CompleteAttempt removes the only retained deletion authority and operation
// atomically. A successful processor run has already completed private cleanup
// and the PDS deleter's final empty scan.
func (store *Store) CompleteAttempt(ctx context.Context, operation ClaimedOperation) error {
	return errors.New("account deletion completion requires lifecycle coordination")
}

func (store *Store) CompleteParticipant(
	operation ClaimedOperation,
) ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if operation.JobID == uuid.Nil || operation.Owner == "" || operation.OwnerGeneration <= 0 ||
			operation.LeaseToken == uuid.Nil || before.Owner != operation.Owner || after.Owner != operation.Owner ||
			before.State != ownerlifecycle.StateDeleting || after.State != ownerlifecycle.StateDeparted {
			return ErrOperationNotFound
		}
		var sessionID string
		var credentialGeneration int64
		if err := tx.QueryRow(ctx, `
			SELECT deletion_oauth_session_id,deletion_credential_generation
			FROM account_deletion_operations
			WHERE id=$1 AND owner_did=$2 AND owner_generation=$3 AND lease_token=$4
			FOR UPDATE
		`, operation.JobID, operation.Owner, operation.OwnerGeneration, operation.LeaseToken).Scan(
			&sessionID, &credentialGeneration,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrOperationNotFound
			}
			return err
		}
		var safetyPending bool
		if err := tx.QueryRow(ctx, `
			SELECT
				EXISTS(
					SELECT 1 FROM account_deletion_safety_tombstones
					WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=$3
					  AND state<>'settled'
				)
				OR EXISTS(
					SELECT 1
					FROM owner_effect_attempts effect
					WHERE effect.owner_did=$2 AND effect.owner_generation=$3
					  AND effect.effect_kind='pds_record'
					  AND effect.effect_action='put_record'
					  AND split_part(effect.deterministic_key, '/', 3)=$2
					  AND split_part(effect.deterministic_key, '/', 4)=ANY(ARRAY[
						'social.craftsky.actor.profile',
						'social.craftsky.feed.post',
						'social.craftsky.feed.like',
						'social.craftsky.feed.repost'
					  ]::text[])
					  AND split_part(effect.deterministic_key, '/', 5)<>''
					  AND effect.remote_outcome IN ('dispatched','outcome_unknown_pre_transition')
					  AND NOT EXISTS(
						SELECT 1 FROM account_deletion_safety_tombstones safety
						WHERE safety.operation_id=$1
						  AND safety.source_attempt_id=effect.operation_id
					  )
				)
		`, operation.JobID, operation.Owner, operation.OwnerGeneration).Scan(&safetyPending); err != nil {
			return fmt.Errorf("verify account deletion safety convergence: %w", err)
		}
		if safetyPending {
			return ErrSafetyPending
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM account_deletion_safety_tombstones
			WHERE operation_id=$1 AND owner_did=$2 AND owner_generation=$3
		`, operation.JobID, operation.Owner, operation.OwnerGeneration); err != nil {
			return fmt.Errorf("remove account deletion safety tombstones: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1`, operation.JobID); err != nil {
			return fmt.Errorf("remove account deletion operation: %w", err)
		}
		return nil
	}
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
			if _, err := tx.Exec(ctx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
				    deletion_credential_generation=NULL,
				    revocation_requested_at=COALESCE(revocation_requested_at,now()),
				    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,now()),
				    row_version=row_version+1,updated_at=now()
				WHERE account_did=$1 AND session_id=$2
			`, owner, oldSessionID); err != nil {
				return err
			}
		}
		refreshed = true
		return nil
	})
	return refreshed, err
}
