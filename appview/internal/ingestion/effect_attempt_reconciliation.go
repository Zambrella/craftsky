package ingestion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

const maxPDSAttemptReconciliationPage = 1000

// PDSAttemptDepartureParticipant durably schedules read-only reconciliation
// in the same transition transaction that makes dispatched effects
// non-repeatable. It performs no remote work.
func (store *Store) PDSAttemptDepartureParticipant() ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if before.Owner == "" || before.Owner != after.Owner {
			return errors.New("invalid PDS effect departure participant authority")
		}
		if before.State != ownerlifecycle.StateActive || after.State == ownerlifecycle.StateActive {
			return nil
		}
		return store.enqueueUnresolvedPDSAttemptJobTx(
			ctx, tx, before.Owner, before.Generation, store.now().UTC().Truncate(time.Microsecond),
		)
	}
}

// PDSAttemptTerminalParticipant is the terminal counterpart. Terminalization
// closes every generation, so its reconciliation job covers all unresolved
// Put attempts for the owner.
func (store *Store) PDSAttemptTerminalParticipant() ownerlifecycle.TerminalParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before *ownerlifecycle.Lifecycle,
		terminal ownerlifecycle.Lifecycle,
	) error {
		if terminal.Owner == "" || terminal.State != ownerlifecycle.StateTerminal {
			return errors.New("invalid PDS effect terminal participant authority")
		}
		if before == nil {
			return nil
		}
		return store.enqueueUnresolvedPDSAttemptJobTx(
			ctx, tx, terminal.Owner, 0, store.now().UTC().Truncate(time.Microsecond),
		)
	}
}

func (store *Store) enqueueUnresolvedPDSAttemptJobTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	generation int64,
	now time.Time,
) error {
	if store == nil || tx == nil || owner == "" || generation < 0 || now.IsZero() {
		return errors.New("invalid unresolved PDS effect job")
	}
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM owner_effect_attempts
			WHERE owner_did=$1
			  AND ($2=0 OR owner_generation=$2)
			  AND effect_kind='pds_record'
			  AND effect_action='put_record'
			  AND remote_outcome IN ('dispatched','outcome_unknown_pre_transition')
		)
	`, owner, generation).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check unresolved PDS effects: %w", err)
	}
	if !exists {
		return nil
	}
	return enqueueRepositoryJob(
		ctx, tx, owner, string(RepositoryJobPDSReconcile), now,
	)
}

// ReconcileUnresolvedPDSAttempts performs at most limit read-only PDS lookups.
// It never repeats a remote mutation. Missing records remain outcome-unknown:
// one absence observation is not proof that an in-flight Put was rejected.
func (service *Service) ReconcileUnresolvedPDSAttempts(
	ctx context.Context,
	owner syntax.DID,
	reader pdseffects.RecordReader,
	limit int,
) (remaining bool, err error) {
	if service == nil || service.lifecycles == nil || owner == "" || reader == nil ||
		limit <= 0 || limit > maxPDSAttemptReconciliationPage {
		return false, errors.New("invalid PDS effect reconciliation request")
	}
	attempts, hasMore, err := service.lifecycles.UnresolvedPDSAttempts(ctx, owner, limit)
	if err != nil {
		return false, err
	}
	remaining = hasMore
	now := service.store.now().UTC()
	for _, attempt := range attempts {
		if attempt.RemoteDeadline.After(now) {
			remaining = true
			continue
		}
		uri, err := syntax.ParseATURI(attempt.DeterministicKey)
		if err != nil || uri.Authority().DID() != owner || uri.Collection() == "" || uri.RecordKey() == "" {
			return false, fmt.Errorf("invalid unresolved PDS effect identity %q", attempt.OperationID)
		}
		var current map[string]any
		cid, readErr := reader.GetRecord(
			ctx, owner, uri.Collection().String(), uri.RecordKey().String(), &current,
		)
		switch {
		case errors.Is(readErr, auth.ErrRecordNotFound):
			remaining = true
			continue
		case readErr != nil:
			return false, fmt.Errorf("read unresolved PDS effect %q: %w", attempt.OperationID, readErr)
		case strings.TrimSpace(cid) == "":
			return false, fmt.Errorf("read unresolved PDS effect %q: empty CID", attempt.OperationID)
		}
		fingerprint, err := pdseffects.RecordContentFingerprint(
			owner, uri.Collection(), uri.RecordKey(), current,
		)
		if err != nil {
			return false, fmt.Errorf("fingerprint unresolved PDS effect %q: %w", attempt.OperationID, err)
		}
		var resolution ownerlifecycle.EffectSourceResolution
		err = service.lifecycles.WithOwnerStates(ctx, []syntax.DID{owner}, func(
			fencedCtx context.Context,
			tx pgx.Tx,
			states map[syntax.DID]ownerlifecycle.Lifecycle,
		) error {
			lifecycle, exists := states[owner]
			if !exists {
				return ownerlifecycle.ErrOwnerNotActive
			}
			var resolveErr error
			resolution, resolveErr = ownerlifecycle.ResolvePDSRecordSourceTx(
				fencedCtx,
				tx,
				lifecycle,
				ownerlifecycle.PDSRecordSourceObservation{
					Owner: owner, URI: uri, CID: syntax.CID(cid),
					RecordFingerprint: fingerprint, Authoritative: true,
				},
				service.store.now().UTC().Truncate(time.Microsecond),
			)
			return resolveErr
		})
		if err != nil {
			return false, err
		}
		if resolution.Match == ownerlifecycle.EffectSourceAmbiguous {
			remaining = true
		}
	}
	return remaining, nil
}
