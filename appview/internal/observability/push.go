package observability

import (
	"context"
	"log/slog"
	"time"
)

func (o *Observer) ObserveNotificationDecision(category, result string) {
	if o == nil {
		return
	}
	o.metricRecorder.NotificationDecision(context.Background(), category, result)
}
func (o *Observer) ObservePushDelivery(platform, result string) {
	if o == nil {
		return
	}
	o.metricRecorder.PushDelivery(context.Background(), platform, result)
	level := slog.LevelInfo
	ctx := EventContext{
		"component": "push",
		"operation": "push.dispatch",
		"result":    result,
	}
	if result != "success" {
		level = slog.LevelWarn
		ctx["error_category"] = "provider"
	}
	o.Log(context.Background(), level, "push delivery attempt completed", ctx)
}
func (o *Observer) ObservePushQueue(pending int, oldestAge time.Duration) {
	if o == nil {
		return
	}
	if oldestAge < 0 {
		oldestAge = 0
	}
	o.metricRecorder.PushQueue(context.Background(), pending, oldestAge)
}
