package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

const defaultServiceName = "craftsky_appview"

// Config contains the safe, bounded process metadata exposed by telemetry.
type Config struct {
	Service             string
	Env                 string
	Release             string
	BuildVersion        string
	LogsEnabled         bool
	TracingEnabled      bool
	TracesSampleRate    float64
	MetricsEnabled      bool
	TapTracingEnabled   bool
	TapTracesSampleRate float64
	MetricRecorder      MetricRecorder
	LogSink             LogSink
	SentryDSN           string
	SentryTransport     sentry.Transport
	Logger              *slog.Logger
	FlushFunc           func(time.Duration) bool
}

// Observer owns AppView observability sinks behind local interfaces.
type Observer struct {
	logsEnabled         bool
	logSink             LogSink
	metricsEnabled      bool
	metricRecorder      MetricRecorder
	tracingEnabled      bool
	tapTracingEnabled   bool
	tapTracesSampleRate float64
	sentryClient        *sentry.Client
	sentryHub           *sentry.Hub
	logger              *slog.Logger
	flushFunc           func(time.Duration) bool
	tapLastEventAt      time.Time
}

func New(cfg Config) *Observer {
	if cfg.Service == "" {
		cfg.Service = defaultServiceName
	}
	observer := &Observer{
		logsEnabled:         cfg.SentryDSN != "" && cfg.LogsEnabled,
		logSink:             cfg.LogSink,
		metricsEnabled:      cfg.SentryDSN != "" && cfg.MetricsEnabled,
		metricRecorder:      cfg.MetricRecorder,
		tracingEnabled:      cfg.TracingEnabled,
		tapTracingEnabled:   cfg.SentryDSN != "" && cfg.TapTracingEnabled,
		tapTracesSampleRate: cfg.TapTracesSampleRate,
		logger:              cfg.Logger,
		flushFunc:           cfg.FlushFunc,
	}

	var sentryClient *sentry.Client
	var sentryHub *sentry.Hub
	if cfg.SentryDSN != "" {
		client, err := sentry.NewClient(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.Env,
			Release:          cfg.Release,
			EnableTracing:    cfg.TracingEnabled,
			TracesSampleRate: cfg.TracesSampleRate,
			SendDefaultPII:   false,
			DisableLogs:      !cfg.LogsEnabled,
			DisableMetrics:   !cfg.MetricsEnabled,
			Transport:        cfg.SentryTransport,
		})
		if err == nil {
			sentryClient = client
			sentryHub = sentry.NewHub(client, sentry.NewScope())
		}
	}
	observer.sentryClient = sentryClient
	observer.sentryHub = sentryHub

	if observer.metricRecorder == nil {
		if observer.metricsEnabled {
			observer.metricRecorder = newSentryMetricRecorder(sentryHub)
		} else {
			observer.metricRecorder = noopMetricRecorder{}
		}
	}
	if observer.logSink == nil {
		if observer.logsEnabled {
			observer.logSink = newSentryLogSink(sentryHub)
		} else {
			observer.logSink = noopLogSink{}
		}
	}
	return observer
}

func (o *Observer) BeginHTTPRequest(method, routePattern string) string {
	if routePattern == "" {
		routePattern = unmatchedRoutePattern
	}
	if o == nil {
		return routePattern
	}
	o.metricRecorder.HTTPRequestStarted(context.Background(), method, routePattern)
	return routePattern
}

func (o *Observer) EndHTTPRequest(method, routePattern string) {
	if o == nil {
		return
	}
	if routePattern == "" {
		routePattern = unmatchedRoutePattern
	}
	o.metricRecorder.HTTPRequestEnded(context.Background(), method, routePattern)
}

func (o *Observer) ObserveHTTPRequest(method, routePattern string, status int, duration time.Duration, responseBytes int) {
	if o == nil {
		return
	}
	if routePattern == "" {
		routePattern = unmatchedRoutePattern
	}
	o.metricRecorder.HTTPRequestFinished(context.Background(), method, routePattern, status, duration, responseBytes)
}

func (o *Observer) Flush(timeout time.Duration) bool {
	if o == nil {
		return true
	}
	if o.flushFunc != nil {
		return o.flushFunc(timeout)
	}
	if o.sentryClient != nil {
		return o.sentryClient.Flush(timeout)
	}
	return true
}
