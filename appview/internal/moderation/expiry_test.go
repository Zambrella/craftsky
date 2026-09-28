package moderation

import (
	"testing"
	"time"
)

func TestStrikeDeadlineUsesClampedUTCCalendarYear(t *testing.T) {
	tests := []struct {
		name   string
		issued time.Time
		want   time.Time
	}{
		{
			name:   "ordinary date",
			issued: time.Date(2026, time.September, 10, 14, 15, 16, 123000000, time.UTC),
			want:   time.Date(2027, time.September, 10, 14, 15, 16, 123000000, time.UTC),
		},
		{
			name:   "leap day clamps",
			issued: time.Date(2024, time.February, 29, 8, 30, 0, 0, time.UTC),
			want:   time.Date(2025, time.February, 28, 8, 30, 0, 0, time.UTC),
		},
		{
			name:   "source offset normalizes first",
			issued: time.Date(2026, time.January, 31, 23, 30, 0, 0, time.FixedZone("minus five", -5*60*60)),
			want:   time.Date(2027, time.February, 1, 4, 30, 0, 0, time.UTC),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := StrikeDeadline(test.issued); !got.Equal(test.want) || got.Location() != time.UTC {
				t.Fatalf("StrikeDeadline(%s) = %s (%s), want %s UTC", test.issued, got, got.Location(), test.want)
			}
		})
	}
}

func TestStrikeRemainsProjectedActiveWhenDueUntilProcessed(t *testing.T) {
	due := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	strike := StrikeProjection{CaseID: "case-one", DueAt: due}
	if !strike.Due(due) || !strike.Active() {
		t.Fatal("due but unprocessed strike must remain projected active")
	}
	processed := due.Add(time.Minute)
	strike.ExpiredAt = &processed
	if strike.Active() {
		t.Fatal("processed strike remained active")
	}
}
