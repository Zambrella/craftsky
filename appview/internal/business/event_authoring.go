package business

import (
	"errors"
	"time"
)

var ErrClientCreatedAt = errors.New("business: createdAt is server-owned")

type EventAuthoringInput struct {
	Event            EventWrite
	CreatedAtPresent bool
	CreatedAt        string
}

type AuthoredEvent struct {
	Event     EventWrite
	CreatedAt string
}

func PrepareEventCreate(input EventAuthoringInput, now time.Time) (AuthoredEvent, error) {
	if input.CreatedAtPresent {
		return AuthoredEvent{}, ErrClientCreatedAt
	}
	return AuthoredEvent{
		Event:     input.Event,
		CreatedAt: now.UTC().Truncate(time.Second).Format(time.RFC3339),
	}, nil
}

func PrepareEventUpdate(input EventAuthoringInput, storedCreatedAt string) (AuthoredEvent, error) {
	if input.CreatedAtPresent {
		return AuthoredEvent{}, ErrClientCreatedAt
	}
	return AuthoredEvent{Event: input.Event, CreatedAt: storedCreatedAt}, nil
}
