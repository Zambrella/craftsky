package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/getsentry/sentry-go"
)

type LogSink interface {
	Emit(ctx context.Context, level slog.Level, message string, attrs EventContext)
}

type noopLogSink struct{}

func (noopLogSink) Emit(context.Context, slog.Level, string, EventContext) {}

type sentryLogSink struct {
	hub *sentry.Hub
}

func newSentryLogSink(hub *sentry.Hub) LogSink {
	if hub == nil {
		return noopLogSink{}
	}
	return sentryLogSink{hub: hub}
}

func (s sentryLogSink) Emit(ctx context.Context, level slog.Level, message string, attrs EventContext) {
	if message == "" {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = sentry.SetHubOnContext(ctx, s.hub)
	entry := sentryLogEntry(sentry.NewLogger(ctx), level).WithCtx(ctx)
	selected := SanitizeEventContext(attrs)
	message = boundDiagnosticText(message, 512)
	for _, key := range []string{"causes", "diagnostic", "panic", "incoming_path", "method", "status", "bytes", "content_length", "count", "suppressed_count"} {
		if value, ok := attrs[key]; ok {
			selected[key] = value
		}
	}
	for key, value := range selected {
		switch v := value.(type) {
		case bool:
			entry = entry.Bool(key, v)
		case int:
			entry = entry.Int(key, v)
		case int64:
			entry = entry.Int64(key, v)
		case float64:
			entry = entry.Float64(key, v)
		case []DiagnosticCause, EventContext, []sentry.Exception:
			if data, err := json.Marshal(value); err == nil {
				entry = entry.String(key, string(data))
			}
		default:
			entry = entry.String(key, fmt.Sprint(value))
		}
	}
	entry.Emit(message)
}

func sentryLogEntry(logger sentry.Logger, level slog.Level) sentry.LogEntry {
	switch {
	case level >= slog.LevelError:
		return logger.Error()
	case level >= slog.LevelWarn:
		return logger.Warn()
	case level <= slog.LevelDebug:
		return logger.Debug()
	default:
		return logger.Info()
	}
}

func (o *Observer) EmitLog(ctx context.Context, level slog.Level, message string, attrs EventContext) {
	o.Log(ctx, level, message, attrs)
}

func (o *Observer) Log(ctx context.Context, level slog.Level, message string, sentryCtx EventContext, localOnlyAttrs ...any) {
	if o == nil {
		return
	}
	safeCtx := SanitizeEventContext(withDiagnosticCorrelation(ctx, sentryCtx))
	if o.logger != nil {
		attrs := eventContextSlogAttrs(safeCtx)
		attrs = append(attrs, localOnlyAttrs...)
		o.logger.Log(ctx, level, message, attrs...)
		return
	}
	if o.logSink == nil {
		return
	}
	o.logSink.Emit(ctx, level, message, safeCtx)
}

func eventContextSlogAttrs(ctx EventContext) []any {
	if len(ctx) == 0 {
		return nil
	}
	keys := make([]string, 0, len(ctx))
	for key := range ctx {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	attrs := make([]any, 0, len(keys))
	for _, key := range keys {
		attrs = append(attrs, slog.Any(key, ctx[key]))
	}
	return attrs
}
