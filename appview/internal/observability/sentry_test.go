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
		"duration", "result", "nsid", "tap_connected", "reconnect_attempt",
		"sentry_trace_id", "sentry_span_id",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("SanitizeEventContext missing allowed key %q in %#v", key, got)
		}
	}
	for _, key := range []string{"did", "handle", "token", "request_body", "raw_path", "run_id"} {
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

func TestSentryPillarGatesDefaultToErrorsOnlyWithDSN(t *testing.T) {
	observer := New(Config{
		Env:       "test",
		SentryDSN: "https://public@example.invalid/1",
	})
	if observer.sentryClient == nil {
		t.Fatal("sentryClient = nil, want configured client when DSN is present")
	}
	if observer.tracingEnabled {
		t.Fatal("tracingEnabled = true, want false for DSN-only config")
	}
	if observer.logsEnabled {
		t.Fatal("logsEnabled = true, want false for DSN-only config")
	}
	if observer.metricsEnabled {
		t.Fatal("metricsEnabled = true, want false for DSN-only config")
	}
	if observer.tapTracingEnabled {
		t.Fatal("tapTracingEnabled = true, want false for DSN-only config")
	}
}

func TestSentryPillarGatesHonorExplicitFlags(t *testing.T) {
	observer := New(Config{
		Env:                 "test",
		SentryDSN:           "https://public@example.invalid/1",
		LogsEnabled:         true,
		MetricsEnabled:      true,
		TracingEnabled:      true,
		TracesSampleRate:    1,
		TapTracingEnabled:   true,
		TapTracesSampleRate: 0.25,
	})
	if observer.sentryClient == nil {
		t.Fatal("sentryClient = nil, want configured client when DSN is present")
	}
	if !observer.tracingEnabled {
		t.Fatal("tracingEnabled = false, want true")
	}
	if !observer.logsEnabled {
		t.Fatal("logsEnabled = false, want true")
	}
	if !observer.metricsEnabled {
		t.Fatal("metricsEnabled = false, want true")
	}
	if !observer.tapTracingEnabled {
		t.Fatal("tapTracingEnabled = false, want true")
	}
	if observer.tapTracesSampleRate != 0.25 {
		t.Fatalf("tapTracesSampleRate = %v, want 0.25", observer.tapTracesSampleRate)
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

func TestStartSpanNormalizesUnsafeOperationAndAttributes(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:              "test",
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
	})

	rootCtx, root := observer.StartSpan(context.Background(), SpanContext{Operation: "http.server", Component: "http"})
	root.SetTransactionName("GET /v1/posts/did:plc:raw?cursor=secret")
	_, child := observer.StartSpan(rootCtx, SpanContext{
		Operation: "/v1/posts/did:plc:raw",
		Component: "http",
		Attributes: EventContext{
			"route_pattern": "/v1/posts/did:plc:raw?cursor=secret",
			"operation":     "SELECT * FROM sessions WHERE token='secret-token'",
			"token":         "secret-token",
		},
	})
	child.Finish("success")
	root.Finish("success")
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d events, want 1 transaction", len(events))
	}
	event := events[0]
	if event.Transaction != "GET unmatched" {
		t.Fatalf("transaction = %q, want GET unmatched", event.Transaction)
	}
	if len(event.Spans) != 1 {
		t.Fatalf("transaction spans = %d, want 1", len(event.Spans))
	}
	if event.Spans[0].Op != "unknown" {
		t.Fatalf("child span op = %q, want unknown; span=%#v", event.Spans[0].Op, event.Spans[0])
	}
	for _, forbidden := range []string{"did:plc:raw", "cursor=secret", "secret-token", "SELECT"} {
		if strings.Contains(event.Transaction, forbidden) {
			t.Fatalf("transaction contains forbidden value %q: %#v", forbidden, event)
		}
		for _, span := range event.Spans {
			if strings.Contains(span.Op, forbidden) || strings.Contains(span.Description, forbidden) {
				t.Fatalf("span contains forbidden value %q: %#v", forbidden, span)
			}
			for key, value := range span.Data {
				if strings.Contains(key, forbidden) || strings.Contains(fmt.Sprint(value), forbidden) {
					t.Fatalf("span data contains forbidden value %q: %s=%#v", forbidden, key, value)
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
	if event.Tags["component"] != "http" || event.Tags["route_pattern"] != "/v1/panic" {
		t.Fatalf("event tags missing sanitized context: %#v", event.Tags)
	}
	for _, forbidden := range []string{"secret-token", "did:plc:raw", "raw_path", "token", "run-123"} {
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

func TestCaptureErrorUsesActiveSpanTraceContext(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:              "test",
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
	})

	rootCtx, root := observer.StartSpan(context.Background(), SpanContext{Operation: "http.server", Component: "http"})
	childCtx, child := observer.StartSpan(rootCtx, SpanContext{Operation: "http.handler", Component: "http"})
	observer.CaptureError(childCtx, EventContext{
		"component":     "http",
		"route_pattern": "/v1/profiles/{handleOrDid}",
	}, errors.New("http server error response"))
	child.Finish("error")
	root.Finish("error")
	if !observer.Flush(time.Second) {
		t.Fatal("Flush returned false")
	}

	events := transport.Events()
	var errorEvent *sentry.Event
	for _, event := range events {
		if event.Level == sentry.LevelError {
			errorEvent = event
			break
		}
	}
	if errorEvent == nil {
		t.Fatalf("missing error event in %#v", events)
	}
	traceCtx := errorEvent.Contexts["trace"]
	if traceCtx == nil {
		t.Fatalf("error event missing trace context: %#v", errorEvent.Contexts)
	}
	if got := fmt.Sprint(traceCtx["trace_id"]); got != child.sentrySpan.TraceID.String() {
		t.Fatalf("trace_id = %q, want %q; trace context=%#v", got, child.sentrySpan.TraceID.String(), traceCtx)
	}
	if got := fmt.Sprint(traceCtx["span_id"]); got != child.sentrySpan.SpanID.String() {
		t.Fatalf("span_id = %q, want %q; trace context=%#v", got, child.sentrySpan.SpanID.String(), traceCtx)
	}
	if errorEvent.Tags["sentry_trace_id"] != child.sentrySpan.TraceID.String() || errorEvent.Tags["sentry_span_id"] != child.sentrySpan.SpanID.String() {
		t.Fatalf("trace tags missing active span IDs: %#v", errorEvent.Tags)
	}
}

func TestCapturePanicIncludesRecoveredTypeWithoutRecoveredValue(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
	})

	observer.CapturePanic(context.Background(), EventContext{
		"component":     "http",
		"route_pattern": "/v1/panic",
		"run_id":        "run-123",
	}, "panic contained did:plc:raw secret-token")
	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d events, want 1", len(events))
	}
	event := events[0]
	if event.Exception[0].Type != "AppViewPanic" {
		t.Fatalf("exception type = %q, want AppViewPanic", event.Exception[0].Type)
	}
	if event.Exception[0].Value != "redacted" {
		t.Fatalf("exception value = %q, want redacted", event.Exception[0].Value)
	}
	if event.Tags["recovered_type"] != "string" {
		t.Fatalf("recovered_type tag = %q, want string; tags=%#v", event.Tags["recovered_type"], event.Tags)
	}
	for _, forbidden := range []string{"did:plc:raw", "secret-token", "run-123", "panic contained"} {
		if strings.Contains(event.Message, forbidden) || strings.Contains(event.Exception[0].Type, forbidden) || strings.Contains(event.Exception[0].Value, forbidden) {
			t.Fatalf("panic event contains forbidden value %q: %#v", forbidden, event)
		}
		for key, value := range event.Tags {
			if strings.Contains(key, forbidden) || strings.Contains(value, forbidden) {
				t.Fatalf("panic tag contains forbidden value %q: %s=%s", forbidden, key, value)
			}
		}
	}
}
