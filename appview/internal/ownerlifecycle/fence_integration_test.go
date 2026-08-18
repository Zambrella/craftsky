package ownerlifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestOwnerFenceSharedEffectsDrainBeforeExclusiveTransition(t *testing.T) {
	poolA, poolB := ownerFenceTestPools(t)
	fenceA, err := NewFencer(poolA, 150*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	fenceB, err := NewFencer(poolB, 150*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:fence-owner")

	entered := make(chan struct{})
	release := make(chan struct{})
	sharedDone := make(chan error, 1)
	go func() {
		sharedDone <- fenceA.WithShared(context.Background(), []syntax.DID{owner}, func(context.Context) error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered

	err = fenceB.WithExclusive(context.Background(), []syntax.DID{owner}, func(context.Context) error {
		return errors.New("exclusive callback must not run while shared fence is held")
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked exclusive error = %v, want deadline exceeded", err)
	}
	if err := poolB.Ping(context.Background()); err != nil {
		t.Fatalf("pool unusable after cancelled acquisition: %v", err)
	}

	close(release)
	if err := <-sharedDone; err != nil {
		t.Fatalf("shared effect: %v", err)
	}
	if err := fenceB.WithExclusive(context.Background(), []syntax.DID{owner}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("exclusive transition after shared drain: %v", err)
	}
}

func TestOwnerFenceSharedOwnersOverlapAndReverseInputCannotDeadlock(t *testing.T) {
	poolA, poolB := ownerFenceTestPools(t)
	fenceA, err := NewFencer(poolA, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	fenceB, err := NewFencer(poolB, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	a := syntax.DID("did:plc:aaa")
	b := syntax.DID("did:plc:bbb")

	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- fenceA.WithShared(context.Background(), []syntax.DID{b, a}, func(context.Context) error {
			close(firstEntered)
			<-releaseFirst
			return nil
		})
	}()
	<-firstEntered

	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- fenceB.WithShared(context.Background(), []syntax.DID{a, b}, func(context.Context) error {
			close(secondEntered)
			return nil
		})
	}()
	select {
	case <-secondEntered:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("shared multi-owner fence did not overlap")
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second shared fence: %v", err)
	}
	close(releaseFirst)
	if err := <-firstDone; err != nil {
		t.Fatalf("first shared fence: %v", err)
	}

	// Both callers supply inverse orders. Canonical acquisition allows one to
	// finish and the other to follow without an AB/BA deadlock.
	start := make(chan struct{})
	done := make(chan error, 2)
	for _, owners := range [][]syntax.DID{{a, b}, {b, a}} {
		owners := owners
		fencer := fenceA
		if owners[0] == b {
			fencer = fenceB
		}
		go func() {
			<-start
			done <- fencer.WithExclusive(context.Background(), owners, func(context.Context) error {
				time.Sleep(25 * time.Millisecond)
				return nil
			})
		}()
	}
	close(start)
	for range 2 {
		if err := <-done; err != nil {
			t.Fatalf("inverse-order exclusive fence: %v", err)
		}
	}
}

func TestOwnerFenceUnlocksOnCallbackErrorAndRejectsReentry(t *testing.T) {
	poolA, _ := ownerFenceTestPools(t)
	fencer, err := NewFencer(poolA, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:fence-owner")
	sentinel := errors.New("callback failed")

	err = fencer.WithShared(context.Background(), []syntax.DID{owner}, func(ctx context.Context) error {
		nestedErr := fencer.WithExclusive(ctx, []syntax.DID{owner}, func(context.Context) error { return nil })
		if !errors.Is(nestedErr, ErrFenceReentry) {
			t.Fatalf("nested fence error = %v, want ErrFenceReentry", nestedErr)
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("callback error = %v, want sentinel", err)
	}
	if err := fencer.WithExclusive(context.Background(), []syntax.DID{owner}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("exclusive fence after callback error: %v", err)
	}
}

func ownerFenceTestPools(t *testing.T) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	poolA := testdb.WithSchema(t, "")
	poolB, err := pgxpool.NewWithConfig(context.Background(), poolA.Config().Copy())
	if err != nil {
		t.Fatalf("second pool: %v", err)
	}
	t.Cleanup(poolB.Close)
	return poolA, poolB
}
