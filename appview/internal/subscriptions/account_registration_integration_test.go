package subscriptions

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func TestEnsureAccountRegistrationPreservesPendingClaimsBackoffAndCompletedState(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := NewStore(pool)
	ctx := t.Context()
	owner := syntax.DID("did:plc:initial-registration")
	account, created, err := store.EnsureAccount(ctx, owner)
	if err != nil || !created || account.RequestedGeneration != 1 || account.ReconciledGeneration != 0 || account.ReconciliationRequested == nil {
		t.Fatalf("new registration = %+v, created=%t error=%v", account, created, err)
	}

	type queueState struct {
		requested, reconciled int64
		requestedAt, next     *time.Time
		claimed               *int64
		token                 *uuid.UUID
		expires               *time.Time
		attempts              int
	}
	readQueue := func() queueState {
		t.Helper()
		var state queueState
		if err := pool.QueryRow(ctx, `SELECT requested_generation,reconciled_generation,reconciliation_requested_at,next_attempt_at,claimed_generation,lease_token,lease_expires_at,reconciliation_attempts FROM billing_accounts WHERE id=$1`, account.ID).Scan(
			&state.requested, &state.reconciled, &state.requestedAt, &state.next,
			&state.claimed, &state.token, &state.expires, &state.attempts,
		); err != nil {
			t.Fatal(err)
		}
		return state
	}
	ensureUnchanged := func() {
		t.Helper()
		before := readQueue()
		for range 3 {
			current, created, err := store.EnsureAccount(ctx, owner)
			if err != nil || created || current.ID != account.ID || current.RevenueCatAppUserID != account.RevenueCatAppUserID {
				t.Fatalf("repeated registration = %+v, created=%t error=%v", current, created, err)
			}
		}
		if after := readQueue(); !reflect.DeepEqual(before, after) {
			t.Fatalf("ensure reset registration queue: before=%+v after=%+v", before, after)
		}
	}
	ensureUnchanged()

	now := time.Now().Add(time.Second)
	claim, found, err := store.ClaimReconciliation(ctx, now, time.Minute)
	if err != nil || !found || claim.BillingAccountID != account.ID || claim.Generation != 1 {
		t.Fatalf("initial registration claim=%+v found=%t error=%v", claim, found, err)
	}
	ensureUnchanged()
	if err := store.FailReconciliation(ctx, claim, now, 5*time.Minute); err != nil {
		t.Fatal(err)
	}
	ensureUnchanged()
	if _, found, err := store.ClaimReconciliation(ctx, now, time.Minute); err != nil || found {
		t.Fatalf("registration backoff bypassed: found=%t error=%v", found, err)
	}

	now = now.Add(5 * time.Minute)
	claim, found, err = store.ClaimReconciliation(ctx, now, time.Minute)
	if err != nil || !found || claim.Generation != 1 {
		t.Fatalf("registration retry=%+v found=%t error=%v", claim, found, err)
	}
	catalog, err := NewCatalog(CatalogConfig{ProjectID: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ApplySnapshot(ctx, claim, CompleteSnapshot{Complete: true}, catalog); err != nil {
		t.Fatal(err)
	}
	ensureUnchanged()
	current, err := store.OwnerAccount(ctx, owner)
	if err != nil || current.RequestedGeneration != 1 || current.ReconciledGeneration != 1 || current.ReconciledAt == nil {
		t.Fatalf("completed registration=%+v error=%v", current, err)
	}
	if _, found, err := store.ClaimReconciliation(ctx, now, time.Minute); err != nil || found {
		t.Fatalf("completed registration queued again: found=%t error=%v", found, err)
	}
}

func TestEnsureAccountRepairsUnqueuedRegistrationOnceUnderConcurrency(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := NewStore(pool)
	ctx := t.Context()
	owner := syntax.DID("did:plc:unqueued-registration")
	id, customerID := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES($1,$2,$3)`, id, owner, customerID); err != nil {
		t.Fatal(err)
	}
	const callers = 8
	results := make(chan BillingAccount, callers)
	errorsCh := make(chan error, callers)
	var ready sync.WaitGroup
	ready.Add(callers)
	start := make(chan struct{})
	for range callers {
		go func() {
			ready.Done()
			<-start
			account, created, err := store.EnsureAccount(ctx, owner)
			if err == nil && created {
				t.Error("repair created a replacement account")
			}
			if err != nil {
				errorsCh <- err
				return
			}
			results <- account
		}()
	}
	ready.Wait()
	close(start)
	for range callers {
		select {
		case err := <-errorsCh:
			t.Fatal(err)
		case account := <-results:
			if account.ID != id || account.RevenueCatAppUserID != customerID || account.RequestedGeneration != 1 || account.ReconciliationRequested == nil {
				t.Fatalf("repaired account=%+v", account)
			}
		}
	}
	claim, found, err := store.ClaimReconciliation(context.Background(), time.Now().Add(time.Second), time.Minute)
	if err != nil || !found || claim.BillingAccountID != id || claim.RevenueCatAppUserID != customerID || claim.Generation != 1 {
		t.Fatalf("repair claim=%+v found=%t error=%v", claim, found, err)
	}
}
