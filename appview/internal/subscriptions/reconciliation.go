package subscriptions

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ReconciliationTrigger string

const (
	TriggerWebhook      ReconciliationTrigger = "webhook"
	TriggerOwnerRefresh ReconciliationTrigger = "owner_refresh"
	TriggerRestore      ReconciliationTrigger = "restore"
	TriggerScheduled    ReconciliationTrigger = "scheduled"
)

type ProviderClient interface {
	ListCustomerSubscriptions(context.Context, uuid.UUID) (CompleteSnapshot, error)
}

type Reconciler struct {
	store         *Store
	provider      ProviderClient
	catalog       *Catalog
	leaseDuration time.Duration
	now           func() time.Time
	observer      ReconciliationObserver
}

type ReconciliationObserver interface {
	ObserveRevenueCatReconciliation(context.Context, string, string, time.Duration)
}

const reconciliationRetryDelay = time.Minute

func NewReconciler(store *Store, provider ProviderClient, catalog *Catalog, leaseDuration time.Duration, now func() time.Time, observers ...ReconciliationObserver) *Reconciler {
	var observer ReconciliationObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &Reconciler{store: store, provider: provider, catalog: catalog, leaseDuration: leaseDuration, now: now, observer: observer}
}

func (r *Reconciler) Trigger(ctx context.Context, accountID uuid.UUID, _ ReconciliationTrigger) error {
	return r.store.RequestReconciliation(ctx, accountID, r.now())
}

func (r *Reconciler) ScheduleActiveAccounts(ctx context.Context) (int64, error) {
	started := time.Now()
	count, err := r.store.ScheduleActiveAccounts(ctx, r.now())
	if r.observer != nil {
		outcome := "success"
		if err != nil {
			outcome = "store_error"
		}
		r.observer.ObserveRevenueCatReconciliation(ctx, "schedule", outcome, time.Since(started))
	}
	return count, err
}

func (r *Reconciler) ProcessOne(ctx context.Context) (bool, error) {
	started := time.Now()
	observe := func(outcome string) {
		if r.observer != nil {
			r.observer.ObserveRevenueCatReconciliation(ctx, "process", outcome, time.Since(started))
		}
	}
	claim, ok, err := r.store.ClaimReconciliation(ctx, r.now(), r.leaseDuration)
	if err != nil || !ok {
		if err != nil {
			observe("store_error")
		} else {
			observe("empty")
		}
		return ok, err
	}
	// Provider I/O occurs after ClaimReconciliation commits and before ApplySnapshot opens its transaction.
	snapshot, err := r.provider.ListCustomerSubscriptions(ctx, claim.RevenueCatAppUserID)
	if err != nil {
		_ = r.store.FailReconciliation(ctx, claim, r.now(), reconciliationRetryDelay)
		observe("provider_error")
		return true, err
	}
	if err := r.store.ApplySnapshot(ctx, claim, snapshot, r.catalog); err != nil {
		_ = r.store.FailReconciliation(ctx, claim, r.now(), reconciliationRetryDelay)
		observe("apply_error")
		return true, err
	}
	observe("success")
	return true, nil
}
