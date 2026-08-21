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

func (o *Observer) ObservePushOperation(
	stage string,
	platform string,
	semantics string,
	outcome string,
	duration time.Duration,
	count int64,
) {
	if o == nil {
		return
	}
	o.metricRecorder.PushOperation(
		context.Background(),
		stage,
		platform,
		semantics,
		outcome,
		duration,
		count,
	)
	labels := pushOperationAttributes(stage, platform, semantics, outcome)
	if labels["outcome"] != "accepted_unfinalized" && labels["outcome"] != "error" {
		return
	}
	o.Log(context.Background(), slog.LevelWarn, "push operation requires attention", EventContext{
		"component":      "push",
		"operation":      "push." + labels["stage"],
		"result":         "error",
		"failure_stage":  labels["stage"],
		"error_category": "storage",
		"duration":       nonNegativeDuration(duration).Seconds(),
	})
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
