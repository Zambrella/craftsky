package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/ctxkeys"
)

func TestOAuthFailureFactsSurviveLocalAndSDKSinks(t *testing.T) {
	for _, test := range []struct {
		name, message, stage, status, code string
	}{
		{"client metadata", "PAR request failed (HTTP 400): invalid_client_metadata", "authorization_request", "400", "oauth.invalid_client_metadata"},
		{"invalid client", "PAR request failed (HTTP 401): invalid_client", "authorization_request", "401", "oauth.invalid_client"},
		{"discovery", "HTTP error fetching protected resource document: 502", "provider_discovery", "502", ""},
		{"metadata", "HTTP error fetching auth server metadata: 503", "provider_metadata", "503", ""},
		{"unknown provider code", "PAR request failed (HTTP 400): private_provider_canary", "authorization_request", "400", ""},
		{"provider prose", "PAR request failed (HTTP 400): invalid_client private_provider_canary", "authorization_request", "", ""},
		{"network error", "private_provider_canary https://private.invalid/?code=private_code_canary", "provider_discovery", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := errors.New(test.message)
			cause := WrapError("operator OAuth dependency failed", OAuthRequestFailure(original))
			if !errors.Is(cause, original) {
				t.Fatal("original cause lost")
			}
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			observer := New(Config{Env: "test", Logger: slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil))), LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			const requestID = "10000000-0000-4000-8000-000000000001"
			ctx := WithRequestObserver(ctxkeys.WithRunID(context.Background(), requestID), observer)
			ReportRequestFailure(ctx, cause, "operator.auth", test.stage)
			observer.Flush(time.Second)
			var issues int
			var providerLog bool
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 {
					issues++
					if event.Tags["failure_stage"] != test.stage || event.Tags["upstream_status"] != test.status || event.Tags["provider_code"] != test.code {
						t.Fatalf("wrong diagnostic tags: %v", event.Tags)
					}
					if event.Exception[0].Type != "*errors.errorString" {
						t.Fatal("original typed cause lost")
					}
				}
				for _, log := range event.Logs {
					if log.Attributes["provider_code"].AsString() == test.code {
						providerLog = true
					}
				}
			}
			if issues != 1 || (test.code != "" && !providerLog) {
				t.Fatalf("issue ownership/provider log lost: issues=%d providerLog=%t", issues, providerLog)
			}
			payload, err := json.Marshal(transport.Events())
			if err != nil {
				t.Fatal(err)
			}
			for _, output := range []string{local.String(), string(payload)} {
				for _, want := range []string{test.stage, "operator.auth", requestID, "operator OAuth dependency failed", test.code, test.status} {
					if want != "" && !strings.Contains(output, want) {
						t.Fatalf("diagnostic missing %s", want)
					}
				}
				for _, private := range []string{"private_provider_canary", "private_code_canary", "private.invalid"} {
					if strings.Contains(output, private) {
						t.Fatalf("protected value escaped: %s", private)
					}
				}
			}
		})
	}
	if OAuthRequestFailure(nil) != nil {
		t.Fatal("success acquired a failure")
	}
}
