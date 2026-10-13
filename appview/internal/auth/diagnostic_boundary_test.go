package auth_test

import (
	"bytes"
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
	"time"
)

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
