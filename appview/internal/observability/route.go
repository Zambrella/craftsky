package observability

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
)

const unmatchedRoutePattern = "unmatched"

// RoutePattern returns the low-cardinality ServeMux route pattern for telemetry.
func RoutePattern(r *http.Request) string {
	pattern := r.Pattern
	if pattern == "" {
		return unmatchedRoutePattern
	}
	return strings.TrimPrefix(pattern, r.Method+" ")
}

type routePatternRecorderKey struct{}

type routePatternRecorder struct {
	value atomic.Value
}

// WithRoutePatternRecorder installs a shared recorder so inner middleware can
// publish the ServeMux pattern back to outer middleware that holds an older
// request copy.
func WithRoutePatternRecorder(ctx context.Context) context.Context {
	if _, ok := ctx.Value(routePatternRecorderKey{}).(*routePatternRecorder); ok {
		return ctx
	}
	return context.WithValue(ctx, routePatternRecorderKey{}, &routePatternRecorder{})
}

func RecordRoutePattern(ctx context.Context, routePattern string) {
	if routePattern == "" {
		routePattern = unmatchedRoutePattern
	}
	if recorder, ok := ctx.Value(routePatternRecorderKey{}).(*routePatternRecorder); ok {
		recorder.value.Store(routePattern)
	}
}

func RecordedRoutePattern(ctx context.Context, fallback string) string {
	if recorder, ok := ctx.Value(routePatternRecorderKey{}).(*routePatternRecorder); ok {
		if routePattern, ok := recorder.value.Load().(string); ok && routePattern != "" {
			return routePattern
		}
	}
	if fallback == "" {
		return unmatchedRoutePattern
	}
	return fallback
}
