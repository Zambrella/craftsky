package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestObserveDBOperationRecordsBoundedComparableTelemetry(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})
	errBoom := errors.New("boom")

	if err := observer.ObserveDB(context.Background(), DBOperation{
		Operation:    "search.posts",
		RoutePattern: "/v1/search/posts",
		RunID:        "run-123",
	}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("ObserveDB success: %v", err)
	}
	if err := observer.ObserveDB(context.Background(), DBOperation{
		Operation:    "search.posts",
		RoutePattern: "/v1/search/posts",
		RunID:        "run-456",
	}, func(context.Context) error {
		return errBoom
	}); !errors.Is(err, errBoom) {
		t.Fatalf("ObserveDB error = %v, want boom", err)
	}

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_db_operation_duration_seconds",
	} {
		if !metricCallsContain(calls, want) {
			t.Fatalf("metric calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
}

func TestObserveDBCreatesBoundedStorageSpan(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:              "test",
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
	})

	rootCtx, root := observer.StartSpan(context.Background(), SpanContext{Operation: "http.server", Component: "http"})
	if err := observer.ObserveDB(rootCtx, DBOperation{
		Operation:    "search.posts",
		RoutePattern: "/v1/search/posts",
		ResultClass:  "some",
		RunID:        "run-123",
	}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("ObserveDB: %v", err)
	}
	root.Finish("success")
	if !observer.Flush(time.Second) {
		t.Fatal("Flush returned false")
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d events, want 1 transaction", len(events))
	}
	event := events[0]
	if len(event.Spans) != 1 {
		t.Fatalf("transaction spans = %d, want 1; event=%#v", len(event.Spans), event)
	}
	span := event.Spans[0]
	if span.Op != "db.search.posts" {
		t.Fatalf("DB span op = %q, want db.search.posts; span=%#v", span.Op, span)
	}
	for key, want := range map[string]any{
		"component":     "db",
		"operation":     "search.posts",
		"route_pattern": "/v1/search/posts",
		"result":        "some",
	} {
		if got := span.Data[key]; got != want {
			t.Fatalf("DB span data %q = %#v, want %#v; data=%#v", key, got, want, span.Data)
		}
	}
	for _, forbidden := range []string{"run-123", "SELECT", "did:plc:", "secret"} {
		if strings.Contains(span.Op, forbidden) {
			t.Fatalf("DB span op contains forbidden value %q: %#v", forbidden, span)
		}
		for key, value := range span.Data {
			if strings.Contains(key, forbidden) || strings.Contains(valueString(value), forbidden) {
				t.Fatalf("DB span data contains forbidden value %q: %s=%#v", forbidden, key, value)
			}
		}
	}
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(value), "\n", " "), "\t", " "), "\r", " "))
}

func TestDBDiagnosticRetainsSQLStateAndCauseWithoutRowText(t *testing.T) {
	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	cause := &pgconn.PgError{Code: "40001", Message: "opaque private row canary", Detail: "access_token=credential-canary"}
	ctx := WithCaptureMarker(context.Background())
	err := observer.ObserveDB(ctx, DBOperation{Operation: "post.read"}, func(context.Context) error { return cause })
	if err != cause {
		t.Fatal("business error changed")
	}
	observer.Flush(time.Second)
	for _, want := range []string{"pgconn.PgError", "40001", "post.read"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("missing %s: %s", want, local.String())
		}
	}
	if len(transport.Events()) != 1 {
		t.Fatal("missing issue")
	}
	data, _ := json.Marshal(transport.Events())
	if strings.Contains(string(data)+local.String(), "canary") {
		t.Fatal("private detail leaked")
	}
}
