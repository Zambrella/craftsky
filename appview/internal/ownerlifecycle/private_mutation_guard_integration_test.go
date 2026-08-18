package ownerlifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/testdb"
)

func TestGuardPrivateMutationTxHoldsOwnerAndUnknownTargetFencesUntilCommit(t *testing.T) {
	pool := testdb.WithSchema(t, ownerLifecycleTransactionLockDDL)
	ctx := context.Background()
	owner := syntax.DID("did:plc:private-mutation-fence-owner")
	unknownTarget := syntax.DID("did:plc:private-mutation-unknown-target")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',5,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	guarded := WithExpectedGeneration(ctx, 5)
	if err := GuardPrivateMutationTx(guarded, tx, owner, []syntax.DID{unknownTarget}); err != nil {
		t.Fatalf("guard private mutation: %v", err)
	}

	fencer, err := NewFencer(pool, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	exclusiveEntered := make(chan struct{})
	exclusiveDone := make(chan error, 1)
	go func() {
		exclusiveDone <- fencer.WithExclusive(ctx, []syntax.DID{unknownTarget, owner}, func(context.Context) error {
			close(exclusiveEntered)
			return nil
		})
	}()
	select {
	case <-exclusiveEntered:
		t.Fatal("exclusive transition entered while private mutation transaction was open")
	case <-time.After(100 * time.Millisecond):
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exclusiveEntered:
	case <-time.After(time.Second):
		t.Fatal("exclusive transition did not enter after private mutation committed")
	}
	if err := <-exclusiveDone; err != nil {
		t.Fatalf("exclusive transition: %v", err)
	}
}

func TestGuardPrivateMutationTxFailsClosedWithoutExpectedGeneration(t *testing.T) {
	pool := testdb.WithSchema(t, ownerLifecycleTransactionLockDDL)
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := GuardPrivateMutationTx(
		context.Background(),
		tx,
		syntax.DID("did:plc:private-mutation-owner"),
		nil,
	); !errors.Is(err, ErrGenerationRequired) {
		t.Fatalf("error = %v, want ErrGenerationRequired", err)
	}
}

func TestWithPreheldNonTerminalOwnerTxReusesExistingAuthFence(t *testing.T) {
	store, _ := ownerLifecycleTestStores(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:preheld-derived-cache-owner")
	if _, err := store.EnsureOnboardingOwner(ctx, owner); err != nil {
		t.Fatal(err)
	}

	called := false
	err := store.WithExistingAuth(ctx, owner, func(fenceCtx context.Context, _ Lifecycle) error {
		used, err := WithPreheldNonTerminalOwnerTx(fenceCtx, owner, func(tx pgx.Tx) error {
			called = true
			var state State
			return tx.QueryRow(fenceCtx, `SELECT state FROM owner_lifecycles WHERE owner_did=$1`, owner).Scan(&state)
		})
		if !used {
			t.Fatal("pre-held owner fence was not reused")
		}
		return err
	})
	if err != nil {
		t.Fatalf("reuse pre-held owner fence: %v", err)
	}
	if !called {
		t.Fatal("fenced transaction callback was not called")
	}
}
