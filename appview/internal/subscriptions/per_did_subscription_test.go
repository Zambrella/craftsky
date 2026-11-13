package subscriptions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func TestPerDIDSubscriptionFollowsOwnerAcrossTierChanges(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	migration = append(migration, []byte(`ALTER TABLE provider_subscriptions ADD COLUMN pending_mapped_tier TEXT;`)...)
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration)+`CREATE TABLE profile_pins(owner_did TEXT,slot TEXT);`)
	ctx := context.Background()
	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	account, _, err := store.EnsureAccount(ctx, alice)
	if err != nil {
		t.Fatal(err)
	}
	other, _, err := store.EnsureAccount(ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	if account.RevenueCatAppUserID == other.RevenueCatAppUserID {
		t.Fatal("DIDs shared RevenueCat identity")
	}
	again, _, err := store.EnsureAccount(ctx, alice)
	if err != nil || again.RevenueCatAppUserID != account.RevenueCatAppUserID {
		t.Fatal("identity was not stable")
	}
	catalog, _ := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{"plus": {AppID: "app", Tier: TierPlus}, "business": {AppID: "app", Tier: TierBusiness}}})
	apply := func(generation int64, subs ...ProviderSubscriptionSnapshot) {
		t.Helper()
		token := uuid.New()
		if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=now()+interval '1 minute' WHERE id=$1`, account.ID, generation, token); err != nil {
			t.Fatal(err)
		}
		if err := store.ApplySnapshot(ctx, SnapshotClaim{BillingAccountID: account.ID, Generation: generation, LeaseToken: token}, CompleteSnapshot{Complete: true, Subscriptions: subs}, catalog); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := ProviderSubscriptionSnapshot{ID: "native", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true}
	apply(1, snapshot)
	assertTier := func(did syntax.DID, want Tier) {
		t.Helper()
		access, err := store.SelfAccess(ctx, did, time.Now())
		if err != nil || access.EffectiveTier != want {
			t.Fatalf("access %s=%+v err=%v want=%s", did, access, err, want)
		}
	}
	assertTier(alice, TierPlus)
	assertTier(bob, TierFree)
	snapshot.PendingProductID = "business"
	apply(2, snapshot)
	assertTier(alice, TierPlus)
	snapshot.ProductID = "business"
	snapshot.PendingProductID = ""
	apply(3, snapshot)
	assertTier(alice, TierBusiness)
	state, err := store.OwnerState(ctx, alice, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Licenses) != 1 || state.Licenses[0].Assignable || state.Licenses[0].AssignedDID == nil || *state.Licenses[0].AssignedDID != alice {
		t.Fatalf("projection=%+v", state.Licenses)
	}
	// Even a corrupt legacy assignment cannot give another account this customer's access.
	if _, err := pool.Exec(ctx, `UPDATE billing_licenses SET assigned_did=$1 WHERE provider_subscription_id=(SELECT id FROM provider_subscriptions WHERE revenuecat_subscription_id='native')`, bob); err != nil {
		t.Fatal(err)
	}
	assertTier(bob, TierFree)
	apply(4, snapshot)
	assertTier(alice, TierBusiness)
	snapshot.GivesAccess = false
	snapshot.Status = "expired"
	apply(5, snapshot)
	assertTier(alice, TierFree)
}

func TestPerDIDMigrationRejectsAmbiguousOwnershipAndEnforcesOwner(t *testing.T) {
	old, err := os.ReadFile("../../migrations/000076_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration, err := os.ReadFile("../../migrations/000124_per_did_subscriptions.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(old))
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES('10000000-0000-4000-8000-000000000001','did:plc:owner','20000000-0000-4000-8000-000000000001')`,
		`INSERT INTO provider_subscriptions(id,billing_account_id,project_id,revenuecat_subscription_id,product_id,app_id,store,environment,status,gives_access,mapped_tier,accepted_generation) VALUES('30000000-0000-4000-8000-000000000001','10000000-0000-4000-8000-000000000001','project','native','plus','app','app_store','production','active',true,'plus',1)`,
		`INSERT INTO billing_licenses(provider_subscription_id,tier,assigned_did,assigned_at) VALUES('30000000-0000-4000-8000-000000000001','plus','did:plc:beneficiary',now())`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, string(migration)); err == nil {
		t.Fatal("migration silently transferred cross-DID subscription")
	}
	var assigned string
	if err := pool.QueryRow(ctx, `SELECT assigned_did FROM billing_licenses`).Scan(&assigned); err != nil || assigned != "did:plc:beneficiary" {
		t.Fatalf("failed migration changed assignment=%s err=%v", assigned, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE billing_licenses SET assigned_did='did:plc:owner'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE billing_licenses SET assigned_did='did:plc:other'`); err == nil {
		t.Fatal("database accepted cross-DID subscription mutation")
	}
	var assignable bool
	if err := pool.QueryRow(ctx, `SELECT assigned_did,assignable FROM billing_licenses`).Scan(&assigned, &assignable); err != nil || assigned != "did:plc:owner" || assignable {
		t.Fatalf("migrated projection=%s assignable=%t err=%v", assigned, assignable, err)
	}
}
