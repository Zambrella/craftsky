package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	if got := attrs["sentry_trace_id"].AsString(); got != "trace-123" {
		t.Fatalf("shared safe trace correlation lost: %#v", attrs)
	}
}

// IT-013 / FR-015: process direct slog uses the same independent export path.
func TestDiagnosticConfiguredDirectSlogExport(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			var local bytes.Buffer
			transport := &sentry.MockTransport{}
			logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: enabled, TracingEnabled: false, Logger: logger})
			logger.With(slog.String("operation", "post.read")).Warn("operation failed", slog.Any("error", errors.New("opaque private prose")))
			observer.Log(context.Background(), slog.LevelError, "HTTP request failed", EventContext{"operation": "http.server"})
			observer.Log(context.Background(), slog.LevelInfo, "post create: response ready", EventContext{"operation": "post.create"})
			if !observer.Flush(time.Second) {
				t.Fatal("flush failed")
			}
			var logs []sentry.Log
			for _, event := range transport.Events() {
				if len(event.Exception) > 0 || event.Type == "transaction" {
					t.Fatal("log created issue/trace")
				}
				logs = append(logs, event.Logs...)
			}
			if enabled && len(logs) != 3 {
				t.Fatalf("exported logs=%d want 3 without duplicates: %#v", len(logs), logs)
			}
			if !enabled && len(logs) != 0 {
				t.Fatal("disabled export emitted")
			}
			if strings.Count(local.String(), "\n") != 3 {
				t.Fatalf("local records: %s", local.String())
			}
			if enabled {
				for _, log := range logs {
					if !strings.Contains(local.String(), log.Body) {
						t.Fatal("export differs from local selected record")
					}
				}
			}
			if strings.Contains(local.String(), "opaque private prose") {
				t.Fatal("private cause leaked")
			}
		})
	}
}

func TestSIMT05SDKLogRetainsTypedOperationalScalars(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
	observer.Log(context.Background(), slog.LevelWarn, "Worker retry scheduled", EventContext{"operation": "post.read", "retryable": true, "attempt": 2})
	observer.Flush(time.Second)
	events := transport.Events()
	if len(events) != 1 || len(events[0].Logs) != 1 {
		t.Fatalf("missing log: %#v", events)
	}
	attrs := events[0].Logs[0].Attributes
	if value, ok := attrs["retryable"].AsInterface().(bool); !ok || !value {
		t.Fatalf("retryable changed: %#v", attrs["retryable"])
	}
	if value := attrs["attempt"].AsInterface(); value != int64(2) {
		t.Fatalf("attempt lost numeric type: %#v", value)
	}
}

func TestSDKOfficialSlogWithRequestTimeline(t *testing.T) {
	transport := &sentry.MockTransport{}
	var local bytes.Buffer
	observer := New(Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
	ctx := sentry.SetHubOnContext(context.Background(), observer.sentryHub.Clone())
	observer.Log(ctx, slog.LevelWarn, "PDS read failed token=private-log-token", EventContext{"operation": "post.read", "http_status": 503})
	observer.CaptureDiagnostic(ctx, DiagnosticInput{Error: errors.New("private-error"), Context: EventContext{"operation": "post.read"}})
	observer.Flush(time.Second)
	data, _ := json.Marshal(transport.Events())
	for _, positive := range []string{"auto.log.slog", "PDS read failed", "503", "breadcrumbs"} {
		if !strings.Contains(string(data), positive) {
			t.Fatalf("missing %s: %s", positive, data)
		}
	}
	for _, private := range []string{"private-log-token", "private-error"} {
		if strings.Contains(string(data)+local.String(), private) {
			t.Fatalf("private value leaked: %s", data)
		}
	}
	events := 0
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			events++
		}
	}
	if events != 1 {
		t.Fatalf("logs implicitly created issues: %d", events)
	}
}

func TestSDKRequestBreadcrumbsAreIsolatedWithoutTracing(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
	first := WithRequestObserver(context.Background(), observer)
	observer.Log(first, slog.LevelInfo, "first request started", EventContext{"operation": "post.read"})
	observer.CaptureDiagnostic(first, DiagnosticInput{Error: errors.New("private-first"), Context: EventContext{"operation": "post.read"}})
	second := WithRequestObserver(context.Background(), observer)
	observer.Log(second, slog.LevelInfo, "second request started", EventContext{"operation": "profile.read"})
	observer.CaptureDiagnostic(second, DiagnosticInput{Error: errors.New("private-second"), Context: EventContext{"operation": "profile.read"}})
	observer.Flush(time.Second)
	var issues []*sentry.Event
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			issues = append(issues, event)
		}
	}
	if len(issues) != 2 {
		t.Fatalf("issues = %d", len(issues))
	}
	for i, event := range issues {
		data, _ := json.Marshal(event)
		own, other := "first request started", "second request started"
		if i == 1 {
			own, other = other, own
		}
		if !strings.Contains(string(data), own) || strings.Contains(string(data), other) {
			t.Fatalf("request timeline mixed or missing: %s", data)
		}
	}
}

// SDK-T06: official typed log attributes survive the final hook.
func TestSDKNativeHTTPLogScalars(t *testing.T) {
	transport := &sentry.MockTransport{}
	var local bytes.Buffer
	observer := New(Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
	observer.Log(context.Background(), slog.LevelWarn, "HTTP request failed", EventContext{"operation": "http.request"}, slog.Int("status", 503), slog.Int64("bytes", 250))
	observer.Flush(time.Second)
	attrs := transport.Events()[0].Logs[0].Attributes
	if attrs["status"].AsInterface() != int64(503) || attrs["bytes"].AsInterface() != int64(250) {
		t.Fatalf("HTTP numeric fields lost: %#v", attrs)
	}
}

func TestFollowerGrowthAlreadyCompleteSurvivesDiagnosticSinks(t *testing.T) {
	var local bytes.Buffer
	logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
	observer.FollowerGrowthCapture(context.Background(), "already_complete", "none", time.Second, 0, nil)
	observer.FollowerGrowthCapture(context.Background(), "private-outcome-canary", "none", time.Second, 0, nil)
	observer.Flush(time.Second)
	if !strings.Contains(local.String(), `"result":"already_complete"`) {
		t.Fatalf("completion outcome lost: %s", local.String())
	}
	sawResult := false
	for _, event := range transport.Events() {
		for _, entry := range event.Logs {
			if entry.Attributes["result"].AsString() == "already_complete" {
				sawResult = true
			}
		}
	}
	if !sawResult {
		t.Fatal("native SDK completion outcome lost")
	}
	data, _ := json.Marshal(transport.Events())
	if strings.Contains(string(data)+local.String(), "private-outcome-canary") {
		t.Fatal("private outcome leaked")
	}
}

func TestReleaseMetadataSurvivesOrIsAbsentAcrossDiagnosticSinks(t *testing.T) {
	for _, release := range []string{"craftsky-appview@1.0.11", "", "private-release-canary@example.invalid"} {
		t.Run(release, func(t *testing.T) {
			var local bytes.Buffer
			logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			transport := &sentry.MockTransport{}
			observer := New(Config{Env: "test", Release: release, Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true})
			observer.FollowerGrowthCapture(context.Background(), "success", "none", time.Second, 0, nil)
			observer.Flush(time.Second)
			var record map[string]any
			if err := json.Unmarshal(local.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if release == "" {
				if _, ok := record["release"]; ok {
					t.Errorf("unavailable release emitted: %s", local.String())
				}
			} else if release == "craftsky-appview@1.0.11" && record["release"] != release {
				t.Errorf("local release lost: %s", local.String())
			}
			count := 0
			for _, event := range transport.Events() {
				for _, entry := range event.Logs {
					count++
					value, ok := entry.Attributes["release"]
					if release == "" && ok {
						t.Error("unavailable SDK release emitted")
					}
					if release == "craftsky-appview@1.0.11" && (!ok || value.AsString() != release) {
						t.Errorf("SDK release lost: %+v", entry)
					}
				}
			}
			if count != 1 {
				t.Errorf("native SDK logs=%d", count)
			}
			data, _ := json.Marshal(transport.Events())
			if strings.Contains(string(data)+local.String(), "private-release-canary") {
				t.Fatal("unreviewed release leaked")
			}
		})
	}
}
