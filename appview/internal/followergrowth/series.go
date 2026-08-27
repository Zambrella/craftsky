package followergrowth

import "time"

type Snapshot struct {
	Date          time.Time
	FollowerCount int64
	CapturedAt    time.Time
}

type History struct {
	AvailableFrom *time.Time
	Latest        *Snapshot
	Snapshots     []Snapshot
}

type Point struct {
	Date  time.Time
	Count *int64
}

type Growth struct {
	Range               DateRange
	AvailableFrom       *time.Time
	LatestSnapshotDate  *time.Time
	LatestCapturedAt    *time.Time
	LatestFollowerCount *int64
	NetChange           *int64
	Points              []Point
}

func BuildSeries(history History, dateRange DateRange) Growth {
	growth := Growth{Range: dateRange}
	if history.AvailableFrom != nil {
		availableFrom := *history.AvailableFrom
		growth.AvailableFrom = &availableFrom
	}
	if history.Latest != nil {
		latestDate := history.Latest.Date
		latestCapturedAt := history.Latest.CapturedAt
		latestCount := history.Latest.FollowerCount
		growth.LatestSnapshotDate = &latestDate
		growth.LatestCapturedAt = &latestCapturedAt
		growth.LatestFollowerCount = &latestCount
	}

	counts := make(map[string]int64, len(history.Snapshots))
	for _, snapshot := range history.Snapshots {
		counts[dateKey(snapshot.Date)] = snapshot.FollowerCount
	}
	growth.Points = make([]Point, 0, dateRange.Days())
	var (
		firstCount       int64
		lastCount        int64
		observationCount int
	)
	for date := dateRange.Start; !date.After(dateRange.End); date = date.AddDate(0, 0, 1) {
		point := Point{Date: date}
		if count, ok := counts[dateKey(date)]; ok {
			value := count
			point.Count = &value
			if observationCount == 0 {
				firstCount = count
			}
			lastCount = count
			observationCount++
		}
		growth.Points = append(growth.Points, point)
	}
	if observationCount >= 2 {
		netChange := lastCount - firstCount
		growth.NetChange = &netChange
	}
	return growth
}

func dateKey(date time.Time) string {
	return date.UTC().Format(time.DateOnly)
}
