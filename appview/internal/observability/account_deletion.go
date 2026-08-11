package observability

import (
	"context"
	"time"
)

func (o *Observer) ObserveAccountDeletion(
	ctx context.Context,
	event, phase, outcome, errorCategory string,
	duration time.Duration,
) {
	if o == nil {
		return
	}
	o.metricRecorder.AccountDeletion(ctx, event, phase, outcome, errorCategory, duration)
}
