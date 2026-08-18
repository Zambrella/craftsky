package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

type expectedGenerationContextKey struct{}

var ErrGenerationRequired = errors.New("owner lifecycle generation required")

// WithExpectedGeneration carries the active generation observed by the HTTP
// membership boundary to the transaction that will persist a private write.
func WithExpectedGeneration(ctx context.Context, generation int64) context.Context {
	return context.WithValue(ctx, expectedGenerationContextKey{}, generation)
}

// ExpectedGeneration returns only positive owner generations.
func ExpectedGeneration(ctx context.Context) (int64, bool) {
	generation, ok := ctx.Value(expectedGenerationContextKey{}).(int64)
	return generation, ok && generation > 0
}

// GuardPrivateMutationTx linearizes an AppView-private write with lifecycle
// transitions. The authenticated owner must still be active at the exact
// generation observed by middleware. Unknown and departed non-terminal
// targets remain addressable, while terminal targets are permanently denied.
func GuardPrivateMutationTx(
	ctx context.Context,
	tx pgx.Tx,
	owner syntax.DID,
	targets []syntax.DID,
) error {
	expectedGeneration, ok := ExpectedGeneration(ctx)
	if !ok {
		return ErrGenerationRequired
	}
	owners := make([]syntax.DID, 0, 1+len(targets))
	owners = append(owners, owner)
	owners = append(owners, targets...)
	states, err := LockOwnerStatesTx(ctx, tx, owners)
	if err != nil {
		return fmt.Errorf("lock private mutation owners: %w", err)
	}
	return RequirePrivateMutationAuthority(states, owner, expectedGeneration, targets)
}

// RequirePrivateMutationAuthority applies the role-aware lifecycle decision to
// states that were re-read while the caller's transaction locks were held.
func RequirePrivateMutationAuthority(
	states map[syntax.DID]Lifecycle,
	owner syntax.DID,
	expectedGeneration int64,
	targets []syntax.DID,
) error {
	if expectedGeneration <= 0 {
		return ErrGenerationRequired
	}
	ownerState, ok := states[owner]
	if !ok || ownerState.State != StateActive {
		return ErrOwnerNotActive
	}
	if ownerState.Generation != expectedGeneration {
		return ErrGenerationChanged
	}
	return RequireNonTerminalTargets(states, targets)
}

// GuardNonTerminalTargetsTx protects a derived cache/materialization write
// from crossing a terminal transition without manufacturing member authority
// for unknown or departed external DIDs.
func GuardNonTerminalTargetsTx(ctx context.Context, tx pgx.Tx, targets []syntax.DID) error {
	states, err := LockOwnerStatesTx(ctx, tx, targets)
	if err != nil {
		return fmt.Errorf("lock derived mutation targets: %w", err)
	}
	return RequireNonTerminalTargets(states, targets)
}

// RequireNonTerminalTargets rejects only the irreversible terminal state.
// Unknown and every known non-terminal state remain eligible for derived data.
func RequireNonTerminalTargets(states map[syntax.DID]Lifecycle, targets []syntax.DID) error {
	for _, target := range targets {
		if targetState, known := states[target]; known && targetState.State == StateTerminal {
			return fmt.Errorf("private mutation target: %w", ErrTerminalOwner)
		}
	}
	return nil
}

// WithPreheldNonTerminalOwnerTx reuses the dedicated connection when the
// caller already holds this owner's lifecycle fence. It prevents a cache write
// nested inside OAuth finalization from opening a second connection and
// waiting on its own owner lock. The boolean is false only when no lifecycle
// fence is present, so callers may then use an ordinary guarded transaction.
func WithPreheldNonTerminalOwnerTx(
	ctx context.Context,
	owner syntax.DID,
	callback func(pgx.Tx) error,
) (bool, error) {
	if owner == "" {
		return true, ErrInvalidOwner
	}
	if callback == nil {
		return true, errors.New("invalid pre-held owner transaction")
	}
	authority, hasAuthority := ctx.Value(authTransitionContextKey{}).(Lifecycle)
	conn, hasConnection := fencedConnection(ctx)
	if !hasAuthority && !hasConnection {
		return false, nil
	}
	if !hasAuthority || !hasConnection || authority.Owner != owner {
		return true, ErrFenceRequired
	}
	if authority.State == StateTerminal {
		return true, ErrTerminalOwner
	}
	return true, pgx.BeginFunc(ctx, conn.Conn(), callback)
}
