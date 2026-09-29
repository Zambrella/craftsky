package imagesafety

import "testing"

func TestParentDisplayEligible(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		states []State
		want   bool
	}{
		{name: "no images", want: true},
		{name: "one clear", states: []State{StateClear}, want: true},
		{name: "all clear", states: []State{StateClear, StateClear}, want: true},
		{name: "pending", states: []State{StatePending}},
		{name: "match", states: []State{StateMatch}},
		{name: "unavailable", states: []State{StateUnavailable}},
		{name: "error", states: []State{StateError}},
		{name: "mixed", states: []State{StateClear, StatePending}},
		{name: "unknown", states: []State{StateClear, State("unknown")}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := ParentDisplayEligible(test.states); got != test.want {
				t.Errorf("ParentDisplayEligible(%v) = %t, want %t", test.states, got, test.want)
			}
		})
	}
}
