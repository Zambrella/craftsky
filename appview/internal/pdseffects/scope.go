package pdseffects

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// ResolveExpectedOwners snapshots the exact generations that a directed PDS
// effect must recheck at its guarded boundary. The bound repository owner is
// always included at ownerGeneration. Unknown external target DIDs receive a
// fence-only missing expectation; a target known to CraftSky but not active
// rejects the new effect.
func (executor *Executor) ResolveExpectedOwners(
	ctx context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	if executor == nil || executor.attempts == nil || executor.owner == "" {
		return nil, errors.New("durable PDS effect scope resolver is unavailable")
	}
	if ownerGeneration <= 0 {
		return nil, ownerlifecycle.ErrGenerationChanged
	}
	owners := make([]syntax.DID, 0, len(targets)+1)
	owners = append(owners, executor.owner)
	owners = append(owners, targets...)
	canonical, err := ownerlifecycle.CanonicalOwners(owners)
	if err != nil {
		return nil, err
	}
	resolved := make([]ownerlifecycle.ExpectedOwner, 0, len(canonical))
	for _, owner := range canonical {
		lifecycle, err := executor.attempts.Get(ctx, owner)
		if errors.Is(err, pgx.ErrNoRows) && owner != executor.owner {
			resolved = append(resolved, ownerlifecycle.ExpectedOwner{
				Owner: owner, AllowMissing: true,
			})
			continue
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ownerlifecycle.ErrOwnerNotActive
		}
		if err != nil {
			return nil, err
		}
		if lifecycle.State == ownerlifecycle.StateTerminal {
			return nil, ownerlifecycle.ErrTerminalOwner
		}
		if lifecycle.State != ownerlifecycle.StateActive {
			return nil, ownerlifecycle.ErrOwnerNotActive
		}
		if owner == executor.owner && lifecycle.Generation != ownerGeneration {
			return nil, ownerlifecycle.ErrGenerationChanged
		}
		resolved = append(resolved, ownerlifecycle.ExpectedOwner{
			Owner: owner, Generation: lifecycle.Generation,
		})
	}
	return resolved, nil
}
