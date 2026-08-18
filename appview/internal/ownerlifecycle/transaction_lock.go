package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// LockOwnerStatesTx acquires transaction-scoped shared owner fences in
// canonical DID order, then re-reads every existing lifecycle row on the same
// caller-owned transaction. Unknown external DIDs are omitted; terminal rows
// remain visible so the caller can apply role-specific suppression.
//
// The locks are released only when tx commits or rolls back. Callers must keep
// the transaction local and short and must not perform remote I/O while the
// fences are held.
func LockOwnerStatesTx(
	ctx context.Context,
	tx pgx.Tx,
	owners []syntax.DID,
) (map[syntax.DID]Lifecycle, error) {
	if tx == nil {
		return nil, errors.New("owner lifecycle transaction lock requires a transaction")
	}
	canonical, err := CanonicalOwners(owners)
	if err != nil {
		return nil, err
	}
	for position, owner := range canonical {
		key, err := FenceKey(owner)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock_shared($1)`, key); err != nil {
			return nil, fmt.Errorf("acquire transaction owner fence at canonical position %d: %w", position, err)
		}
	}

	states := make(map[syntax.DID]Lifecycle, len(canonical))
	for _, owner := range canonical {
		lifecycle, err := scanLifecycle(tx.QueryRow(
			ctx,
			lifecycleSelect+` WHERE owner_did=$1 FOR SHARE`,
			owner,
		))
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read transaction owner lifecycle: %w", err)
		}
		states[owner] = lifecycle
	}
	return states, nil
}
