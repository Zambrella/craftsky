package observability

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
)

// EventContext is the safe technical context shape used before handing data
// to external error/tracing backends.
type EventContext map[string]any

var allowedEventContextKeys = map[string]struct{}{
	"service":           {},
	"run_id":            {},
	"environment":       {},
	"release":           {},
	"component":         {},
	"operation":         {},
	"route_pattern":     {},
	"http_method":       {},
	"http_status":       {},
	"http_status_class": {},
	"error_category":    {},
	"error_code":        {},
	"failure_stage":     {},
	"duration":          {},
	"result":            {},
	"reason":            {},
	"retryable":         {},
	"attempt":           {},
	"nsid":              {},
	"tap_connected":     {},
	"reconnect_attempt": {},
	"recovered_type":    {},
	"sentry_trace_id":   {},
	"sentry_span_id":    {},
	"actor_type":        {},
	"request_id":        {},
	"case_reference":    {},
	"occurred_at":       {},
}

func SanitizeEventContext(ctx EventContext) EventContext {
	out := EventContext{}
	keys := make([]string, 0, len(allowedEventContextKeys))
	for key := range allowedEventContextKeys {
		if _, ok := ctx[key]; ok {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, pj := logFieldPriority(keys[i]), logFieldPriority(keys[j])
		if pi != pj {
			return pi < pj
		}
		return keys[i] < keys[j]
	})
	for _, key := range keys {
		if len(out) >= 32 {
			break
		}
		value := ctx[key]
		switch v := value.(type) {
		case string:
			if len(v) > MaxDiagnosticTextBytes {
				out[key] = "[OMITTED: oversized field]"
			} else {
				out[key] = sanitizeEventContextValue(key, v)
			}
		case bool, int, int64, float64, time.Duration:
			out[key] = sanitizeEventContextValue(key, value)
		}
	}
	return out
}

func sanitizeEventContextValue(key string, value any) any {
	switch key {
	case "run_id":
		if _, err := uuid.Parse(fmt.Sprint(value)); err != nil {
			return "[OMITTED: invalid request ID]"
		}
		return value
	case "duration":
		if duration, err := time.ParseDuration(fmt.Sprint(value)); err == nil {
			return duration.String()
		}
		return "[OMITTED: invalid duration]"
	case "component", "operation":
		return safeMetricOperation(fmt.Sprint(value))
	case "route_pattern":
		return safeMetricRoute(fmt.Sprint(value))
	case "http_method":
		return safeHTTPMethod(fmt.Sprint(value))
	case "http_status":
		switch v := value.(type) {
		case int:
			return safeHTTPStatus(v)
		case string:
			n, err := strconv.Atoi(v)
			if err != nil {
				return "000"
			}
			return safeHTTPStatus(n)
		default:
			return "000"
		}
	case "http_status_class":
		return safeHTTPStatusClassString(fmt.Sprint(value))
	case "error_category":
		return safeMetricCategory(fmt.Sprint(value))
	case "error_code":
		return safeMetricOperation(fmt.Sprint(value))
	case "failure_stage":
		return safeMetricStage(fmt.Sprint(value))
	case "result":
		if text, ok := value.(string); ok {
			switch text {
			case "retry", "terminal", "exhausted", "quarantine":
				return text
			}
		}
		if strings.TrimSpace(fmt.Sprint(value)) == "alert" {
			return "alert"
		}
		moderationResult := safeModerationResult(fmt.Sprint(value))
		if moderationResult != "unknown" {
			return moderationResult
		}
		return safeMetricResult(fmt.Sprint(value))
	case "actor_type":
		return safeModerationActorType(fmt.Sprint(value))
	case "request_id":
		return safeModerationRequestIdentifier(fmt.Sprint(value))
	case "case_reference":
		return safeModerationCaseReference(fmt.Sprint(value))
	case "occurred_at":
		parsed, err := time.Parse(time.RFC3339Nano, fmt.Sprint(value))
		if err != nil {
			return "unknown"
		}
		return parsed.UTC().Format(time.RFC3339Nano)
	case "reason":
		return safeMigrationReason(fmt.Sprint(value))
	case "retryable":
		retryable, ok := value.(bool)
		return ok && retryable
	default:
		switch v := value.(type) {
		case string:
			if diagnosticTechnicalPattern.MatchString(v) {
				return v
			}
			return "[OMITTED]"
		case bool, int, int64, float64, time.Duration:
			return value
		default:
			return "[OMITTED]"
		}
	}
}

func (o *Observer) CaptureError(ctx context.Context, eventCtx EventContext, err error) {
	o.CaptureDiagnostic(ctx, DiagnosticInput{Error: err, Context: eventCtx})
}

func (o *Observer) CapturePanic(ctx context.Context, eventCtx EventContext, recovered any) {
	if o == nil || recovered == nil || !claimCapture(ctx) {
		return
	}
	if o.sentryClient == nil {
		return
	}
	if eventCtx == nil {
		eventCtx = EventContext{}
	}
	eventCtx["recovered_type"] = fmt.Sprintf("%T", recovered)
	eventCtx["result"] = "error"

	exceptions := diagnosticPanicExceptions(recovered, eventCtx)
	original, _ := recovered.(error)
	o.captureExceptions(ctx, "appview panic recovered", exceptions, eventCtx, original)
}

func (o *Observer) capture(ctx context.Context, message, exceptionType, exceptionValue string, eventCtx EventContext) {
	o.captureExceptions(ctx, message, []sentry.Exception{{Type: exceptionType, Value: exceptionValue}}, eventCtx, nil)
}

func (o *Observer) captureExceptions(ctx context.Context, message string, exceptions []sentry.Exception, eventCtx EventContext, original error, workflowContexts ...EventContext) {
	defer func() {
		if recover() != nil {
			causes := make([]DiagnosticCause, 0, len(exceptions))
			for _, exception := range exceptions {
				causes = append(causes, DiagnosticCause{Type: exception.Type, Message: exception.Value})
			}
			o.localFailure(ctx, "telemetry capture failed", eventCtx, causes)
		}
	}()
	eventCtx = withDiagnosticCorrelation(ctx, eventCtx)
	tags := map[string]string{}
	correlation := EventContext{}
	for key, value := range SanitizeEventContext(eventCtx) {
		if correlationFieldNames[key] {
			correlation[key] = value
		} else {
			tags[key] = fmt.Sprint(value)
		}
	}
	event := &sentry.Event{
		Message:   message,
		Level:     sentry.LevelError,
		Tags:      tags,
		Exception: exceptions,
	}
	if len(workflowContexts) > 0 && len(workflowContexts[0]) > 0 {
		event.Contexts = map[string]sentry.Context{"diagnostic": sentry.Context(workflowContexts[0])}
	}
	if event.Contexts == nil {
		event.Contexts = map[string]sentry.Context{}
	}
	if len(correlation) > 0 {
		event.Contexts["correlation"] = sentry.Context(correlation)
	}
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = o.sentryHub
	}
	if hub == nil {
		o.sentryClient.CaptureEvent(event, &sentry.EventHint{Context: ctx, OriginalException: original}, nil)
		return
	}
	captureHub := hub.Clone()
	captureHub.ConfigureScope(func(scope *sentry.Scope) {
		if span := sentry.SpanFromContext(ctx); span != nil && span.TraceID.String() != "" {
			scope.SetSpan(span)
		}
	})
	captureHub.CaptureEventWithHint(event, &sentry.EventHint{Context: ctx, OriginalException: original})
}

type captureMarkerKey struct{}

type captureMarker struct {
	captured atomic.Bool
	panicked atomic.Bool
}

// MarkPanicRecovered records the request outcome independently of remote capture.
func MarkPanicRecovered(ctx context.Context) {
	if marker, ok := ctx.Value(captureMarkerKey{}).(*captureMarker); ok {
		marker.panicked.Store(true)
	}
}

func PanicRecovered(ctx context.Context) bool {
	marker, ok := ctx.Value(captureMarkerKey{}).(*captureMarker)
	return ok && marker.panicked.Load()
}

func WithCaptureMarker(ctx context.Context) context.Context {
	if _, ok := ctx.Value(captureMarkerKey{}).(*captureMarker); ok {
		return ctx
	}
	return context.WithValue(ctx, captureMarkerKey{}, &captureMarker{})
}

// claimCapture assigns one owner to this occurrence even without a backend.
// Contexts without a marker represent independent standalone occurrences.
func claimCapture(ctx context.Context) bool {
	if marker, ok := ctx.Value(captureMarkerKey{}).(*captureMarker); ok {
		return marker.captured.CompareAndSwap(false, true)
	}
	return true
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
	name = safeTransactionName(name)
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
	spanCtx.Operation = safeMetricOperation(spanCtx.Operation)
	spanCtx.Component = safeMetricOperation(spanCtx.Component)
	if o.sentryHub != nil {
		ctx = sentry.SetHubOnContext(ctx, o.sentryHub)
		options := []sentry.SpanOption{}
		if sentry.SpanFromContext(ctx) == nil {
			options = append(options, sentry.WithTransactionName(spanCtx.Operation))
		}
		sdkSpan := sentry.StartSpan(ctx, spanCtx.Operation, options...)
		sdkSpan.SetData("component", spanCtx.Component)
		sdkSpan.SetData("operation", spanCtx.Operation)
		attributes := withDiagnosticCorrelation(ctx, spanCtx.Attributes)
		attributes["sentry_trace_id"] = sdkSpan.TraceID.String()
		attributes["sentry_span_id"] = sdkSpan.SpanID.String()
		for key, value := range SanitizeEventContext(attributes) {
			sdkSpan.SetData(key, value)
		}
		ctx = context.WithValue(sdkSpan.Context(), traceContextKey{}, traceIDs{
			traceID: sdkSpan.TraceID.String(),
			spanID:  sdkSpan.SpanID.String(),
		})
		return ctx, &Span{enabled: true, sentrySpan: sdkSpan}
	}
	return ctx, &Span{}
}

func TraceIDs(ctx context.Context) (string, string) {
	ids, ok := ctx.Value(traceContextKey{}).(traceIDs)
	if !ok {
		return "", ""
	}
	return ids.traceID, ids.spanID
}

func safeTransactionName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown"
	}
	parts := strings.SplitN(name, " ", 2)
	if len(parts) == 2 {
		method := safeHTTPMethod(parts[0])
		if method != "OTHER" {
			return method + " " + safeMetricRoute(parts[1])
		}
	}
	return safeMetricOperation(name)
}
