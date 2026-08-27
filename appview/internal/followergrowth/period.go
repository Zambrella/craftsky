package followergrowth

import (
	"errors"
	"time"
)

var ErrInvalidPeriod = errors.New("follower growth: invalid period")

type Period string

const (
	PeriodSevenDays  Period = "7d"
	PeriodThirtyDays Period = "30d"
	PeriodOneYear    Period = "1y"
)

func ParsePeriod(raw string) (Period, error) {
	period := Period(raw)
	switch period {
	case PeriodSevenDays, PeriodThirtyDays, PeriodOneYear:
		return period, nil
	default:
		return "", ErrInvalidPeriod
	}
}

type DateRange struct {
	Start time.Time
	End   time.Time
}

func (r DateRange) Days() int {
	return int(r.End.Sub(r.Start)/(24*time.Hour)) + 1
}

func (p Period) Range(now time.Time) DateRange {
	current := now.UTC()
	end := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, time.UTC)
	var start time.Time
	switch p {
	case PeriodSevenDays:
		start = end.AddDate(0, 0, -6)
	case PeriodThirtyDays:
		start = end.AddDate(0, 0, -29)
	case PeriodOneYear:
		if end.Month() == time.February && end.Day() == 29 {
			start = time.Date(end.Year()-1, time.February, 28, 0, 0, 0, 0, time.UTC)
		} else {
			start = time.Date(end.Year()-1, end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
		}
	}
	return DateRange{Start: start, End: end}
}
