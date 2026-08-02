package observability

import (
	"context"
	"time"
)

func (o *Observer) ObserveScheduledQueue(
	status string,
	count int,
	due int,
	overdue int,
	oldestDueAge time.Duration,
) {
	if o == nil {
		return
	}
	o.metricRecorder.ScheduledQueue(
		context.Background(),
		status,
		count,
		due,
		overdue,
		oldestDueAge,
	)
}

func (o *Observer) ObserveScheduledOperation(
	operation string,
	result string,
	errorClass string,
	duration time.Duration,
) {
	if o == nil {
		return
	}
	o.metricRecorder.ScheduledOperation(
		context.Background(),
		operation,
		result,
		errorClass,
		duration,
	)
}

func (o *Observer) ObserveScheduledPublication(
	attempt int,
	startLatency time.Duration,
	duration time.Duration,
) {
	if o == nil {
		return
	}
	o.metricRecorder.ScheduledPublication(
		context.Background(),
		attempt,
		startLatency,
		duration,
	)
}

func (o *Observer) ObserveScheduledCleanupQueue(pending int, oldestAge time.Duration) {
	if o == nil {
		return
	}
	o.metricRecorder.ScheduledCleanupQueue(
		context.Background(),
		pending,
		oldestAge,
	)
}
