// Package tap consumes atproto events from a Tap sidecar over WebSocket.
//
// The real WSConsumer lands in a later commit. This file defines the public
// types so other packages (internal/index, internal/app, internal/api) can
// compile against them.
package tap

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Event is one decoded record-event from Tap's /channel WebSocket.
// Identity events are consumed internally by the consumer and are not
// surfaced to indexers.
type Event struct {
	URI        string          // at://did/collection/rkey
	CID        string          // content identifier of the record
	DID        string          // repo owner
	Collection string          // e.g. "app.bsky.feed.post"
	Rkey       string          // record key
	Action     string          // "create" | "update" | "delete"
	Record     json.RawMessage // opaque JSON; nil or empty on Action == "delete"
	Live       bool            // false during backfill, true for steady-state
	ID         uint64          // Tap's per-event "id" field from the envelope
	Rev        string          // repo rev at time of event
}

// Consumer is the interface the appview uses to consume events from Tap.
type Consumer interface {
	// Run blocks until ctx is cancelled, continuously connecting to Tap
	// and dispatching events to the configured indexer. It always returns
	// a non-nil error; on graceful shutdown the error is ctx.Err().
	Run(ctx context.Context) error

	// State returns a snapshot of the consumer's current connection state.
	// Safe to call concurrently with Run.
	State() ConnState
}

// ConnState describes the consumer's current connection state; used by the
// /healthz handler and the `cli tap status` command.
type ConnState struct {
	Connected        bool
	LastEventAt      time.Time
	LastError        string
	ReconnectAttempt int
}

// NotImplemented is a stub consumer used until WSConsumer lands.
// Run returns an error immediately; State reports disconnected.
type NotImplemented struct{}

var _ Consumer = NotImplemented{}

func (NotImplemented) Run(ctx context.Context) error {
	return errors.New("tap: consumer not yet implemented")
}

func (NotImplemented) State() ConnState {
	return ConnState{LastError: "not implemented"}
}
