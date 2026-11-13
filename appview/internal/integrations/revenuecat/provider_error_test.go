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
