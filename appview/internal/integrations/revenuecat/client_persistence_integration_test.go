package revenuecat

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestPendingProductChangeReachesPersistedFailClosedAnomaly(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	store := subscriptions.NewStore(pool)
	account, _, err := store.EnsureAccount(ctx, syntax.DID("did:plc:pending-change-owner"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{
		ProjectID: "project", AppIDs: []string{"app"},
		Products: map[string]subscriptions.ProductMapping{
			"prod-plus":     {AppID: "app", Tier: subscriptions.TierPlus},
			"prod-business": {AppID: "app", Tier: subscriptions.TierBusiness},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	responses := []string{
		`{"items":[{"id":"sub-plus","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
		`{"items":[{"id":"sub-plus","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false,"auto_renewal_status":"will_change_product","pending_changes":{"product":{"id":"prod-business","future_product_field":"ignored"},"future_change_field":{"ignored":true}},"future_field":true},{"id":"sub-business","product_id":"prod-business","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
	}
	responseIndex := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(responses[responseIndex]))
		responseIndex++
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "secret", ProjectID: "project"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	applyNext := func(at time.Time) subscriptions.CompleteSnapshot {
		t.Helper()
		snapshot, err := client.ListCustomerSubscriptions(ctx, account.RevenueCatAppUserID)
		if err != nil {
			t.Fatal(err)
		}
		if err := store.RequestReconciliation(ctx, account.ID, at); err != nil {
			t.Fatal(err)
		}
		claim, ok, err := store.ClaimReconciliation(ctx, at, time.Minute)
		if err != nil || !ok {
			t.Fatalf("claim = %+v/%t, error %v", claim, ok, err)
		}
		if err := store.ApplySnapshot(ctx, claim, snapshot, catalog); err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	applyNext(now)
	beneficiary := syntax.DID("did:plc:pending-change-beneficiary")
	if _, err := pool.Exec(ctx, `
		UPDATE billing_licenses SET assigned_did=$2,assigned_at=$3
		WHERE provider_subscription_id=(
			SELECT id FROM provider_subscriptions
			WHERE billing_account_id=$1 AND revenuecat_subscription_id='sub-plus'
		)
	`, account.ID, beneficiary, now); err != nil {
		t.Fatal(err)
	}
	snapshot := applyNext(now.Add(time.Minute))
	if len(snapshot.Subscriptions) != 2 || snapshot.Subscriptions[0].PendingProductID != "prod-business" {
		t.Fatalf("client snapshot = %+v, want nested pending product ID and destination subscription", snapshot)
	}
	rows, err := pool.Query(ctx, `
		SELECT subscription.revenuecat_subscription_id,subscription.pending_product_id,
			subscription.anomaly,license.anomaly,license.assignable,license.assigned_did
		FROM provider_subscriptions subscription
		JOIN billing_licenses license ON license.provider_subscription_id=subscription.id
		WHERE subscription.billing_account_id=$1
		ORDER BY subscription.revenuecat_subscription_id
	`, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var subscriptionID, providerAnomaly, licenseAnomaly string
		var pendingProductID *string
		var assignedDID *string
		var assignable bool
		if err := rows.Scan(&subscriptionID, &pendingProductID, &providerAnomaly, &licenseAnomaly, &assignable, &assignedDID); err != nil {
			t.Fatal(err)
		}
		if providerAnomaly != "cross_tier_conflict" || licenseAnomaly != "cross_tier_conflict" || assignable {
			t.Fatalf("persisted %s = pending %v anomalies %q/%q assignable=%t", subscriptionID, pendingProductID, providerAnomaly, licenseAnomaly, assignable)
		}
		if subscriptionID == "sub-plus" {
			if pendingProductID == nil || *pendingProductID != "prod-business" || assignedDID == nil || *assignedDID != beneficiary.String() {
				t.Fatalf("persisted Plus transition = pending %v assigned %v", pendingProductID, assignedDID)
			}
		} else if assignedDID != nil {
			t.Fatalf("destination assignment moved automatically to %q", *assignedDID)
		}
		seen++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen != 2 {
		t.Fatalf("persisted subscription count = %d, want 2", seen)
	}
	access, err := store.SelfAccess(ctx, beneficiary, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if access.GivesAccess || access.EffectiveTier != subscriptions.TierFree || access.AssignedTier == nil || *access.AssignedTier != subscriptions.TierPlus {
		t.Fatalf("beneficiary access = %+v, want fail-closed free with preserved Plus assignment", access)
	}
}

func TestCompletedSameIDProductChangesRemainFailClosed(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	tests := []struct {
		name      string
		responses []string
	}{
		{
			name: "pending change completes",
			responses: []string{
				`{"items":[{"id":"sub-change","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"auto_renewal_status":"will_change_product","pending_changes":{"product":{"id":"prod-business"}}}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"prod-business","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
			},
		},
		{
			name: "supported becomes unsupported then different tier",
			responses: []string{
				`{"items":[{"id":"sub-change","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"unknown-product","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"prod-business","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":null}`,
			},
		},
	}

	for testIndex, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, string(migration))
			store := subscriptions.NewStore(pool)
			catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{
				ProjectID: "project", AppIDs: []string{"app"},
				Products: map[string]subscriptions.ProductMapping{
					"prod-plus":     {AppID: "app", Tier: subscriptions.TierPlus},
					"prod-business": {AppID: "app", Tier: subscriptions.TierBusiness},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			owner := syntax.DID(fmt.Sprintf("did:plc:same-id-owner-%d", testIndex))
			beneficiary := syntax.DID(fmt.Sprintf("did:plc:same-id-beneficiary-%d", testIndex))
			account, _, err := store.EnsureAccount(ctx, owner)
			if err != nil {
				t.Fatal(err)
			}
			responseIndex := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(test.responses[responseIndex]))
				responseIndex++
			}))
			t.Cleanup(server.Close)
			client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "secret", ProjectID: "project"}, server.Client())
			if err != nil {
				t.Fatal(err)
			}

			applyNext := func(generation int64) {
				t.Helper()
				snapshot, err := client.ListCustomerSubscriptions(ctx, account.RevenueCatAppUserID)
				if err != nil {
					t.Fatal(err)
				}
				now := time.Date(2026, 9, 9, 12, int(generation), 0, 0, time.UTC)
				if err := store.RequestReconciliation(ctx, account.ID, now); err != nil {
					t.Fatal(err)
				}
				claim, ok, err := store.ClaimReconciliation(ctx, now, time.Minute)
				if err != nil || !ok {
					t.Fatalf("claim generation %d = %+v/%t, error %v", generation, claim, ok, err)
				}
				if err := store.ApplySnapshot(ctx, claim, snapshot, catalog); err != nil {
					t.Fatal(err)
				}
			}

			applyNext(1)
			if _, err := pool.Exec(ctx, `
				UPDATE billing_licenses SET assigned_did=$2,assigned_at=now()
				WHERE provider_subscription_id=(
					SELECT id FROM provider_subscriptions
					WHERE billing_account_id=$1 AND revenuecat_subscription_id='sub-change'
				)
			`, account.ID, beneficiary); err != nil {
				t.Fatal(err)
			}
			applyNext(2)
			applyNext(3)

			access, err := store.SelfAccess(ctx, beneficiary, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			if access.EffectiveTier != subscriptions.TierFree || access.GivesAccess || access.AssignedTier == nil || *access.AssignedTier != subscriptions.TierPlus {
				t.Fatalf("completed same-ID access = %+v, want free with preserved Plus assignment", access)
			}
			var productID, providerAnomaly, licenseAnomaly, assignedDID string
			var mappedTier *string
			var licenseTier string
			var assignable bool
			if err := pool.QueryRow(ctx, `
				SELECT subscription.product_id,subscription.mapped_tier,subscription.anomaly,
					license.tier,license.anomaly,license.assignable,license.assigned_did
				FROM provider_subscriptions subscription
				JOIN billing_licenses license ON license.provider_subscription_id=subscription.id
				WHERE subscription.billing_account_id=$1 AND subscription.revenuecat_subscription_id='sub-change'
			`, account.ID).Scan(&productID, &mappedTier, &providerAnomaly, &licenseTier, &licenseAnomaly, &assignable, &assignedDID); err != nil {
				t.Fatal(err)
			}
			if productID != "prod-business" || mappedTier == nil || *mappedTier != "business" || providerAnomaly != "cross_tier_conflict" || licenseTier != "plus" || licenseAnomaly != "cross_tier_conflict" || assignable || assignedDID != beneficiary.String() {
				t.Fatalf("completed same-ID state = product %q mapped %v provider anomaly %q license %q/%q assignable=%t assigned=%q", productID, mappedTier, providerAnomaly, licenseTier, licenseAnomaly, assignable, assignedDID)
			}
		})
	}
}

func TestPaginationFailureDoesNotApplyPartialSnapshot(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	store := subscriptions.NewStore(pool)
	account, _, err := store.EnsureAccount(ctx, syntax.DID("did:plc:pagination-owner"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{
		ProjectID: "project", AppIDs: []string{"app"},
		Products: map[string]subscriptions.ProductMapping{"prod-plus": {AppID: "app", Tier: subscriptions.TierPlus}},
	})
	if err != nil {
		t.Fatal(err)
	}
	path := "/v2/projects/project/customers/" + account.RevenueCatAppUserID.String() + "/subscriptions"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		if requests == 1 {
			_, _ = w.Write([]byte(`{"items":[{"id":"sub-partial","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true}],"next_page":"` + path + `?starting_after=sub-partial"}`))
			return
		}
		http.Error(w, "provider-private-canary", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{BaseURL: server.URL + "/v2", APIKey: "secret", ProjectID: "project"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	if err := store.RequestReconciliation(ctx, account.ID, now); err != nil {
		t.Fatal(err)
	}
	reconciler := subscriptions.NewReconciler(store, client, catalog, time.Minute, func() time.Time { return now })
	processed, err := reconciler.ProcessOne(ctx)
	if !processed || err == nil {
		t.Fatalf("process result = %t, error %v; want failed claimed reconciliation", processed, err)
	}
	if requests != 2 {
		t.Fatalf("provider request count = %d, want 2 to prove relative pagination reached the failing page", requests)
	}
	var subscriptionsCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM provider_subscriptions WHERE billing_account_id=$1`, account.ID).Scan(&subscriptionsCount); err != nil {
		t.Fatal(err)
	}
	if subscriptionsCount != 0 {
		t.Fatalf("persisted subscriptions = %d, want no partial snapshot commit", subscriptionsCount)
	}
	accountAfter, err := store.OwnerAccount(ctx, syntax.DID("did:plc:pagination-owner"))
	if err != nil {
		t.Fatal(err)
	}
	if accountAfter.ReconciledGeneration != 0 {
		t.Fatalf("reconciled generation = %d, want 0 after pagination failure", accountAfter.ReconciledGeneration)
	}
}
