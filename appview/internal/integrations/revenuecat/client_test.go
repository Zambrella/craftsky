package revenuecat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestClientMapsCompleteCustomerSubscriptionSnapshot(t *testing.T) {
	customerID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer provider-secret" {
			t.Fatalf("authorization header was not configured")
		}
		wantPath := "/v2/projects/proj/customers/" + customerID.String() + "/subscriptions"
		if r.URL.Path != wantPath {
			t.Fatalf("request path = %q, want %q", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("starting_after") {
		case "":
			_, _ = w.Write([]byte(`{
				"object":"list",
				"items":[{
					"object":"subscription","id":"sub-plus","customer_id":"customer",
					"product_id":"prod-plus","store":"app_store","environment":"production",
					"status":"in_grace_period","gives_access":true,"pending_payment":false,
					"auto_renewal_status":"will_renew","starts_at":1788220800000,
					"pending_changes":{"product":{"id":"prod-business","future_product_field":"ignored"},"future_change_field":"ignored"},
					"current_period_starts_at":1788220800000,
					"current_period_ends_at":1790812800000,"ends_at":1790812800000,
					"future_additive_field":{"ignored":true}
				}],
				"next_page":"` + wantPath + `?starting_after=sub-plus"
			}`))
		case "sub-plus":
			_, _ = w.Write([]byte(`{
				"object":"list",
				"items":[{
					"object":"subscription","id":"sub-business","customer_id":"customer",
					"product_id":"prod-business","store":"play_store","environment":"production",
					"status":"active","gives_access":false,"pending_payment":true,
					"auto_renewal_status":"will_not_renew"
				}],
				"next_page":null
			}`))
		default:
			http.Error(w, "unexpected page", http.StatusBadRequest)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		BaseURL:   server.URL + "/v2",
		APIKey:    "provider-secret",
		ProjectID: "proj",
		PageLimit: 1,
		MaxPages:  3,
	}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.ListCustomerSubscriptions(context.Background(), customerID)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Complete || len(snapshot.Subscriptions) != 2 {
		t.Fatalf("snapshot = %+v, want two complete subscriptions", snapshot)
	}
	plus := snapshot.Subscriptions[0]
	if plus.ID != "sub-plus" || plus.ProductID != "prod-plus" || plus.Store != "app_store" ||
		plus.Environment != "production" || plus.Status != "in_grace_period" || !plus.GivesAccess ||
		plus.PendingPayment || plus.AutoRenewalStatus != "will_renew" || plus.PendingProductID != "prod-business" {
		t.Fatalf("mapped Plus subscription = %+v", plus)
	}
	wantEnd := time.UnixMilli(1790812800000).UTC()
	if plus.CurrentPeriodEndsAt == nil || !plus.CurrentPeriodEndsAt.Equal(wantEnd) ||
		plus.EndsAt == nil || !plus.EndsAt.Equal(wantEnd) {
		t.Fatalf("mapped informational ends = %v / %v, want %v", plus.CurrentPeriodEndsAt, plus.EndsAt, wantEnd)
	}
	business := snapshot.Subscriptions[1]
	if business.ID != "sub-business" || business.GivesAccess || !business.PendingPayment {
		t.Fatalf("mapped Business subscription = %+v", business)
	}
}

func TestClientRejectsUntrustedOrUnscopedPaginationURLs(t *testing.T) {
	customerID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	path := "/v2/projects/proj/customers/" + customerID.String() + "/subscriptions"

	tests := []struct {
		name     string
		nextPage string
	}{
		{name: "cross-origin absolute", nextPage: `https://example.invalid/v2/projects/proj/customers/` + customerID.String() + `/subscriptions?starting_after=sub-one`},
		{name: "scheme-relative external", nextPage: `//example.invalid/v2/projects/proj/customers/` + customerID.String() + `/subscriptions?starting_after=sub-one`},
		{name: "different project", nextPage: `/v2/projects/other/customers/` + customerID.String() + `/subscriptions?starting_after=sub-one`},
		{name: "different customer", nextPage: `/v2/projects/proj/customers/30000000-0000-0000-0000-000000000003/subscriptions?starting_after=sub-one`},
		{name: "endpoint escape", nextPage: `/v2/projects/proj/customers/` + customerID.String() + `/purchases?starting_after=sub-one`},
		{name: "traversal", nextPage: `/v2/projects/proj/customers/` + customerID.String() + `/subscriptions/../purchases?starting_after=sub-one`},
		{name: "malformed", nextPage: `/v2/projects/proj/customers/%zz/subscriptions?starting_after=sub-one`},
		{name: "fragment", nextPage: path + `?starting_after=sub-one#private`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				if requests > 1 {
					t.Fatalf("untrusted pagination URL was requested: %s", r.URL.String())
				}
				_, _ = w.Write([]byte(`{"items":[{"id":"sub-one"}],"next_page":"` + test.nextPage + `"}`))
			}))
			t.Cleanup(server.Close)
			client, err := NewClient(ClientConfig{BaseURL: server.URL + "/v2", APIKey: "secret", ProjectID: "proj"}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := client.ListCustomerSubscriptions(context.Background(), customerID)
			if err == nil || snapshot.Complete || len(snapshot.Subscriptions) != 0 {
				t.Fatalf("result = %+v, error %v; want empty failed snapshot", snapshot, err)
			}
			if requests != 1 {
				t.Fatalf("request count = %d, want 1", requests)
			}
		})
	}
}

func TestClientRejectsPaginationLoopsAndBoundsPages(t *testing.T) {
	customerID := uuid.MustParse("40000000-0000-0000-0000-000000000004")
	path := "/v2/projects/proj/customers/" + customerID.String() + "/subscriptions"

	t.Run("loop", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests++
			_, _ = w.Write([]byte(`{"items":[{"id":"sub-one"}],"next_page":"` + path + `?starting_after=sub-one"}`))
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(ClientConfig{BaseURL: server.URL + "/v2", APIKey: "secret", ProjectID: "proj", PageLimit: 1, MaxPages: 3}, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := client.ListCustomerSubscriptions(context.Background(), customerID)
		if err == nil || snapshot.Complete || len(snapshot.Subscriptions) != 0 || requests != 2 {
			t.Fatalf("loop result = %+v, error %v, requests %d; want empty failure after two requests", snapshot, err, requests)
		}
	})

	t.Run("page limit", func(t *testing.T) {
		requests := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests++
			next := path + `?starting_after=sub-` + string(rune('0'+requests))
			_, _ = w.Write([]byte(`{"items":[{"id":"sub"}],"next_page":"` + next + `"}`))
		}))
		t.Cleanup(server.Close)
		client, err := NewClient(ClientConfig{BaseURL: server.URL + "/v2", APIKey: "secret", ProjectID: "proj", MaxPages: 2}, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := client.ListCustomerSubscriptions(context.Background(), customerID)
		if err == nil || snapshot.Complete || len(snapshot.Subscriptions) != 0 || requests != 2 {
			t.Fatalf("bounded result = %+v, error %v, requests %d; want empty failure after two pages", snapshot, err, requests)
		}
	})
}

func TestClientRejectsOversizedAndErrorPaginationPages(t *testing.T) {
	customerID := uuid.MustParse("50000000-0000-0000-0000-000000000005")
	path := "/v2/projects/proj/customers/" + customerID.String() + "/subscriptions"
	for _, test := range []struct {
		name       string
		secondPage func(http.ResponseWriter)
	}{
		{name: "provider error", secondPage: func(w http.ResponseWriter) { http.Error(w, "private-provider-canary", http.StatusBadGateway) }},
		{name: "oversized", secondPage: func(w http.ResponseWriter) { _, _ = w.Write([]byte(strings.Repeat("x", 257))) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests++
				if requests == 1 {
					_, _ = w.Write([]byte(`{"items":[{"id":"sub-one"}],"next_page":"` + path + `?starting_after=sub-one"}`))
					return
				}
				test.secondPage(w)
			}))
			t.Cleanup(server.Close)
			client, err := NewClient(ClientConfig{BaseURL: server.URL + "/v2", APIKey: "secret", ProjectID: "proj", MaxResponseBytes: 256}, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := client.ListCustomerSubscriptions(context.Background(), customerID)
			if err == nil || strings.Contains(err.Error(), "private-provider-canary") || snapshot.Complete || len(snapshot.Subscriptions) != 0 || requests != 2 {
				t.Fatalf("result = %+v, error %v, requests %d; want private empty failure on second page", snapshot, err, requests)
			}
		})
	}
}

func TestClientDoesNotForwardAuthorizationOnRedirect(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path == "/private" && r.Header.Get("Authorization") != "" {
			t.Fatal("authorization was forwarded outside the customer subscriptions endpoint")
		}
		if requests == 1 {
			http.Redirect(w, r, "/private", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte(`{"items":[],"next_page":null}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{BaseURL: server.URL + "/v2", APIKey: "secret", ProjectID: "proj"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
	if err == nil || snapshot.Complete || requests != 1 {
		t.Fatalf("redirect result = %+v, error %v, requests %d; want one failed scoped request", snapshot, err, requests)
	}
}

func TestClientPreservesAutoRenewalStatusEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"items":[
				{"id":"missing","product_id":"product","store":"app_store","environment":"production","status":"expired","gives_access":false},
				{"id":"empty","product_id":"product","store":"app_store","environment":"production","status":"expired","gives_access":false,"auto_renewal_status":""},
				{"id":"unknown","product_id":"product","store":"app_store","environment":"production","status":"expired","gives_access":false,"auto_renewal_status":"future_provider_state"},
				{"id":"resumable","product_id":"product","store":"app_store","environment":"production","status":"expired","gives_access":false,"auto_renewal_status":"will_pause"},
				{"id":"non-renewing","product_id":"product","store":"app_store","environment":"production","status":"expired","gives_access":false,"auto_renewal_status":"will_not_renew"}
			],
			"next_page":null
		}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: "secret", ProjectID: "project"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"", "", "future_provider_state", "will_pause", "will_not_renew"}
	if len(snapshot.Subscriptions) != len(want) {
		t.Fatalf("subscription count = %d, want %d", len(snapshot.Subscriptions), len(want))
	}
	for i, subscription := range snapshot.Subscriptions {
		if subscription.AutoRenewalStatus != want[i] {
			t.Fatalf("subscription %q auto-renewal status = %q, want %q", subscription.ID, subscription.AutoRenewalStatus, want[i])
		}
	}
}

func TestClientProviderErrorsDoNotExposeResponseCanaries(t *testing.T) {
	const canary = "provider-private-receipt-secret-canary"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, canary, http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(ClientConfig{BaseURL: server.URL, APIKey: canary, ProjectID: "project"}, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListCustomerSubscriptions(context.Background(), uuid.New())
	if err == nil || strings.Contains(err.Error(), canary) {
		t.Fatalf("provider error = %v, want generic error without canary", err)
	}
}
