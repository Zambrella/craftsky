package ownerlifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/postfence"
	"social.craftsky/appview/internal/testdb"
)

// IT-012: the nested post fence reuses the owner-fenced connection; another
// pool acquisition would deadlock at capacity during concurrent deliveries.
func TestPostPublicationFenceReusesOwnerConnectionAtCapacity(t *testing.T) {
	base := testdb.WithSchema(t, "")
	cfg := base.Config().Copy()
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	fencer, err := NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	uri := syntax.ATURI("at://did:plc:actor/social.craftsky.feed.post/3aaaaaaaaaaa2")
	called := false
	err = fencer.WithShared(ctx, []syntax.DID{"did:plc:actor"}, func(ownerCtx context.Context) error {
		return store.WithPostPublicationFence(ownerCtx, uri, func(context.Context) error {
			called = true
			var acquired bool
			if err := base.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, postfence.Key(uri)).Scan(&acquired); err != nil {
				return err
			}
			if acquired {
				t.Error("post fence not held across callback")
				_, _ = base.Exec(ctx, `SELECT pg_advisory_unlock($1)`, postfence.Key(uri))
			}
			return nil
		})
	})
	if err != nil || !called {
		t.Fatalf("nested post fence blocked at capacity: called=%v error=%v", called, err)
	}
	var acquired bool
	conn, err := base.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if err := conn.QueryRow(context.Background(), `SELECT pg_try_advisory_lock($1)`, postfence.Key(uri)).Scan(&acquired); err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("post fence leaked after callback")
	}
	if _, err := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, postfence.Key(uri)); err != nil {
		t.Fatal(err)
	}
}

func TestPostPublicationFenceReleasesAfterCallbackFailure(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	fencer, err := NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	uri := syntax.ATURI("at://did:plc:actor/social.craftsky.feed.post/3aaaaaaaaaaa2")
	failure := errors.New("provider failed")
	err = fencer.WithShared(context.Background(), []syntax.DID{"did:plc:actor"}, func(ctx context.Context) error {
		return store.WithPostPublicationFence(ctx, uri, func(context.Context) error { return failure })
	})
	if !errors.Is(err, failure) {
		t.Fatalf("callback cause lost: %v", err)
	}
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var acquired bool
	if err := conn.QueryRow(context.Background(), `SELECT pg_try_advisory_lock($1)`, postfence.Key(uri)).Scan(&acquired); err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("failed callback leaked post fence")
	}
	if _, err := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, postfence.Key(uri)); err != nil {
		t.Fatal(err)
	}
}
