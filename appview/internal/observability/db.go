package observability

import (
	"context"
	"time"
)

// DBOperation describes a bounded database operation for telemetry.
type DBOperation struct {
	Operation    string
	RoutePattern string
	RunID        string
}

// ObserveDB records duration and result for a bounded DB operation.
func (o *Observer) ObserveDB(ctx context.Context, op DBOperation, fn func(context.Context) error) error {
	started := time.Now()
	err := fn(ctx)
	if o != nil {
		result := "success"
		if err != nil {
			result = "error"
		}
		operation := op.Operation
		if operation == "" {
			operation = "unknown"
		}
		routePattern := op.RoutePattern
		if routePattern == "" {
			routePattern = unmatchedRoutePattern
		}
		o.dbDuration.WithLabelValues(operation, routePattern, result).Observe(time.Since(started).Seconds())
	}
	return err
}
