package followergrowth

import (
	"testing"
	"time"
)

func TestBuildSeriesPreservesSparseHistory(t *testing.T) {
	rangeStart := growthDate(2026, time.August, 19)
	rangeEnd := growthDate(2026, time.August, 25)
	dateRange := DateRange{Start: rangeStart, End: rangeEnd}

	t.Run("no history returns every date without values or metadata", func(t *testing.T) {
		got := BuildSeries(History{}, dateRange)
		if got.AvailableFrom != nil || got.LatestSnapshotDate != nil || got.LatestCapturedAt != nil || got.LatestFollowerCount != nil {
			t.Fatalf("no-history metadata = %+v, want all nil", got)
		}
		assertSeriesCounts(t, got.Points, []countExpectation{
			{rangeStart, nil},
			{growthDate(2026, time.August, 20), nil},
			{growthDate(2026, time.August, 21), nil},
			{growthDate(2026, time.August, 22), nil},
			{growthDate(2026, time.August, 23), nil},
			{growthDate(2026, time.August, 24), nil},
			{rangeEnd, nil},
		})
	})

	t.Run("global availability remains older than leading and interior gaps", func(t *testing.T) {
		availableFrom := growthDate(2026, time.July, 1)
		latest := Snapshot{
			Date:          growthDate(2026, time.August, 24),
			FollowerCount: 11,
			CapturedAt:    time.Date(2026, time.August, 24, 0, 0, 2, 0, time.UTC),
		}
		got := BuildSeries(History{
			AvailableFrom: &availableFrom,
			Latest:        &latest,
			Snapshots: []Snapshot{
				{Date: growthDate(2026, time.August, 20), FollowerCount: 10},
				{Date: growthDate(2026, time.August, 22), FollowerCount: 12},
				latest,
			},
		}, dateRange)

		assertTimePointer(t, "availableFrom", got.AvailableFrom, availableFrom)
		assertTimePointer(t, "latestSnapshotDate", got.LatestSnapshotDate, latest.Date)
		assertTimePointer(t, "latestCapturedAt", got.LatestCapturedAt, latest.CapturedAt)
		if got.LatestFollowerCount == nil || *got.LatestFollowerCount != 11 {
			t.Fatalf("latest follower count = %v, want 11", got.LatestFollowerCount)
		}
		assertSeriesCounts(t, got.Points, []countExpectation{
			{rangeStart, nil},
			{growthDate(2026, time.August, 20), countPointer(10)},
			{growthDate(2026, time.August, 21), nil},
			{growthDate(2026, time.August, 22), countPointer(12)},
			{growthDate(2026, time.August, 23), nil},
			{growthDate(2026, time.August, 24), countPointer(11)},
			{rangeEnd, nil},
		})
	})
}

func TestBuildSeriesCalculatesSelectedRangeNetChange(t *testing.T) {
	dateRange := DateRange{
		Start: growthDate(2026, time.August, 19),
		End:   growthDate(2026, time.August, 25),
	}
	availableBeforeRange := growthDate(2026, time.July, 1)
	tests := []struct {
		name      string
		snapshots []Snapshot
		want      *int64
	}{
		{name: "no observations", want: nil},
		{
			name:      "one observation",
			snapshots: []Snapshot{{Date: growthDate(2026, time.August, 22), FollowerCount: 10}},
			want:      nil,
		},
		{
			name: "increase across gaps",
			snapshots: []Snapshot{
				{Date: growthDate(2026, time.August, 20), FollowerCount: 10},
				{Date: growthDate(2026, time.August, 22), FollowerCount: 50},
				{Date: growthDate(2026, time.August, 24), FollowerCount: 13},
			},
			want: countPointer(3),
		},
		{
			name: "decrease",
			snapshots: []Snapshot{
				{Date: growthDate(2026, time.August, 19), FollowerCount: 12},
				{Date: growthDate(2026, time.August, 25), FollowerCount: 7},
			},
			want: countPointer(-5),
		},
		{
			name: "equal counts",
			snapshots: []Snapshot{
				{Date: growthDate(2026, time.August, 19), FollowerCount: 7},
				{Date: growthDate(2026, time.August, 25), FollowerCount: 7},
			},
			want: countPointer(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSeries(History{
				AvailableFrom: &availableBeforeRange,
				Snapshots:     tt.snapshots,
			}, dateRange).NetChange
			switch {
			case got == nil && tt.want == nil:
			case got == nil || tt.want == nil:
				t.Fatalf("net change = %v, want %v", got, tt.want)
			case *got != *tt.want:
				t.Fatalf("net change = %d, want %d", *got, *tt.want)
			}
		})
	}
}

type countExpectation struct {
	date  time.Time
	count *int64
}

func assertSeriesCounts(t *testing.T, got []Point, want []countExpectation) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("point count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if !got[i].Date.Equal(want[i].date) {
			t.Errorf("point %d date = %s, want %s", i, got[i].Date, want[i].date)
		}
		switch {
		case got[i].Count == nil && want[i].count == nil:
		case got[i].Count == nil || want[i].count == nil:
			t.Errorf("point %d count = %v, want %v", i, got[i].Count, want[i].count)
		case *got[i].Count != *want[i].count:
			t.Errorf("point %d count = %d, want %d", i, *got[i].Count, *want[i].count)
		}
	}
}

func assertTimePointer(t *testing.T, name string, got *time.Time, want time.Time) {
	t.Helper()
	if got == nil || !got.Equal(want) {
		t.Fatalf("%s = %v, want %s", name, got, want)
	}
}

func countPointer(value int64) *int64 {
	return &value
}

func growthDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
