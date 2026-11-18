package subscriptions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/testdb"
)

func TestFencedFeatureAccessUsesHeldPoolConnectionAndReleasesIt(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	did := syntax.DID("did:plc:fenced-publisher")
	store := NewStore(pool)
	account, _, err := store.EnsureAccount(t.Context(), did)
	if err != nil {
		t.Fatal(err)
	}
	subscriptionID := uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO provider_subscriptions(
		id,billing_account_id,project_id,revenuecat_subscription_id,product_id,
		app_id,store,environment,status,gives_access,mapped_tier,accepted_generation
	) VALUES($1,$2,'project','subscription','plus','app','app_store','production','active',true,'plus',1)`, subscriptionID, account.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES($1,'plus',$2,now())`, subscriptionID, did); err != nil {
		t.Fatal(err)
	}
	config := pool.Config()
	config.MaxConns = 1
	config.MinConns = 0
	single, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(single.Close)
	store.pool = single
	effectFailure := errors.New("publication effect failed")
	for _, tc := range []struct {
		name       string
		paid       bool
		effectErr  error
		wantErr    error
		wantEffect bool
	}{
		{name: "paid", paid: true, wantEffect: true},
		{name: "free", wantErr: ErrFeatureAccessRequired},
		{name: "effect failure", paid: true, effectErr: effectFailure, wantErr: effectFailure, wantEffect: true},
		{name: "reuse after failure", paid: true, wantEffect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := single.Exec(t.Context(), `UPDATE provider_subscriptions SET gives_access=$2 WHERE id=$1`, subscriptionID, tc.paid); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			called := false
			err := store.WithPlusAccess(ctx, did, func(context.Context) error {
				called = true
				return tc.effectErr
			})
			if !errors.Is(err, tc.wantErr) || called != tc.wantEffect {
				t.Fatalf("effect called=%t error=%v, want called=%t error=%v", called, err, tc.wantEffect, tc.wantErr)
			}
			// A failed access check or effect must release both the connection
			// and the session-level fence before another publication can run.
			if single.Stat().AcquiredConns() != 0 {
				t.Fatal("publication retained its database connection")
			}
			conn, err := single.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Release()
			var unlocked bool
			if err := conn.QueryRow(ctx, `SELECT pg_advisory_unlock_shared($1)`, AccessFenceKey(did)).Scan(&unlocked); err != nil || unlocked {
				t.Fatalf("publication retained its access fence: unlocked=%t err=%v", unlocked, err)
			}
		})
	}
}
