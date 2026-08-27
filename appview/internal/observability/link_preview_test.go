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
)

func TestUT020LinkPreviewTelemetryUsesOnlyBoundedFields(t *testing.T) {
	t.Parallel()
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	canary := "https://private.example/path?token=secret"

	observer.ObserveLinkPreview(canary, canary, canary, 599, 99, 2_000_000, -time.Second)
	calls := recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("metric calls = %d, want 2", len(calls))
	}
	for _, call := range calls {
		if strings.Contains(call.Name, canary) {
			t.Fatalf("metric name leaked canary: %q", call.Name)
		}
		for key, value := range call.Attributes {
			if strings.Contains(key, canary) || strings.Contains(value, canary) {
				t.Fatalf("metric leaked canary: %q=%q", key, value)
			}
		}
		want := map[string]string{
			"stage": "unknown", "result": "unknown", "error_class": "unknown",
			"status": "5xx", "redirects": "6+", "bytes": "1m+",
		}
		for key, value := range want {
			if call.Attributes[key] != value {
				t.Fatalf("%s = %q, want %q", key, call.Attributes[key], value)
			}
		}
		if call.Value < 0 {
			t.Fatalf("negative metric value = %v", call.Value)
		}
	}
}

func TestAT009PreviewCanariesStayOutOfEveryTelemetrySink(t *testing.T) {
	t.Parallel()
	canaries := []string{
		"url-canary.invalid/private", "host-canary.invalid", "query-canary=secret",
		"redirect-canary.invalid/final", "title-canary", "description-canary",
		"thumbnail-byte-canary", "post-text-canary", "did:plc:identity-canary",
		"bearer-token-canary", "device-id-canary", "dependency-error-canary",
	}
	unsafe := strings.Join(canaries, "|")
	var localLogs bytes.Buffer
	logs := &recordingPrivacyLogSink{}
	metrics := NewInMemoryMetricRecorder()
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1",
		SentryTransport: transport, TracingEnabled: true, TracesSampleRate: 1,
		MetricRecorder: metrics, LogSink: logs,
		Logger: slog.New(slog.NewJSONHandler(&localLogs, nil)),
	})
	unsafeContext := EventContext{
		"operation": unsafe, "component": unsafe, "route_pattern": unsafe,
		"error_category": unsafe, "failure_stage": unsafe, "result": unsafe,
		"url": canaries[0], "host": canaries[1], "query": canaries[2],
		"redirect": canaries[3], "title": canaries[4], "description": canaries[5],
		"thumbnail": canaries[6], "post_text": canaries[7], "did": canaries[8],
		"token": canaries[9], "device_id": canaries[10],
	}
	observer.Log(context.Background(), slog.LevelError, "preview operation failed", unsafeContext)
	ctx, span := observer.StartSpan(context.Background(), SpanContext{
		Operation: unsafe, Component: unsafe, Attributes: unsafeContext,
	})
	span.SetAttributes(unsafeContext)
	observer.CaptureError(ctx, unsafeContext, errors.New(canaries[11]))
	span.Finish("error")
	observer.ObserveLinkPreview(unsafe, unsafe, unsafe, 599, 99, 2_000_000, time.Second)
	observer.ObserveScheduledOperation(unsafe, unsafe, unsafe, time.Second)
	observer.ObserveScheduledPublication(99, time.Second, time.Second)
	observer.ObserveScheduledCleanupQueue(1, time.Second)
	if !observer.Flush(time.Second) {
		t.Fatal("observer flush failed")
	}

	events, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	captured := strings.Join([]string{
		localLogs.String(), logs.String(), metricCallsString(metrics.Calls()), string(events),
	}, "\n")
	for _, canary := range canaries {
		if strings.Contains(captured, canary) {
			t.Fatalf("telemetry leaked %q:\n%s", canary, captured)
		}
	}
}

type recordingPrivacyLogSink struct {
	events []EventContext
}

func (sink *recordingPrivacyLogSink) Emit(_ context.Context, _ slog.Level, _ string, attrs EventContext) {
	sink.events = append(sink.events, attrs)
}

func (sink *recordingPrivacyLogSink) String() string {
	body, _ := json.Marshal(sink.events)
	return string(body)
}

func metricCallsString(calls []MetricCall) string {
	body, _ := json.Marshal(calls)
	return string(body)
}
