package observability

import (
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
