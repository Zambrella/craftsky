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

type scriptedProvider struct {
	snapshot CompleteSnapshot
	calls    int
	onFetch  func()
}

func (provider *scriptedProvider) ListCustomerSubscriptions(context.Context, uuid.UUID) (CompleteSnapshot, error) {
	provider.calls++
	if provider.onFetch != nil {
		provider.onFetch()
	}
	return provider.snapshot, nil
}

func TestReconciliationTriggersUseOneCompleteSnapshotPath(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	store := NewStore(pool)
	account, _, err := store.EnsureAccount(context.Background(), syntax.DID("did:plc:reconcile-owner"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, _ := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{
		"plus": {AppID: "app", Tier: TierPlus},
	}})
	provider := &scriptedProvider{snapshot: CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		{ID: "subscription", ProductID: "plus", Environment: "production", Store: "app_store", Status: "active", GivesAccess: true},
	}}}
	reconciler := NewReconciler(store, provider, catalog, time.Minute, func() time.Time {
		return time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	})

	for _, trigger := range []ReconciliationTrigger{TriggerWebhook, TriggerOwnerRefresh, TriggerRestore, TriggerScheduled} {
		if err := reconciler.Trigger(context.Background(), account.ID, trigger); err != nil {
			t.Fatalf("trigger %s: %v", trigger, err)
		}
	}
	provider.onFetch = func() {
		provider.onFetch = nil
		if err := reconciler.Trigger(context.Background(), account.ID, TriggerWebhook); err != nil {
			t.Errorf("trigger during fetch: %v", err)
		}
	}
	processed, err := reconciler.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("process first claim = %t, %v", processed, err)
	}
	var requested, reconciled int64
	if err := pool.QueryRow(context.Background(), `
		SELECT requested_generation,reconciled_generation FROM billing_accounts WHERE id=$1
	`, account.ID).Scan(&requested, &reconciled); err != nil {
		t.Fatal(err)
	}
	if requested != 5 || reconciled != 4 {
		t.Fatalf("generations after trigger during fetch = requested %d reconciled %d, want 5/4", requested, reconciled)
	}
	processed, err = reconciler.ProcessOne(context.Background())
	if err != nil || !processed || provider.calls != 2 {
		t.Fatalf("process pending generation = %t, calls %d, error %v", processed, provider.calls, err)
	}
}
