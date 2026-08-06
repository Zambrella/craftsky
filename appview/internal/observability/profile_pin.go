package observability

import (
	"context"
	"time"
)

// ObserveProfilePin records only bounded domain dimensions. It deliberately
// cannot accept an owner, target, state token, content value, or timestamp.
func (o *Observer) ObserveProfilePin(
	operation string,
	slot string,
	result string,
	errorClass string,
	duration time.Duration,
) {
	if o == nil {
		return
	}
	o.metricRecorder.ProfilePinOperation(
		context.Background(),
		operation,
		slot,
		result,
		errorClass,
		nonNegativeDuration(duration),
	)
}
