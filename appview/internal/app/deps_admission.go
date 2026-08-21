package app

import (
	"log/slog"
	"time"

	"social.craftsky/appview/internal/middleware"
)

type admissionDependencies struct {
	rateLimiter *middleware.LocalRateLimiter
}

func newAdmissionDependencies(cfg Config, logger *slog.Logger) *admissionDependencies {
	logger.Warn("rate limiter is process-local; run one AppView instance or configure shared/edge enforcement before horizontal scaling")
	return &admissionDependencies{
		rateLimiter: middleware.NewLocalRateLimiter(cfg.RateLimits, time.Now),
	}
}
