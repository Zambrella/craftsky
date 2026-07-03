package observability

import (
	"context"
	"time"
)

// DBOperation describes a bounded database operation for telemetry.
type DBOperation struct {
	Operation    string
	RoutePattern string
	ResultClass  string
	RunID        string
}

// ObserveDB records duration and result for a bounded DB operation.
func (o *Observer) ObserveDB(ctx context.Context, op DBOperation, fn func(context.Context) error) error {
	if o == nil {
		return fn(ctx)
	}
	started := time.Now()
	operation := op.Operation
	if operation == "" {
		operation = "unknown"
	}
	operation = safeMetricOperation(operation)
	routePattern := op.RoutePattern
	if routePattern == "" {
		routePattern = unmatchedRoutePattern
	}
	routePattern = safeMetricRoute(routePattern)
	spanCtx, span := o.StartSpan(ctx, SpanContext{
		Operation: "db." + operation,
		Component: "db",
		Attributes: EventContext{
			"component":     "db",
			"operation":     operation,
			"route_pattern": routePattern,
		},
	})
	err := fn(spanCtx)
	result := "success"
	if op.ResultClass != "" {
		result = safeMetricResult(op.ResultClass)
	}
	if err != nil {
		result = "error"
	}
	duration := time.Since(started)
	if span != nil {
		span.SetAttributes(EventContext{
			"component":     "db",
			"operation":     operation,
			"route_pattern": routePattern,
			"result":        result,
			"duration":      duration.String(),
		})
		span.Finish(result)
	}
	o.metricRecorder.DBOperation(spanCtx, operation, routePattern, result, duration)
	return err
}
