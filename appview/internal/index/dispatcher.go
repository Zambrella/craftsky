package index

import (
	"context"

	"social.craftsky/appview/internal/tap"
)

// Dispatcher routes tap.Events to indexers keyed by the event's atproto
// collection NSID. It itself implements Indexer so it can be handed to
// the Tap consumer in place of a single concrete indexer.
//
// Dispatcher is NOT safe for concurrent Register calls. Register is
// expected to be called once during startup wiring, before Run on the
// consumer. Handle is called serially by the Tap consumer (one event at
// a time per connection), so no locking is needed on the read path.
type Dispatcher struct {
	handlers map[string]Indexer
	fallback Indexer
}

// NewDispatcher returns a Dispatcher with the given fallback for events
// whose collection has no registered handler. fallback must be non-nil;
// a wiring mistake that passes nil is a loud panic (we prefer that to a
// silent drop in prod).
func NewDispatcher(fallback Indexer) *Dispatcher {
	if fallback == nil {
		panic("index.NewDispatcher: fallback must not be nil")
	}
	return &Dispatcher{
		handlers: map[string]Indexer{},
		fallback: fallback,
	}
}

// Register associates collection (e.g. "social.craftsky.feed.post") with
// idx. A later Register for the same collection replaces the previous
// handler; this is convenient in tests and startup-only in prod.
func (d *Dispatcher) Register(collection string, idx Indexer) {
	d.handlers[collection] = idx
}

// Handle routes ev to the indexer registered for ev.Collection, or to
// the fallback if none matches. Downstream errors propagate unchanged.
func (d *Dispatcher) Handle(ctx context.Context, ev tap.Event) error {
	if h, ok := d.handlers[ev.Collection]; ok {
		return h.Handle(ctx, ev)
	}
	return d.fallback.Handle(ctx, ev)
}

var _ Indexer = (*Dispatcher)(nil)
