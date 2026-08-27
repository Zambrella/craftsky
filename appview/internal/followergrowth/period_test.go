package followergrowth

import (
	"testing"
	"time"
)

func TestPeriodRange(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		now       time.Time
		wantStart time.Time
		wantEnd   time.Time
		wantDays  int
	}{
		{
			name:      "seven days",
			raw:       "7d",
			now:       time.Date(2026, time.August, 25, 18, 30, 0, 0, time.UTC),
			wantStart: time.Date(2026, time.August, 19, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC),
			wantDays:  7,
		},
		{
			name:      "thirty days crosses year",
			raw:       "30d",
			now:       time.Date(2026, time.January, 10, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2025, time.December, 12, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, time.January, 10, 0, 0, 0, 0, time.UTC),
			wantDays:  30,
		},
		{
			name:      "one year ordinary anniversary",
			raw:       "1y",
			now:       time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2025, time.August, 25, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC),
			wantDays:  366,
		},
		{
			name:      "one year leap day clamps to February 28",
			raw:       "1y",
			now:       time.Date(2028, time.February, 29, 12, 0, 0, 0, time.UTC),
			wantStart: time.Date(2027, time.February, 28, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC),
			wantDays:  367,
		},
		{
			name: "uses current UTC date instead of local date",
			raw:  "7d",
			now: time.Date(2026, time.August, 25, 0, 30, 0, 0,
				time.FixedZone("UTC+14", 14*60*60)),
			wantStart: time.Date(2026, time.August, 18, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC),
			wantDays:  7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period, err := ParsePeriod(tt.raw)
			if err != nil {
				t.Fatalf("ParsePeriod(%q): %v", tt.raw, err)
			}
			got := period.Range(tt.now)
			if !got.Start.Equal(tt.wantStart) || !got.End.Equal(tt.wantEnd) {
				t.Fatalf("range = %s through %s, want %s through %s", got.Start, got.End, tt.wantStart, tt.wantEnd)
			}
			if got.Days() != tt.wantDays {
				t.Fatalf("range days = %d, want %d", got.Days(), tt.wantDays)
			}
			if got.Days() > 367 {
				t.Fatalf("range days = %d, exceeds bounded maximum", got.Days())
			}
		})
	}
}

func TestParsePeriodRejectsUnsupportedValues(t *testing.T) {
	for _, raw := range []string{"", "7", "365d", "1Y", " 7d"} {
		if _, err := ParsePeriod(raw); err == nil {
			t.Errorf("ParsePeriod(%q) succeeded", raw)
		}
	}
}
