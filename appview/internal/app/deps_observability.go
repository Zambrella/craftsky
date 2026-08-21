package app

import (
	"log/slog"
	"time"

	"social.craftsky/appview/internal/observability"
)

// newObservabilityDependencies owns process telemetry construction and its
// bounded shutdown flush. Feature constructors receive the resulting observer
// explicitly and do not register their own process-global cleanup.
func newObservabilityDependencies(
	cfg Config,
	logger *slog.Logger,
	resources *dependencyCleanup,
) *observability.Observer {
	observer := observability.New(observability.Config{
		Env:                 string(cfg.Env),
		Release:             cfg.SentryRelease,
		LogsEnabled:         cfg.SentryLogsEnabled,
		TracingEnabled:      cfg.SentryTracingEnabled,
		TracesSampleRate:    cfg.SentryTracesSampleRate,
		MetricsEnabled:      cfg.SentryMetricsEnabled,
		TapTracingEnabled:   cfg.SentryTapTracingEnabled,
		TapTracesSampleRate: cfg.SentryTapTracesSampleRate,
		SentryDSN:           cfg.SentryDSN,
		Logger:              logger,
	})
	resources.add(func() { observer.Flush(2 * time.Second) })
	return observer
}
