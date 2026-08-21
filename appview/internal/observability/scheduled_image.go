package observability

import (
	"context"
	"time"
)

func (o *Observer) ObserveScheduledImageValidation(
	result string,
	format string,
	duration time.Duration,
	inFlight int,
) {
	if o == nil {
		return
	}
	o.metricRecorder.ScheduledImageValidation(
		context.Background(),
		result,
		format,
		duration,
		inFlight,
	)
}
