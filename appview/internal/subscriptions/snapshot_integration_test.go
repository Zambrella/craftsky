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

func TestApplyCompleteSnapshotsIsFencedIdempotentAndFailClosed(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	observer := &recordingBillingObserver{}
	store := NewStore(pool, observer)
	ctx := context.Background()
	account, _, err := store.EnsureAccount(ctx, syntax.DID("did:plc:snapshot-owner"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(CatalogConfig{
		ProjectID: "proj-production",
		AppIDs:    []string{"app-ios"},
		Products: map[string]ProductMapping{
			"prod-plus":     {AppID: "app-ios", Tier: TierPlus},
			"prod-business": {AppID: "app-ios", Tier: TierBusiness},
			"prod-bad-app":  {AppID: "app-unconfigured", Tier: TierPlus},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	claim := func(generation int64) SnapshotClaim {
		t.Helper()
		token := uuid.New()
		_, err := pool.Exec(ctx, `
			UPDATE billing_accounts
			SET requested_generation=GREATEST(requested_generation,$2),
				claimed_generation=$2, lease_token=$3,
				lease_expires_at=now()+interval '1 minute', reconciliation_requested_at=now()
			WHERE id=$1
		`, account.ID, generation, token)
		if err != nil {
			t.Fatal(err)
		}
		return SnapshotClaim{BillingAccountID: account.ID, Generation: generation, LeaseToken: token}
	}

	periodEnd := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	first := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		{ID: "sub-plus-old", ProductID: "prod-plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true, CurrentPeriodEndsAt: &periodEnd},
	}}
	if err := store.ApplySnapshot(ctx, claim(1), first, catalog); err != nil {
		t.Fatalf("apply first snapshot: %v", err)
	}
	beneficiary := syntax.DID("did:plc:snapshot-beneficiary")
	if _, err := pool.Exec(ctx, `
		UPDATE billing_licenses SET assigned_did=$2, assigned_at=now()
		WHERE provider_subscription_id=(SELECT id FROM provider_subscriptions WHERE revenuecat_subscription_id=$1)
	`, "sub-plus-old", beneficiary); err != nil {
		t.Fatal(err)
	}

	second := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		{ID: "sub-plus-old", ProductID: "prod-plus", Store: "app_store", Environment: "production", Status: "expired"},
		{ID: "sub-business-new", ProductID: "prod-business", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
		{ID: "sub-sandbox", ProductID: "prod-plus", Store: "app_store", Environment: "sandbox", Status: "active", GivesAccess: true},
		{ID: "sub-unknown", ProductID: "prod-unknown", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
		{ID: "sub-bad-app", ProductID: "prod-bad-app", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
	}}
	if err := store.ApplySnapshot(ctx, claim(2), second, catalog); err != nil {
		t.Fatalf("apply second snapshot: %v", err)
	}
	if err := store.ApplySnapshot(ctx, SnapshotClaim{BillingAccountID: account.ID, Generation: 1, LeaseToken: uuid.New()}, first, catalog); !errors.Is(err, ErrStaleSnapshot) {
		t.Fatalf("older snapshot error = %v, want ErrStaleSnapshot", err)
	}

	var assigned *string
	var givesAccess bool
	if err := pool.QueryRow(ctx, `
		SELECT license.assigned_did, subscription.gives_access
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id='sub-plus-old'
	`).Scan(&assigned, &givesAccess); err != nil {
		t.Fatal(err)
	}
	if assigned == nil || *assigned != beneficiary.String() || givesAccess {
		t.Fatalf("dormant same-ID assignment = %v, access=%t", assigned, givesAccess)
	}
	if err := pool.QueryRow(ctx, `
		SELECT assigned_did FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id='sub-business-new'
	`).Scan(&assigned); err != nil || assigned != nil {
		t.Fatalf("new subscription assignment = %v, error %v; want nil", assigned, err)
	}
	var unsupportedLicenses int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id IN ('sub-sandbox','sub-unknown','sub-bad-app')
	`).Scan(&unsupportedLicenses); err != nil || unsupportedLicenses != 0 {
		t.Fatalf("unsupported production licenses = %d, error %v", unsupportedLicenses, err)
	}

	recovery := first
	if err := store.ApplySnapshot(ctx, claim(3), recovery, catalog); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT license.assigned_did, subscription.gives_access
		FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id='sub-plus-old'
	`).Scan(&assigned, &givesAccess); err != nil || assigned == nil || *assigned != beneficiary.String() || !givesAccess {
		t.Fatalf("same-ID recovery assignment = %v, access=%t, error=%v", assigned, givesAccess, err)
	}

	duplicate := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		first.Subscriptions[0],
		{ID: "sub-plus-extra", ProductID: "prod-plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
	}}
	if err := store.ApplySnapshot(ctx, claim(4), duplicate, catalog); err != nil {
		t.Fatal(err)
	}
	var assignable bool
	var anomaly string
	if err := pool.QueryRow(ctx, `
		SELECT license.assignable, license.anomaly FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id='sub-plus-extra'
	`).Scan(&assignable, &anomaly); err != nil || assignable || anomaly != "same_tier_duplicate" {
		t.Fatalf("duplicate decision = assignable %t anomaly %q error %v", assignable, anomaly, err)
	}

	crossTier := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		{ID: "sub-plus-old", ProductID: "prod-plus", PendingProductID: "prod-business", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
		{ID: "sub-business-new", ProductID: "prod-business", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
	}}
	if err := store.ApplySnapshot(ctx, claim(5), crossTier, catalog); err != nil {
		t.Fatal(err)
	}
	var crossTierCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM billing_licenses license
		JOIN provider_subscriptions subscription ON subscription.id=license.provider_subscription_id
		WHERE subscription.revenuecat_subscription_id IN ('sub-plus-old','sub-business-new')
		  AND license.assignable=false AND license.anomaly='cross_tier_conflict'
	`).Scan(&crossTierCount); err != nil || crossTierCount != 2 {
		t.Fatalf("cross-tier contained licenses = %d, error %v", crossTierCount, err)
	}

	for generation, unsupported := range []ProviderSubscriptionSnapshot{
		{ID: "sub-plus-old", ProductID: "prod-unknown", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
		{ID: "sub-plus-old", ProductID: "prod-plus", Store: "app_store", Environment: "sandbox", Status: "active", GivesAccess: true},
	} {
		if err := store.ApplySnapshot(ctx, claim(int64(generation+6)), CompleteSnapshot{
			Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{unsupported},
		}, catalog); err != nil {
			t.Fatalf("apply previously licensed unsupported snapshot: %v", err)
		}
		access, err := store.SelfAccess(ctx, beneficiary, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if access.EffectiveTier != TierFree || access.GivesAccess || access.AssignedTier == nil || *access.AssignedTier != TierPlus {
			t.Fatalf("unsupported assigned access = %+v, want free with dormant Plus assignment", access)
		}
		var productID, environment, providerAnomaly string
		if err := pool.QueryRow(ctx, `
			SELECT subscription.product_id,subscription.environment,subscription.anomaly
			FROM provider_subscriptions subscription
			JOIN billing_licenses license ON license.provider_subscription_id=subscription.id
			WHERE subscription.revenuecat_subscription_id='sub-plus-old'
		`).Scan(&productID, &environment, &providerAnomaly); err != nil {
			t.Fatal(err)
		}
		if productID != unsupported.ProductID || environment != unsupported.Environment || providerAnomaly != "unsupported" {
			t.Fatalf("owner-visible unsupported state = %q/%q/%q", productID, environment, providerAnomaly)
		}
	}

	concurrentClaim := claim(8)
	concurrentSnapshot := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{first.Subscriptions[0]}}
	results := make(chan error, 2)
	for range 2 {
		go func() { results <- store.ApplySnapshot(ctx, concurrentClaim, concurrentSnapshot, catalog) }()
	}
	var applied, rejected int
	for range 2 {
		err := <-results
		switch {
		case err == nil:
			applied++
		case errors.Is(err, ErrStaleSnapshot):
			rejected++
		default:
			t.Fatalf("concurrent snapshot apply error = %v", err)
		}
	}
	if applied != 1 || rejected != 1 {
		t.Fatalf("concurrent snapshot ordering = applied %d rejected %d, want 1/1", applied, rejected)
	}
	if len(observer.anomalies) != 8 || observer.anomalies[len(observer.anomalies)-1] != 0 {
		t.Fatalf("committed snapshot anomaly observations = %v", observer.anomalies)
	}
}
