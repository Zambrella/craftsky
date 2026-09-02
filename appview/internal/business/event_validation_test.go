package business

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestEventCatalogValidation(t *testing.T) {
	if got, want := EventRoleCatalog(), []string{"organizer", "instructor", "vendor", "exhibitor", "speaker", "demonstrator"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EventRoleCatalog() = %v, want %v", got, want)
	}
	if got, want := EventModeCatalog(), []string{"in-person", "online", "hybrid"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EventModeCatalog() = %v, want %v", got, want)
	}
	if got, want := EventStatusCatalog(), []string{"scheduled", "cancelled", "postponed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("EventStatusCatalog() = %v, want %v", got, want)
	}

	valid := EventWrite{Mode: "online", Status: "scheduled", TimeZone: "UTC", Roles: []string{"vendor", "organizer"}}
	if err := ValidateEventCatalogs(valid); err != nil {
		t.Fatalf("ValidateEventCatalogs(valid): %v", err)
	}
	for name, event := range map[string]EventWrite{
		"missing mode":     {Status: "scheduled", TimeZone: "UTC", Roles: []string{"vendor"}},
		"missing status":   {Mode: "online", TimeZone: "UTC", Roles: []string{"vendor"}},
		"missing timezone": {Mode: "online", Status: "scheduled", Roles: []string{"vendor"}},
		"empty roles":      {Mode: "online", Status: "scheduled", TimeZone: "UTC"},
		"unknown role":     {Mode: "online", Status: "scheduled", TimeZone: "UTC", Roles: []string{"attendee"}},
		"duplicate role":   {Mode: "online", Status: "scheduled", TimeZone: "UTC", Roles: []string{"vendor", "vendor"}},
		"five roles":       {Mode: "online", Status: "scheduled", TimeZone: "UTC", Roles: []string{"organizer", "instructor", "vendor", "exhibitor", "speaker"}},
		"unknown mode":     {Mode: "virtual", Status: "scheduled", TimeZone: "UTC", Roles: []string{"vendor"}},
		"unknown status":   {Mode: "online", Status: "completed", TimeZone: "UTC", Roles: []string{"vendor"}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateEventCatalogs(event); !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("ValidateEventCatalogs(%+v) error = %v, want ErrInvalidEvent", event, err)
			}
		})
	}

	defaults := HydrateIndependentEventDefaults(nil, nil, nil, nil)
	if defaults.Status != "scheduled" || defaults.Mode != nil || defaults.TimeZone != nil || defaults.IsAllDay {
		t.Fatalf("independent defaults = %+v", defaults)
	}
	roles := ClassifyIndependentRoles([]string{"organizer", "future-role"})
	if len(roles) != 2 || !roles[0].Known || roles[1].Known || roles[1].Value != "future-role" {
		t.Fatalf("independent roles = %+v", roles)
	}
}

func TestEventTemporalPolicy(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		event    EventWrite
		existing bool
		wantErr  bool
	}{
		{
			name:  "future create at 31 days",
			event: EventWrite{StartsAt: "2026-09-02T12:00:00Z", EndsAt: "2026-10-03T12:00:00Z", TimeZone: "UTC"},
		},
		{
			name:  "ongoing create",
			event: EventWrite{StartsAt: "2026-09-01T11:00:00Z", EndsAt: "2026-09-01T13:00:00Z", TimeZone: "UTC"},
		},
		{
			name:    "over 31 days",
			event:   EventWrite{StartsAt: "2026-09-02T12:00:00Z", EndsAt: "2026-10-03T12:00:01Z", TimeZone: "UTC"},
			wantErr: true,
		},
		{
			name:    "ended create",
			event:   EventWrite{StartsAt: "2026-08-31T10:00:00Z", EndsAt: "2026-09-01T11:59:59Z", TimeZone: "UTC"},
			wantErr: true,
		},
		{
			name:     "past update",
			event:    EventWrite{StartsAt: "2026-08-31T10:00:00Z", EndsAt: "2026-08-31T11:00:00Z", TimeZone: "UTC"},
			existing: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEventTemporalPolicy(tt.event, now, tt.existing)
			if tt.wantErr && !errors.Is(err, ErrInvalidEvent) {
				t.Fatalf("error = %v, want ErrInvalidEvent", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEventMediaValidation(t *testing.T) {
	valid := EventWrite{
		EventURI:        "https://events.example/details",
		RegistrationURI: "https://tickets.example/register",
		Image:           &Image{MIMEType: "image/webp", Size: MaxImageBytes},
	}
	if err := ValidateEventMedia(valid); err != nil {
		t.Fatalf("ValidateEventMedia(valid): %v", err)
	}
	if err := ValidateEventMedia(EventWrite{}); err != nil {
		t.Fatalf("ValidateEventMedia(empty): %v", err)
	}

	invalid := []EventWrite{
		{EventURI: "http://events.example/details"},
		{RegistrationURI: "https://user@tickets.example/register"},
		{EventURI: "https://events.example/same", RegistrationURI: "https://events.example/same"},
		{Image: &Image{MIMEType: "image/gif", Size: 1}},
	}
	for _, event := range invalid {
		if err := ValidateEventMedia(event); !errors.Is(err, ErrInvalidEvent) {
			t.Errorf("ValidateEventMedia(%+v) error = %v, want ErrInvalidEvent", event, err)
		}
	}

	hydrated := HydrateIndependentEventDestinations(
		"https://events.example/same",
		"https://events.example/same",
	)
	if hydrated.EventURI != "https://events.example/same" || hydrated.RegistrationURI != "" {
		t.Fatalf("duplicate independent links hydrated as %+v", hydrated)
	}
	unsafe := HydrateIndependentEventDestinations("http://events.example", "https://tickets.example/register")
	if unsafe.EventURI != "" || unsafe.RegistrationURI != "https://tickets.example/register" {
		t.Fatalf("unsafe independent links hydrated as %+v", unsafe)
	}

	eventType := reflect.TypeOf(EventWrite{})
	if _, ok := eventType.FieldByName("OnlineURI"); ok {
		t.Fatal("EventWrite exposes forbidden OnlineURI field")
	}
}
