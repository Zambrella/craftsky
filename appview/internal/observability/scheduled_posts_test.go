package observability

import (
	"strings"
	"testing"
	"time"
)

func TestScheduledPostOperationalSignalsAreCompleteAndContentFree(t *testing.T) {
	t.Parallel()

	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{Env: "test", MetricRecorder: recorder})
	observer.ObserveScheduledQueue("scheduled", 3, 2, 1, 90*time.Second)
	observer.ObserveScheduledOperation("claim", "success", "none", 5*time.Millisecond)
	observer.ObserveScheduledOperation("needs_attention", "failure", "auth_unavailable", time.Second)
	observer.ObserveScheduledOperation("recover", "success", "lease_expired", time.Millisecond)
	observer.ObserveScheduledOperation("stale_worker", "stale", "stale_worker", time.Millisecond)
	observer.ObserveScheduledPublication(2, 90*time.Second, 3*time.Second)
	observer.ObserveScheduledCleanupQueue(4, 20*time.Minute)

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_scheduled_posts_status",
		"craftsky_appview_scheduled_posts_due",
		"craftsky_appview_scheduled_posts_overdue",
		"craftsky_appview_scheduled_posts_oldest_due_age_seconds",
		"craftsky_appview_scheduled_posts_operations_total",
		"craftsky_appview_scheduled_posts_operation_duration_seconds",
		"craftsky_appview_scheduled_posts_publication_start_latency_seconds",
		"craftsky_appview_scheduled_posts_publication_duration_seconds",
		"craftsky_appview_scheduled_posts_cleanup_pending",
		"craftsky_appview_scheduled_posts_cleanup_oldest_age_seconds",
	} {
		if !metricCallsContain(calls, want) {
			t.Fatalf("metric calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("invalid scheduled metric: %v; call=%#v", err, call)
		}
		for key, value := range call.Attributes {
			joined := key + "=" + value
			if strings.Contains(joined, "did:") || strings.Contains(joined, "secret") {
				t.Fatalf("scheduled metric leaked private value: %#v", call)
			}
		}
	}
}
