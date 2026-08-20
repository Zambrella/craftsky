package observability

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestPushTelemetryUsesOnlyBoundedLabels(t *testing.T) {
	r := NewInMemoryMetricRecorder()
	o := New(Config{SentryDSN: "test", MetricsEnabled: true, MetricRecorder: r})
	o.ObserveNotificationDecision("like", "created")
	o.ObservePushDelivery("ios", "retryable")
	o.ObservePushQueue(3, 2*time.Second)
	for _, call := range r.Calls() {
		for _, value := range call.Attributes {
			if value == "secret-token" || value == "did:plc:actor" {
				t.Fatalf("sensitive metric attribute: %+v", call)
			}
		}
	}
}

func TestPushOperationTelemetryBoundsEveryLabelAndRecordsLatency(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{
		SentryDSN: "test", MetricsEnabled: true, MetricRecorder: recorder,
	})
	observer.ObservePushOperation(
		"send", "ios", "unique_event", "success", 25*time.Millisecond, 1,
	)
	observer.ObservePushOperation(
		"SENTINEL_STAGE", "secret-token", "notification-id",
		"raw provider error for did:plc:actor", -time.Second, -5,
	)

	calls := recorder.Calls()
	if len(calls) != 4 {
		t.Fatalf("metric calls = %+v, want counter and duration per operation", calls)
	}
	for index, call := range calls {
		keys := make([]string, 0, len(call.Attributes))
		for key, value := range call.Attributes {
			keys = append(keys, key)
			for _, forbidden := range []string{
				"SENTINEL", "secret-token", "notification-id",
				"provider error", "did:plc:",
			} {
				if strings.Contains(value, forbidden) {
					t.Fatalf("metric attribute leaked %q: %+v", forbidden, call)
				}
			}
		}
		slices.Sort(keys)
		if !slices.Equal(keys, []string{"outcome", "platform", "semantics", "stage"}) {
			t.Fatalf("metric attributes = %+v", call)
		}
		if call.Value < 0 {
			t.Fatalf("negative metric value at %d: %+v", index, call)
		}
	}
	if got := calls[0].Attributes; got["stage"] != "send" ||
		got["platform"] != "ios" || got["semantics"] != "unique_event" ||
		got["outcome"] != "success" {
		t.Fatalf("allowed labels = %+v", got)
	}
	if got := calls[2].Attributes; got["stage"] != "other" ||
		got["platform"] != "other" || got["semantics"] != "other" ||
		got["outcome"] != "other" {
		t.Fatalf("unbounded labels = %+v, want all other", got)
	}
}
