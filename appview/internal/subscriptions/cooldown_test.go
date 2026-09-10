package subscriptions

import (
	"testing"
	"time"
)

func TestCanChangeAssignmentTargetUsesSevenDayBoundary(t *testing.T) {
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)
	justInside := sevenDaysAgo.Add(time.Nanosecond)
	justOutside := sevenDaysAgo.Add(-time.Nanosecond)

	for _, test := range []struct {
		name       string
		lastChange *time.Time
		want       bool
	}{
		{name: "initial assignment", want: true},
		{name: "inside boundary", lastChange: &justInside, want: false},
		{name: "exact boundary", lastChange: &sevenDaysAgo, want: true},
		{name: "outside boundary", lastChange: &justOutside, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := CanChangeAssignmentTarget(test.lastChange, now); got != test.want {
				t.Fatalf("CanChangeAssignmentTarget(%v, %v) = %t, want %t", test.lastChange, now, got, test.want)
			}
		})
	}
}
