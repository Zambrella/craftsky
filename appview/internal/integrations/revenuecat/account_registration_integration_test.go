package revenuecat

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestEnsuredAccountRegistersExactCustomerThroughWorkerWithoutSDK(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	store := subscriptions.NewStore(pool)
	ctx := t.Context()
	account, created, err := store.EnsureAccount(ctx, "did:plc:eager-registration")
	if err != nil || !created {
		t.Fatalf("ensure account: created=%t error=%v", created, err)
	}
	requests := 0
	registered := false
	creationUnavailable := true
	client, err := NewClient(ClientConfig{BaseURL: "https://revenuecat.invalid/v2", APIKey: "secret", ProjectID: "project"}, &http.Client{Transport: responseTransport(func(request *http.Request) *http.Response {
		requests++
		status := http.StatusOK
		body := `{"items":[]}`
		switch request.Method {
		case http.MethodGet:
			if request.URL.Path != "/v2/projects/project/customers/"+account.RevenueCatAppUserID.String()+"/subscriptions" {
				t.Fatalf("registration read used an unexpected customer path")
			}
			if !registered {
				status = http.StatusNotFound
			}
		case http.MethodPost:
			if request.URL.Path != "/v2/projects/project/customers" {
				t.Fatal("registration used an unexpected creation path")
			}
			var customer struct {
				ID string `json:"id"`
			}
			if err := json.NewDecoder(request.Body).Decode(&customer); err != nil || customer.ID != account.RevenueCatAppUserID.String() {
				t.Fatalf("registration did not use the server-owned UUID: error=%v", err)
			}
			if creationUnavailable {
				status = http.StatusServiceUnavailable
			} else {
				registered = true
				status = http.StatusCreated
			}
		default:
			t.Fatal("registration used an unexpected request method")
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
	})})
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{ProjectID: "project"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(time.Second)
	worker := subscriptions.NewReconciler(store, client, catalog, time.Minute, func() time.Time { return now })
	if requests != 0 {
		t.Fatal("ensuring an account performed provider I/O")
	}
	if processed, err := worker.ProcessOne(ctx); !processed || err == nil || requests != 2 || registered {
		t.Fatalf("failed registration: processed=%t requests=%d registered=%t error=%v", processed, requests, registered, err)
	}
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `SELECT next_attempt_at FROM billing_accounts WHERE id=$1`, account.ID).Scan(&nextAttempt); err != nil {
		t.Fatal(err)
	}
	repeated, created, err := store.EnsureAccount(ctx, account.OwnerDID)
	if err != nil || created || repeated.RequestedGeneration != 1 || repeated.RevenueCatAppUserID != account.RevenueCatAppUserID || requests != 2 {
		t.Fatalf("repeat after provider failure: account=%+v created=%t requests=%d error=%v", repeated, created, requests, err)
	}
	if processed, err := worker.ProcessOne(ctx); processed || err != nil || requests != 2 {
		t.Fatalf("registration bypassed durable backoff: processed=%t requests=%d error=%v", processed, requests, err)
	}
	creationUnavailable = false
	now = nextAttempt
	if processed, err := worker.ProcessOne(ctx); !processed || err != nil || requests != 5 || !registered {
		t.Fatalf("registration retry: processed=%t requests=%d registered=%t error=%v", processed, requests, registered, err)
	}
	state, err := store.OwnerState(ctx, account.OwnerDID, now)
	if err != nil || state.ReconciliationStale || state.RequestedGeneration != 1 || state.ReconciledGeneration != 1 || state.ReconciledAt == nil {
		t.Fatalf("registered customer state=%+v error=%v", state, err)
	}
	if _, created, err := store.EnsureAccount(ctx, account.OwnerDID); err != nil || created {
		t.Fatalf("ensure registered customer: created=%t error=%v", created, err)
	}
	if processed, err := worker.ProcessOne(ctx); processed || err != nil || requests != 5 {
		t.Fatalf("registered customer was queued again: processed=%t requests=%d error=%v", processed, requests, err)
	}
}
