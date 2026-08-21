package observability

import (
	"context"
	"time"
)

// ObserveAuthRequestSweep records only bounded aggregate queue state. OAuth
// state, request URIs, owners, and device identifiers are deliberately absent.
func (o *Observer) ObserveAuthRequestSweep(
	pending int64,
	oldestAge time.Duration,
	deleted int64,
	failed bool,
) {
	if o == nil {
		return
	}
	o.metricRecorder.AuthRequestSweep(
		context.Background(), pending, nonNegativeDuration(oldestAge), deleted, failed,
	)
}
