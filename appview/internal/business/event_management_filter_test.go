package business

import (
	"testing"
	"time"
)

func TestClassifyOwnerEventAtTraversalCutoff(t *testing.T) {
	cutoff := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name   string
		status string
		endsAt time.Time
		want   OwnerEventFilter
	}{
		{name: "scheduled future", status: "scheduled", endsAt: cutoff.Add(time.Hour), want: OwnerEventUpcoming},
		{name: "default status future", endsAt: cutoff.Add(time.Hour), want: OwnerEventUpcoming},
		{name: "scheduled equal cutoff", status: "scheduled", endsAt: cutoff, want: OwnerEventHistory},
		{name: "scheduled ended", status: "scheduled", endsAt: cutoff.Add(-time.Second), want: OwnerEventHistory},
		{name: "cancelled future", status: "cancelled", endsAt: cutoff.Add(time.Hour), want: OwnerEventHistory},
		{name: "postponed future", status: "postponed", endsAt: cutoff.Add(time.Hour), want: OwnerEventHistory},
		{name: "unknown future", status: "rescheduled-externally", endsAt: cutoff.Add(time.Hour), want: OwnerEventHistory},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ClassifyOwnerEvent(test.status, test.endsAt, cutoff); got != test.want {
				t.Fatalf("ClassifyOwnerEvent(%q, %s, %s) = %q, want %q", test.status, test.endsAt, cutoff, got, test.want)
			}
		})
	}
}
