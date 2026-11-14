package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/observability"
)

type failingDiagnosticDeviceTouch struct{ cause error }

func (touch failingDiagnosticDeviceTouch) TouchDeviceID(context.Context, string, string, string) error {
	return touch.cause
}

func TestMiddlewareFailuresRetainCausesWithoutPrivatePayloads(t *testing.T) {
	cause := fmt.Errorf("PRIVATE_WRAPPER: %w", &pgconn.PgError{Code: "40001", Message: "PRIVATE_DATABASE recipient@example.invalid", Detail: "PRIVATE_QUERY"})
	for _, test := range []struct {
		name, operation, stage string
		wrap                   func(*slog.Logger) func(http.Handler) http.Handler
		status, issues         int
	}{
		{"authentication", "auth.authenticate", "session_lookup", func(logger *slog.Logger) func(http.Handler) http.Handler {
			return Authenticated(&errorAuthSvc{err: cause}, logger, DevAuthPolicy{})
		}, http.StatusServiceUnavailable, 1},
		{"membership", "membership.check", "membership", func(logger *slog.Logger) func(http.Handler) http.Handler {
			return CurrentMember(&stubCurrentMemberChecker{err: cause}, logger)
		}, http.StatusServiceUnavailable, 1},
		{"lifecycle", "membership.check", "lifecycle", func(logger *slog.Logger) func(http.Handler) http.Handler {
			return CurrentMember(&stubCurrentMemberChecker{current: map[syntax.DID]bool{"did:plc:initiator": true}}, logger, stubOwnerLifecycleReader{err: cause})
		}, http.StatusServiceUnavailable, 1},
		{"device touch", "auth.device_touch", "store", func(logger *slog.Logger) func(http.Handler) http.Handler {
			return DeviceID(failingDiagnosticDeviceTouch{cause: cause}, logger)
		}, http.StatusNoContent, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var local bytes.Buffer
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			handler := Logging(logger)(HTTPMetrics(observer)(test.wrap(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			}))))
			req := httptest.NewRequest(http.MethodPost, "/v1/private/PRIVATE_TARGET?secret=PRIVATE_QUERY", strings.NewReader("PRIVATE_BODY"))
			req.Header.Set("Authorization", "Bearer PRIVATE_CREDENTIAL")
			req.Header.Set("X-Craftsky-Device-Id", "PRIVATE_DEVICE")
			req = req.WithContext(WithOAuthSessionID(WithDID(req.Context(), "did:plc:initiator"), "PRIVATE_SESSION"))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != test.status {
				t.Fatalf("status=%d want%d", response.Code, test.status)
			}
			observer.Flush(time.Second)
			issues := 0
			for _, event := range transport.Events() {
				if len(event.Exception) == 0 {
					continue
				}
				issues++
				if event.Tags["operation"] != test.operation || event.Tags["failure_stage"] != test.stage || event.Exception[0].Type != "*pgconn.PgError" || !strings.Contains(event.Exception[0].Value, "40001") {
					t.Fatalf("original failure lost: %+v", event)
				}
				if event.Contexts["correlation"]["run_id"] != response.Header().Get("X-Request-ID") {
					t.Fatal("request correlation lost")
				}
			}
			if issues != test.issues {
				t.Fatalf("issues=%d want%d", issues, test.issues)
			}
			serialized, err := json.Marshal(transport.Events())
			if err != nil {
				t.Fatal(err)
			}
			for _, wanted := range []string{test.operation, test.stage, "40001", "*pgconn.PgError"} {
				if !strings.Contains(local.String(), wanted) {
					t.Fatalf("missing local field %q: %s", wanted, local.String())
				}
			}
			if strings.Contains(local.String()+string(serialized)+response.Body.String(), "PRIVATE_") || strings.Contains(local.String()+string(serialized), "recipient@example.invalid") {
				t.Fatal("private diagnostic canary leaked")
			}
		})
	}
}

func TestDeviceTouchFailureDoesNotClaimTheHandlerFailure(t *testing.T) {
	transport := &sentry.MockTransport{}
	logger := discardLogger()
	observer := observability.New(observability.Config{Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	cause := &pgconn.PgError{Code: "40001"}
	handler := Logging(logger)(HTTPMetrics(observer)(DeviceID(failingDiagnosticDeviceTouch{cause: cause}, logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observability.ReportRequestFailure(r.Context(), cause, "post.read", "query")
		w.WriteHeader(http.StatusInternalServerError)
	}))))
	req := httptest.NewRequest(http.MethodGet, "/v1/private", nil)
	req.Header.Set("X-Craftsky-Device-Id", "PRIVATE_DEVICE")
	req = req.WithContext(WithOAuthSessionID(WithDID(req.Context(), "did:plc:initiator"), "PRIVATE_SESSION"))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 || events[0].Tags["operation"] != "post.read" {
		t.Fatalf("handler issue ownership lost: %+v", events)
	}
}
