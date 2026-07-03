package observability

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

func TestSentryLogsRequireExplicitGateAndFilterAttributes(t *testing.T) {
	disabledTransport := &sentry.MockTransport{}
	disabled := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: disabledTransport,
	})
	disabled.EmitLog(context.Background(), slog.LevelWarn, "pds write completed", EventContext{
		"component": "pds",
		"run_id":    "run-123",
	})
	if !disabled.Flush(50 * time.Millisecond) {
		t.Fatal("disabled Flush returned false")
	}
	if events := disabledTransport.Events(); len(events) != 0 {
		t.Fatalf("disabled logs captured %d events, want 0", len(events))
	}

	enabledTransport := &sentry.MockTransport{}
	enabled := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: enabledTransport,
		LogsEnabled:     true,
	})
	enabled.EmitLog(context.Background(), slog.LevelWarn, "pds write completed", EventContext{
		"component":      "pds",
		"operation":      "post.create",
		"failure_stage":  "pds_request",
		"result":         "error",
		"error_category": "unexpected",
		"error_code":     "appview.unexpected",
		"run_id":         "run-123",
		"token":          "secret-token",
		"raw_path":       "/v1/posts/did:plc:raw?cursor=secret",
	})
	if !enabled.Flush(time.Second) {
		t.Fatal("enabled Flush returned false")
	}

	events := enabledTransport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d log events, want 1; events=%#v", len(events), events)
	}
	if len(events[0].Logs) != 1 {
		t.Fatalf("event logs = %d, want 1; event=%#v", len(events[0].Logs), events[0])
	}
	log := events[0].Logs[0]
	if log.Body != "pds write completed" || log.Level != sentry.LogLevelWarn {
		t.Fatalf("log body/level = %q/%q, want pds write completed/warn", log.Body, log.Level)
	}
	for _, want := range []string{"component", "operation", "failure_stage", "result", "error_category", "error_code"} {
		if _, ok := log.Attributes[want]; !ok {
			t.Fatalf("log missing attribute %q: %#v", want, log.Attributes)
		}
	}
	for _, forbidden := range []string{"run-123", "secret-token", "did:plc:raw", "cursor=secret", "raw_path", "token"} {
		if strings.Contains(log.Body, forbidden) {
			t.Fatalf("log body contains forbidden value %q: %#v", forbidden, log)
		}
		for key, value := range log.Attributes {
			if strings.Contains(key, forbidden) || strings.Contains(fmt.Sprint(value), forbidden) {
				t.Fatalf("log attribute contains forbidden value %q: %s=%#v", forbidden, key, value)
			}
		}
	}
}

func TestObserverLogWritesSafeContextToStdoutAndSentry(t *testing.T) {
	var stdout bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
		LogsEnabled:     true,
		Logger:          slog.New(slog.NewJSONHandler(&stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})

	observer.Log(context.Background(), slog.LevelWarn, "pds write completed", EventContext{
		"component":      "pds",
		"operation":      "post.create",
		"failure_stage":  "pds_request",
		"result":         "error",
		"error_category": "unexpected",
		"run_id":         "run-123",
		"token":          "secret-token",
	}, slog.String("sentry_trace_id", "trace-123"))

	logged := stdout.String()
	for _, want := range []string{
		`"msg":"pds write completed"`,
		`"component":"pds"`,
		`"operation":"post.create"`,
		`"failure_stage":"pds_request"`,
		`"result":"error"`,
		`"error_category":"unexpected"`,
		`"sentry_trace_id":"trace-123"`,
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("stdout log missing %q:\n%s", want, logged)
		}
	}
	for _, forbidden := range []string{"run-123", "secret-token"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("stdout log contains forbidden value %q:\n%s", forbidden, logged)
		}
	}

	if !observer.Flush(time.Second) {
		t.Fatal("observer Flush returned false")
	}
	events := transport.Events()
	if len(events) != 1 || len(events[0].Logs) != 1 {
		t.Fatalf("captured logs = %#v, want one Sentry log", events)
	}
	attrs := events[0].Logs[0].Attributes
	for _, want := range []string{"component", "operation", "failure_stage", "result", "error_category"} {
		if _, ok := attrs[want]; !ok {
			t.Fatalf("Sentry log missing %q: %#v", want, attrs)
		}
	}
	if _, ok := attrs["sentry_trace_id"]; ok {
		t.Fatalf("Sentry log included local-only trace id: %#v", attrs)
	}
}
