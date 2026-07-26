package observability

import (
	"testing"
	"time"
)

func TestRelationshipTelemetryUsesOnlyBoundedLabels(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})

	observer.ObserveRelationship("block", "success", 25*time.Millisecond)
	observer.ObserveRelationshipOutcome("authorization_like", "policy", "denied", "policy", 5*time.Millisecond)
	observer.ObserveRelationship("did:plc:private-target", "secret-result", time.Second)

	calls := recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(calls))
	}
	if calls[0].Name != "craftsky_appview_relationship_operation_duration_seconds" ||
		calls[0].Attributes["operation"] != "block" || calls[0].Attributes["result"] != "success" {
		t.Fatalf("bounded relationship call = %#v", calls[0])
	}
	if calls[1].Attributes["operation"] != "authorization_like" ||
		calls[1].Attributes["stage"] != "policy" ||
		calls[1].Attributes["result"] != "denied" ||
		calls[1].Attributes["error_class"] != "policy" {
		t.Fatalf("detailed relationship call = %#v", calls[1])
	}
	if calls[2].Attributes["operation"] != "unknown" || calls[2].Attributes["result"] != "unknown" {
		t.Fatalf("unbounded relationship values escaped: %#v", calls[2])
	}
	for _, call := range calls {
		if err := ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
}
