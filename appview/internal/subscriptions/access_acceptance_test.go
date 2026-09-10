package subscriptions

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestReconciledAccessUsesProviderTruthWithoutLocalExpiry(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	account, _, err := store.EnsureAccount(ctx, syntax.DID("did:plc:authority-owner"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, _ := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{
		"plus": {AppID: "app", Tier: TierPlus},
	}})
	beneficiary := syntax.DID("did:plc:authority-beneficiary")
	periodEnd := now.Add(-time.Hour)
	apply := func(generation int64, givesAccess bool) {
		t.Helper()
		token := uuid.New()
		if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=$4 WHERE id=$1`, account.ID, generation, token, now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		if err := store.ApplySnapshot(ctx, SnapshotClaim{BillingAccountID: account.ID, Generation: generation, LeaseToken: token}, CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
			{ID: "stable-subscription", ProductID: "plus", Store: "app_store", Environment: "production", Status: "provider-state", GivesAccess: givesAccess, CurrentPeriodEndsAt: &periodEnd},
		}}, catalog); err != nil {
			t.Fatal(err)
		}
	}
	apply(1, true)
	if _, err := pool.Exec(ctx, `UPDATE billing_licenses SET assigned_did=$2,assigned_at=$3 WHERE provider_subscription_id=(SELECT id FROM provider_subscriptions WHERE revenuecat_subscription_id=$1)`, "stable-subscription", beneficiary, now); err != nil {
		t.Fatal(err)
	}

	provider := &retryProvider{err: errors.New("provider unavailable")}
	reconciler := NewReconciler(store, provider, catalog, time.Minute, func() time.Time { return now })
	if err := reconciler.Trigger(ctx, account.ID, TriggerOwnerRefresh); err != nil {
		t.Fatal(err)
	}
	if processed, err := reconciler.ProcessOne(ctx); !processed || err == nil {
		t.Fatalf("provider failure = processed %t, error %v", processed, err)
	}
	paid, err := store.SelfAccess(ctx, beneficiary, now)
	if err != nil || !paid.GivesAccess || paid.EffectiveTier != TierPlus || paid.AccessEndsAt == nil || !paid.AccessEndsAt.Equal(periodEnd) {
		t.Fatalf("outage access = %+v, error %v", paid, err)
	}
	var nextAttempt *time.Time
	if err := pool.QueryRow(ctx, `SELECT next_attempt_at FROM billing_accounts WHERE id=$1`, account.ID).Scan(&nextAttempt); err != nil || nextAttempt == nil {
		t.Fatalf("observable retry state = %v, error %v", nextAttempt, err)
	}

	apply(3, false)
	dormant, err := store.SelfAccess(ctx, beneficiary, now)
	if err != nil || dormant.GivesAccess || dormant.EffectiveTier != TierFree || dormant.AssignedTier == nil {
		t.Fatalf("dormant access = %+v, error %v", dormant, err)
	}
	apply(4, true)
	restored, err := store.SelfAccess(ctx, beneficiary, now)
	if err != nil || !restored.GivesAccess || restored.EffectiveTier != TierPlus {
		t.Fatalf("restored access = %+v, error %v", restored, err)
	}
}
