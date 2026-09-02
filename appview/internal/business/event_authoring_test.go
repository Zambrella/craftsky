package business

import (
	"errors"
	"testing"
	"time"
)

func TestEventCreatedAtIsServerOwned(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 34, 56, 987654321, time.FixedZone("test", -7*60*60))
	wantCreatedAt := "2026-09-01T19:34:56Z"
	event := EventWrite{Name: "Autumn market"}

	created, err := PrepareEventCreate(EventAuthoringInput{Event: event}, now)
	if err != nil {
		t.Fatalf("PrepareEventCreate without createdAt: %v", err)
	}
	if created.CreatedAt != wantCreatedAt || created.Event.Name != event.Name {
		t.Fatalf("created event = %+v, want createdAt %q and original event", created, wantCreatedAt)
	}
	if _, err := PrepareEventCreate(EventAuthoringInput{
		Event:            event,
		CreatedAtPresent: true,
		CreatedAt:        wantCreatedAt,
	}, now); !errors.Is(err, ErrClientCreatedAt) {
		t.Fatalf("PrepareEventCreate with createdAt error = %v, want ErrClientCreatedAt", err)
	}

	updated, err := PrepareEventUpdate(EventAuthoringInput{Event: event}, wantCreatedAt)
	if err != nil {
		t.Fatalf("PrepareEventUpdate without createdAt: %v", err)
	}
	if updated.CreatedAt != wantCreatedAt {
		t.Fatalf("updated createdAt = %q, want stored value %q", updated.CreatedAt, wantCreatedAt)
	}
	for name, supplied := range map[string]string{
		"same":      wantCreatedAt,
		"different": "2026-09-02T00:00:00Z",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := PrepareEventUpdate(EventAuthoringInput{
				Event:            event,
				CreatedAtPresent: true,
				CreatedAt:        supplied,
			}, wantCreatedAt); !errors.Is(err, ErrClientCreatedAt) {
				t.Fatalf("PrepareEventUpdate with %s createdAt error = %v, want ErrClientCreatedAt", name, err)
			}
		})
	}
}
