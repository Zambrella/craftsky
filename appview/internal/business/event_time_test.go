package business

import (
	"errors"
	"testing"
)

func TestEventInstantValidation(t *testing.T) {
	valid := EventWrite{
		StartsAt: "2026-09-01T10:00:00Z",
		EndsAt:   "2026-09-01T11:00:00Z",
		TimeZone: "America/New_York",
	}
	times, err := ValidateEventTimes(valid)
	if err != nil {
		t.Fatalf("ValidateEventTimes(valid): %v", err)
	}
	if !times.EndsAt.After(times.StartsAt) {
		t.Fatalf("validated times = %+v, want ordered", times)
	}
	valid.TimeZone = "UTC"
	if _, err := ValidateEventTimes(valid); err != nil {
		t.Fatalf("ValidateEventTimes(UTC): %v", err)
	}

	for name, mutate := range map[string]func(*EventWrite){
		"offset start":      func(event *EventWrite) { event.StartsAt = "2026-09-01T06:00:00-04:00" },
		"fractional start":  func(event *EventWrite) { event.StartsAt = "2026-09-01T10:00:00.1Z" },
		"lowercase z":       func(event *EventWrite) { event.StartsAt = "2026-09-01T10:00:00z" },
		"equal end":         func(event *EventWrite) { event.EndsAt = event.StartsAt },
		"end before start":  func(event *EventWrite) { event.EndsAt = "2026-09-01T09:59:59Z" },
		"missing timezone":  func(event *EventWrite) { event.TimeZone = "" },
		"invalid timezone":  func(event *EventWrite) { event.TimeZone = "Mars/Olympus" },
		"local pseudo zone": func(event *EventWrite) { event.TimeZone = "Local" },
	} {
		t.Run(name, func(t *testing.T) {
			event := valid
			mutate(&event)
			if _, err := ValidateEventTimes(event); !errors.Is(err, ErrInvalidEventTime) {
				t.Fatalf("ValidateEventTimes(%+v) error = %v, want ErrInvalidEventTime", event, err)
			}
		})
	}
}

func TestAllDayEventValidation(t *testing.T) {
	valid := []EventWrite{
		{StartsAt: "2026-02-01T05:00:00Z", EndsAt: "2026-02-02T05:00:00Z", TimeZone: "America/New_York", IsAllDay: true},
		{StartsAt: "2026-03-08T05:00:00Z", EndsAt: "2026-03-09T04:00:00Z", TimeZone: "America/New_York", IsAllDay: true},
		{StartsAt: "2026-11-01T04:00:00Z", EndsAt: "2026-11-02T05:00:00Z", TimeZone: "America/New_York", IsAllDay: true},
	}
	for _, event := range valid {
		if err := ValidateAllDayEvent(event); err != nil {
			t.Errorf("ValidateAllDayEvent(%+v): %v", event, err)
		}
	}

	invalid := []EventWrite{
		{StartsAt: "2026-02-01T05:00:01Z", EndsAt: "2026-02-02T05:00:00Z", TimeZone: "America/New_York", IsAllDay: true},
		{StartsAt: "2026-02-01T05:00:00Z", EndsAt: "2026-02-02T05:00:01Z", TimeZone: "America/New_York", IsAllDay: true},
		{StartsAt: "2026-02-01T05:00:00Z", EndsAt: "2026-02-01T05:00:00Z", TimeZone: "America/New_York", IsAllDay: true},
	}
	for _, event := range invalid {
		if err := ValidateAllDayEvent(event); !errors.Is(err, ErrInvalidEventTime) {
			t.Errorf("ValidateAllDayEvent(%+v) error = %v, want ErrInvalidEventTime", event, err)
		}
	}

	notAllDay := EventWrite{StartsAt: "2026-02-01T05:30:00Z", EndsAt: "2026-02-01T06:30:00Z", TimeZone: "America/New_York"}
	if err := ValidateAllDayEvent(notAllDay); err != nil {
		t.Fatalf("non-all-day event rejected: %v", err)
	}
}
