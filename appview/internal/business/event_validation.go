package business

import (
	"errors"
	"time"
)

var ErrInvalidEvent = errors.New("business: invalid event")

var (
	eventRoleCatalog   = []string{"organizer", "instructor", "vendor", "exhibitor", "speaker", "demonstrator"}
	eventModeCatalog   = []string{"in-person", "online", "hybrid"}
	eventStatusCatalog = []string{"scheduled", "cancelled", "postponed"}
)

type EventWrite struct {
	Name            string
	StartsAt        string
	EndsAt          string
	Roles           []string
	Mode            string
	Status          string
	TimeZone        string
	IsAllDay        bool
	Summary         string
	VenueName       string
	EventURI        string
	RegistrationURI string
	Image           *Image
}

type IndependentEventDefaults struct {
	Status   string
	Mode     *string
	TimeZone *string
	IsAllDay bool
}

type EventDestinations struct {
	EventURI        string `json:"eventUri,omitempty"`
	RegistrationURI string `json:"registrationUri,omitempty"`
}

func EventRoleCatalog() []string {
	return append([]string(nil), eventRoleCatalog...)
}

func EventModeCatalog() []string {
	return append([]string(nil), eventModeCatalog...)
}

func EventStatusCatalog() []string {
	return append([]string(nil), eventStatusCatalog...)
}

func ValidateEventCatalogs(event EventWrite) error {
	if !catalogContains(eventModeCatalog, event.Mode) || !catalogContains(eventStatusCatalog, event.Status) || event.TimeZone == "" {
		return ErrInvalidEvent
	}
	if len(event.Roles) == 0 || len(event.Roles) > 4 {
		return ErrInvalidEvent
	}
	seen := make(map[string]struct{}, len(event.Roles))
	for _, role := range event.Roles {
		if _, duplicate := seen[role]; duplicate || !catalogContains(eventRoleCatalog, role) {
			return ErrInvalidEvent
		}
		seen[role] = struct{}{}
	}
	return nil
}

func ValidateEventTemporalPolicy(event EventWrite, now time.Time, existing bool) error {
	times, err := ValidateEventTimes(event)
	if err != nil || times.EndsAt.Sub(times.StartsAt) > 31*24*time.Hour {
		return ErrInvalidEvent
	}
	if !existing && !times.EndsAt.After(now) {
		return ErrInvalidEvent
	}
	return nil
}

func ValidateEventMedia(event EventWrite) error {
	if event.EventURI != "" && ValidateWebDestination(event.EventURI) != nil {
		return ErrInvalidEvent
	}
	if event.RegistrationURI != "" && ValidateWebDestination(event.RegistrationURI) != nil {
		return ErrInvalidEvent
	}
	if event.EventURI != "" && event.EventURI == event.RegistrationURI {
		return ErrInvalidEvent
	}
	if ValidateProductImage(event.Image, false) != nil {
		return ErrInvalidEvent
	}
	return nil
}

func HydrateIndependentEventDestinations(eventURI, registrationURI string) EventDestinations {
	var hydrated EventDestinations
	if ValidateWebDestination(eventURI) == nil {
		hydrated.EventURI = eventURI
	}
	if ValidateWebDestination(registrationURI) == nil && registrationURI != hydrated.EventURI {
		hydrated.RegistrationURI = registrationURI
	}
	return hydrated
}

func HydrateIndependentEventDefaults(status, mode, timeZone *string, isAllDay *bool) IndependentEventDefaults {
	hydrated := IndependentEventDefaults{Status: "scheduled", Mode: mode, TimeZone: timeZone}
	if status != nil {
		hydrated.Status = *status
	}
	if isAllDay != nil {
		hydrated.IsAllDay = *isAllDay
	}
	return hydrated
}

func ClassifyIndependentRoles(roles []string) []OpenValue {
	classified := make([]OpenValue, len(roles))
	for i, role := range roles {
		classified[i] = ClassifyOpenValue(role, eventRoleCatalog)
	}
	return classified
}
