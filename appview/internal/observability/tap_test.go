package observability

import (
	"net/http/httptest"
	"strings"
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

func TestTapMetricsExposeIngestionAndIndexerSignals(t *testing.T) {
	observer := New(Config{Env: "test"})

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

	rec := httptest.NewRecorder()
	observer.MetricsHandler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	body := rec.Body.String()

	for _, want := range []string{
		"# HELP craftsky_appview_tap_connected Whether the Tap consumer is currently connected.",
		`craftsky_appview_tap_connected 1`,
		`craftsky_appview_tap_reconnects_total 1`,
		`craftsky_appview_tap_events_received_total{type="record"} 1`,
		`craftsky_appview_tap_events_received_total{type="identity"} 1`,
		`craftsky_appview_tap_events_acknowledged_total 1`,
		`craftsky_appview_tap_ack_failures_total 1`,
		`craftsky_appview_tap_indexer_records_total{nsid="malformed",reason="malformed",result="skipped"} 1`,
		`craftsky_appview_tap_indexer_records_total{nsid="social.craftsky.feed.post",reason="none",result="indexed"} 1`,
		`craftsky_appview_tap_indexer_records_total{nsid="social.craftsky.feed.like",reason="indexer_error",result="error"} 1`,
		`craftsky_appview_tap_indexer_handling_duration_seconds_bucket{nsid="social.craftsky.feed.post",result="indexed",le="`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"did:plc:raw", "rkey"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("metrics output contains forbidden raw value %q:\n%s", forbidden, body)
		}
	}
}

type assertErr struct{}

func (assertErr) Error() string { return "assert err" }
