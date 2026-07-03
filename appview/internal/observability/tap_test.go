package observability

import (
	"context"
	"testing"
	"time"
)

func TestSafeNSIDLabelUsesKnownNSIDsOrBoundedFallbacks(t *testing.T) {
	cases := []struct {
		name string
		nsid string
		want string
	}{
		{name: "craftsky post", nsid: "social.craftsky.feed.post", want: "social.craftsky.feed.post"},
		{name: "craftsky like", nsid: "social.craftsky.feed.like", want: "social.craftsky.feed.like"},
		{name: "bsky follow", nsid: "app.bsky.graph.follow", want: "app.bsky.graph.follow"},
		{name: "unsupported", nsid: "com.example.raw.identifier", want: "unsupported"},
		{name: "empty", nsid: "", want: "malformed"},
		{name: "path-like raw value", nsid: "did:plc:raw/rkey123", want: "malformed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SafeNSIDLabel(tc.nsid); got != tc.want {
				t.Fatalf("SafeNSIDLabel(%q) = %q, want %q", tc.nsid, got, tc.want)
			}
		})
	}
}

func TestTapTraceControlsSampleSuccessButKeepForcedErrors(t *testing.T) {
	disabled := New(Config{Env: "test", TracingEnabled: true})
	_, disabledSpan := disabled.StartTapSpan(context.Background(), "tap.consume", false)
	if disabledSpan.Enabled() {
		t.Fatal("disabled Tap success span Enabled = true, want false")
	}

	sampledOut := New(Config{
		Env:                 "test",
		TracingEnabled:      true,
		SentryDSN:           "https://public@example.invalid/1",
		TapTracingEnabled:   true,
		TapTracesSampleRate: 0,
	})
	_, sampledOutSuccess := sampledOut.StartTapSpan(context.Background(), "tap.consume", false)
	if sampledOutSuccess.Enabled() {
		t.Fatal("sampled-out Tap success span Enabled = true, want false")
	}
	_, sampledOutError := sampledOut.StartTapSpan(context.Background(), "tap.consume", true)
	if !sampledOutError.Enabled() {
		t.Fatal("forced Tap error span Enabled = false, want true")
	}

	sampledIn := New(Config{
		Env:                 "test",
		TracingEnabled:      true,
		SentryDSN:           "https://public@example.invalid/1",
		TapTracingEnabled:   true,
		TapTracesSampleRate: 1,
	})
	_, sampledInSpan := sampledIn.StartTapSpan(context.Background(), "tap.consume", false)
	if !sampledInSpan.Enabled() {
		t.Fatal("sampled-in Tap success span Enabled = false, want true")
	}
}

func TestTapMetricsExposeIngestionAndIndexerSignals(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})

	observer.SetTapConnected(true)
	observer.ObserveTapReconnect()
	observer.ObserveTapEventReceived("record")
	observer.ObserveTapEventReceived("identity")
	observer.ObserveTapEventAcknowledged(nil)
	observer.ObserveTapEventAcknowledged(assertErr{})
	observer.ObserveTapLastEventAt(time.Now().Add(-2 * time.Second))
	observer.ObserveIndexerSkipped("did:plc:raw/rkey", "malformed")
	observer.ObserveIndexerHandled("social.craftsky.feed.post", nil, 10*time.Millisecond)
	observer.ObserveIndexerHandled("social.craftsky.feed.like", assertErr{}, 20*time.Millisecond)

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_tap_connected",
		"craftsky_appview_tap_last_event_age_seconds",
		"craftsky_appview_tap_reconnects_total",
		"craftsky_appview_tap_events_received_total",
		"craftsky_appview_tap_events_acknowledged_total",
		"craftsky_appview_tap_ack_failures_total",
		"craftsky_appview_tap_indexer_records_total",
		"craftsky_appview_tap_indexer_handling_duration_seconds",
	} {
		if !metricCallsContain(calls, want) {
			t.Fatalf("metric calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "assert err" }
