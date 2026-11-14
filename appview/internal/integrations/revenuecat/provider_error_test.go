package revenuecat

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type diagnosticRoundTrip func(*http.Request) (*http.Response, error)

func (f diagnosticRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type diagnosticErrorBody struct{ cause error }

func (body diagnosticErrorBody) Read([]byte) (int, error) { return 0, body.cause }
func (diagnosticErrorBody) Close() error                  { return nil }

func TestClientPreservesNativeFailureWithoutFormattingPrivateData(t *testing.T) {
	const canary = "private-provider-payload-canary"
	original := errors.New(canary)
	for _, stage := range []string{"request", "read", "decode", "response"} {
		t.Run(stage, func(t *testing.T) {
			client, err := NewClient(ClientConfig{BaseURL: "https://provider.invalid", APIKey: "private-token", ProjectID: "private-project"}, &http.Client{Transport: diagnosticRoundTrip(func(request *http.Request) (*http.Response, error) {
				if stage == "request" {
					return nil, original
				}
				status := http.StatusOK
				var body io.ReadCloser = io.NopCloser(strings.NewReader(`{"items":"` + canary + `"}`))
				if stage == "read" {
					body = diagnosticErrorBody{cause: original}
				}
				if stage == "response" {
					status = http.StatusBadGateway
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: body, Request: request}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := client.ListCustomerSubscriptions(context.Background(), uuid.New())
			var failure *ProviderError
			if !errors.As(err, &failure) || snapshot.Complete || failure.stage != stage {
				t.Fatalf("missing typed %s failure: %T", stage, err)
			}
			if strings.Contains(err.Error(), canary) || strings.Contains(err.Error(), "private-project") {
				t.Fatal("ordinary error formatting exposed private provider data")
			}
			switch stage {
			case "request", "read":
				if !errors.Is(err, original) {
					t.Fatal("original cause was discarded")
				}
			case "decode":
				var parse *json.UnmarshalTypeError
				if !errors.As(err, &parse) {
					t.Fatal("native JSON cause was discarded")
				}
			case "response":
				if failure.StatusCode() != http.StatusBadGateway || failure.Unwrap() != nil {
					t.Fatal("provider status was lost or raw response was retained")
				}
			}
		})
	}
}

func TestClientRetainsNativeProductAndExpiryDecodeCauses(t *testing.T) {
	for _, field := range []string{"product_id", "current_period_ends_at", "ends_at"} {
		t.Run(field, func(t *testing.T) {
			item := promoResource()
			item[field] = map[string]any{"private": "PRIVATE_PROVIDER_CANARY"}
			client, _ := promoClient(t, 3, func(*http.Request, int) (int, any) {
				return http.StatusOK, map[string]any{"items": []any{item}}
			})
			snapshot, err := client.ListCustomerSubscriptions(t.Context(), uuid.New())
			var failure *ProviderError
			var cause *json.UnmarshalTypeError
			if snapshot.Complete || !errors.As(err, &failure) || !errors.As(err, &cause) || failure.StatusCode() != http.StatusOK {
				t.Fatalf("field %s lost native decode cause: %T", field, err)
			}
			if strings.Contains(err.Error(), "PRIVATE_PROVIDER_CANARY") {
				t.Fatal("ordinary decode error exposed private response data")
			}
		})
	}
}

func TestClientRetainsCauseDuringCustomerCreationAndNestedEntitlementPagination(t *testing.T) {
	for _, operation := range []string{"customer_creation", "promotional_pagination"} {
		t.Run(operation, func(t *testing.T) {
			original := errors.New("PRIVATE_TRANSPORT_CANARY")
			requests := 0
			client, err := NewClient(ClientConfig{BaseURL: "https://provider.invalid/v2", APIKey: "private-token", ProjectID: "project"}, &http.Client{Transport: diagnosticRoundTrip(func(request *http.Request) (*http.Response, error) {
				requests++
				if requests == 2 {
					if operation == "customer_creation" && request.Method != http.MethodPost {
						t.Fatal("missing exact-customer registration attempt")
					}
					if operation == "promotional_pagination" && request.URL.Path != "/v2/projects/project/subscriptions/sub-promo/entitlements" {
						t.Fatal("missing nested entitlement page request")
					}
					return nil, original
				}
				status := http.StatusNotFound
				body := []byte(`{}`)
				if operation == "promotional_pagination" {
					status = http.StatusOK
					item := promoResource()
					item["entitlements"].(map[string]any)["next_page"] = "/v2/projects/project/subscriptions/sub-promo/entitlements?starting_after=ent-plus"
					body, _ = json.Marshal(map[string]any{"items": []any{item}})
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(body))), Request: request}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			snapshot, err := client.ListCustomerSubscriptions(t.Context(), uuid.New())
			var failure *ProviderError
			if requests != 2 || snapshot.Complete || !errors.Is(err, original) || !errors.As(err, &failure) || failure.stage != "request" {
				t.Fatalf("lost %s failure: requests=%d type=%T", operation, requests, err)
			}
			if strings.Contains(err.Error(), "PRIVATE_TRANSPORT_CANARY") || strings.Contains(err.Error(), "provider.invalid") {
				t.Fatal("ordinary provider error exposed transport or customer routing")
			}
		})
	}
}
