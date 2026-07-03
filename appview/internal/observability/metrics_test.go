package observability

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/auth"
)

func TestInMemoryMetricsRecordAppViewDomainMethods(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	ctx := context.Background()

	recorder.HTTPRequestStarted(ctx, "GET", "/v1/whoami")
	recorder.HTTPRequestFinished(ctx, "GET", "/v1/whoami", 200, 150*time.Millisecond, 42)
	recorder.DBOperation(ctx, "search.posts", "/v1/search/posts", "some", 25*time.Millisecond)
	recorder.PDSOperation(ctx, "post.create", "pds_request", "success", "none", 40*time.Millisecond)
	recorder.TapConnected(ctx, true)
	recorder.TapEventReceived(ctx, "record")
	recorder.TapEventAcknowledged(ctx, "success")
	recorder.TapIndexerRecord(ctx, "social.craftsky.feed.post", "indexed", "none", 10*time.Millisecond)

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_http_requests_in_flight",
		"craftsky_appview_http_requests_total",
		"craftsky_appview_http_request_duration_seconds",
		"craftsky_appview_http_response_size_bytes",
		"craftsky_appview_db_operation_duration_seconds",
		"craftsky_appview_pds_write_duration_seconds",
		"craftsky_appview_tap_connected",
		"craftsky_appview_tap_events_received_total",
		"craftsky_appview_tap_events_acknowledged_total",
		"craftsky_appview_tap_indexer_records_total",
		"craftsky_appview_tap_indexer_handling_duration_seconds",
	} {
		if !metricCallsContain(calls, want) {
			t.Fatalf("metric calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		for key, value := range call.Attributes {
			if key == "run_id" {
				t.Fatalf("metric call includes high-cardinality run_id attribute: %#v", call)
			}
			if strings.Contains(value, "did:") || strings.Contains(value, "secret") {
				t.Fatalf("metric call includes forbidden attribute value: %#v", call)
			}
		}
	}
}

func TestObserverSentryMetricsRequireExplicitMetricsGate(t *testing.T) {
	ctx := context.Background()

	disabledTransport := &sentry.MockTransport{}
	disabled := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: disabledTransport,
	})
	disabled.ObserveHTTPRequest("GET", "/v1/whoami", 200, time.Millisecond, 12)
	if !disabled.Flush(50 * time.Millisecond) {
		t.Fatal("disabled Flush returned false")
	}
	if metrics := sentryMetricNames(disabledTransport.Events()); len(metrics) != 0 {
		t.Fatalf("DSN-only observer emitted metrics %v, want none", metrics)
	}

	enabledTransport := &sentry.MockTransport{}
	enabled := New(Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: enabledTransport,
		MetricsEnabled:  true,
	})
	enabled.BeginHTTPRequest("GET", "/v1/whoami")
	enabled.EndHTTPRequest("GET", "/v1/whoami")
	enabled.ObserveHTTPRequest("GET", "/v1/whoami", 200, time.Millisecond, 12)
	enabled.ObserveDB(ctx, DBOperation{Operation: "search.posts", RoutePattern: "/v1/search/posts"}, func(context.Context) error {
		return nil
	})
	if !enabled.Flush(time.Second) {
		t.Fatal("enabled Flush returned false")
	}

	names := sentryMetricNames(enabledTransport.Events())
	for _, want := range []string{
		"craftsky_appview_http_requests_total",
		"craftsky_appview_http_request_duration_seconds",
		"craftsky_appview_http_response_size_bytes",
		"craftsky_appview_db_operation_duration_seconds",
	} {
		if !slices.Contains(names, want) {
			t.Fatalf("Sentry metric names missing %q in %v", want, names)
		}
	}
}

func TestObserverHTTPInFlightRecordsConcurrentActiveCounts(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})

	firstRoutePattern := observer.BeginHTTPRequest("GET", "/v1/whoami")
	secondRoutePattern := observer.BeginHTTPRequest("GET", "/v1/whoami")
	observer.EndHTTPRequest("GET", firstRoutePattern)
	observer.EndHTTPRequest("GET", secondRoutePattern)

	var values []float64
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_http_requests_in_flight" {
			values = append(values, call.Value)
		}
	}
	if !slices.Equal(values, []float64{1, 2, 1, 0}) {
		t.Fatalf("in-flight values = %v, want [1 2 1 0]", values)
	}
}

func TestObserverMetricRecorderCoversRepresentativeOperations(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})
	ctx := context.Background()

	observer.BeginHTTPRequest("GET", "/v1/whoami")
	observer.EndHTTPRequest("GET", "/v1/whoami")
	observer.ObserveHTTPRequest("GET", "/v1/whoami", 200, time.Millisecond, 12)
	if err := observer.ObserveDB(ctx, DBOperation{Operation: "search.posts", RoutePattern: "/v1/search/posts"}, func(context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("ObserveDB: %v", err)
	}
	wrappedFactory := observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
		return fakePDSClient{}, nil
	})
	client, err := wrappedFactory(ctx, syntax.DID("did:plc:writer"), "session-secret")
	if err != nil {
		t.Fatalf("wrappedFactory: %v", err)
	}
	_, _, _ = client.CreateRecord(ctx, syntax.DID("did:plc:writer"), "social.craftsky.feed.post", map[string]any{"text": "secret body"})
	observer.SetTapConnected(true)
	observer.ObserveTapEventReceived("record")
	observer.ObserveTapEventAcknowledged(nil)
	observer.ObserveIndexerHandled("social.craftsky.feed.post", nil, time.Millisecond)

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_http_requests_total",
		"craftsky_appview_db_operation_duration_seconds",
		"craftsky_appview_pds_write_duration_seconds",
		"craftsky_appview_tap_connected",
		"craftsky_appview_tap_events_received_total",
		"craftsky_appview_tap_events_acknowledged_total",
		"craftsky_appview_tap_indexer_records_total",
	} {
		if !metricCallsContain(calls, want) {
			t.Fatalf("metric recorder calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
}

func metricCallsContain(calls []MetricCall, name string) bool {
	for _, call := range calls {
		if call.Name == name {
			return true
		}
	}
	return false
}

func sentryMetricNames(events []*sentry.Event) []string {
	var names []string
	for _, event := range events {
		for _, metric := range event.Metrics {
			names = append(names, metric.Name)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}
