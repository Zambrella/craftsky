package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

type SessionExpiryProcessorOptions struct {
	Lifecycle *SessionLifecycleService
	BatchSize int
}

// SessionExpiryProcessor eagerly applies the same local denial states that
// request-time authentication enforces. Correctness does not depend on this
// processor: every bearer and privileged parent load still checks its own
// deadline. The processor only bounds how long expired rows remain live.
type SessionExpiryProcessor struct {
	lifecycle *SessionLifecycleService
	batchSize int
}

func NewSessionExpiryProcessor(options SessionExpiryProcessorOptions) (*SessionExpiryProcessor, error) {
	if options.Lifecycle == nil || options.Lifecycle.pool == nil || options.Lifecycle.owners == nil ||
		options.BatchSize < 1 || options.BatchSize > maximumAuthCleanupBatchSize {
		return nil, errors.New("invalid session expiry processor options")
	}
	return &SessionExpiryProcessor{lifecycle: options.Lifecycle, batchSize: options.BatchSize}, nil
}

func (processor *SessionExpiryProcessor) ProcessBatch(ctx context.Context) (int, error) {
	if processor == nil || processor.lifecycle == nil {
		return 0, errors.New("session expiry processor is unavailable")
	}
	return processor.lifecycle.sweepExpiredSessions(ctx, processor.batchSize)
}

type expiredParentCandidate struct {
	Owner       syntax.DID
	SessionID   string
	OperationID *uuid.UUID
}

type lockedHandoffExchange struct {
	ID        uuid.UUID
	State     string
	ExpiresAt time.Time
}

type lockedHandoffReceipt struct {
	ID        uuid.UUID
	State     string
	ConfirmBy time.Time
}

// sweepExpiredSessions spends at most batch transition attempts. Discovery is
// lock-free and every candidate is revalidated under the repository-wide
// owner fence and row-lock order before mutation.
func (service *SessionLifecycleService) sweepExpiredSessions(ctx context.Context, batch int) (int, error) {
	if service == nil || service.pool == nil || service.owners == nil || batch < 1 ||
		batch > maximumAuthCleanupBatchSize {
		return 0, errors.New("invalid session expiry sweep")
	}
	now := service.now().UTC()
	parents, err := service.discoverExpiredParents(ctx, now, batch)
	if err != nil {
		return 0, err
	}
	processed := 0
	var processingErrors []error
	for _, candidate := range parents {
		changed, err := service.expireParentCandidate(ctx, candidate, now)
		if err != nil {
			processingErrors = append(processingErrors, err)
			continue
		}
		if changed {
			processed++
		}
	}

	remaining := batch - len(parents)
	if remaining <= 0 {
		return processed, errors.Join(processingErrors...)
	}
	children, err := service.discoverExpiredChildren(ctx, now, remaining)
	if err != nil {
		processingErrors = append(processingErrors, err)
		return processed, errors.Join(processingErrors...)
	}
	for _, candidate := range children {
		changed, err := service.expireChildCandidate(ctx, candidate, now)
		if err != nil {
			processingErrors = append(processingErrors, err)
			continue
		}
		if changed {
			processed++
		}
	}
	return processed, errors.Join(processingErrors...)
}

func (service *SessionLifecycleService) discoverExpiredParents(
	ctx context.Context,
	now time.Time,
	batch int,
) ([]expiredParentCandidate, error) {
	rows, err := service.pool.Query(ctx, `
		SELECT parent.account_did,parent.session_id,parent.deletion_operation_id
		FROM oauth_sessions parent
		LEFT JOIN account_deletion_operations operation
		  ON operation.id=parent.deletion_operation_id AND operation.owner_did=parent.account_did
		WHERE (
		    parent.lifecycle_state IN ('active','pending_handoff')
		    AND (
		      parent.absolute_expires_at <= $1
		      OR (
		        parent.lifecycle_state='pending_handoff'
		        AND EXISTS (
		          SELECT 1 FROM oauth_handoff_exchanges exchange
		          LEFT JOIN oauth_handoff_receipts receipt ON receipt.exchange_id=exchange.id
		          WHERE exchange.owner_did=parent.account_did
		            AND exchange.oauth_session_id=parent.session_id
		            AND (
		              (exchange.state IN ('ready','redeemed') AND exchange.expires_at <= $1)
		              OR (receipt.state='pending' AND receipt.confirm_by <= $1)
		            )
		        )
		      )
		    )
		  ) OR (
		    parent.lifecycle_state='deletion_only'
		    AND parent.absolute_expires_at <= $1
		    AND operation.state IN ('active','retrying')
		    AND operation.deletion_oauth_session_id=parent.session_id
		  )
		ORDER BY parent.absolute_expires_at,parent.account_did,parent.session_id
		LIMIT $2
	`, now, batch)
	if err != nil {
		return nil, fmt.Errorf("discover expired OAuth parents: %w", err)
	}
	defer rows.Close()
	candidates := make([]expiredParentCandidate, 0, batch)
	for rows.Next() {
		var candidate expiredParentCandidate
		if err := rows.Scan(&candidate.Owner, &candidate.SessionID, &candidate.OperationID); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (service *SessionLifecycleService) expireParentCandidate(
	ctx context.Context,
	candidate expiredParentCandidate,
	now time.Time,
) (bool, error) {
	changed := false
	err := service.owners.WithExistingAuth(ctx, candidate.Owner, func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
		return service.owners.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
			var ownerState ownerlifecycle.State
			var authEpoch int64
			if err := tx.QueryRow(authCtx, `
				SELECT state,auth_epoch FROM owner_lifecycles
				WHERE owner_did=$1 FOR SHARE
			`, candidate.Owner).Scan(&ownerState, &authEpoch); err != nil {
				return err
			}
			if ownerState == ownerlifecycle.StateTerminal {
				return ownerlifecycle.ErrTerminalOwner
			}

			// Authorization requests precede account-deletion operations and
			// OAuth parents in the global row-lock order.
			requestRows, err := tx.Query(authCtx, `
				SELECT state FROM oauth_auth_requests
				WHERE state=$1 ORDER BY state FOR UPDATE
			`, candidate.SessionID)
			if err != nil {
				return fmt.Errorf("lock expired parent auth request: %w", err)
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

			var operationState string
			var operationDeletionSession *string
			if candidate.OperationID != nil {
				if err := tx.QueryRow(authCtx, `
					SELECT state,deletion_oauth_session_id
					FROM account_deletion_operations
					WHERE id=$1 AND owner_did=$2 FOR UPDATE
				`, *candidate.OperationID, candidate.Owner).Scan(
					&operationState, &operationDeletionSession,
				); errors.Is(err, pgx.ErrNoRows) {
					return nil
				} else if err != nil {
					return fmt.Errorf("lock expired deletion credential operation: %w", err)
				}
			}

			var parentState string
			var parentVersion int64
			var parentExpiry time.Time
			var currentOperationID *uuid.UUID
			if err := tx.QueryRow(authCtx, `
				SELECT lifecycle_state,row_version,absolute_expires_at,deletion_operation_id
				FROM oauth_sessions
				WHERE account_did=$1 AND session_id=$2 FOR UPDATE
			`, candidate.Owner, candidate.SessionID).Scan(
				&parentState, &parentVersion, &parentExpiry, &currentOperationID,
			); errors.Is(err, pgx.ErrNoRows) {
				return nil
			} else if err != nil {
				return fmt.Errorf("lock expired OAuth parent: %w", err)
			}
			if parentState != "active" && parentState != "pending_handoff" && parentState != "deletion_only" {
				return nil
			}
			if parentState == "deletion_only" {
				if candidate.OperationID == nil || currentOperationID == nil ||
					*currentOperationID != *candidate.OperationID ||
					(operationState != "active" && operationState != "retrying") ||
					operationDeletionSession == nil || *operationDeletionSession != candidate.SessionID {
					return nil
				}
			} else if currentOperationID != nil {
				return nil
			}

			childRows, err := tx.Query(authCtx, `
				SELECT token_hash FROM craftsky_sessions
				WHERE account_did=$1 AND oauth_session_id=$2
				ORDER BY token_hash FOR UPDATE
			`, candidate.Owner, candidate.SessionID)
			if err != nil {
				return fmt.Errorf("lock expired parent children: %w", err)
			}
			for childRows.Next() {
				var tokenHash []byte
				if err := childRows.Scan(&tokenHash); err != nil {
					childRows.Close()
					return err
				}
			}
			if err := childRows.Err(); err != nil {
				childRows.Close()
				return err
			}
			childRows.Close()

			exchanges, err := lockParentHandoffExchanges(authCtx, tx, candidate.Owner, candidate.SessionID)
			if err != nil {
				return err
			}
			receipts, err := lockParentHandoffReceipts(authCtx, tx, candidate.Owner, candidate.SessionID)
			if err != nil {
				return err
			}
			expired := !parentExpiry.After(now)
			if parentState == "pending_handoff" && !expired {
				for _, exchange := range exchanges {
					if (exchange.State == "ready" || exchange.State == "redeemed") && !exchange.ExpiresAt.After(now) {
						expired = true
						break
					}
				}
				if !expired {
					for _, receipt := range receipts {
						if receipt.State == "pending" && !receipt.ConfirmBy.After(now) {
							expired = true
							break
						}
					}
				}
			}
			if !expired {
				return nil
			}

			if parentState == "deletion_only" {
				command, err := tx.Exec(authCtx, `
					UPDATE account_deletion_operations
					SET state='reauth_required',deletion_oauth_session_id=NULL,
					    deletion_credential_generation=NULL,error_category='reauthentication',
					    next_attempt_at=NULL,lease_owner=NULL,lease_token=NULL,
					    lease_expires_at=NULL,updated_at=$3
					WHERE id=$1 AND owner_did=$2 AND state IN ('active','retrying')
					  AND deletion_oauth_session_id=$4
				`, *candidate.OperationID, candidate.Owner, now, candidate.SessionID)
				if err != nil {
					return fmt.Errorf("expire deletion credential operation: %w", err)
				}
				if command.RowsAffected() != 1 {
					return nil
				}
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_auth_requests
				SET request_state='revoked',consumed_at=COALESCE(consumed_at,$2),
				    exchange_finished_at=CASE
				      WHEN exchange_started_at IS NOT NULL THEN COALESCE(exchange_finished_at,$2)
				      ELSE exchange_finished_at END
				WHERE state=$1 AND request_state<>'exchange_ambiguous'
			`, candidate.SessionID, now); err != nil {
				return fmt.Errorf("revoke expired parent auth request: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE craftsky_sessions
				SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,$3)
				WHERE account_did=$1 AND oauth_session_id=$2 AND lifecycle_state<>'revoked'
			`, candidate.Owner, candidate.SessionID, now); err != nil {
				return fmt.Errorf("revoke expired parent children: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_handoff_exchanges
				SET state='expired',code_hash=NULL,consumed_at=COALESCE(consumed_at,$3),updated_at=$3
				WHERE owner_did=$1 AND oauth_session_id=$2 AND state IN ('ready','redeemed')
			`, candidate.Owner, candidate.SessionID, now); err != nil {
				return fmt.Errorf("expire parent handoff exchanges: %w", err)
			}
			if _, err := tx.Exec(authCtx, `
				UPDATE oauth_handoff_receipts receipt
				SET state='expired',ciphertext=NULL,nonce=NULL,
				    consumed_at=COALESCE(receipt.consumed_at,$3),updated_at=$3
				FROM oauth_handoff_exchanges exchange
				WHERE exchange.id=receipt.exchange_id AND exchange.owner_did=$1
				  AND exchange.oauth_session_id=$2 AND receipt.state='pending'
			`, candidate.Owner, candidate.SessionID, now); err != nil {
				return fmt.Errorf("expire parent handoff receipts: %w", err)
			}
			command, err := tx.Exec(authCtx, `
				UPDATE oauth_sessions
				SET lifecycle_state='revocation_pending',deletion_operation_id=NULL,
				    deletion_credential_generation=NULL,
				    revocation_requested_at=COALESCE(revocation_requested_at,$4),
				    cleanup_next_attempt_at=COALESCE(cleanup_next_attempt_at,$4),
				    cleanup_lease_token=NULL,cleanup_lease_expires_at=NULL,
				    row_version=row_version+1,updated_at=$4
				WHERE account_did=$1 AND session_id=$2 AND row_version=$3
				  AND lifecycle_state IN ('active','pending_handoff','deletion_only')
			`, candidate.Owner, candidate.SessionID, parentVersion, now)
			if err != nil {
				return fmt.Errorf("queue expired OAuth parent: %w", err)
			}
			if command.RowsAffected() != 1 {
				return ErrSessionVersionChanged
			}
			if err := enqueueInactiveParentDeviceCleanup(authCtx, tx, candidate.Owner, candidate.SessionID, authEpoch, now); err != nil {
				return err
			}
			changed = true
			return nil
		})
	})
	if errors.Is(err, ownerlifecycle.ErrTerminalOwner) || errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return changed, err
}

func lockParentHandoffExchanges(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	sessionID string,
) ([]lockedHandoffExchange, error) {
	rows, err := tx.Query(ctx, `
		SELECT id,state,expires_at FROM oauth_handoff_exchanges
		WHERE owner_did=$1 AND oauth_session_id=$2 ORDER BY id FOR UPDATE
	`, owner, sessionID)
	if err != nil {
		return nil, fmt.Errorf("lock expired parent handoff exchanges: %w", err)
	}
	defer rows.Close()
	var exchanges []lockedHandoffExchange
	for rows.Next() {
		var exchange lockedHandoffExchange
		if err := rows.Scan(&exchange.ID, &exchange.State, &exchange.ExpiresAt); err != nil {
			return nil, err
		}
		exchanges = append(exchanges, exchange)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return exchanges, nil
}

func lockParentHandoffReceipts(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	sessionID string,
) ([]lockedHandoffReceipt, error) {
	rows, err := tx.Query(ctx, `
		SELECT receipt.id,receipt.state,receipt.confirm_by
		FROM oauth_handoff_receipts receipt
		JOIN oauth_handoff_exchanges exchange ON exchange.id=receipt.exchange_id
		WHERE exchange.owner_did=$1 AND exchange.oauth_session_id=$2
		ORDER BY receipt.id FOR UPDATE OF receipt
	`, owner, sessionID)
	if err != nil {
		return nil, fmt.Errorf("lock expired parent handoff receipts: %w", err)
	}
	defer rows.Close()
	var receipts []lockedHandoffReceipt
	for rows.Next() {
		var receipt lockedHandoffReceipt
		if err := rows.Scan(&receipt.ID, &receipt.State, &receipt.ConfirmBy); err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return receipts, nil
}

type expiredChildCandidate struct {
	Owner     syntax.DID
	SessionID string
	TokenHash []byte
}

func (service *SessionLifecycleService) discoverExpiredChildren(
	ctx context.Context,
	now time.Time,
	batch int,
) ([]expiredChildCandidate, error) {
	rows, err := service.pool.Query(ctx, `
		SELECT account_did,oauth_session_id,token_hash
		FROM craftsky_sessions
		WHERE lifecycle_state='active' AND idle_expires_at <= $1
		ORDER BY idle_expires_at,account_did,oauth_session_id,token_hash
		LIMIT $2
	`, now, batch)
	if err != nil {
		return nil, fmt.Errorf("discover expired CraftSky children: %w", err)
	}
	defer rows.Close()
	candidates := make([]expiredChildCandidate, 0, batch)
	for rows.Next() {
		var candidate expiredChildCandidate
		if err := rows.Scan(&candidate.Owner, &candidate.SessionID, &candidate.TokenHash); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (service *SessionLifecycleService) expireChildCandidate(
	ctx context.Context,
	candidate expiredChildCandidate,
	now time.Time,
) (bool, error) {
	changed := false
	err := service.owners.WithExistingAuth(ctx, candidate.Owner, func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
		return service.owners.WithAuthTransaction(authCtx, func(tx pgx.Tx) error {
			var ownerState ownerlifecycle.State
			var authEpoch int64
			if err := tx.QueryRow(authCtx, `
				SELECT state,auth_epoch FROM owner_lifecycles
				WHERE owner_did=$1 FOR SHARE
			`, candidate.Owner).Scan(&ownerState, &authEpoch); err != nil {
				return err
			}
			if ownerState == ownerlifecycle.StateTerminal {
				return ownerlifecycle.ErrTerminalOwner
			}
			var parentOwner syntax.DID
			if err := tx.QueryRow(authCtx, `
				SELECT account_did FROM oauth_sessions
				WHERE account_did=$1 AND session_id=$2 FOR UPDATE
			`, candidate.Owner, candidate.SessionID).Scan(&parentOwner); errors.Is(err, pgx.ErrNoRows) {
				return nil
			} else if err != nil {
				return fmt.Errorf("lock idle child OAuth parent: %w", err)
			}
			var childState string
			var idleExpiry time.Time
			var deviceID *string
			if err := tx.QueryRow(authCtx, `
				SELECT lifecycle_state,idle_expires_at,last_device_id
				FROM craftsky_sessions WHERE token_hash=$1 FOR UPDATE
			`, candidate.TokenHash).Scan(&childState, &idleExpiry, &deviceID); errors.Is(err, pgx.ErrNoRows) {
				return nil
			} else if err != nil {
				return fmt.Errorf("lock idle CraftSky child: %w", err)
			}
			if childState != "active" || idleExpiry.After(now) {
				return nil
			}
			command, err := tx.Exec(authCtx, `
				UPDATE craftsky_sessions
				SET lifecycle_state='revoked',revoked_at=COALESCE(revoked_at,$2)
				WHERE token_hash=$1 AND lifecycle_state='active' AND idle_expires_at <= $2
			`, candidate.TokenHash, now)
			if err != nil {
				return fmt.Errorf("expire idle CraftSky child: %w", err)
			}
			if command.RowsAffected() != 1 {
				return nil
			}
			if deviceID != nil && *deviceID != "" {
				if err := enqueueInactiveDeviceCleanup(authCtx, tx, candidate.Owner, *deviceID, authEpoch, now); err != nil {
					return err
				}
			}
			changed = true
			return nil
		})
	})
	if errors.Is(err, ownerlifecycle.ErrTerminalOwner) || errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return changed, err
}

func enqueueInactiveParentDeviceCleanup(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	sessionID string,
	authEpoch int64,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO auth_auxiliary_cleanup_jobs(
			id,owner_did,auth_epoch,kind,installation_id,state,next_attempt_at,created_at,updated_at
		)
		SELECT gen_random_uuid(),$1,$3,'installation_push',expired.last_device_id,
		       'pending',$4,$4,$4
		FROM (
		  SELECT DISTINCT last_device_id
		  FROM craftsky_sessions
		  WHERE account_did=$1 AND oauth_session_id=$2
		    AND last_device_id IS NOT NULL AND btrim(last_device_id)<>''
		) expired
		WHERE NOT EXISTS (
		  SELECT 1 FROM craftsky_sessions live
		  WHERE live.account_did=$1 AND live.last_device_id=expired.last_device_id
		    AND live.lifecycle_state='active'
		)
	`, owner, sessionID, authEpoch, now)
	if err != nil {
		return fmt.Errorf("enqueue expired parent device cleanup: %w", err)
	}
	return nil
}

func enqueueInactiveDeviceCleanup(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	deviceID string,
	authEpoch int64,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO auth_auxiliary_cleanup_jobs(
			id,owner_did,auth_epoch,kind,installation_id,state,next_attempt_at,created_at,updated_at
		)
		SELECT $1,$2,$3,'installation_push',$4,'pending',$5,$5,$5
		WHERE NOT EXISTS (
		  SELECT 1 FROM craftsky_sessions
		  WHERE account_did=$2 AND last_device_id=$4 AND lifecycle_state='active'
		)
	`, uuid.New(), owner, authEpoch, deviceID, now)
	if err != nil {
		return fmt.Errorf("enqueue expired child device cleanup: %w", err)
	}
	return nil
}
