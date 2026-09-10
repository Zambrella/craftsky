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

type retryProvider struct {
	snapshot CompleteSnapshot
	err      error
	calls    int
}

func (provider *retryProvider) ListCustomerSubscriptions(context.Context, uuid.UUID) (CompleteSnapshot, error) {
	provider.calls++
	if provider.err != nil {
		return CompleteSnapshot{}, provider.err
	}
	return provider.snapshot, nil
}

func TestReconciliationProcessorRetriesAndRejectsSupersededLeases(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	store := NewStore(pool)
	account, _, err := store.EnsureAccount(context.Background(), syntax.DID("did:plc:worker-owner"))
	if err != nil {
		t.Fatal(err)
	}
	catalog, _ := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{
		"plus": {AppID: "app", Tier: TierPlus},
	}})
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	provider := &retryProvider{err: errors.New("provider unavailable"), snapshot: CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		{ID: "subscription", ProductID: "plus", Environment: "production", Store: "app_store", Status: "active", GivesAccess: true},
	}}}
	reconciler := NewReconciler(store, provider, catalog, time.Minute, func() time.Time { return now })
	if err := reconciler.Trigger(context.Background(), account.ID, TriggerScheduled); err != nil {
		t.Fatal(err)
	}
	if processed, err := reconciler.ProcessOne(context.Background()); !processed || err == nil {
		t.Fatalf("failed provider attempt = processed %t, error %v", processed, err)
	}
	var leaseToken *uuid.UUID
	var nextAttempt *time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT lease_token,next_attempt_at FROM billing_accounts WHERE id=$1
	`, account.ID).Scan(&leaseToken, &nextAttempt); err != nil {
		t.Fatal(err)
	}
	if leaseToken != nil || nextAttempt == nil || !nextAttempt.After(now) {
		t.Fatalf("retry state = lease %v next %v", leaseToken, nextAttempt)
	}
	provider.err = nil
	now = *nextAttempt
	if processed, err := reconciler.ProcessOne(context.Background()); !processed || err != nil {
		t.Fatalf("retry success = processed %t, error %v", processed, err)
	}

	if err := reconciler.Trigger(context.Background(), account.ID, TriggerRestore); err != nil {
		t.Fatal(err)
	}
	oldClaim, ok, err := store.ClaimReconciliation(context.Background(), now, time.Minute)
	if err != nil || !ok {
		t.Fatalf("old claim = %+v, %t, %v", oldClaim, ok, err)
	}
	now = now.Add(time.Minute + time.Second)
	newClaim, ok, err := store.ClaimReconciliation(context.Background(), now, time.Minute)
	if err != nil || !ok || newClaim.LeaseToken == oldClaim.LeaseToken {
		t.Fatalf("replacement claim = %+v, %t, %v; old %+v", newClaim, ok, err, oldClaim)
	}
	if err := store.ApplySnapshot(context.Background(), oldClaim, provider.snapshot, catalog); !errors.Is(err, ErrStaleSnapshot) {
		t.Fatalf("superseded apply = %v, want ErrStaleSnapshot", err)
	}
	if err := store.ApplySnapshot(context.Background(), newClaim, provider.snapshot, catalog); err != nil {
		t.Fatalf("replacement apply: %v", err)
	}
}
