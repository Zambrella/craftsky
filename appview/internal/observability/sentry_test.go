package observability

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestSanitizeEventContextKeepsOnlyAllowedTechnicalFields(t *testing.T) {
	ctx := EventContext{
		"service":           "craftsky_appview",
		"environment":       "prod",
		"release":           "abc123",
		"component":         "http",
		"operation":         "post.create",
		"route_pattern":     "/v1/posts",
		"http_method":       "POST",
		"http_status":       500,
		"http_status_class": "5xx",
		"error_category":    "server",
		"failure_stage":     "pds_response",
		"duration":          "10ms",
		"result":            "error",
		"nsid":              "social.craftsky.feed.post",
		"tap_connected":     false,
		"reconnect_attempt": 2,
		"run_id":            "run-123",
		"sentry_trace_id":   "trace-123",
		"sentry_span_id":    "span-123",
		"did":               "did:plc:raw",
		"handle":            "raw.example",
		"token":             "secret-token",
		"request_body":      `{"secret":"payload"}`,
		"raw_path":          "/v1/posts/did:plc:raw/rkey",
	}

	got := SanitizeEventContext(ctx)
	for _, key := range []string{
		"service", "environment", "release", "component", "operation", "route_pattern",
		"http_method", "http_status", "http_status_class", "error_category", "failure_stage",
		"duration", "result", "nsid", "tap_connected", "reconnect_attempt", "run_id",
		"sentry_trace_id", "sentry_span_id",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("SanitizeEventContext missing allowed key %q in %#v", key, got)
		}
	}
	for _, key := range []string{"did", "handle", "token", "request_body", "raw_path"} {
		if _, ok := got[key]; ok {
			t.Fatalf("SanitizeEventContext retained disallowed key %q in %#v", key, got)
		}
	}
}

func TestStartSpanAddsTraceIDsOnlyWhenTracingEnabled(t *testing.T) {
	disabled := New(Config{Env: "test"})
	disabledCtx, disabledSpan := disabled.StartSpan(context.Background(), SpanContext{Operation: "post.create", Component: "pds"})
	if disabledSpan.Enabled() {
		t.Fatal("disabled span Enabled = true, want false")
	}
	if traceID, spanID := TraceIDs(disabledCtx); traceID != "" || spanID != "" {
		t.Fatalf("disabled TraceIDs = (%q, %q), want empty", traceID, spanID)
	}

	enabled := New(Config{Env: "test", TracingEnabled: true})
	enabledCtx, span := enabled.StartSpan(context.Background(), SpanContext{Operation: "post.create", Component: "pds"})
	if !span.Enabled() {
		t.Fatal("enabled span Enabled = false, want true")
	}
	traceID, spanID := TraceIDs(enabledCtx)
	if traceID == "" || spanID == "" {
		t.Fatalf("enabled TraceIDs = (%q, %q), want populated", traceID, spanID)
	}
	span.Finish("success")
	if span.Result() != "success" {
		t.Fatalf("span result = %q, want success", span.Result())
	}
}

func TestStartSpanExportsSentryTransactionAndChildSpan(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:              "test",
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
	})

	rootCtx, root := observer.StartSpan(context.Background(), SpanContext{Operation: "http.server", Component: "http"})
	childCtx, child := observer.StartSpan(rootCtx, SpanContext{Operation: "post.create", Component: "pds"})
	child.Finish("success")
	root.Finish("success")
	if traceID, spanID := TraceIDs(childCtx); traceID == "" || spanID == "" {
		t.Fatalf("child TraceIDs = (%q, %q), want populated", traceID, spanID)
	}
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1 transaction", len(events))
	}
	event := events[0]
	if event.Transaction != "http.server" {
		t.Fatalf("transaction name = %q, want http.server", event.Transaction)
	}
	if len(event.Spans) != 1 {
		t.Fatalf("transaction spans = %d, want 1; event=%#v", len(event.Spans), event)
	}
	span := event.Spans[0]
	if span.Op != "post.create" {
		t.Fatalf("child span op = %q, want post.create", span.Op)
	}
	for key, want := range map[string]any{
		"component": "pds",
		"operation": "post.create",
		"result":    "success",
	} {
		if got := span.Data[key]; got != want {
			t.Fatalf("child span data %q = %#v, want %#v; all data=%#v", key, got, want, span.Data)
		}
	}
	for _, forbidden := range []string{"did:plc:raw", "session-secret", "request_body"} {
		if strings.Contains(event.Transaction, forbidden) {
			t.Fatalf("transaction contains forbidden value %q: %#v", forbidden, event)
		}
		for _, span := range event.Spans {
			if strings.Contains(span.Op, forbidden) || strings.Contains(span.Description, forbidden) {
				t.Fatalf("span contains forbidden value %q: %#v", forbidden, span)
			}
			for key, value := range span.Data {
				if strings.Contains(key, forbidden) || strings.Contains(fmt.Sprint(value), forbidden) {
					t.Fatalf("span data contains forbidden value %q: %q=%#v", forbidden, key, value)
				}
			}
		}
	}
}

func TestFlushUsesConfiguredExternalTelemetryHook(t *testing.T) {
	disabled := New(Config{Env: "test"})
	if !disabled.Flush(50 * time.Millisecond) {
		t.Fatal("disabled Flush = false, want true no-op")
	}

	var gotTimeout time.Duration
	enabled := New(Config{
		Env:       "test",
		SentryDSN: "https://public@example.invalid/1",
		FlushFunc: func(timeout time.Duration) bool {
			gotTimeout = timeout
			return true
		},
	})
	if !enabled.Flush(2 * time.Second) {
		t.Fatal("enabled Flush = false, want true")
	}
	if gotTimeout != 2*time.Second {
		t.Fatalf("flush timeout = %v, want 2s", gotTimeout)
	}
}

func TestCaptureErrorUsesSentryTransportWithSanitizedContext(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:             "test",
		Release:         "abc123",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
	})

	observer.CaptureError(context.Background(), EventContext{
		"component":     "http",
		"route_pattern": "/v1/panic",
		"run_id":        "run-123",
		"token":         "secret-token",
		"raw_path":      "/v1/posts/did:plc:raw/rkey",
	}, errors.New("database contained raw did:plc:raw"))
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1", len(events))
	}
	event := events[0]
	if event.Environment != "test" || event.Release != "abc123" {
		t.Fatalf("event metadata env=%q release=%q, want test/abc123", event.Environment, event.Release)
	}
	if event.Tags["component"] != "http" || event.Tags["route_pattern"] != "/v1/panic" || event.Tags["run_id"] != "run-123" {
		t.Fatalf("event tags missing sanitized context: %#v", event.Tags)
	}
	for _, forbidden := range []string{"secret-token", "did:plc:raw", "raw_path", "token"} {
		if strings.Contains(event.Message, forbidden) {
			t.Fatalf("event message contains forbidden value %q: %#v", forbidden, event)
		}
		for key, value := range event.Tags {
			if strings.Contains(key, forbidden) || strings.Contains(value, forbidden) {
				t.Fatalf("event tag contains forbidden value %q: %q=%q", forbidden, key, value)
			}
		}
	}
}
