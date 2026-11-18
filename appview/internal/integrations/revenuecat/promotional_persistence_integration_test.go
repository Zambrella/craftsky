package revenuecat

import (
	"net/http"
	"os"
	"testing"
	"time"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func promotionalGrantSchema(t *testing.T) string {
	t.Helper()
	schema, err := os.ReadFile("../../../migrations/000126_billing_promotional_grants.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	return string(schema)
}

func TestProviderPromotionalGrantReconcilesSeparateAccessAndPaidOverlap(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := subscriptions.NewStore(pool)
	ctx := t.Context()
	const owner = "did:plc:provider-promotional-owner"
	account, _, err := store.EnsureAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	promo := promoResource()
	promo["entitlements"] = map[string]any{"items": []any{map[string]any{"id": "ent-business", "project_id": "project", "lookup_key": "business"}}}
	items := []any{promo}
	client, _ := promoClient(t, 3, func(*http.Request, int) (int, any) { return 200, map[string]any{"items": items} })
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{
		ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]subscriptions.ProductMapping{"prod-plus": {AppID: "app", Tier: subscriptions.TierPlus}},
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	reconciler := subscriptions.NewReconciler(store, client, catalog, time.Minute, func() time.Time { return now })
	refresh := func() {
		t.Helper()
		if err := reconciler.Trigger(ctx, account.ID, subscriptions.TriggerOwnerRefresh); err != nil {
			t.Fatal(err)
		}
		if processed, err := reconciler.ProcessOne(ctx); !processed || err != nil {
			t.Fatalf("reconcile processed=%t error=%v", processed, err)
		}
	}
	refresh()
	state, err := store.OwnerState(ctx, owner, now)
	if err != nil || len(state.Subscriptions) != 0 || len(state.Licenses) != 0 || len(state.PromotionalGrants) != 1 || !state.PromotionalGrants[0].GivesAccess {
		t.Fatalf("gift state %+v, error %v", state, err)
	}
	access, err := store.SelfAccess(ctx, owner, now)
	if err != nil || !access.AllowsBusiness() {
		t.Fatalf("gift access %+v, error %v", access, err)
	}
	paid := map[string]any{"id": "sub-paid", "product_id": "prod-plus", "store": "app_store", "environment": "production", "status": "active", "gives_access": true, "pending_payment": false, "auto_renewal_status": "will_renew"}
	items = []any{promo, paid}
	now = now.Add(time.Minute)
	refresh()
	state, err = store.OwnerState(ctx, owner, now)
	if err != nil || len(state.Subscriptions) != 1 || len(state.Licenses) != 1 || state.Subscriptions[0].Anomaly != "none" {
		t.Fatalf("overlap state %+v, error %v", state, err)
	}
	access, err = store.SelfAccess(ctx, owner, now)
	if err != nil || !access.AllowsBusiness() {
		t.Fatalf("overlap access %+v, error %v", access, err)
	}
	// A malformed grant must fail the entire snapshot, including the paid part.
	delete(promo, "ends_at")
	now = now.Add(2 * time.Minute)
	if err := reconciler.Trigger(ctx, account.ID, subscriptions.TriggerOwnerRefresh); err != nil {
		t.Fatal(err)
	}
	if processed, err := reconciler.ProcessOne(ctx); !processed || err == nil {
		t.Fatalf("malformed processed=%t error=%v", processed, err)
	}
	access, err = store.SelfAccess(ctx, owner, now)
	if err != nil || !access.AllowsBusiness() {
		t.Fatalf("preserved access %+v, error %v", access, err)
	}
	// Revocation/removal falls back to the independent paid Plus subscription.
	items = []any{paid}
	now = now.Add(2 * time.Minute)
	refresh()
	access, err = store.SelfAccess(ctx, owner, now)
	if err != nil || !access.AllowsPlus() || access.AllowsBusiness() {
		t.Fatalf("fallback access %+v, error %v", access, err)
	}
	state, err = store.OwnerState(ctx, owner, now)
	if err != nil || len(state.PromotionalGrants) != 1 || state.PromotionalGrants[0].GivesAccess {
		t.Fatalf("revoked state %+v, error %v", state, err)
	}
}
