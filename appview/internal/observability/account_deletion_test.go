package observability

import (
	"context"
	"testing"
	"time"
)

func TestAccountDeletionMetricsUseOnlyCoarseBoundedAttributes(t *testing.T) {
	t.Parallel()

	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	observer.ObserveAccountDeletion(
		context.Background(),
		"automaticRetry",
		"removingCraftskyRecords",
		"",
		"pds",
		2*time.Second,
	)

	calls := recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("account deletion metric calls=%d, want counter and duration", len(calls))
	}
	for _, call := range calls {
		if call.Attributes["event"] != "automaticRetry" ||
			call.Attributes["phase"] != "removingCraftskyRecords" ||
			call.Attributes["error_category"] != "pds" {
			t.Fatalf("account deletion metric attributes=%v", call.Attributes)
		}
		for _, prohibited := range []string{"did", "handle", "uri", "token", "url"} {
			if _, exists := call.Attributes[prohibited]; exists {
				t.Fatalf("account deletion metric exposes prohibited attribute %q", prohibited)
			}
		}
	}
}
