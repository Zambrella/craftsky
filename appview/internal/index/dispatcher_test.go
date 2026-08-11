package index

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
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

type fakePostHandleObserver struct {
	calls  *[]string
	err    error
	events []tap.Event
}

func (observer *fakePostHandleObserver) ObserveHandled(_ context.Context, event tap.Event) error {
	*observer.calls = append(*observer.calls, "observer")
	observer.events = append(observer.events, event)
	return observer.err
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

func TestDispatcher_ObservesOnlyAfterSuccessfulIndexingAndBeforeAck(t *testing.T) {
	var calls []string
	indexer := &orderingIndexer{calls: &calls}
	observer := &fakePostHandleObserver{calls: &calls}
	d := NewDispatcher(NotImplemented{})
	d.Register("social.craftsky.feed.post", indexer)
	d.AddPostHandleObserver(observer)
	event := tap.Event{
		URI: "at://did:plc:alice/social.craftsky.feed.post/post1", Collection: "social.craftsky.feed.post",
		Action: "delete", ID: 41, Rev: "rev-1",
	}
	if err := d.Handle(context.Background(), event); err != nil {
		t.Fatalf("handle observed event: %v", err)
	}
	if got, want := strings.Join(calls, ","), "indexer,observer"; got != want {
		t.Fatalf("call order=%q want=%q", got, want)
	}
	if len(observer.events) != 1 || observer.events[0].ID != 41 || observer.events[0].Rev != "rev-1" {
		t.Fatalf("observed events=%+v", observer.events)
	}

	calls = nil
	indexer.err = errors.New("index transaction failed")
	if err := d.Handle(context.Background(), event); !errors.Is(err, indexer.err) {
		t.Fatalf("indexer failure=%v", err)
	}
	if got, want := strings.Join(calls, ","), "indexer"; got != want {
		t.Fatalf("failed-index call order=%q want=%q", got, want)
	}

	calls = nil
	indexer.err = nil
	observer.err = errors.New("receipt transaction failed")
	if err := d.Handle(context.Background(), event); !errors.Is(err, observer.err) || !errors.Is(err, tap.ErrMustReplay) {
		t.Fatalf("observer failure=%v", err)
	}
	if got, want := strings.Join(calls, ","), "indexer,observer"; got != want {
		t.Fatalf("observer-failure call order=%q want=%q", got, want)
	}
}

type orderingIndexer struct {
	calls *[]string
	err   error
}

func (indexer *orderingIndexer) Handle(context.Context, tap.Event) error {
	*indexer.calls = append(*indexer.calls, "indexer")
	return indexer.err
}

func TestDispatcher_LogOmitsRawRecordIdentity(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	d := NewDispatcher(&fakeIndexer{name: "fallback"})
	d.Register("social.craftsky.feed.post", &fakeIndexer{name: "post"})

	err := d.Handle(context.Background(), tap.Event{
		URI:        "at://did:plc:alice/social.craftsky.feed.post/post1",
		CID:        "bafySecret",
		DID:        "did:plc:alice",
		Rkey:       "post1",
		Collection: "social.craftsky.feed.post",
		Action:     "create",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := logs.String()
	for _, want := range []string{
		`"collection":"social.craftsky.feed.post"`,
		`"action":"create"`,
		`"fallback":false`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing %s:\n%s", want, out)
		}
	}

	for _, forbidden := range []string{
		"did:plc:alice",
		"post1",
		"at://",
		"bafySecret",
		`"uri"`,
		`"did"`,
		`"rkey"`,
		`"cid"`,
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("logs contain raw identity/content field %q:\n%s", forbidden, out)
		}
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
