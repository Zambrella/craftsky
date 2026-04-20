package index

import (
	"context"
	"errors"
	"testing"

	"social.craftsky/appview/internal/tap"
)

// fakeIndexer records every event it's asked to handle and returns a
// configurable error.
type fakeIndexer struct {
	name   string
	events []tap.Event
	err    error
}

func (f *fakeIndexer) Handle(_ context.Context, ev tap.Event) error {
	f.events = append(f.events, ev)
	return f.err
}

func TestDispatcher_RoutesByCollection(t *testing.T) {
	a := &fakeIndexer{name: "a"}
	b := &fakeIndexer{name: "b"}
	fallback := &fakeIndexer{name: "fallback"}

	d := NewDispatcher(fallback)
	d.Register("social.craftsky.test.post", a)
	d.Register("app.bsky.feed.post", b)

	if err := d.Handle(context.Background(), tap.Event{Collection: "social.craftsky.test.post", URI: "at://x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := d.Handle(context.Background(), tap.Event{Collection: "app.bsky.feed.post", URI: "at://y"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(a.events); got != 1 {
		t.Errorf("indexer a: got %d events, want 1", got)
	}
	if got := len(b.events); got != 1 {
		t.Errorf("indexer b: got %d events, want 1", got)
	}
	if got := len(fallback.events); got != 0 {
		t.Errorf("fallback: got %d events, want 0", got)
	}
}

func TestDispatcher_UnregisteredGoesToFallback(t *testing.T) {
	fallback := &fakeIndexer{}
	d := NewDispatcher(fallback)

	if err := d.Handle(context.Background(), tap.Event{Collection: "com.example.unknown"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := len(fallback.events); got != 1 {
		t.Fatalf("fallback: got %d events, want 1", got)
	}
}

func TestDispatcher_PropagatesDownstreamError(t *testing.T) {
	boom := errors.New("boom")
	a := &fakeIndexer{err: boom}
	d := NewDispatcher(NotImplemented{})
	d.Register("x.y.z", a)

	err := d.Handle(context.Background(), tap.Event{Collection: "x.y.z"})
	if !errors.Is(err, boom) {
		t.Fatalf("got %v, want boom", err)
	}
}

func TestDispatcher_NilFallbackPanicsOnMiss(t *testing.T) {
	// A nil fallback is a wiring bug — prefer a loud panic at boot over a
	// silent drop in prod. We don't test the happy path needing a fallback;
	// this test documents the contract: every dispatcher must have one.
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on nil fallback")
		}
	}()
	NewDispatcher(nil)
}
