package main

import (
	"context"
	"log/slog"
	"time"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
)

type authRequestSweeper interface {
	SweepAuthRequests(context.Context, int) (auth.AuthRequestSweepStats, error)
}

func runAuthRequestSweeper(
	ctx context.Context,
	sweeper authRequestSweeper,
	logger *slog.Logger,
	observer *observability.Observer,
	batch int,
	interval time.Duration,
) {
	if sweeper == nil || batch <= 0 || interval <= 0 {
		return
	}
	for {
		stats, err := sweeper.SweepAuthRequests(ctx, batch)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			observer.ObserveAuthRequestSweep(0, 0, 0, true)
			if logger != nil {
				logger.Error("OAuth auth-request sweep failed",
					slog.String("component", "oauth_auth_requests"),
					slog.String("operation", "sweep"),
					slog.String("result", "error"),
					slog.String("error_category", "store"))
			}
		} else {
			var oldestAge time.Duration
			if stats.OldestPendingCreatedAt != nil {
				oldestAge = time.Since(*stats.OldestPendingCreatedAt)
				if oldestAge < 0 {
					oldestAge = 0
				}
			}
			observer.ObserveAuthRequestSweep(stats.Pending, oldestAge, stats.Deleted, false)
			if stats.Deleted > 0 {
				continue
			}
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}
