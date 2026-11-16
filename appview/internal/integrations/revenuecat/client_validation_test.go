package revenuecat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type responseTransport func(*http.Request) *http.Response

func (transport responseTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request), nil
}

func TestClientRejectsMalformedSuccessfulSubscriptionPages(t *testing.T) {
	valid := map[string]any{
		"id": "sub-valid", "product_id": "prod-plus", "store": "app_store",
		"environment": "production", "status": "active", "gives_access": true,
		"pending_payment": false, "auto_renewal_status": "will_renew",
	}
	tests := []struct{ name, body string }{
		{"null response", `null`},
		{"missing items", `{}`},
		{"null items", `{"items":null}`},
		{"null subscription", `{"items":[null]}`},
		{"empty pagination URL", `{"items":[],"next_page":""}`},
	}
	for _, key := range []string{"id", "product_id", "store", "environment", "status", "gives_access", "pending_payment"} {
		for _, value := range []string{"missing", "null"} {
			item := make(map[string]any, len(valid))
			for key, value := range valid {
				item[key] = value
			}
			if value == "missing" {
				delete(item, key)
			} else {
				item[key] = nil
			}
			body, err := json.Marshal(map[string]any{"items": []any{item}})
			if err != nil {
				t.Fatal(err)
			}
			tests = append(tests, struct{ name, body string }{key + " " + value, string(body)})
		}
	}
	for _, test := range tests {
		for _, laterPage := range []bool{false, true} {
			name := test.name + "/first page"
			if laterPage {
				name = test.name + "/later page"
			}
			t.Run(name, func(t *testing.T) {
				requests := 0
				client, err := NewClient(ClientConfig{BaseURL: "https://revenuecat.invalid/v2", APIKey: "secret", ProjectID: "project"}, &http.Client{Transport: responseTransport(func(request *http.Request) *http.Response {
					requests++
					body := test.body
					if laterPage && requests == 1 {
						first, err := json.Marshal(map[string]any{
							"items": []any{valid}, "next_page": request.URL.Path + "?starting_after=sub-valid",
						})
						if err != nil {
							t.Fatal(err)
						}
						body = string(first)
					}
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
				})})
				if err != nil {
					t.Fatal(err)
				}
				snapshot, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
				if err == nil || snapshot.Complete || len(snapshot.Subscriptions) != 0 {
					t.Fatalf("malformed response = %+v, error %v; want failed empty snapshot", snapshot, err)
				}
				wantRequests := 1
				if laterPage {
					wantRequests = 2
				}
				if requests != wantRequests {
					t.Fatalf("requests = %d, want %d", requests, wantRequests)
				}
			})
		}
	}
}

func TestClientRejectsRepeatedSubscriptionResources(t *testing.T) {
	for _, store := range []string{"app_store", "promotional"} {
		for _, laterPage := range []bool{false, true} {
			t.Run(store+map[bool]string{false: "/same page", true: "/later page"}[laterPage], func(t *testing.T) {
				item := promoResource()
				item["store"] = store
				if store != "promotional" {
					item["product_id"] = "prod-plus"
				}
				client, requests := promoClient(t, 3, func(request *http.Request, count int) (int, any) {
					if !laterPage {
						return http.StatusOK, map[string]any{"items": []any{item, item}}
					}
					body := map[string]any{"items": []any{item}}
					if count == 1 {
						body["next_page"] = request.URL.Path + "?starting_after=sub-promo"
					}
					return http.StatusOK, body
				})
				snapshot, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
				if err == nil || snapshot.Complete || len(snapshot.Subscriptions) != 0 {
					t.Fatalf("duplicate resource produced authoritative snapshot: %+v, error %v", snapshot, err)
				}
				wantRequests := 1
				if laterPage {
					wantRequests = 2
				}
				if *requests != wantRequests {
					t.Fatalf("requests = %d, want %d", *requests, wantRequests)
				}
			})
		}
	}
}

func TestClientAcceptsExplicitFalseNullableProductsAndAdditiveFields(t *testing.T) {
	transport := responseTransport(func(*http.Request) *http.Response {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
			"items":[{"id":"sub-promo","product_id":null,"store":"promotional",
			"environment":"production","status":"future_provider_state","gives_access":false,
			"current_period_ends_at":null,"ends_at":null,"entitlements":{"items":[]},
			"pending_payment":false,"future_field":{"ignored":true}}],"future_list_field":true
		}`)), Header: make(http.Header)}
	})
	client, err := NewClient(ClientConfig{BaseURL: "https://revenuecat.invalid/v2", APIKey: "secret", ProjectID: "project"}, &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
	if err != nil || !snapshot.Complete || len(snapshot.Subscriptions) != 1 {
		t.Fatalf("valid future response = %+v, error %v", snapshot, err)
	}
	if item := snapshot.Subscriptions[0]; item.ProductID != "" || item.GivesAccess || item.PendingPayment || item.Status != "future_provider_state" {
		t.Fatalf("valid future subscription = %+v", item)
	}
}
