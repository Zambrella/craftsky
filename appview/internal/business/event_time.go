package business

import (
	"errors"
	"time"
)

var ErrInvalidEventTime = errors.New("business: invalid event time")

type EventTimes struct {
	StartsAt time.Time
	EndsAt   time.Time
	Location *time.Location
}

func ValidateEventTimes(event EventWrite) (EventTimes, error) {
	startsAt, err := parseCanonicalEventInstant(event.StartsAt)
	if err != nil {
		return EventTimes{}, err
	}
	endsAt, err := parseCanonicalEventInstant(event.EndsAt)
	if err != nil || !endsAt.After(startsAt) {
		return EventTimes{}, ErrInvalidEventTime
	}
	if event.TimeZone == "" || event.TimeZone == "Local" {
		return EventTimes{}, ErrInvalidEventTime
	}
	location, err := time.LoadLocation(event.TimeZone)
	if err != nil {
		return EventTimes{}, ErrInvalidEventTime
	}
	return EventTimes{StartsAt: startsAt, EndsAt: endsAt, Location: location}, nil
}

func ValidateAllDayEvent(event EventWrite) error {
	times, err := ValidateEventTimes(event)
	if err != nil {
		return err
	}
	if !event.IsAllDay {
		return nil
	}
	if !isLocalMidnight(times.StartsAt, times.Location) || !isLocalMidnight(times.EndsAt, times.Location) {
		return ErrInvalidEventTime
	}
	return nil
}

func isLocalMidnight(instant time.Time, location *time.Location) bool {
	local := instant.In(location)
	return local.Hour() == 0 && local.Minute() == 0 && local.Second() == 0 && local.Nanosecond() == 0
}

func parseCanonicalEventInstant(raw string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil || parsed.Nanosecond() != 0 || parsed.UTC().Format(time.RFC3339) != raw {
		return time.Time{}, ErrInvalidEventTime
	}
	return parsed, nil
}
