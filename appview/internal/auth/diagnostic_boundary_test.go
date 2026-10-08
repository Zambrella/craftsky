package auth_test

import (
	"bytes"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
