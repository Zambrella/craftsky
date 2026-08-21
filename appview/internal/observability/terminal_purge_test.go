package observability

import (
	"testing"

	"social.craftsky/appview/internal/ownerlifecycle"
)

func TestTerminalPurgeMetricsContainOnlyBoundedAggregateLabels(t *testing.T) {
	t.Parallel()
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	observer.ObserveTerminalPurge(ownerlifecycle.TerminalPurgeObservation{
		Operation: "component", Result: "success", ErrorCategory: "none",
		Component: "scheduled_posts", DIDRole: "owner", Claims: 2,
		RowsAffected: 7, Complete: true,
	})
	calls := recorder.Calls()
	if len(calls) != 3 {
		t.Fatalf("metric calls = %d, want 3", len(calls))
	}
	for _, call := range calls {
		for key, value := range call.Attributes {
			if key == "owner" || key == "did" || value == "did:plc:alice" {
				t.Fatalf("terminal purge metric leaked owner identity: %+v", call)
			}
		}
	}
}

func TestTerminalPurgeBacklogRecordsGauge(t *testing.T) {
	t.Parallel()
	recorder := NewInMemoryMetricRecorder()
	observer := New(Config{MetricRecorder: recorder})
	observer.ObserveTerminalPurge(ownerlifecycle.TerminalPurgeObservation{
		Operation: "backlog", Result: "success", Remaining: 11,
	})
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_terminal_purge_remaining" && call.Value == 11 {
			return
		}
	}
	t.Fatal("terminal purge backlog gauge was not recorded")
}
