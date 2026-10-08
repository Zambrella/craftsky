package observability

import (
	"context"
	"social.craftsky/appview/internal/ctxkeys"
)

var correlationFieldNames = map[string]bool{"run_id": true, "request_id": true, "sentry_trace_id": true, "sentry_span_id": true, "case_reference": true}

func withDiagnosticCorrelation(ctx context.Context, fields EventContext) EventContext {
	selected := make(EventContext, len(fields)+3)
	for key, value := range fields {
		selected[key] = value
	}
	if ctx != nil {
		if runID := ctxkeys.GetRunID(ctx); runID != "" {
			selected["run_id"] = runID
		}
		traceID, spanID := TraceIDs(ctx)
		if traceID != "" {
			selected["sentry_trace_id"] = traceID
		}
		if spanID != "" {
			selected["sentry_span_id"] = spanID
		}
	}
	return selected
}
