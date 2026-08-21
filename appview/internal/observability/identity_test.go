package observability

import (
	"testing"
	"time"
)

func TestIdentityTelemetryUsesBoundedNonIdentityLabels(t *testing.T) {
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	observer.ObserveIdentityResolution("authoritative", "handle_to_did", "success", 25*time.Millisecond)
	observer.ObserveIdentityCache("refresh_retry", 25*time.Hour)

	calls := recorder.Calls()
	if len(calls) != 4 {
		t.Fatalf("identity metric calls=%+v, want two lookup and two cache metrics", calls)
	}
	for _, call := range calls {
		for key, value := range call.Attributes {
			if key == "did" || key == "handle" || value == "did:plc:secret" || value == "alice.example" {
				t.Fatalf("identity metric leaked identity data: %+v", call)
			}
		}
	}
}
