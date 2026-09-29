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
	migration, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration)+`CREATE TABLE profile_pins(owner_did TEXT, slot TEXT);`)
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

func TestSelfAccessIsIsolatedToConfiguredRevenueCatEnvironment(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	beneficiary := syntax.DID("did:plc:sandbox-beneficiary")
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES('10000000-0000-4000-8000-000000000071','did:plc:sandbox-owner','20000000-0000-4000-8000-000000000071')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES('30000000-0000-4000-8000-000000000071','10000000-0000-4000-8000-000000000071','project','sandbox-subscription','sandbox-plus','test-app','test_store','sandbox','active',true,'plus',1)`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES('30000000-0000-4000-8000-000000000071','plus',$1,now())`, beneficiary); err != nil {
		t.Fatal(err)
	}

	sandboxAccess, err := NewStoreForEnvironment(pool, "sandbox").SelfAccess(ctx, beneficiary, time.Now())
	if err != nil || !sandboxAccess.GivesAccess || sandboxAccess.EffectiveTier != TierPlus {
		t.Fatalf("sandbox access = %+v, error %v", sandboxAccess, err)
	}
	productionAccess, err := NewStore(pool).SelfAccess(ctx, beneficiary, time.Now())
	if err != nil || productionAccess.GivesAccess || productionAccess.EffectiveTier != TierFree || productionAccess.AssignedTier == nil {
		t.Fatalf("production access = %+v, error %v", productionAccess, err)
	}
}
