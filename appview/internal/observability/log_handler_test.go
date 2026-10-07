package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// IT-011 / FR-010, RULE-001–003: direct slog/With/local-only attrs share policy.
func TestDiagnosticDirectSlogProtection(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug})))
	logger.With(slog.Any("nested", map[string]any{"draft": "private-draft-canary"}), slog.String("token", "token-canary")).ErrorContext(context.Background(), "Worker boundary failed token=private-log-canary", slog.Any("error", errors.New("opaque private exception")), slog.Any("unknown", struct{ Body string }{"private-body-canary"}))
	logger.Error("operation failed", slog.String("operation", "post.read"), slog.Any("causes", DescribeError(errors.New("opaque private cause"), EventContext{})), slog.Any("diagnostic", PublicRecordContext{}))
	observer := New(Config{Logger: logger})
	observer.Log(context.Background(), slog.LevelWarn, "pds write completed", EventContext{"component": "pds"}, slog.Any("private", map[string]any{"draft": "private-draft-canary"}), slog.String("password", "password-canary"))
	for _, bad := range []string{"opaque private", "private-log-canary", "private-draft-canary", "token-canary", "private-body-canary", "password-canary"} {
		if strings.Contains(output.String(), bad) {
			t.Fatalf("direct log leak %s: %s", bad, output.String())
		}
	}
	for _, want := range []string{"post.read", "*errors.errorString", "operation failed", "pds write completed"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("positive missing %s: %s", want, output.String())
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if len(line) > 16384 {
			t.Fatal("log record over budget")
		}
	}
}

func TestDiagnosticFinalSDKLogsAndTrace(t *testing.T) {
	transport := &sentry.MockTransport{}
	observer := New(Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true, TracingEnabled: true, TracesSampleRate: 1})
	ctx, span := observer.StartSpan(context.Background(), SpanContext{Component: "api", Operation: "post.read"})
	sdkSpan := sentry.SpanFromContext(ctx)
	sdkSpan.SetData("private", map[string]any{"draft": "draft-canary"})
	sdkSpan.Description = "opaque private trace"
	child := sentry.StartSpan(ctx, "private-capability-canary")
	child.Description = "opaque private trace"
	child.SetData("secret", "token-canary")
	child.Finish()
	sentry.NewLogger(ctx).Error().String("password", "password-canary").Emit("SDK boundary failed token=private-log-canary")
	observer.Log(ctx, slog.LevelWarn, "pds write completed", EventContext{"operation": "post.read"})
	span.Finish("error")
	observer.Flush(time.Second)
	var logs []sentry.Log
	serialized := ""
	for _, event := range transport.Events() {
		logs = append(logs, event.Logs...)
		data, _ := json.Marshal(event)
		serialized += string(data)
		data, _ = json.Marshal(event.Logs)
		serialized += string(data)
	}
	if len(logs) != 2 {
		t.Fatalf("logs=%d", len(logs))
	}
	for _, bad := range []string{"private-log-canary", "draft-canary", "private-capability-canary", "opaque private", "token-canary", "password-canary"} {
		if strings.Contains(serialized, bad) {
			t.Fatalf("SDK sink leak %s: %s", bad, serialized)
		}
	}
	for _, want := range []string{"pds write completed", "post.read"} {
		if !strings.Contains(serialized, want) {
			t.Fatalf("positive missing %s", want)
		}
	}
}

type panickingDiagnosticSink struct{ calls int }

func (s *panickingDiagnosticSink) Emit(context.Context, slog.Level, string, EventContext) {
	s.calls++
	panic("PRIVATE_EXPORT_FAILURE")
}

func TestIT012ThrowingExporterKeepsLocalCauseAndNonrecursiveFallback(t *testing.T) {
	var output bytes.Buffer
	sink := &panickingDiagnosticSink{}
	observer := New(Config{Logger: slog.New(slog.NewJSONHandler(&output, nil)), SentryDSN: "https://public@example.invalid/1", LogsEnabled: true, LogSink: sink})
	observer.ObservePrivateFailure(context.Background(), &pgconn.PgError{Code: "40001", Message: "PRIVATE_ORIGINAL"}, syntax.DID("did:plc:owner"), "40000000-0000-4000-8000-000000000001", "schedule.publish", "record_write", "retry", 1)
	if sink.calls != 1 {
		t.Fatalf("recursive exporter calls=%d", sink.calls)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[1], "telemetry export failed") || !strings.Contains(lines[1], "40001") {
		t.Fatalf("missing safe fallback: %s", output.String())
	}
	if strings.Contains(output.String(), "PRIVATE_") {
		t.Fatal("private telemetry prose leaked")
	}
}

type throwingCaptureTransport struct{ sentry.MockTransport }

func (*throwingCaptureTransport) SendEvent(*sentry.Event) { panic("PRIVATE_TRANSPORT_PANIC") }

func TestIT012ThrowingCaptureTransportPreservesCallerAndLocalCause(t *testing.T) {
	var output bytes.Buffer
	observer := New(Config{Logger: slog.New(slog.NewJSONHandler(&output, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: &throwingCaptureTransport{}})
	func() {
		defer func() {
			if recover() != nil {
				t.Error("telemetry changed caller control flow")
			}
		}()
		observer.CaptureDiagnostic(context.Background(), DiagnosticInput{Error: &pgconn.PgError{Code: "40001", Message: "PRIVATE_ORIGINAL"}, Context: EventContext{"operation": "post.read", "failure_stage": "query", "result": "error"}})
	}()
	for _, selected := range []string{"telemetry capture failed", "40001", "post.read"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s in %s", selected, output.String())
		}
	}
	if strings.Contains(output.String(), "PRIVATE_") {
		t.Fatal("transport/source prose leaked")
	}
}

func TestIT012FailedFlushIsLocalAndDoesNotReenterTelemetry(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(fmt.Sprint(panics), func(t *testing.T) {
			var output bytes.Buffer
			calls := 0
			observer := New(Config{Logger: slog.New(slog.NewJSONHandler(&output, nil)), FlushFunc: func(time.Duration) bool {
				calls++
				if panics {
					panic("PRIVATE_FLUSH")
				}
				return false
			}})
			result := true
			func() {
				defer func() {
					if recover() != nil {
						t.Error("flush changed caller control flow")
					}
				}()
				result = observer.Flush(time.Millisecond)
			}()
			if result || calls != 1 {
				t.Fatalf("result=%t calls=%d", result, calls)
			}
			if !strings.Contains(output.String(), "telemetry flush failed") {
				t.Fatal("failed flush has no local indicator")
			}
			if strings.Contains(output.String(), "PRIVATE_FLUSH") {
				t.Fatal("private flush prose leaked")
			}
		})
	}
}

func TestIT012AsyncSDKTransportFailureIsLocalAndNonrecursive(t *testing.T) {
	if os.Getenv("CRAFTSKY_IT012_ASYNC_CHILD") != "1" {
		command := exec.Command(os.Args[0], "-test.run=^TestIT012AsyncSDKTransportFailureIsLocalAndNonrecursive$")
		command.Env = append(os.Environ(), "CRAFTSKY_IT012_ASYNC_CHILD=1")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("asynchronous SDK failure interrupted process: %v\n%s", err, output)
		}
		return
	}
	var output bytes.Buffer
	observer := New(Config{Logger: slog.New(slog.NewJSONHandler(&output, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: &throwingCaptureTransport{}, LogsEnabled: true})
	observer.Log(context.Background(), slog.LevelWarn, "operation failed", EventContext{"operation": "post.read", "result": "error"})
	observer.Flush(time.Second)
	if !strings.Contains(output.String(), "telemetry export failed") {
		t.Fatalf("no local async export failure: %s", output.String())
	}
	if strings.Contains(output.String(), "PRIVATE_") {
		t.Fatal("transport prose leaked")
	}
}

type panickingLocalHandler struct{ slog.Handler }

func (panickingLocalHandler) Handle(context.Context, slog.Record) error {
	panic("PRIVATE_LOCAL_FAILURE")
}

func TestIT012LocalSinkFailureDoesNotChangeCallerOrExport(t *testing.T) {
	sink := &moderationLogSink{}
	observer := New(Config{Logger: slog.New(panickingLocalHandler{slog.NewJSONHandler(&bytes.Buffer{}, nil)}), SentryDSN: "https://public@example.invalid/1", LogsEnabled: true, LogSink: sink})
	func() {
		defer func() {
			if recover() != nil {
				t.Error("local sink interrupted application")
			}
		}()
		observer.Log(context.Background(), slog.LevelError, "operation failed", EventContext{"operation": "post.read", "result": "error"})
	}()
	if len(sink.events) != 1 {
		t.Fatalf("selected remote record lost: %d", len(sink.events))
	}
}

func TestSIMT05DeveloperMessagesNeedNoRegistry(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(NewDiagnosticHandler(slog.NewJSONHandler(&output, nil)))
	logger.Warn("New worker boundary failed token=PRIVATE_TOKEN", slog.String("operation", "post.read"))
	if !strings.Contains(output.String(), "New worker boundary failed") || strings.Contains(output.String(), "PRIVATE_TOKEN") {
		t.Fatalf("developer message or redaction lost: %s", output.String())
	}
}
