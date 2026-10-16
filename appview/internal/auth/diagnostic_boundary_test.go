package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"strings"
	"testing"
	"time"
)

func TestOAuthCallbackConsumedRequestDoesNotCaptureIssue(t *testing.T) {
	pool := withAuthSchema(t)
	owners := newAuthOwnerStore(t, pool)
	config := testStoreConfig()
	config.OwnerLifecycles = owners
	store := auth.NewPostgresAuthStore(pool, config)
	owner := syntax.DID("did:plc:consumed-callback")
	seedAuthOwner(t, pool, owner)
	err := owners.WithOnboardingAuth(context.Background(), owner, func(ctx context.Context, authority ownerlifecycle.Lifecycle) error {
		ctx = auth.WithLoginAuthRequest(ctx, owner, authority.Generation, authority.AuthEpoch,
			"https://pds.example", "https://issuer.example", auth.HandoffVerifiedLink, "private-device", "")
		if err := store.SaveAuthRequestInfo(ctx, oauth.AuthRequestData{
			State: "expired-private-state", RequestURI: "urn:private-request",
		}); err != nil {
			return err
		}
		_, err := store.BeginExchange(ctx, "expired-private-state")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	clientConfig := oauth.NewPublicConfig("https://appview.example/oauth-client-metadata.json", "https://appview.example/oauth/callback", auth.DefaultScopes())
	flow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App:   &oauth.ClientApp{Config: &clientConfig, Resolver: oauth.NewResolver(), Dir: &callbackDirectory{}},
		Store: store, Owners: owners, AuthorityVerifier: &fakeOAuthAuthorityVerifier{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	handlers := &auth.HTTPHandlers{Logger: logger, OAuthFlow: flow}
	request := httptest.NewRequest("GET", "/oauth/callback?state=expired-private-state&code=private-code&iss=https://private-issuer", nil)
	response := httptest.NewRecorder()
	middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.CallbackHandler())).ServeHTTP(response, request)
	observer.Flush(time.Second)
	if response.Code != http.StatusBadRequest || len(transport.Events()) != 0 {
		t.Fatalf("stale callback status/issues = %d/%d", response.Code, len(transport.Events()))
	}
	for _, want := range []string{"callback_request_unavailable", "OAuthCallbackDiagnostic", "run_id"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("local diagnostics missing %q", want)
		}
	}
	serialized, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	for _, canary := range []string{"expired-private-state", "private-code", "private-issuer"} {
		if strings.Contains(local.String(), canary) || strings.Contains(string(serialized), canary) || strings.Contains(response.Body.String(), canary) {
			t.Fatalf("protected callback value escaped: %s", canary)
		}
	}
}

func TestAuthDiagnosticRetainsUnderlyingCause(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	flow := &registrationValidationFlow{err: &pgconn.PgError{Code: "40001", Message: "opaque private auth canary"}}
	handlers := &auth.HTTPHandlers{OAuthFlow: flow, Logger: logger}
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/registrations", strings.NewReader(`{"handoffMode":"verified_link"}`))
	request = request.WithContext(middleware.WithDeviceID(request.Context(), "private-device-canary"))
	response := httptest.NewRecorder()
	middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.RegistrationHandler())).ServeHTTP(response, request)
	observer.Flush(time.Second)
	if response.Code != 502 {
		t.Fatalf("status %d", response.Code)
	}
	for _, want := range []string{"pgconn.PgError", "40001", "registration.start"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("local missing %s: %s", want, local.String())
		}
	}
	events := transport.Events()
	if len(events) != 1 || events[0].Exception[0].Type != "*pgconn.PgError" {
		t.Fatalf("cause owner missing: %+v", events)
	}
	for _, private := range []string{"opaque private auth canary", "private-device-canary"} {
		if strings.Contains(local.String(), private) || strings.Contains(response.Body.String(), private) {
			t.Fatal("protected data leaked")
		}
	}
}

// Exercise the actual HTTP diagnostic boundary and serialized sinks with the
// error produced by the real callback service above, including protected values.
func verifyCallbackDiagnosticSink(t *testing.T, cause error) {
	t.Helper()
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	handlers := &auth.HTTPHandlers{Logger: logger, OAuthFlow: &callbackDiagnosticFlow{cause: cause}}
	request := httptest.NewRequest("GET", "/oauth/callback?state=oauth-state-canary&code=authorization-code-canary&iss=https://private-issuer-canary", nil)
	recorder := httptest.NewRecorder()
	middleware.Logging(logger)(middleware.HTTPMetrics(observer)(handlers.CallbackHandler())).ServeHTTP(recorder, request)
	observer.Flush(time.Second)
	if recorder.Code != 400 {
		t.Fatalf("diagnostics changed callback status: %d", recorder.Code)
	}
	for _, expected := range []string{`"failure_reason":"scope_required_missing"`, "missing=craftsky", "required=8 granted=1", "alternate_forms=0", `"run_id"`, "OAuthCallbackDiagnostic"} {
		if !strings.Contains(local.String(), expected) {
			t.Fatalf("missing %q in serialized diagnostics: %s", expected, local.String())
		}
	}
	if len(transport.Events()) != 0 {
		t.Fatal("expected scope refusal created a Sentry issue")
	}
	for _, protected := range []string{"oauth-state-canary", "authorization-code-canary", "private-issuer-canary", "old-access-token", "old-refresh-token", "pkce-verifier", "callback-device"} {
		if strings.Contains(local.String(), protected) || strings.Contains(recorder.Body.String(), protected) {
			t.Fatalf("protected callback value %q escaped", protected)
		}
	}
}

type callbackDiagnosticFlow struct {
	registrationValidationFlow
	cause error
}

func (f *callbackDiagnosticFlow) CompleteCallback(context.Context, url.Values, auth.OAuthCallbackFinalizer) error {
	return f.cause
}
