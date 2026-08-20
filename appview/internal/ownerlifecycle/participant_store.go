package ownerlifecycle

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// OwnerStatesParticipant runs a short local transaction while shared owner
// fences keep the supplied lifecycle rows stable. Unlike
// WithNonTerminalOwners, terminal rows are deliberately returned to the
// participant so role-aware cleanup/outbox code can cancel local effects
// instead of attempting to create new work. Missing rows are omitted for
// explicitly trusted external DIDs.
type OwnerStatesParticipant func(
	context.Context,
	pgx.Tx,
	map[syntax.DID]Lifecycle,
) error

// WithOwnerStates acquires canonical shared owner fences and supplies every
// existing lifecycle state in one transaction. It performs local database
// work only; callers must not perform remote I/O in participant.
func (store *Store) WithOwnerStates(
	ctx context.Context,
	owners []syntax.DID,
	participant OwnerStatesParticipant,
) error {
	if store == nil || participant == nil {
		return errors.New("invalid owner-state participant")
	}
	canonical, err := CanonicalOwners(owners)
	if err != nil {
		return err
	}
	return store.fencer.WithShared(ctx, canonical, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			states := make(map[syntax.DID]Lifecycle, len(canonical))
			for _, owner := range canonical {
				lifecycle, err := scanLifecycle(tx.QueryRow(
					fenceCtx,
					lifecycleSelect+` WHERE owner_did=$1 FOR SHARE`,
					owner,
				))
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				if err != nil {
					return err
				}
				states[owner] = lifecycle
			}
			return participant(fenceCtx, tx, states)
		})
	})
}

// WithExclusiveOwnerStates is the local destructive counterpart to
// WithOwnerStates. It takes every source/target fence in canonical DID order,
// locks existing lifecycle rows before later operation/outbox rows, and runs
// one short database-only transaction. Missing trusted external DIDs remain
// represented by their advisory fence but are omitted from states.
func (store *Store) WithExclusiveOwnerStates(
	ctx context.Context,
	owners []syntax.DID,
	participant OwnerStatesParticipant,
) error {
	if store == nil || participant == nil {
		return errors.New("invalid exclusive owner-state participant")
	}
	canonical, err := CanonicalOwners(owners)
	if err != nil {
		return err
	}
	return store.fencer.WithExclusive(ctx, canonical, func(fenceCtx context.Context) error {
		return store.beginFenced(fenceCtx, func(tx pgx.Tx) error {
			states := make(map[syntax.DID]Lifecycle, len(canonical))
			for _, owner := range canonical {
				lifecycle, err := scanLifecycle(tx.QueryRow(
					fenceCtx,
					lifecycleSelect+` WHERE owner_did=$1 FOR UPDATE`,
					owner,
				))
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				if err != nil {
					return err
				}
				states[owner] = lifecycle
			}
			return participant(fenceCtx, tx, states)
		})
	})
}
