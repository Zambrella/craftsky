// Package index defines the contract for writing atproto records into
// Postgres. Implementations are dispatched by the Tap consumer, one event
// at a time. Implementations MUST be idempotent on (URI, CID) because Tap
// delivers events at least once.
package index

import (
	"context"
	"errors"

	"social.craftsky/appview/internal/tap"
)

// Indexer writes records into the application's Postgres store.
type Indexer interface {
	// Handle processes a single Tap event. Returns nil on success;
	// any non-nil error causes the Tap consumer to skip the ack, so
	// Tap will redeliver the event after its configured retry timeout.
	Handle(ctx context.Context, ev tap.Event) error
}

// NotImplemented is a stub indexer that errors on every event.
// Used during construction before the real indexer is wired in.
type NotImplemented struct{}

var _ Indexer = NotImplemented{}

func (NotImplemented) Handle(ctx context.Context, ev tap.Event) error {
	return errors.New("indexer: not yet implemented")
}
