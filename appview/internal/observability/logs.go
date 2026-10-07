package observability

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	sentryslog "github.com/getsentry/sentry-go/slog"
)

type LogSink interface {
	Emit(ctx context.Context, level slog.Level, message string, attrs EventContext)
}

type noopLogSink struct{}

func (noopLogSink) Emit(context.Context, slog.Level, string, EventContext) {}

type sentryLogSink struct{ handler slog.Handler }

func newSentryLogSink(hub *sentry.Hub) LogSink {
	if hub == nil {
		return noopLogSink{}
	}
	ctx := sentry.SetHubOnContext(context.Background(), hub)
	return sentryLogSink{handler: sentryslog.Option{}.NewSentryHandler(ctx)}
}
func (s sentryLogSink) Emit(ctx context.Context, level slog.Level, message string, attrs EventContext) {
	if ctx == nil {
		ctx = context.Background()
	}
	if message == "" {
		return
	}
	selected := SanitizeEventContext(attrs)
	for _, key := range []string{"diagnostic", "incoming_path", "method", "status", "bytes", "content_length", "count", "suppressed_count"} {
		if value, ok := attrs[key]; ok {
			selected[key] = value
		}
	}
	// Logs need one brief cause when a retry deliberately creates no issue.
	// Full chains and origin stacks remain exclusively on native exceptions.
	if causes, ok := attrs["causes"].([]DiagnosticCause); ok && len(causes) > 0 {
		cause := causes[0]
		for _, candidate := range causes {
			if candidate.Code != "" {
				cause = candidate
				break
			}
		}
		selected["error.type"] = safeDiagnosticType(cause.Type)
		if sqlStatePattern.MatchString(cause.Code) {
			selected["error.code"] = cause.Code
		} else if code, err := strconv.Atoi(cause.Code); err == nil && code >= 100 && code <= 599 {
			selected["http_status"] = code
		}
	}

	record := slog.NewRecord(time.Now(), level, boundDiagnosticText(message, 512), 0)
	for key, value := range selected {
		switch value.(type) {
		case EventContext:
			if data, err := json.Marshal(value); err == nil {
				record.AddAttrs(slog.String(key, string(data)))
			}
		default:
			record.AddAttrs(slog.Any(key, value))
		}
	}
	// Breadcrumbs belong to a request/operation hub, never shared process history.
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{Category: "log", Message: record.Message, Level: sentry.LevelInfo, Data: map[string]any(SanitizeEventContext(attrs))}, nil)
	}
	// The official handler emits Logs only; issue ownership remains explicit.
	_ = s.handler.Handle(ctx, record)
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
