package observability

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// EventContext is the safe technical context shape used before handing data
// to external error/tracing backends.
type EventContext map[string]any

var allowedEventContextKeys = map[string]struct{}{
	"service":           {},
	"environment":       {},
	"release":           {},
	"component":         {},
	"operation":         {},
	"route_pattern":     {},
	"http_method":       {},
	"http_status":       {},
	"http_status_class": {},
	"error_category":    {},
	"failure_stage":     {},
	"duration":          {},
	"result":            {},
	"nsid":              {},
	"tap_connected":     {},
	"reconnect_attempt": {},
	"run_id":            {},
	"sentry_trace_id":   {},
	"sentry_span_id":    {},
}

func SanitizeEventContext(ctx EventContext) EventContext {
	out := EventContext{}
	for key, value := range ctx {
		if _, ok := allowedEventContextKeys[key]; ok {
			out[key] = value
		}
	}
	return out
}

func (o *Observer) CaptureError(ctx context.Context, eventCtx EventContext, err error) {
	if o == nil || o.sentryClient == nil || err == nil {
		return
	}
	MarkCaptured(ctx)
	o.capture(ctx, "appview actionable error", fmt.Sprintf("%T", err), "redacted", eventCtx)
}

func (o *Observer) CapturePanic(ctx context.Context, eventCtx EventContext, recovered any) {
	if o == nil || o.sentryClient == nil || recovered == nil {
		return
	}
	MarkCaptured(ctx)
	o.capture(ctx, "appview panic recovered", fmt.Sprintf("%T", recovered), "redacted", eventCtx)
}

func (o *Observer) capture(ctx context.Context, message, exceptionType, exceptionValue string, eventCtx EventContext) {
	tags := map[string]string{}
	for key, value := range SanitizeEventContext(eventCtx) {
		tags[key] = fmt.Sprint(value)
	}
	if traceID, spanID := TraceIDs(ctx); traceID != "" || spanID != "" {
		if traceID != "" {
			tags["sentry_trace_id"] = traceID
		}
		if spanID != "" {
			tags["sentry_span_id"] = spanID
		}
	}
	o.sentryClient.CaptureEvent(&sentry.Event{
		Message: message,
		Level:   sentry.LevelError,
		Tags:    tags,
		Exception: []sentry.Exception{{
			Type:  exceptionType,
			Value: exceptionValue,
		}},
	}, nil, nil)
}

type captureMarkerKey struct{}

type captureMarker struct {
	captured atomic.Bool
}

func WithCaptureMarker(ctx context.Context) context.Context {
	if _, ok := ctx.Value(captureMarkerKey{}).(*captureMarker); ok {
		return ctx
	}
	return context.WithValue(ctx, captureMarkerKey{}, &captureMarker{})
}

func MarkCaptured(ctx context.Context) {
	marker, ok := ctx.Value(captureMarkerKey{}).(*captureMarker)
	if ok {
		marker.captured.Store(true)
	}
}

// CaptureRecorded reports whether a request has either emitted a Sentry event
// or deliberately handled a classified error without capture.
func CaptureRecorded(ctx context.Context) bool {
	marker, ok := ctx.Value(captureMarkerKey{}).(*captureMarker)
	return ok && marker.captured.Load()
}

type traceContextKey struct{}

type traceIDs struct {
	traceID string
	spanID  string
}

type SpanContext struct {
	Operation  string
	Component  string
	Attributes EventContext
}

type Span struct {
	enabled    bool
	result     string
	sentrySpan *sentry.Span
}

func (s *Span) Enabled() bool {
	return s != nil && s.enabled
}

func (s *Span) Finish(result string) {
	if s != nil {
		s.result = result
		if s.sentrySpan != nil {
			if result != "" {
				s.sentrySpan.SetData("result", result)
			}
			switch result {
			case "success":
				s.sentrySpan.Status = sentry.SpanStatusOK
			case "error":
				s.sentrySpan.Status = sentry.SpanStatusInternalError
			}
			s.sentrySpan.Finish()
		}
	}
}

func (s *Span) SetAttributes(attrs EventContext) {
	if s == nil || s.sentrySpan == nil {
		return
	}
	for key, value := range SanitizeEventContext(attrs) {
		s.sentrySpan.SetData(key, value)
	}
}

func (s *Span) SetTransactionName(name string) {
	if s == nil || s.sentrySpan == nil || name == "" {
		return
	}
	transaction := s.sentrySpan.GetTransaction()
	if transaction == nil {
		s.sentrySpan.Name = name
		return
	}
	transaction.Name = name
}

func (s *Span) Result() string {
	if s == nil {
		return ""
	}
	return s.result
}

func (o *Observer) StartSpan(ctx context.Context, spanCtx SpanContext) (context.Context, *Span) {
	if o == nil || !o.tracingEnabled {
		return ctx, &Span{}
	}
	if spanCtx.Operation == "" {
		spanCtx.Operation = "unknown"
	}
	if spanCtx.Component == "" {
		spanCtx.Component = "unknown"
	}
	if o.sentryHub != nil {
		ctx = sentry.SetHubOnContext(ctx, o.sentryHub)
		options := []sentry.SpanOption{}
		if sentry.SpanFromContext(ctx) == nil {
			options = append(options, sentry.WithTransactionName(spanCtx.Operation))
		}
		sdkSpan := sentry.StartSpan(ctx, spanCtx.Operation, options...)
		sdkSpan.SetData("component", spanCtx.Component)
		sdkSpan.SetData("operation", spanCtx.Operation)
		for key, value := range SanitizeEventContext(spanCtx.Attributes) {
			sdkSpan.SetData(key, value)
		}
		ctx = context.WithValue(sdkSpan.Context(), traceContextKey{}, traceIDs{
			traceID: sdkSpan.TraceID.String(),
			spanID:  sdkSpan.SpanID.String(),
		})
		return ctx, &Span{enabled: true, sentrySpan: sdkSpan}
	}
	ids := traceIDs{traceID: uuid.NewString(), spanID: uuid.NewString()}
	return context.WithValue(ctx, traceContextKey{}, ids), &Span{enabled: true}
}

func TraceIDs(ctx context.Context) (string, string) {
	ids, ok := ctx.Value(traceContextKey{}).(traceIDs)
	if !ok {
		return "", ""
	}
	return ids.traceID, ids.spanID
}
