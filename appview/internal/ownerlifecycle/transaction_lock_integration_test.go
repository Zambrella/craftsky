package ownerlifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestLockOwnerStatesTxReturnsKnownStatesAndOmitsUnknownOwners(t *testing.T) {
	pool := testdb.WithSchema(t, ownerLifecycleTransactionLockDDL)
	ctx := context.Background()
	active := syntax.DID("did:plc:active-transaction-owner")
	terminal := syntax.DID("did:plc:terminal-transaction-owner")
	unknown := syntax.DID("did:plc:unknown-transaction-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,terminal_at,created_at,updated_at
		) VALUES
			($1,'active',3,4,'test',now(),NULL,now(),now()),
			($2,'terminal',7,8,'test',now(),now(),now(),now())
	`, active, terminal); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	states, err := LockOwnerStatesTx(ctx, tx, []syntax.DID{terminal, unknown, active, terminal})
	if err != nil {
		t.Fatalf("lock owner states: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("states=%+v, want only two known owners", states)
	}
	if got := states[active]; got.State != StateActive || got.Generation != 3 || got.AuthEpoch != 4 {
		t.Fatalf("active lifecycle=%+v", got)
	}
	if got := states[terminal]; got.State != StateTerminal || got.Generation != 7 || got.AuthEpoch != 8 {
		t.Fatalf("terminal lifecycle=%+v", got)
	}
	if _, exists := states[unknown]; exists {
		t.Fatal("unknown external owner received manufactured lifecycle authority")
	}
}

func TestLockOwnerStatesTxHoldsKnownAndUnknownSharedFencesUntilCallerTransactionEnds(t *testing.T) {
	pool := testdb.WithSchema(t, ownerLifecycleTransactionLockDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:transaction-fence-owner")
	unknownTarget := syntax.DID("did:plc:transaction-fence-unknown-target")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LockOwnerStatesTx(ctx, tx, []syntax.DID{unknownTarget, owner}); err != nil {
		t.Fatalf("lock owner states: %v", err)
	}

	fencer, err := NewFencer(pool, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	exclusiveEntered := make(chan struct{})
	exclusiveDone := make(chan error, 1)
	go func() {
		exclusiveDone <- fencer.WithExclusive(ctx, []syntax.DID{owner, unknownTarget}, func(context.Context) error {
			close(exclusiveEntered)
			return nil
		})
	}()
	select {
	case <-exclusiveEntered:
		t.Fatal("exclusive transition entered while projector transaction was open")
	case <-time.After(100 * time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exclusiveEntered:
	case <-time.After(time.Second):
		t.Fatal("exclusive transition did not enter after projector transaction committed")
	}
	if err := <-exclusiveDone; err != nil {
		t.Fatalf("exclusive transition: %v", err)
	}
}

const ownerLifecycleTransactionLockDDL = `
CREATE TABLE owner_lifecycles (
	owner_did TEXT PRIMARY KEY,
	state TEXT NOT NULL,
	generation BIGINT NOT NULL,
	auth_epoch BIGINT NOT NULL,
	transition_reason TEXT NOT NULL,
	transitioned_at TIMESTAMPTZ NOT NULL,
	terminal_at TIMESTAMPTZ,
	purge_completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);
`
