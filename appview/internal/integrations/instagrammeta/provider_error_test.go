package instagrammeta

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type diagnosticRoundTrip func(*http.Request) (*http.Response, error)

func (f diagnosticRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type diagnosticErrorBody struct{ cause error }

func (body diagnosticErrorBody) Read([]byte) (int, error) { return 0, body.cause }
func (diagnosticErrorBody) Close() error                  { return nil }

func TestClientPreservesNativeFailureWithoutFormattingPrivateData(t *testing.T) {
	t.Parallel()
	const canary = "private-provider-payload-canary"
	original := errors.New(canary)
	for _, stage := range []string{"request", "read", "decode"} {
		t.Run(stage, func(t *testing.T) {
			client, err := NewHTTPClient(HTTPClientConfig{
				APIVersion: "v99.0", AccessToken: "private-token", OfficialAccountID: "official",
				HTTPClient: &http.Client{Transport: diagnosticRoundTrip(func(request *http.Request) (*http.Response, error) {
					if stage == "request" {
						return nil, original
					}
					var body io.ReadCloser = io.NopCloser(strings.NewReader(`{"username":{"private":"` + canary + `"}}`))
					if stage == "read" {
						body = diagnosticErrorBody{cause: original}
					}
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body, Request: request}, nil
				})},
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.LookupUsername(context.Background(), "private-sender")
			var failure *ProviderError
			if !errors.As(err, &failure) || strings.Contains(err.Error(), canary) || strings.Contains(err.Error(), "private-sender") {
				t.Fatal("missing protected typed provider failure")
			}
			if stage == "decode" {
				var parse *json.UnmarshalTypeError
				if !errors.As(err, &parse) || failure.Kind() != ProviderErrorInvalidResponse {
					t.Fatal("native JSON cause or existing response classification lost")
				}
			} else if !errors.Is(err, original) || !IsRetryableProviderError(err) {
				t.Fatal("native cause or retry classification lost")
			}
		})
	}
	if err := classifyContextError(context.DeadlineExceeded); !errors.Is(err, context.DeadlineExceeded) || !IsRetryableProviderError(err) {
		t.Fatal("native deadline cause or retry classification lost")
	}
}
