package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/linkpreview"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/routes"
	"social.craftsky/appview/internal/testdb"
)

func TestIR017LinkPreviewAdmissionExcludesCanariesFromEveryApplicableSink(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		INSERT INTO craftsky_profiles (did) VALUES ('did:plc:test');
	`)
	canaries := []string{
		"admission-source-canary.invalid/private?query=secret",
		"admission-final-canary.invalid/redirect",
		"admission-title-canary",
		"admission-description-canary",
		"admission-thumbnail-canary",
		"admission-bearer-canary",
		"admission-device-canary",
	}
	finalURL, err := url.Parse("https://" + canaries[1])
	if err != nil {
		t.Fatal(err)
	}

	var localLogs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&localLogs, nil))
	metrics := observability.NewInMemoryMetricRecorder()
	externalLogs := &admissionLogSink{}
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env: "test", SentryDSN: "https://public@example.invalid/1",
		SentryTransport: transport, TracingEnabled: true, TracesSampleRate: 1,
		LogsEnabled: true, MetricRecorder: metrics, LogSink: externalLogs, Logger: logger,
	})
	service := admissionPreviewService{preview: linkpreview.Preview{
		URL: finalURL, Title: canaries[2], Description: canaries[3],
		Thumbnail: &linkpreview.Thumbnail{Bytes: []byte(canaries[4]), MIMEType: "image/png", Width: 1, Height: 1},
	}}
	build := func(enabled bool, limit int) http.Handler {
		config := routes.Config{Env: routes.EnvDev, AllowedOrigins: []string{"*"}, LinkPreviewsEnabled: enabled}
		deps := &routes.Dependencies{
			Config: config, Logger: logger, DB: pool,
			AuthService:    &auth.MockAuthService{DefaultDID: "did:plc:test"},
			HandleResolver: admissionResolver{}, LinkPreviews: service, Observability: observer,
			RateLimiter: middleware.NewLocalRateLimiter(middleware.RateLimitConfig{Classes: map[middleware.RateClass]middleware.ClassLimit{
				middleware.RateClassLinkPreview: {Window: time.Hour, PerToken: limit, PerDevice: limit},
			}}, func() time.Time { return time.Unix(1_000, 0) }),
		}
		mux := http.NewServeMux()
		routes.AddRoutes(context.Background(), mux, deps)
		catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(config.Env, config))
		if err != nil {
			t.Fatal(err)
		}
		return middleware.Logging(logger)(middleware.HTTPMetrics(observer)(catalogue.RoutingHandler(mux)))
	}
	serve := func(handler http.Handler, withAuth, withDevice bool) {
		request := httptest.NewRequest(
			http.MethodPost,
			"/v1/link-previews",
			strings.NewReader(`{"url":"https://`+canaries[0]+`"}`),
		)
		if withAuth {
			request.Header.Set("Authorization", "Bearer "+canaries[5])
		}
		if withDevice {
			request.Header.Set("X-Craftsky-Device-Id", canaries[6])
		}
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
	serve(build(true, 10), false, true)
	serve(build(true, 10), true, false)
	serve(build(false, 10), true, true)
	rateLimited := build(true, 1)
	serve(rateLimited, true, true)
	serve(rateLimited, true, true)
	if !observer.Flush(time.Second) {
		t.Fatal("observer flush failed")
	}

	metricJSON, _ := json.Marshal(metrics.Calls())
	eventJSON, _ := json.Marshal(transport.Events())
	captured := strings.Join([]string{localLogs.String(), externalLogs.String(), string(metricJSON), string(eventJSON)}, "\n")
	for _, canary := range canaries {
		if strings.Contains(captured, canary) {
			t.Fatalf("link-preview admission telemetry leaked %q:\n%s", canary, captured)
		}
	}
	if localLogs.Len() == 0 || len(metrics.Calls()) == 0 || len(transport.Events()) == 0 {
		t.Fatalf("expected local logs, metrics, and Sentry trace/error events; captured=%s", captured)
	}
	var sawError, sawTransaction bool
	for _, event := range transport.Events() {
		sawError = sawError || event.Level == sentry.LevelError
		sawTransaction = sawTransaction || event.Type == "transaction"
	}
	if !sawError || !sawTransaction {
		t.Fatalf("Sentry events missing error or transaction: %#v", transport.Events())
	}
	if len(externalLogs.events) != 0 {
		t.Fatalf("admission unexpectedly emitted external logs: %s", externalLogs.String())
	}
}

type admissionPreviewService struct {
	preview linkpreview.Preview
}

func (service admissionPreviewService) FetchPreview(context.Context, string) (linkpreview.Preview, error) {
	return service.preview, nil
}

type admissionResolver struct{}

func (admissionResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return "stub-handle.example", nil
}

func (admissionResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "did:plc:test", nil
}

type admissionLogSink struct {
	events []observability.EventContext
}

func (sink *admissionLogSink) Emit(_ context.Context, _ slog.Level, _ string, attrs observability.EventContext) {
	sink.events = append(sink.events, attrs)
}

func (sink *admissionLogSink) String() string {
	body, _ := json.Marshal(sink.events)
	return string(body)
}

var _ api.HandleResolver = admissionResolver{}
