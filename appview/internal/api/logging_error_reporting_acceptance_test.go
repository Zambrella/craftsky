package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/routes"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

// AT-001 / BR-001, FR-001, FR-002 / AC-001 exercises the real handler and
// fallback middleware, not just a diagnostic helper.
func TestLoggingErrorReportingPublicRead(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, Logger: logger})
	cause := fmt.Errorf("Bearer credential-canary: %w", &pgconn.PgError{Code: "23505", Message: "opaque linen sentence"})
	handler := middleware.HTTPMetrics(observer)(api.GetPostHandler(&fakePostStore{oneErr: cause}, fakeResolver{}, logger))
	request := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:target/record", "", "did:plc:actor")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	observer.Flush(time.Second)
	if response.Code != 500 {
		t.Fatalf("status=%d", response.Code)
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("issue events=%d", len(events))
	}
	event := events[0]
	fields := event.Contexts["diagnostic"]
	if fields["actor_did"] != "did:plc:actor" || fields["target_did"] != "did:plc:target" || fields["record_uri"] != "at://did:plc:target/social.craftsky.feed.post/record" {
		t.Fatalf("public references lost: %#v", fields)
	}
	if len(event.Exception) != 2 || event.Exception[0].Type != "*pgconn.PgError" || event.Exception[1].Type != "*fmt.wrapError" {
		t.Fatalf("actionable cause replaced by fallback: %#v", event.Exception)
	}
	for _, want := range []string{"23505", "did:plc:target", "did:plc:actor", "post.get"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("local diagnostics missing %s", want)
		}
	}
	data, _ := json.Marshal(events)
	combined := logs.String() + string(data)
	for _, secret := range []string{"credential-canary", "opaque linen sentence"} {
		if strings.Contains(combined, secret) {
			t.Fatalf("protected prose leaked: %s", secret)
		}
	}
	if strings.Contains(response.Body.String(), "23505") {
		t.Fatal("internal cause disclosed through API")
	}
}

// AT-003 / FR-003 produces an actual envelope for the optional paired Dart harness.
func TestLoggingErrorReportingRequestCorrelation(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"disabled", "unsampled", "sampled"} {
		t.Run(mode, func(t *testing.T) {
			var local bytes.Buffer
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			transport := &sentry.MockTransport{}
			dsn := "https://public@example.invalid/1"
			if mode == "disabled" {
				dsn = ""
			}
			rate := 0.0
			if mode == "sampled" {
				rate = 1
			}
			observer := observability.New(observability.Config{Env: "test", Release: "test-release", Logger: logger, SentryDSN: dsn, SentryTransport: transport, LogsEnabled: true, TracingEnabled: mode != "disabled", TracesSampleRate: rate})
			store := &fakePostStore{oneErr: &pgconn.PgError{Code: "23505", Message: "opaque private row"}}
			handler := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(api.GetPostHandler(store, fakeResolver{}, logger)))
			req := authedPostPathReq("GET", "/v1/posts/did:plc:target/record", "", "did:plc:actor")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			observer.Flush(time.Second)
			var envelope map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			id, ok := envelope["requestId"].(string)
			if !ok || id == "" {
				t.Fatalf("request ID absent: %s", response.Body.String())
			}
			for _, line := range strings.Split(strings.TrimSpace(local.String()), "\n") {
				var record map[string]any
				if err := json.Unmarshal([]byte(line), &record); err != nil {
					t.Fatal(err)
				}
				if record["environment"] != "test" || record["release"] != "test-release" {
					t.Fatalf("local process metadata lost: %#v", record)
				}
				if record["run_id"] != id {
					t.Fatalf("local ID changed: %#v envelope=%s", record, id)
				}
			}
			var issue *sentry.Event
			for _, event := range transport.Events() {
				for _, log := range event.Logs {
					if log.Attributes["run_id"].AsString() != id {
						t.Fatal("export ID changed")
					}
				}
				if len(event.Exception) > 0 {
					issue = event
					if event.Contexts["correlation"]["run_id"] != id {
						t.Fatal("issue ID changed")
					}
				}
			}
			if mode != "disabled" && issue == nil {
				t.Fatal("issue missing")
			}
			if mode == "sampled" {
				for _, event := range transport.Events() {
					for _, span := range event.Spans {
						if span.Data["run_id"] != id {
							t.Fatalf("span request ID lost: %#v", span.Data)
						}
					}
				}
			}
			if mode == "sampled" && issue.Contexts["correlation"]["sentry_trace_id"] == nil {
				t.Fatal("sampled trace lost")
			}
			if mode == "unsampled" {
				if output := os.Getenv("CRAFTSKY_CORRELATION_FIXTURE"); output != "" {
					fixture, _ := json.Marshal(map[string]any{"envelope": envelope, "events": transport.Events(), "local": local.String()})
					if err := os.WriteFile(output, fixture, 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
		})
	}
}

// IT-006: a second real API caller uses shared cause propagation, not the
// earlier post-read special case. No row content is serialized.
func TestAuthorListDiagnosticRetainsCauseAndPublicTarget(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	handler := api.ListPostsByAuthorHandler(&fakePostStore{listErr: &pgconn.PgError{Code: "40001", Message: "opaque private row canary"}}, fakeResolver{didFor: "did:plc:author", handleFor: "author.test"}, logger, nil)
	mux := http.NewServeMux()
	mux.Handle("GET /v1/profiles/{handleOrDid}/posts", handler)
	chain := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(mux))
	request := authedReq("GET", "/v1/profiles/did:plc:author/posts", "", "did:plc:viewer")
	response := httptest.NewRecorder()
	chain.ServeHTTP(response, request)
	observer.Flush(time.Second)
	if response.Code != 500 {
		t.Fatalf("status=%d", response.Code)
	}
	for _, want := range []string{"pgconn.PgError", "40001", "did:plc:author"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("local missing %s: %s", want, local.String())
		}
	}
	data, _ := json.Marshal(transport.Events())
	for _, want := range []string{"pgconn.PgError", "40001", "did:plc:author"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("event missing %s: %s", want, data)
		}
	}
	if len(transport.Events()) != 1 {
		t.Fatal("duplicate issue")
	}
	if strings.Contains(local.String()+string(data)+response.Body.String(), "opaque private row canary") {
		t.Fatal("private row leaked")
	}
	if os.Getenv("CRAFTSKY_DIAGNOSTIC_EVIDENCE") == "1" {
		t.Logf("public read local diagnostics:\n%s\nmock issue envelope:\n%s", local.String(), data)
	}
}

type diagnosticEventReader struct{ cause error }

func (s diagnosticEventReader) ReadEvent(context.Context, business.EventReadInput) (business.EventView, error) {
	return business.EventView{}, s.cause
}

func TestBusinessEventFailureRetainsUnderlyingCause(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	handler := middleware.Logging(slog.New(slog.NewJSONHandler(&local, nil)), catalogue)(middleware.HTTPMetrics(observer)(api.GetBusinessEventHandler(diagnosticEventReader{&pgconn.PgError{Code: "40001", Message: "private event row canary"}}, time.Now)))
	req := authedPostPathReq("GET", "/v1/events/did:plc:target/record", "", "did:plc:actor")
	req.SetPathValue("did", "did:plc:target")
	req.SetPathValue("rkey", "record")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	observer.Flush(time.Second)
	if response.Code != 500 {
		t.Fatalf("status=%d", response.Code)
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("events=%d", len(events))
	}
	encoded, _ := json.Marshal(events)
	for _, want := range []string{"pgconn.PgError", "40001", "did:plc:target"} {
		if !strings.Contains(local.String(), want) || !strings.Contains(string(encoded), want) {
			t.Fatalf("missing %s: local=%s event=%s", want, local.String(), encoded)
		}
	}
	if strings.Contains(local.String()+string(encoded)+response.Body.String(), "private event row canary") {
		t.Fatal("private cause disclosed")
	}
}

func TestCommandFailureRetainsUnderlyingCause(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	ctx := observability.WithRequestObserver(observability.WithCaptureMarker(context.Background()), observer)
	cause := &pgconn.PgError{Code: "40001", Message: "private command intent canary"}
	response := httptest.NewRecorder()
	api.WriteCommandError(response, "test-command-request", cause, ctx)
	observer.Flush(time.Second)
	events := transport.Events()
	encoded, _ := json.Marshal(events)
	if len(events) != 1 || !strings.Contains(string(encoded), "pgconn.PgError") || !strings.Contains(local.String(), "40001") {
		t.Fatalf("command cause lost: %s %s", local.String(), encoded)
	}
	if response.Code != 503 || strings.Contains(local.String()+string(encoded)+response.Body.String(), "private command intent canary") {
		t.Fatal("command contract or privacy changed")
	}
}
