package observability

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsUseCraftskyAppViewNamesAndUnits(t *testing.T) {
	observer := New(Config{Env: "test"})
	inFlightRoute := observer.BeginHTTPRequest("GET", "/v1/whoami")
	observer.EndHTTPRequest("GET", inFlightRoute)
	observer.ObserveHTTPRequest("GET", "/v1/whoami", 200, 150*time.Millisecond, 42)

	rec := httptest.NewRecorder()
	observer.MetricsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		"# HELP craftsky_appview_build_info AppView process metadata as a constant gauge with value 1.",
		"# TYPE craftsky_appview_build_info gauge",
		"# HELP craftsky_appview_http_requests_total Total HTTP requests handled by AppView.",
		"# TYPE craftsky_appview_http_requests_total counter",
		"# HELP craftsky_appview_http_request_duration_seconds Duration of AppView HTTP requests in seconds.",
		"# TYPE craftsky_appview_http_request_duration_seconds histogram",
		"# HELP craftsky_appview_http_response_size_bytes AppView HTTP response sizes in bytes.",
		"# TYPE craftsky_appview_http_response_size_bytes histogram",
		"# HELP craftsky_appview_http_requests_in_flight Current AppView HTTP requests in flight.",
		"# TYPE craftsky_appview_http_requests_in_flight gauge",
		"# HELP go_goroutines Number of goroutines that currently exist.",
		"# TYPE go_goroutines gauge",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"sentry_application", "exemplar"} {
		if strings.Contains(strings.ToLower(body), forbidden) {
			t.Fatalf("metrics output contains first-slice non-goal %q:\n%s", forbidden, body)
		}
	}
}
