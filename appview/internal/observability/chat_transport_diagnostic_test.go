package observability_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/chat"
	"social.craftsky/appview/internal/observability"
)

type refreshDiagnosticRoundTripper func(*http.Request) (*http.Response, error)

func (f refreshDiagnosticRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestChatTransportReportsOAuthRefreshFailureSeparatelyFromChatRPC(t *testing.T) {
	key, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	const owner = "did:plc:owner"
	session := &oauth.ClientSession{
		Config: &oauth.ClientConfig{ClientID: "https://client.example"},
		Data: &oauth.ClientSessionData{
			AccountDID:              owner,
			HostURL:                 "https://pds.example",
			AuthServerURL:           "https://auth.example",
			AuthServerTokenEndpoint: "https://auth.example/oauth/token",
			AccessToken:             "PRIVATE_ACCESS_TOKEN_CANARY",
			RefreshToken:            "PRIVATE_REFRESH_TOKEN_CANARY",
		},
		DPoPPrivateKey: key,
		Client: &http.Client{Transport: refreshDiagnosticRoundTripper(func(r *http.Request) (*http.Response, error) {
			header := make(http.Header)
			status, body := http.StatusServiceUnavailable, `{"error":"temporarily_unavailable","error_description":"PRIVATE_PROVIDER_PROSE_CANARY"}`
			if r.Method == http.MethodGet {
				status, body = http.StatusUnauthorized, `{}`
				header.Set("WWW-Authenticate", `DPoP error="invalid_token"`)
			}
			return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
		})},
	}
	err = chat.NewTransport(session.APIClient()).Query(t.Context(), "chat.bsky.actor.getStatus", nil, nil)
	original := errors.Unwrap(err)
	if original == nil || errors.Unwrap(original) == nil || err.Error() != original.Error() {
		t.Fatal("refresh diagnostic changed the original auth translation input")
	}

	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	observer.ObservePrivateFailure(t.Context(), err, owner, "", "chat.sync", "get_status", "error", 2)
	observer.Flush(time.Second)

	var issues, logs int
	for _, event := range transport.Events() {
		if len(event.Exception) != 0 {
			issues++
			if event.Tags["upstream_status"] != "503" || event.Tags["provider_code"] != "oauth.temporarily_unavailable" || event.Tags["failure_stage"] != "get_status" {
				t.Fatalf("missing refresh facts in event tags: %v", event.Tags)
			}
		}
		for _, entry := range event.Logs {
			if entry.Attributes["provider_code"].AsString() == "oauth.temporarily_unavailable" {
				logs++
			}
		}
	}
	if issues != 1 || logs != 1 {
		t.Fatalf("capture ownership or provider log missing, got %d issues and %d logs", issues, logs)
	}
	serialized, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	for _, sink := range []string{local.String(), string(serialized)} {
		for _, want := range []string{"OAuth token refresh failed (HTTP 503)", "oauth.temporarily_unavailable", "get_status"} {
			if !strings.Contains(sink, want) {
				t.Errorf("refresh fact %q missing from diagnostic", want)
			}
		}
		if strings.Contains(sink, "PRIVATE_") || strings.Contains(sink, "auth.example") {
			t.Fatal("private refresh input escaped")
		}
	}
}
