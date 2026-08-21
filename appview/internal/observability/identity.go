package observability

import (
	"context"
	"time"
)

func (o *Observer) ObserveIdentityResolution(mode, direction, result string, duration time.Duration) {
	if o == nil {
		return
	}
	o.metricRecorder.IdentityResolution(context.Background(), mode, direction, result, duration)
}

func (o *Observer) ObserveIdentityCache(result string, age time.Duration) {
	if o == nil {
		return
	}
	o.metricRecorder.IdentityCache(context.Background(), result, age)
}
