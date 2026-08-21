package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Terminalize atomically records the irreversible owner tombstone, advances
// the auth epoch, denies all known effects, and installs the finite purge
// component ledger. It performs no per-owner data scan or deletion.
func (store *Store) Terminalize(ctx context.Context, request TerminalizeRequest) (Lifecycle, error) {
	return store.TerminalizeWith(ctx, request, nil)
}

// TerminalizeWith lets the auth/session layer invalidate later row-lock
// classes in the same cardinality-independent tombstone transaction.
func (store *Store) TerminalizeWith(
	ctx context.Context,
	request TerminalizeRequest,
	participant TerminalParticipant,
) (Lifecycle, error) {
	components := TerminalPurgeCatalogue()
	if request.Owner == "" {
		return Lifecycle{}, ErrInvalidOwner
	}
	if !validReason(request.Reason) {
		return Lifecycle{}, errors.New("invalid terminal owner transition")
	}
	var terminal Lifecycle
	var err error
	err = store.fencer.WithExclusive(ctx, []syntax.DID{request.Owner}, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			now := store.now().UTC().Truncate(time.Microsecond)
			current, readErr := scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, request.Owner,
			))
			var before *Lifecycle
			switch {
			case errors.Is(readErr, pgx.ErrNoRows):
				terminal, err = scanLifecycle(tx.QueryRow(fenceCtx, `
					INSERT INTO owner_lifecycles(
						owner_did,state,generation,auth_epoch,transition_reason,
						transitioned_at,terminal_at,created_at,updated_at
					) VALUES($1,'terminal',1,1,$2,$3,$3,$3,$3)
					RETURNING owner_did,state,generation,auth_epoch,transition_reason,
					          transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
				`, request.Owner, strings.TrimSpace(request.Reason), now))
				if err != nil {
					return fmt.Errorf("insert terminal owner tombstone: %w", err)
				}
			case readErr != nil:
				return readErr
			case current.State == StateTerminal:
				before = &current
				terminal = current
			default:
				before = &current
				terminal, err = scanLifecycle(tx.QueryRow(fenceCtx, `
					UPDATE owner_lifecycles
					SET state='terminal',generation=generation+1,auth_epoch=auth_epoch+1,
					    transition_reason=$2,transitioned_at=$3,terminal_at=$3,
					    purge_completed_at=NULL,updated_at=$3
					WHERE owner_did=$1
					RETURNING owner_did,state,generation,auth_epoch,transition_reason,
					          transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
				`, request.Owner, strings.TrimSpace(request.Reason), now))
				if err != nil {
					return fmt.Errorf("write terminal owner tombstone: %w", err)
				}
			}
			if participant != nil {
				if err := participant(fenceCtx, tx, before, terminal); err != nil {
					return err
				}
			}

			if err := closeOwnerEffectsTx(fenceCtx, tx, request.Owner, terminal.Generation, true, now); err != nil {
				return err
			}
			inserted := int64(0)
			for _, component := range components {
				result, err := tx.Exec(fenceCtx, `
					INSERT INTO owner_purge_components(
						owner_did,owner_generation,component,did_role,state,
						next_attempt_at,created_at,updated_at
					) VALUES($1,$2,$3,$4,'pending',$5,$5,$5)
					ON CONFLICT (owner_did,owner_generation,component,did_role) DO NOTHING
				`, request.Owner, terminal.Generation, component.Component, component.DIDRole, now)
				if err != nil {
					return fmt.Errorf("install terminal purge component: %w", err)
				}
				inserted += result.RowsAffected()
			}
			if inserted > 0 && terminal.PurgeCompletedAt != nil {
				terminal.PurgeCompletedAt = nil
				if _, err := tx.Exec(fenceCtx, `
					UPDATE owner_lifecycles
					SET purge_completed_at=NULL,updated_at=$2
					WHERE owner_did=$1 AND state='terminal'
				`, request.Owner, now); err != nil {
					return fmt.Errorf("reopen terminal purge ledger: %w", err)
				}
			}
			return nil
		})
	})
	return terminal, err
}

func (store *Store) ClaimPurgeComponents(
	ctx context.Context,
	request PurgeClaimRequest,
) ([]PurgeClaim, error) {
	if !validBoundedString(request.Worker, 256) || request.LeaseToken == uuid.Nil ||
		request.LeaseDuration <= 0 || request.Limit <= 0 {
		return nil, errors.New("invalid owner purge claim request")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	leaseExpiresAt := now.Add(request.LeaseDuration).Truncate(time.Microsecond)
	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT owner_did,owner_generation,component,did_role
			FROM owner_purge_components
			WHERE next_attempt_at <= $1
			  AND (state='pending' OR (state='running' AND lease_expires_at <= $1))
			ORDER BY next_attempt_at,owner_did,owner_generation,component,did_role
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE owner_purge_components AS component
		SET state='running',attempts=component.attempts+1,
		    lease_owner=$3,lease_token=$4,lease_expires_at=$5,updated_at=$1
		FROM candidates
		WHERE component.owner_did=candidates.owner_did
		  AND component.owner_generation=candidates.owner_generation
		  AND component.component=candidates.component
		  AND component.did_role=candidates.did_role
		RETURNING component.owner_did,component.owner_generation,component.component,
		          component.did_role,component.state,component.attempts,
		          component.lease_owner,component.lease_token,component.lease_expires_at
	`, now, request.Limit, strings.TrimSpace(request.Worker), request.LeaseToken, leaseExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("claim owner purge components: %w", err)
	}
	defer rows.Close()
	claims := make([]PurgeClaim, 0, request.Limit)
	for rows.Next() {
		var claim PurgeClaim
		if err := rows.Scan(
			&claim.Owner,
			&claim.OwnerGeneration,
			&claim.Component,
			&claim.DIDRole,
			&claim.State,
			&claim.Attempts,
			&claim.LeaseOwner,
			&claim.LeaseToken,
			&claim.LeaseExpiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan owner purge claim: %w", err)
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate owner purge claims: %w", err)
	}
	slices.SortFunc(claims, comparePurgeClaims)
	return claims, nil
}

func (store *Store) CompletePurgeComponent(
	ctx context.Context,
	claim PurgeClaim,
	leaseToken uuid.UUID,
) error {
	if claim.Owner == "" || claim.OwnerGeneration <= 0 ||
		!validBoundedString(claim.Component, 128) || !validBoundedString(claim.DIDRole, 64) ||
		leaseToken == uuid.Nil {
		return ErrPurgeLeaseLost
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	result, err := store.pool.Exec(ctx, `
		UPDATE owner_purge_components
		SET state='complete',lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    completed_at=$6,last_error_category=NULL,updated_at=$6
		WHERE owner_did=$1 AND owner_generation=$2 AND component=$3 AND did_role=$4
		  AND state='running' AND lease_token=$5 AND lease_expires_at>$6
	`, claim.Owner, claim.OwnerGeneration, claim.Component, claim.DIDRole, leaseToken, now)
	if err != nil {
		return fmt.Errorf("complete owner purge component: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPurgeLeaseLost
	}
	return nil
}

// ReschedulePurgeComponent durably releases a failed leased component for a
// bounded later retry. The error category is operational metadata only; it
// must not contain owner identifiers or private payload details.
func (store *Store) ReschedulePurgeComponent(
	ctx context.Context,
	claim PurgeClaim,
	leaseToken uuid.UUID,
	nextAttemptAt time.Time,
	errorCategory string,
) error {
	now := store.now().UTC().Truncate(time.Microsecond)
	nextAttemptAt = nextAttemptAt.UTC().Truncate(time.Microsecond)
	if claim.Owner == "" || claim.OwnerGeneration <= 0 ||
		!validBoundedString(claim.Component, 128) || !validBoundedString(claim.DIDRole, 64) ||
		leaseToken == uuid.Nil || !nextAttemptAt.After(now) ||
		!validBoundedString(errorCategory, 128) {
		return ErrPurgeLeaseLost
	}
	result, err := store.pool.Exec(ctx, `
		UPDATE owner_purge_components
		SET state='pending',next_attempt_at=$6,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_error_category=$7,updated_at=$8
		WHERE owner_did=$1 AND owner_generation=$2 AND component=$3 AND did_role=$4
		  AND state='running' AND lease_token=$5 AND lease_expires_at>$8
	`, claim.Owner, claim.OwnerGeneration, claim.Component, claim.DIDRole,
		leaseToken, nextAttemptAt, strings.TrimSpace(errorCategory), now)
	if err != nil {
		return fmt.Errorf("reschedule owner purge component: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrPurgeLeaseLost
	}
	return nil
}

func (store *Store) FinalizeTerminalPurge(
	ctx context.Context,
	owner syntax.DID,
	generation int64,
) (Lifecycle, error) {
	if owner == "" {
		return Lifecycle{}, ErrInvalidOwner
	}
	if generation <= 0 {
		return Lifecycle{}, ErrGenerationChanged
	}
	var lifecycle Lifecycle
	err := store.fencer.WithExclusive(ctx, []syntax.DID{owner}, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			current, err := scanLifecycle(tx.QueryRow(
				fenceCtx, lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`, owner,
			))
			if err != nil {
				return err
			}
			if current.State != StateTerminal {
				return ErrTerminalOwner
			}
			if current.Generation != generation {
				return ErrGenerationChanged
			}
			var total, incomplete int
			if err := tx.QueryRow(fenceCtx, `
				SELECT count(*),count(*) FILTER (WHERE state <> 'complete')
				FROM owner_purge_components
				WHERE owner_did=$1 AND owner_generation=$2
			`, owner, generation).Scan(&total, &incomplete); err != nil {
				return fmt.Errorf("inspect terminal purge ledger: %w", err)
			}
			if total == 0 || incomplete != 0 {
				return ErrPurgeIncomplete
			}
			now := store.now().UTC().Truncate(time.Microsecond)
			lifecycle, err = scanLifecycle(tx.QueryRow(fenceCtx, `
				UPDATE owner_lifecycles
				SET purge_completed_at=COALESCE(purge_completed_at,$3),updated_at=$3
				WHERE owner_did=$1 AND generation=$2 AND state='terminal'
				RETURNING owner_did,state,generation,auth_epoch,transition_reason,
				          transitioned_at,terminal_at,purge_completed_at,created_at,updated_at
			`, owner, generation, now))
			return err
		})
	})
	return lifecycle, err
}

func canonicalPurgeComponents(components []PurgeComponent) ([]PurgeComponent, error) {
	if len(components) == 0 {
		return nil, errors.New("terminal purge catalogue is empty")
	}
	canonical := slices.Clone(components)
	for index := range canonical {
		canonical[index].Component = strings.TrimSpace(canonical[index].Component)
		canonical[index].DIDRole = strings.TrimSpace(canonical[index].DIDRole)
		if !validBoundedString(canonical[index].Component, 128) ||
			!validBoundedString(canonical[index].DIDRole, 64) {
			return nil, errors.New("invalid terminal purge component")
		}
	}
	slices.SortFunc(canonical, func(a, b PurgeComponent) int {
		if comparison := strings.Compare(a.Component, b.Component); comparison != 0 {
			return comparison
		}
		return strings.Compare(a.DIDRole, b.DIDRole)
	})
	canonical = slices.CompactFunc(canonical, func(a, b PurgeComponent) bool {
		return a.Component == b.Component && a.DIDRole == b.DIDRole
	})
	return canonical, nil
}

func comparePurgeClaims(a, b PurgeClaim) int {
	if comparison := strings.Compare(a.Owner.String(), b.Owner.String()); comparison != 0 {
		return comparison
	}
	if a.OwnerGeneration < b.OwnerGeneration {
		return -1
	}
	if a.OwnerGeneration > b.OwnerGeneration {
		return 1
	}
	if comparison := strings.Compare(a.Component, b.Component); comparison != 0 {
		return comparison
	}
	return strings.Compare(a.DIDRole, b.DIDRole)
}
