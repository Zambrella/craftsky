package observability

import (
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	defaultServiceName = "craftsky_appview"
)

// Config contains the safe, bounded process metadata exposed by telemetry.
type Config struct {
	Service          string
	Env              string
	Release          string
	BuildVersion     string
	TracingEnabled   bool
	TracesSampleRate float64
	SentryDSN        string
	SentryTransport  sentry.Transport
	Logger           *slog.Logger
	FlushFunc        func(time.Duration) bool
}

// Observer owns AppView observability sinks. Prometheus metrics are always
// local to this process registry; optional external backends are added later.
type Observer struct {
	registry             *prometheus.Registry
	httpRequests         *prometheus.CounterVec
	httpDuration         *prometheus.HistogramVec
	httpResponseSize     *prometheus.HistogramVec
	httpInFlight         *prometheus.GaugeVec
	dbDuration           *prometheus.HistogramVec
	pdsWriteDuration     *prometheus.HistogramVec
	tapConnected         prometheus.Gauge
	tapReconnects        prometheus.Counter
	tapEventsReceived    *prometheus.CounterVec
	tapEventsAcked       prometheus.Counter
	tapAckFailures       prometheus.Counter
	tapIndexerRecords    *prometheus.CounterVec
	tapIndexerDuration   *prometheus.HistogramVec
	tapLastEventUnixNano atomic.Int64
	tracingEnabled       bool
	sentryClient         *sentry.Client
	sentryHub            *sentry.Hub
	logger               *slog.Logger
	flushFunc            func(time.Duration) bool
}

// New builds an Observer with an isolated Prometheus registry.
func New(cfg Config) *Observer {
	if cfg.Service == "" {
		cfg.Service = defaultServiceName
	}
	registry := prometheus.NewRegistry()
	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "craftsky_appview_build_info",
		Help: "AppView process metadata as a constant gauge with value 1.",
	}, []string{"service", "environment", "release", "build_version"})
	httpRequests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "craftsky_appview_http_requests_total",
		Help: "Total HTTP requests handled by AppView.",
	}, []string{"method", "route_pattern", "status", "status_class"})
	httpDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "craftsky_appview_http_request_duration_seconds",
		Help:    "Duration of AppView HTTP requests in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route_pattern", "status", "status_class"})
	httpResponseSize := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "craftsky_appview_http_response_size_bytes",
		Help:    "AppView HTTP response sizes in bytes.",
		Buckets: []float64{0, 100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
	}, []string{"method", "route_pattern", "status", "status_class"})
	httpInFlight := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "craftsky_appview_http_requests_in_flight",
		Help: "Current AppView HTTP requests in flight.",
	}, []string{"method", "route_pattern"})
	dbDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "craftsky_appview_db_operation_duration_seconds",
		Help:    "Duration of bounded AppView DB operations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation", "route_pattern", "result"})
	pdsWriteDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "craftsky_appview_pds_write_duration_seconds",
		Help:    "Duration of bounded AppView PDS write-proxy operations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation", "stage", "result", "category"})
	tapConnected := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "craftsky_appview_tap_connected",
		Help: "Whether the Tap consumer is currently connected.",
	})
	tapReconnects := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "craftsky_appview_tap_reconnects_total",
		Help: "Total Tap consumer reconnect attempts.",
	})
	tapEventsReceived := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "craftsky_appview_tap_events_received_total",
		Help: "Total Tap events received by envelope type.",
	}, []string{"type"})
	tapEventsAcked := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "craftsky_appview_tap_events_acknowledged_total",
		Help: "Total Tap events acknowledged successfully.",
	})
	tapAckFailures := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "craftsky_appview_tap_ack_failures_total",
		Help: "Total Tap acknowledgement failures.",
	})
	tapIndexerRecords := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "craftsky_appview_tap_indexer_records_total",
		Help: "Total Tap records indexed, skipped, or failed by bounded NSID.",
	}, []string{"nsid", "result", "reason"})
	tapIndexerDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "craftsky_appview_tap_indexer_handling_duration_seconds",
		Help:    "Duration of Tap indexer handling in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"nsid", "result"})

	observer := &Observer{
		registry:           registry,
		httpRequests:       httpRequests,
		httpDuration:       httpDuration,
		httpResponseSize:   httpResponseSize,
		httpInFlight:       httpInFlight,
		dbDuration:         dbDuration,
		pdsWriteDuration:   pdsWriteDuration,
		tapConnected:       tapConnected,
		tapReconnects:      tapReconnects,
		tapEventsReceived:  tapEventsReceived,
		tapEventsAcked:     tapEventsAcked,
		tapAckFailures:     tapAckFailures,
		tapIndexerRecords:  tapIndexerRecords,
		tapIndexerDuration: tapIndexerDuration,
		tracingEnabled:     cfg.TracingEnabled,
		logger:             cfg.Logger,
		flushFunc:          cfg.FlushFunc,
	}
	tapLastEventAge := prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "craftsky_appview_tap_last_event_age_seconds",
		Help: "Age of the last Tap event observed by this AppView in seconds.",
	}, func() float64 {
		last := observer.tapLastEventUnixNano.Load()
		if last == 0 {
			return 0
		}
		return time.Since(time.Unix(0, last)).Seconds()
	})

	// The process collector emits process_* metrics on supported platforms
	// (notably Linux, which is what the AppView container runs in).
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		buildInfo,
		httpRequests,
		httpDuration,
		httpResponseSize,
		httpInFlight,
		dbDuration,
		pdsWriteDuration,
		tapConnected,
		tapReconnects,
		tapEventsReceived,
		tapEventsAcked,
		tapAckFailures,
		tapIndexerRecords,
		tapIndexerDuration,
		tapLastEventAge,
	)
	buildInfo.WithLabelValues(cfg.Service, cfg.Env, cfg.Release, cfg.BuildVersion).Set(1)

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
			Transport:        cfg.SentryTransport,
		})
		if err == nil {
			sentryClient = client
			sentryHub = sentry.NewHub(client, sentry.NewScope())
		}
	}

	observer.sentryClient = sentryClient
	observer.sentryHub = sentryHub
	return observer
}

// MetricsHandler returns a Prometheus text exposition handler.
func (o *Observer) MetricsHandler() http.Handler {
	if o == nil {
		return New(Config{}).MetricsHandler()
	}
	return promhttp.HandlerFor(o.registry, promhttp.HandlerOpts{})
}

// BeginHTTPRequest records an active HTTP request and returns the bounded
// route label that must be used for the matching EndHTTPRequest call.
func (o *Observer) BeginHTTPRequest(method, routePattern string) string {
	if routePattern == "" {
		routePattern = "unmatched"
	}
	if o == nil {
		return routePattern
	}
	o.httpInFlight.WithLabelValues(method, routePattern).Inc()
	return routePattern
}

// EndHTTPRequest records that a previously started HTTP request completed.
func (o *Observer) EndHTTPRequest(method, routePattern string) {
	if o == nil {
		return
	}
	if routePattern == "" {
		routePattern = "unmatched"
	}
	o.httpInFlight.WithLabelValues(method, routePattern).Dec()
}

// ObserveHTTPRequest records the completed HTTP request metrics with bounded labels.
func (o *Observer) ObserveHTTPRequest(method, routePattern string, status int, duration time.Duration, responseBytes int) {
	if o == nil {
		return
	}
	if routePattern == "" {
		routePattern = "unmatched"
	}
	statusCode := strconv.Itoa(status)
	statusClass := strconv.Itoa(status/100) + "xx"
	o.httpRequests.WithLabelValues(method, routePattern, statusCode, statusClass).Inc()
	o.httpDuration.WithLabelValues(method, routePattern, statusCode, statusClass).Observe(duration.Seconds())
	o.httpResponseSize.WithLabelValues(method, routePattern, statusCode, statusClass).Observe(float64(responseBytes))
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
