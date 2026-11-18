package revenuecat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func promoResource() map[string]any {
	return map[string]any{
		"id": "sub-promo", "product_id": nil, "store": "promotional",
		"environment": "production", "status": "active", "gives_access": true,
		"pending_payment": false, "auto_renewal_status": "will_not_renew",
		"current_period_ends_at": int64(1815523200000), "ends_at": int64(1815523200000),
		"entitlements": map[string]any{"items": []any{map[string]any{"id": "ent-plus", "project_id": "project", "lookup_key": "plus"}}},
	}
}

func promoClient(t *testing.T, maxPages int, respond func(*http.Request, int) (int, any)) (*Client, *int) {
	t.Helper()
	requests := 0
	client, err := NewClient(ClientConfig{BaseURL: "https://revenuecat.invalid/v2", APIKey: "secret", ProjectID: "project", MaxPages: maxPages}, &http.Client{Transport: responseTransport(func(r *http.Request) *http.Response {
		requests++
		status, body := respond(r, requests)
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(string(encoded))), Header: make(http.Header)}
	})})
	if err != nil {
		t.Fatal(err)
	}
	return client, &requests
}

func TestClientMapsPromotionalEntitlementsAndExplicitExpiry(t *testing.T) {
	for _, lifetime := range []bool{false, true} {
		t.Run(map[bool]string{false: "timed", true: "lifetime"}[lifetime], func(t *testing.T) {
			item := promoResource()
			if lifetime {
				item["current_period_ends_at"], item["ends_at"] = nil, nil
			}
			client, _ := promoClient(t, 3, func(*http.Request, int) (int, any) { return 200, map[string]any{"items": []any{item}} })
			got, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
			if err != nil || !got.Complete || len(got.Subscriptions) != 1 {
				t.Fatalf("result %+v, error %v", got, err)
			}
			sub := got.Subscriptions[0]
			if sub.ProductID != "" || sub.Store != "promotional" || len(sub.Entitlements) != 1 || sub.Entitlements[0].LookupKey != "plus" || sub.Entitlements[0].ProjectID != "project" {
				t.Fatalf("promo %+v", sub)
			}
			if lifetime {
				if sub.EndsAt != nil || sub.CurrentPeriodEndsAt != nil {
					t.Fatal("lifetime expiry changed")
				}
			} else if sub.EndsAt == nil || !sub.EndsAt.Equal(time.UnixMilli(1815523200000)) {
				t.Fatalf("expiry %v", sub.EndsAt)
			}
		})
	}
}

func TestClientRejectsIncompletePromotionalMetadata(t *testing.T) {
	changes := map[string]func(map[string]any){
		"missing entitlements":  func(x map[string]any) { delete(x, "entitlements") },
		"null entitlements":     func(x map[string]any) { x["entitlements"] = nil },
		"missing items":         func(x map[string]any) { x["entitlements"] = map[string]any{} },
		"null items":            func(x map[string]any) { x["entitlements"] = map[string]any{"items": nil} },
		"null item":             func(x map[string]any) { x["entitlements"] = map[string]any{"items": []any{nil}} },
		"missing expiry":        func(x map[string]any) { delete(x, "ends_at") },
		"missing period expiry": func(x map[string]any) { delete(x, "current_period_ends_at") },
		"malformed expiry":      func(x map[string]any) { x["ends_at"] = "tomorrow" },
	}
	for _, field := range []string{"id", "project_id", "lookup_key"} {
		field := field
		changes["missing entitlement "+field] = func(x map[string]any) {
			delete(x["entitlements"].(map[string]any)["items"].([]any)[0].(map[string]any), field)
		}
	}
	changes["wrong project"] = func(x map[string]any) {
		x["entitlements"].(map[string]any)["items"].([]any)[0].(map[string]any)["project_id"] = "other"
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			item := promoResource()
			change(item)
			client, _ := promoClient(t, 3, func(*http.Request, int) (int, any) { return 200, map[string]any{"items": []any{item}} })
			got, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
			if err == nil || got.Complete || len(got.Subscriptions) != 0 {
				t.Fatalf("result %+v, error %v", got, err)
			}
		})
	}
}

func TestClientCompletelyPaginatesPromotionalEntitlements(t *testing.T) {
	item := promoResource()
	next := "/v2/projects/project/subscriptions/sub-promo/entitlements?starting_after=ent-plus"
	item["entitlements"].(map[string]any)["next_page"] = next
	client, requests := promoClient(t, 3, func(r *http.Request, count int) (int, any) {
		if count == 1 {
			return 200, map[string]any{"items": []any{item}}
		}
		if r.URL.RequestURI() != next || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("unexpected request %s", r.URL.RequestURI())
		}
		return 200, map[string]any{"items": []any{map[string]any{"id": "ent-business", "project_id": "project", "lookup_key": "business"}, map[string]any{"id": "ent-other", "project_id": "project", "lookup_key": "unknown"}}}
	})
	got, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
	if err != nil || !got.Complete || len(got.Subscriptions) != 1 || len(got.Subscriptions[0].Entitlements) != 3 || *requests != 2 {
		t.Fatalf("result %+v, error %v, requests %d", got, err, *requests)
	}
}

func TestClientRejectsIncompletePromotionalPagination(t *testing.T) {
	path := "/v2/projects/project/subscriptions/sub-promo/entitlements"
	tests := []struct {
		name, next       string
		maxPages, status int
		later            any
		requests         int
	}{
		{"cross origin", "https://evil.invalid" + path + "?starting_after=a", 3, 200, nil, 1},
		{"wrong subscription", "/v2/projects/project/subscriptions/other/entitlements?starting_after=a", 3, 200, nil, 1},
		{"wrong endpoint", "/v2/projects/project/customers/other/subscriptions?starting_after=a", 3, 200, nil, 1},
		{"missing cursor", path, 3, 200, nil, 1},
		{"empty next", "", 3, 200, nil, 1},
		{"error later", path + "?starting_after=a", 3, 503, map[string]any{"private": "canary"}, 2},
		{"malformed later", path + "?starting_after=a", 3, 200, map[string]any{}, 2},
		{"loop", path + "?starting_after=a", 3, 200, map[string]any{"items": []any{}, "next_page": path + "?starting_after=a"}, 2},
		{"page bound", path + "?starting_after=a", 1, 200, nil, 1},
		{"duplicate entitlement", path + "?starting_after=a", 3, 200, promoResource()["entitlements"], 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item := promoResource()
			item["entitlements"].(map[string]any)["next_page"] = tc.next
			client, requests := promoClient(t, tc.maxPages, func(_ *http.Request, count int) (int, any) {
				if count == 1 {
					return 200, map[string]any{"items": []any{item}}
				}
				return tc.status, tc.later
			})
			got, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
			if err == nil || got.Complete || len(got.Subscriptions) != 0 || *requests != tc.requests || strings.Contains(err.Error(), "canary") {
				t.Fatalf("result %+v, error %v, requests %d", got, err, *requests)
			}
		})
	}
}
