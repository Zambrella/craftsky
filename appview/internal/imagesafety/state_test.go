package imagesafety

import "testing"

func TestScanState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		state           State
		valid           bool
		terminal        bool
		displayEligible bool
	}{
		{name: "pending", state: StatePending, valid: true},
		{name: "clear", state: StateClear, valid: true, terminal: true, displayEligible: true},
		{name: "match", state: StateMatch, valid: true, terminal: true},
		{name: "unavailable", state: StateUnavailable, valid: true},
		{name: "error", state: StateError, valid: true},
		{name: "unknown", state: State("unknown")},
		{name: "empty", state: State("")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.state.Valid(); got != test.valid {
				t.Errorf("State(%q).Valid() = %t, want %t", test.state, got, test.valid)
			}
			if got := test.state.Terminal(); got != test.terminal {
				t.Errorf("State(%q).Terminal() = %t, want %t", test.state, got, test.terminal)
			}
			if got := test.state.DisplayEligible(); got != test.displayEligible {
				t.Errorf("State(%q).DisplayEligible() = %t, want %t", test.state, got, test.displayEligible)
			}
		})
	}
}
