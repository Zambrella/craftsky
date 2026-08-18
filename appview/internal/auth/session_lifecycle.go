package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ownerlifecycle"
)

type DeletionCredentialExemption interface {
	EligibleDeletionCredential(
		context.Context,
		pgx.Tx,
		syntax.DID,
		ownerlifecycle.Lifecycle,
	) (sessionID string, credentialGeneration int64, eligible bool, err error)
}

type SessionLifecycleOptions struct {
	Pool              *pgxpool.Pool
	Owners            *ownerlifecycle.Store
	Sessions          *CraftskySessionStore
	DeletionExemption DeletionCredentialExemption
	Now               func() time.Time
}

type SessionLifecycleService struct {
	pool              *pgxpool.Pool
	owners            *ownerlifecycle.Store
	sessions          *CraftskySessionStore
	deletionExemption DeletionCredentialExemption
	now               func() time.Time
}

// DeletionCredentialBinding identifies the sole OAuth parent that may survive
// an accepted deletion transition. The operation and generation are part of
// the authority; a session ID alone is never sufficient.
type DeletionCredentialBinding struct {
	OperationID          uuid.UUID
	SessionID            string
	CredentialGeneration int64
}

func (binding DeletionCredentialBinding) valid() bool {
	return binding.OperationID != uuid.Nil && binding.SessionID != "" && binding.CredentialGeneration > 0
}

// DeletionSessionAuthority adds the claimed worker generation and lease to an
// accepted credential binding. A stale worker must not retain remote-delete
// authority merely because the same OAuth parent is still current.
type DeletionSessionAuthority struct {
	DeletionCredentialBinding
	OwnerGeneration int64
	LeaseToken      uuid.UUID
}

func (authority DeletionSessionAuthority) valid() bool {
	return authority.DeletionCredentialBinding.valid() &&
		authority.OwnerGeneration > 0 && authority.LeaseToken != uuid.Nil
}

func NewSessionLifecycleService(options SessionLifecycleOptions) (*SessionLifecycleService, error) {
	if options.Pool == nil || options.Owners == nil || options.Sessions == nil {
		return nil, errors.New("session lifecycle requires database, owners, and child sessions")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &SessionLifecycleService{
		pool: options.Pool, owners: options.Owners, sessions: options.Sessions,
		deletionExemption: options.DeletionExemption, now: options.Now,
	}, nil
}

// OwnerTransitionParticipant invalidates owner-scoped authentication in the
// same transaction as an owner lifecycle transition. If preserve is non-nil,
// only that exact, childless deletion_only parent survives. The optional
// operation participant runs after authorization-request locks and before
// parent locks, which is the account-deletion-operation position in the
// repository-wide row-lock order.
func (service *SessionLifecycleService) OwnerTransitionParticipant(
	preserve *DeletionCredentialBinding,
	operationParticipant ...ownerlifecycle.TransitionParticipant,
) ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if service == nil || service.pool == nil || service.sessions == nil || before.Owner == "" ||
			before.Owner != after.Owner || len(operationParticipant) > 1 {
			return errors.New("invalid auth lifecycle transition participant")
		}
		if preserve != nil && !preserve.valid() {
			return errors.New("invalid deletion credential preservation binding")
		}
		now := service.now().UTC()

		requestRows, err := tx.Query(ctx, `
			SELECT state FROM oauth_auth_requests
			WHERE owner_did=$1 ORDER BY state FOR UPDATE
		`, after.Owner)
		if err != nil {
			return fmt.Errorf("lock transition auth requests: %w", err)
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
			UPDATE oauth_auth_requests
			SET request_state='revoked',consumed_at=COALESCE(consumed_at,$2),
			    exchange_finished_at=CASE
			      WHEN exchange_started_at IS NOT NULL THEN COALESCE(exchange_finished_at,$2)
			      ELSE exchange_finished_at END
			WHERE owner_did=$1 AND request_state<>'exchange_ambiguous'
		`, after.Owner, now); err != nil {
			return fmt.Errorf("invalidate transition auth requests: %w", err)
		}

		if len(operationParticipant) == 1 && operationParticipant[0] != nil {
			if err := operationParticipant[0](ctx, tx, before, after); err != nil {
				return err
			}
		}

		parentRows, err := tx.Query(ctx, `
			SELECT session_id,lifecycle_state,deletion_operation_id,deletion_credential_generation,auth_epoch
			FROM oauth_sessions WHERE account_did=$1 ORDER BY session_id FOR UPDATE
		`, after.Owner)
		if err != nil {
			return fmt.Errorf("lock transition OAuth parents: %w", err)
		}
		preserved := preserve == nil
		for parentRows.Next() {
			var sessionID, lifecycleState string
			var operationID *uuid.UUID
			var credentialGeneration *int64
			var authEpoch int64
			if err := parentRows.Scan(&sessionID, &lifecycleState, &operationID, &credentialGeneration, &authEpoch); err != nil {
				parentRows.Close()
				return err
			}
			if preserve != nil && sessionID == preserve.SessionID {
				preserved = lifecycleState == "deletion_only" && operationID != nil && *operationID == preserve.OperationID &&
					credentialGeneration != nil && *credentialGeneration == preserve.CredentialGeneration &&
					authEpoch == before.AuthEpoch
			}
		}
		if err := parentRows.Err(); err != nil {
			parentRows.Close()
			return err
		}
		parentRows.Close()
		if !preserved {
			return errors.New("deletion credential preservation binding changed")
		}

		childRows, err := tx.Query(ctx, `
			SELECT token_hash FROM craftsky_sessions
			WHERE account_did=$1 ORDER BY token_hash FOR UPDATE
		`, after.Owner)
		if err != nil {
			return fmt.Errorf("lock transition CraftSky children: %w", err)
		}
		for childRows.Next() {
			var hash []byte
			if err := childRows.Scan(&hash); err != nil {
				childRows.Close()
				return err
			}
		}
		if err := childRows.Err(); err != nil {
			childRows.Close()
			return err
		}
		childRows.Close()
		if preserve != nil {
			var liveChildren int
			if err := tx.QueryRow(ctx, `
				SELECT count(*) FROM craftsky_sessions
				WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
			`, after.Owner, preserve.SessionID).Scan(&liveChildren); err != nil {
				return err
			}
			if liveChildren != 0 {
				return errors.New("deletion credential has a live CraftSky child")
			}
		}

		exchangeRows, err := tx.Query(ctx, `
			SELECT id FROM oauth_handoff_exchanges
			WHERE owner_did=$1 ORDER BY id FOR UPDATE
		`, after.Owner)
		if err != nil {
			return fmt.Errorf("lock transition handoff exchanges: %w", err)
		}
		for exchangeRows.Next() {
			var id uuid.UUID
			if err := exchangeRows.Scan(&id); err != nil {
				exchangeRows.Close()
				return err
			}
		}
		if err := exchangeRows.Err(); err != nil {
			exchangeRows.Close()
			return err
		}
		exchangeRows.Close()
		receiptRows, err := tx.Query(ctx, `
			SELECT receipt.id FROM oauth_handoff_receipts receipt
			JOIN oauth_handoff_exchanges exchange ON exchange.id=receipt.exchange_id
			WHERE exchange.owner_did=$1 ORDER BY receipt.id FOR UPDATE OF receipt
		`, after.Owner)
		if err != nil {
			return fmt.Errorf("lock transition handoff receipts: %w", err)
		}
		for receiptRows.Next() {
			var id uuid.UUID
			if err := receiptRows.Scan(&id); err != nil {
				receiptRows.Close()
				return err
			}
		}
		if err := receiptRows.Err(); err != nil {
			receiptRows.Close()
			return err
		}
		receiptRows.Close()

		if _, err := tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,$2)
			WHERE account_did=$1 AND lifecycle_state<>'revoked'
		`, after.Owner, now); err != nil {
			return fmt.Errorf("revoke transition children: %w", err)
		}
		preservedSession := ""
		if preserve != nil {
			preservedSession = preserve.SessionID
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_sessions
			SET lifecycle_state='revocation_pending',
			    deletion_operation_id=NULL,deletion_credential_generation=NULL,
			    revocation_requested_at=COALESCE(revocation_requested_at,$2),
			    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,$2),
			    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
			    row_version=row_version+1,updated_at=$2
			WHERE account_did=$1 AND session_id<>$3 AND lifecycle_state<>'revocation_pending'
		`, after.Owner, now, preservedSession); err != nil {
			return fmt.Errorf("queue transition OAuth parents: %w", err)
		}
		if preserve != nil {
			command, err := tx.Exec(ctx, `
				UPDATE oauth_sessions
				SET auth_epoch=$5,row_version=row_version+1,updated_at=$6
				WHERE account_did=$1 AND session_id=$2
				  AND lifecycle_state='deletion_only'
				  AND deletion_operation_id=$3
				  AND deletion_credential_generation=$4
				  AND auth_epoch=$7
			`, after.Owner, preserve.SessionID, preserve.OperationID,
				preserve.CredentialGeneration, after.AuthEpoch, now, before.AuthEpoch)
			if err != nil {
				return fmt.Errorf("rebase accepted deletion credential: %w", err)
			}
			if command.RowsAffected() != 1 {
				return errors.New("deletion credential preservation binding changed")
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_handoff_exchanges
			SET state='revoked',code_hash=NULL,consumed_at=COALESCE(consumed_at,$2),updated_at=$2
			WHERE owner_did=$1 AND state IN ('ready','redeemed')
		`, after.Owner, now); err != nil {
			return fmt.Errorf("revoke transition exchanges: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE oauth_handoff_receipts receipt
			SET state='revoked',ciphertext=NULL,nonce=NULL,
			    consumed_at=COALESCE(receipt.consumed_at,$2),updated_at=$2
			FROM oauth_handoff_exchanges exchange
			WHERE exchange.id=receipt.exchange_id AND exchange.owner_did=$1 AND receipt.state='pending'
		`, after.Owner, now); err != nil {
			return fmt.Errorf("revoke transition receipts: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO auth_auxiliary_cleanup_jobs(
				id,owner_did,auth_epoch,kind,state,next_attempt_at,created_at,updated_at
			) VALUES($1,$2,$3,'account_push','pending',$4,$4,$4)
		`, uuid.New(), after.Owner, after.AuthEpoch, now); err != nil {
			return fmt.Errorf("enqueue transition auth cleanup: %w", err)
		}
		return nil
	}
}

// RevokeOne commits child invalidation and its installation-cleanup job in one
// local transaction. It does not call push or the authorization server.
func (service *SessionLifecycleService) RevokeOne(
	ctx context.Context,
	owner syntax.DID,
	token string,
	installationID string,
) error {
	if owner == "" || token == "" || installationID == "" {
		return ErrCraftskySessionNotFound
	}
	hash := sha256.Sum256([]byte(token))
	var discoveredOwner syntax.DID
	var parentSessionID string
	if err := service.pool.QueryRow(ctx, `
		SELECT account_did,oauth_session_id
		FROM craftsky_sessions WHERE token_hash=$1
	`, hash[:]).Scan(&discoveredOwner, &parentSessionID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCraftskySessionNotFound
		}
		return fmt.Errorf("discover child logout session: %w", err)
	}
	if discoveredOwner != owner {
		return ErrCraftskySessionNotFound
	}
	return pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		var authEpoch int64
		if err := tx.QueryRow(ctx, `
			SELECT auth_epoch FROM owner_lifecycles WHERE owner_did=$1 FOR SHARE
		`, owner).Scan(&authEpoch); err != nil {
			return fmt.Errorf("lock logout owner epoch: %w", err)
		}
		var parentOwner syntax.DID
		if err := tx.QueryRow(ctx, `
			SELECT account_did FROM oauth_sessions
			WHERE account_did=$1 AND session_id=$2 FOR SHARE
		`, owner, parentSessionID).Scan(&parentOwner); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCraftskySessionNotFound
			}
			return fmt.Errorf("lock child logout parent: %w", err)
		}
		var storedOwner syntax.DID
		var storedParent string
		var childState string
		if err := tx.QueryRow(ctx, `
			SELECT account_did,oauth_session_id,lifecycle_state
			FROM craftsky_sessions WHERE token_hash=$1 FOR UPDATE
		`, hash[:]).Scan(&storedOwner, &storedParent, &childState); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCraftskySessionNotFound
			}
			return fmt.Errorf("lock child logout session: %w", err)
		}
		if storedOwner != owner || parentOwner != owner || storedParent != parentSessionID {
			return ErrCraftskySessionNotFound
		}
		if childState == "revoked" {
			return nil
		}
		now := service.now().UTC()
		if _, err := tx.Exec(ctx, `
			UPDATE craftsky_sessions
			SET lifecycle_state='revoked',revoked_at=$2
			WHERE token_hash=$1 AND lifecycle_state<>'revoked'
		`, hash[:], now); err != nil {
			return fmt.Errorf("revoke child session: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO auth_auxiliary_cleanup_jobs(
				id,owner_did,auth_epoch,kind,installation_id,state,next_attempt_at,created_at,updated_at
			) VALUES($1,$2,$3,'installation_push',$4,'pending',$5,$5,$5)
		`, uuid.New(), owner, authEpoch, installationID, now); err != nil {
			return fmt.Errorf("enqueue installation auth cleanup: %w", err)
		}
		return nil
	})
}

// RevokeAllForDID uses the owner auth epoch as its DID-wide linearization
// point, then invalidates every old-epoch login artifact before returning.
// The optional deletion exemption can preserve only the exact childless,
// accepted-job deletion credential and rebase that parent to the new epoch.
func (service *SessionLifecycleService) RevokeAllForDID(ctx context.Context, owner syntax.DID) error {
	if owner == "" {
		return ErrCraftskySessionNotFound
	}
	return service.owners.WithExistingAuth(ctx, owner, func(authCtx context.Context, authority ownerlifecycle.Lifecycle) error {
		return service.owners.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
			var current ownerlifecycle.Lifecycle
			if err := tx.QueryRow(authCtx, `
				SELECT owner_did,state,generation,auth_epoch,transition_reason,
				       transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
				FROM owner_lifecycles WHERE owner_did=$1 FOR UPDATE
			`, owner).Scan(
				&current.Owner, &current.State, &current.Generation, &current.AuthEpoch,
				&current.TransitionReason, &current.TransitionedAt, &current.TerminalAt,
				&current.PurgeCompletedAt, &current.CreatedAt, &current.UpdatedAt,
			); err != nil {
				return err
			}
			if current.State == ownerlifecycle.StateTerminal || current.AuthEpoch != authority.AuthEpoch {
				return ownerlifecycle.ErrTerminalOwner
			}

			var exemptSession string
			var exemptGeneration int64
			if service.deletionExemption != nil {
				var eligible bool
				var err error
				exemptSession, exemptGeneration, eligible, err = service.deletionExemption.EligibleDeletionCredential(
					authCtx, tx, owner, current,
				)
				if err != nil {
					return err
				}
				if !eligible {
					exemptSession = ""
					exemptGeneration = 0
				}
			}

			now := service.now().UTC()
			var newEpoch int64
			if err := tx.QueryRow(authCtx, `
				UPDATE owner_lifecycles
				SET auth_epoch=auth_epoch+1,transition_reason='logoutAll',
				    transitioned_at=$2,updated_at=$2
				WHERE owner_did=$1 AND auth_epoch=$3
				RETURNING auth_epoch
			`, owner, now, current.AuthEpoch).Scan(&newEpoch); err != nil {
				return fmt.Errorf("advance logout-all auth epoch: %w", err)
			}

			if _, err := tx.Exec(authCtx, `
				SELECT state FROM oauth_auth_requests
				WHERE owner_did=$1 ORDER BY state FOR UPDATE
			`, owner); err != nil {
				return fmt.Errorf("lock logout-all auth requests: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_auth_requests
				SET request_state='revoked',consumed_at=COALESCE(consumed_at,$2),
				    exchange_finished_at=CASE
				      WHEN exchange_started_at IS NOT NULL THEN COALESCE(exchange_finished_at,$2)
				      ELSE exchange_finished_at END
				WHERE owner_did=$1 AND auth_epoch<$3 AND request_state<>'exchange_ambiguous'
			`, owner, now, newEpoch); err != nil {
				return fmt.Errorf("invalidate logout-all auth requests: %w", err)
			}

			rows, err := tx.Query(authCtx, `
				SELECT session_id FROM oauth_sessions
				WHERE account_did=$1 ORDER BY session_id FOR UPDATE
			`, owner)
			if err != nil {
				return fmt.Errorf("lock logout-all parents: %w", err)
			}
			for rows.Next() {
				var ignored string
				if err := rows.Scan(&ignored); err != nil {
					rows.Close()
					return err
				}
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			rows.Close()

			if _, err := tx.Exec(authCtx, `
				SELECT token_hash FROM craftsky_sessions
				WHERE account_did=$1 ORDER BY token_hash FOR UPDATE
			`, owner); err != nil {
				return fmt.Errorf("lock logout-all children: %w", err)
			}
			if exemptSession != "" {
				var children int
				if err := tx.QueryRow(authCtx, `
					SELECT count(*) FROM craftsky_sessions
					WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
				`, owner, exemptSession).Scan(&children); err != nil {
					return err
				}
				if children != 0 {
					return errors.New("eligible deletion credential has an active child")
				}
			}

			if _, err := tx.Exec(authCtx, `
				SELECT id FROM oauth_handoff_exchanges
				WHERE owner_did=$1 ORDER BY id FOR UPDATE
			`, owner); err != nil {
				return fmt.Errorf("lock logout-all exchanges: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				SELECT receipt.id FROM oauth_handoff_receipts receipt
				JOIN oauth_handoff_exchanges exchange ON exchange.id=receipt.exchange_id
				WHERE exchange.owner_did=$1 ORDER BY receipt.id FOR UPDATE OF receipt
			`, owner); err != nil {
				return fmt.Errorf("lock logout-all receipts: %w", err)
			}

			if _, err := tx.Exec(authCtx, `
				UPDATE craftsky_sessions
				SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,$2)
				WHERE account_did=$1 AND lifecycle_state<>'revoked'
			`, owner, now); err != nil {
				return fmt.Errorf("revoke logout-all children: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',
				    deletion_operation_id=NULL,deletion_credential_generation=NULL,
				    revocation_requested_at=COALESCE(revocation_requested_at,$2),
				    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,$2),
				    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
				    row_version=row_version+1,updated_at=$2
				WHERE account_did=$1 AND session_id<>$3 AND lifecycle_state<>'revocation_pending'
			`, owner, now, exemptSession); err != nil {
				return fmt.Errorf("queue logout-all parents: %w", err)
			}
			if exemptSession != "" {
				command, err := tx.Exec(authCtx, `
					UPDATE oauth_sessions
					SET auth_epoch=$4,row_version=row_version+1,updated_at=$5
					WHERE account_did=$1 AND session_id=$2
					  AND lifecycle_state='deletion_only'
					  AND deletion_credential_generation=$3
				`, owner, exemptSession, exemptGeneration, newEpoch, now)
				if err != nil {
					return err
				}
				if command.RowsAffected() != 1 {
					return errors.New("deletion credential exemption changed during logout")
				}
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_handoff_exchanges
				SET state='revoked',code_hash=NULL,consumed_at=COALESCE(consumed_at,$2),updated_at=$2
				WHERE owner_did=$1 AND state IN ('ready','redeemed')
			`, owner, now); err != nil {
				return fmt.Errorf("revoke logout-all exchanges: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_handoff_receipts receipt
				SET state='revoked',ciphertext=NULL,nonce=NULL,
				    consumed_at=COALESCE(receipt.consumed_at,$2),updated_at=$2
				FROM oauth_handoff_exchanges exchange
				WHERE exchange.id=receipt.exchange_id AND exchange.owner_did=$1 AND receipt.state='pending'
			`, owner, now); err != nil {
				return fmt.Errorf("revoke logout-all receipts: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				INSERT INTO auth_auxiliary_cleanup_jobs(
					id,owner_did,auth_epoch,kind,state,next_attempt_at,created_at,updated_at
				) VALUES($1,$2,$3,'account_push','pending',$4,$4,$4)
			`, uuid.New(), owner, newEpoch, now); err != nil {
				return fmt.Errorf("enqueue account auth cleanup: %w", err)
			}
			return nil
		})
	})
}
