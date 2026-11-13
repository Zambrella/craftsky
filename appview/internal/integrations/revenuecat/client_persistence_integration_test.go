package revenuecat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestMalformedProviderResponsePreservesReconciledAccessAndPins(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:malformed-provider-owner")
	postURI := "at://" + owner.String() + "/social.craftsky.feed.post/pinned"
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'cid')`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES($1,'active',1,1,'test',now(),now(),now())`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES($1,$2,'pinned','cid','Pinned','{}',now())`, postURI, owner); err != nil {
		t.Fatal(err)
	}
	store := subscriptions.NewStore(pool)
	account, _, err := store.EnsureAccount(ctx, owner)
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
	response := `{"items":[{"id":"sub-preserve","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false,"auto_renewal_status":"will_renew"}]}`
	var firstPage string
	requests := 0
	client, err := NewClient(ClientConfig{BaseURL: "https://revenuecat.invalid/v2", APIKey: "secret", ProjectID: "project"}, &http.Client{Transport: responseTransport(func(request *http.Request) *http.Response {
		requests++
		body := response
		if firstPage != "" && request.URL.Query().Get("starting_after") == "" {
			body = firstPage
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
	})})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	reconciler := subscriptions.NewReconciler(store, client, catalog, time.Minute, func() time.Time { return now })
	if err := reconciler.Trigger(ctx, account.ID, subscriptions.TriggerOwnerRefresh); err != nil {
		t.Fatal(err)
	}
	if processed, err := reconciler.ProcessOne(ctx); !processed || err != nil {
		t.Fatalf("initial paid reconciliation = %t, error %v", processed, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at) VALUES($1,'standard',$2,gen_random_uuid(),now(),now())`, owner, postURI); err != nil {
		t.Fatal(err)
	}
	before, err := store.OwnerAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, malformed := range []string{
		`null`, `{}`, `{"items":null}`,
		`{"items":[{"id":"sub-preserve","product_id":"prod-plus","store":"app_store","environment":"production","status":"expired","pending_payment":false}]}`,
		`{"items":[{"id":"sub-preserve","product_id":"prod-plus","store":"app_store","environment":"production","status":"expired","gives_access":false}]}`,
	} {
		for _, laterPage := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/later=%t", malformed, laterPage), func(t *testing.T) {
				response, firstPage, requests = malformed, "", 0
				if laterPage {
					path := "/v2/projects/project/customers/" + account.RevenueCatAppUserID.String() + "/subscriptions"
					firstPage = `{"items":[{"id":"sub-preserve","product_id":"prod-plus","store":"app_store","environment":"production","status":"expired","gives_access":false,"pending_payment":false}],"next_page":"` + path + `?starting_after=sub-preserve"}`
				}
				now = now.Add(2 * time.Minute)
				if err := reconciler.Trigger(ctx, account.ID, subscriptions.TriggerOwnerRefresh); err != nil {
					t.Fatal(err)
				}
				if processed, err := reconciler.ProcessOne(ctx); !processed || err == nil {
					t.Fatalf("malformed reconciliation = %t, error %v; want failed claim", processed, err)
				}
				wantRequests := 1
				if laterPage {
					wantRequests = 2
				}
				if requests != wantRequests {
					t.Fatalf("provider requests = %d, want %d", requests, wantRequests)
				}
				access, err := store.SelfAccess(ctx, owner, now)
				if err != nil || !access.AllowsPlus() {
					t.Fatalf("preserved access = %+v, error %v", access, err)
				}
				var pins int
				if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did=$1 AND post_uri=$2`, owner, postURI).Scan(&pins); err != nil || pins != 1 {
					t.Fatalf("preserved pins = %d, error %v", pins, err)
				}
				after, err := store.OwnerAccount(ctx, owner)
				if err != nil {
					t.Fatal(err)
				}
				if after.ReconciledGeneration != before.ReconciledGeneration || after.ReconciledAt == nil || before.ReconciledAt == nil || !after.ReconciledAt.Equal(*before.ReconciledAt) || after.RequestedGeneration <= after.ReconciledGeneration {
					t.Fatalf("malformed response changed confirmed reconciliation: before=%+v after=%+v", before, after)
				}
			})
		}
	}
}

func TestPendingProductChangeReachesPersistedFailClosedAnomaly(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000076_subscription_accounts.up.sql")
	migration = append(migration, []byte(`ALTER TABLE provider_subscriptions ADD COLUMN pending_mapped_tier TEXT;`)...)
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration)+`CREATE TABLE profile_pins(owner_did TEXT, slot TEXT);`)
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
		`{"items":[{"id":"sub-plus","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
		`{"items":[{"id":"sub-plus","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false,"auto_renewal_status":"will_change_product","pending_changes":{"product":{"id":"prod-business","future_product_field":"ignored"},"future_change_field":{"ignored":true}},"future_field":true},{"id":"sub-business","product_id":"prod-business","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
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
	beneficiary := account.OwnerDID
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

func TestCompletedSameIDProductChangesUpdateOwnedTier(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000076_subscription_accounts.up.sql")
	migration = append(migration, []byte(`ALTER TABLE provider_subscriptions ADD COLUMN pending_mapped_tier TEXT;`)...)
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
				`{"items":[{"id":"sub-change","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false,"auto_renewal_status":"will_change_product","pending_changes":{"product":{"id":"prod-business"}}}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"prod-business","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
			},
		},
		{
			name: "supported becomes unsupported then different tier",
			responses: []string{
				`{"items":[{"id":"sub-change","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"unknown-product","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
				`{"items":[{"id":"sub-change","product_id":"prod-business","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":null}`,
			},
		},
	}

	for testIndex, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, string(migration)+`CREATE TABLE profile_pins(owner_did TEXT, slot TEXT);`)
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
			beneficiary := owner
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
			if access.EffectiveTier != subscriptions.TierBusiness || !access.GivesAccess || access.AssignedTier == nil || *access.AssignedTier != subscriptions.TierBusiness {
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
			if productID != "prod-business" || mappedTier == nil || *mappedTier != "business" || providerAnomaly != "none" || licenseTier != "business" || licenseAnomaly != "none" || assignable || assignedDID != beneficiary.String() {
				t.Fatalf("completed same-ID state = product %q mapped %v provider anomaly %q license %q/%q assignable=%t assigned=%q", productID, mappedTier, providerAnomaly, licenseTier, licenseAnomaly, assignable, assignedDID)
			}
		})
	}
}

func TestPaginationFailureDoesNotApplyPartialSnapshot(t *testing.T) {
	migration, err := os.ReadFile("../../../migrations/000076_subscription_accounts.up.sql")
	migration = append(migration, []byte(`ALTER TABLE provider_subscriptions ADD COLUMN pending_mapped_tier TEXT;`)...)
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration)+`CREATE TABLE profile_pins(owner_did TEXT, slot TEXT);`)
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
			_, _ = w.Write([]byte(`{"items":[{"id":"sub-partial","product_id":"prod-plus","store":"app_store","environment":"production","status":"active","gives_access":true,"pending_payment":false}],"next_page":"` + path + `?starting_after=sub-partial"}`))
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

func TestScheduledDowngradeSerializesMappedPendingTierWithoutChangingAccess(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	store := subscriptions.NewStore(pool)
	owner := syntax.DID("did:plc:scheduled-downgrade")
	account, _, err := store.EnsureAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{ProjectID: "project", AppIDs: []string{"app", "other-app"}, Products: map[string]subscriptions.ProductMapping{
		"provider-plus":     {AppID: "app", Tier: subscriptions.TierPlus},
		"provider-business": {AppID: "app", Tier: subscriptions.TierBusiness},
		"other-store-plus":  {AppID: "other-app", Tier: subscriptions.TierPlus},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, product, pending, renewal string
		wantTier                        subscriptions.Tier
		wantPending                     string
	}{
		{"scheduled downgrade", "provider-business", "provider-plus", "will_change_product", subscriptions.TierBusiness, "plus"},
		{"unmapped pending product", "provider-business", "private-unmapped-pending-canary", "will_change_product", subscriptions.TierBusiness, ""},
		{"pending product from another provider app", "provider-business", "other-store-plus", "will_change_product", subscriptions.TierBusiness, ""},
		{"cancelled scheduled change", "provider-business", "", "will_renew", subscriptions.TierBusiness, ""},
		{"completed downgrade", "provider-plus", "", "will_renew", subscriptions.TierPlus, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("unexpected provider mutation %s", r.Method)
				}
				_, _ = fmt.Fprintf(w, `{"items":[{"id":"subscription","product_id":%q,"store":"play_store","environment":"production","status":"active","gives_access":true,"pending_payment":false,"auto_renewal_status":%q,"pending_changes":{"product":{"id":%q}}}],"next_page":null}`, tc.product, tc.renewal, tc.pending)
			}))
			defer server.Close()
			client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "secret", ProjectID: "project"}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := client.ListCustomerSubscriptions(ctx, account.RevenueCatAppUserID)
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			if err := store.RequestReconciliation(ctx, account.ID, now); err != nil {
				t.Fatal(err)
			}
			claim, ok, err := store.ClaimReconciliation(ctx, now, time.Minute)
			if err != nil || !ok {
				t.Fatalf("claim=%+v ok=%t err=%v", claim, ok, err)
			}
			if err := store.ApplySnapshot(ctx, claim, snapshot, catalog); err != nil {
				t.Fatal(err)
			}
			state, err := store.OwnerState(ctx, owner, now)
			if err != nil {
				t.Fatal(err)
			}
			if len(state.Subscriptions) != 1 || state.Subscriptions[0].AutoRenewalStatus == nil || *state.Subscriptions[0].AutoRenewalStatus != tc.renewal || state.ReconciledGeneration != claim.Generation {
				t.Fatalf("state=%+v", state)
			}
			encoded, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Subscriptions []map[string]any `json:"subscriptions"`
			}
			if err := json.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			actual, present := wire.Subscriptions[0]["pendingTier"]
			if (tc.wantPending != "") != present || (present && actual != tc.wantPending) {
				t.Fatalf("pendingTier=%v present=%t want=%s", actual, present, tc.wantPending)
			}
			if strings.Contains(string(encoded), "private-unmapped-pending-canary") || strings.Contains(string(encoded), "pendingProductId") {
				t.Fatal("pending provider identifier leaked into billing response")
			}
			access, err := store.SelfAccess(ctx, owner, now)
			if err != nil || !access.GivesAccess || access.EffectiveTier != tc.wantTier {
				t.Fatalf("effective access=%+v err=%v want=%s", access, err, tc.wantTier)
			}
		})
	}
}
