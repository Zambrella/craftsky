package observability

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

func safeFollowerGrowthResult(result string) string {
	switch strings.TrimSpace(result) {
	case "success", "error", "already_complete":
		return strings.TrimSpace(result)
	default:
		return "unknown"
	}
}

func safeFollowerGrowthErrorCategory(category string) string {
	switch strings.TrimSpace(category) {
	case "none", "capture":
		return strings.TrimSpace(category)
	default:
		return "unknown"
	}
}

func (o *Observer) FollowerGrowthCapture(
	ctx context.Context,
	result string,
	errorCategory string,
	duration time.Duration,
	capturedProfileCount int64,
	latestSuccessfulRunAge *time.Duration,
) {
	if o == nil {
		return
	}
	safeResult := safeFollowerGrowthResult(result)
	safeCategory := safeFollowerGrowthErrorCategory(errorCategory)
	level := slog.LevelInfo
	if safeResult == "error" {
		level = slog.LevelError
	}
	o.Log(ctx, level, "follower growth capture completed", EventContext{
		"component":      "follower_growth",
		"operation":      "capture",
		"result":         safeResult,
		"error_category": safeCategory,
	})
	o.metricRecorder.FollowerGrowthCapture(
		ctx,
		safeResult,
		safeCategory,
		duration,
		capturedProfileCount,
		latestSuccessfulRunAge,
	)
}
