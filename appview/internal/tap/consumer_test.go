package tap_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"social.craftsky/appview/internal/tap"
)

// fakeIndexer records Handle calls and can be configured to fail.
type fakeIndexer struct {
	mu       sync.Mutex
	events   []tap.Event
	failOnce bool // if true, next Handle returns error (then resets to false)
}

func (f *fakeIndexer) Handle(ctx context.Context, ev tap.Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOnce {
		f.failOnce = false
		return errTest
	}
	f.events = append(f.events, ev)
	return nil
}

func (f *fakeIndexer) Events() []tap.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]tap.Event, len(f.events))
	copy(out, f.events)
	return out
}

var errTest = &testErr{msg: "intentional test error"}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }

// fakeTap is a minimal /channel WS server. It sends the provided frames
// on connect, then listens for ack frames from the client.
type fakeTap struct {
	frames []string
	acks   chan uint64
}

func newFakeTap(frames []string) *fakeTap {
	return &fakeTap{frames: frames, acks: make(chan uint64, 32)}
}

func (f *fakeTap) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("websocket accept: %v", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		ctx := r.Context()

		// Send all frames up front.
		for _, fr := range f.frames {
			if err := conn.Write(ctx, websocket.MessageText, []byte(fr)); err != nil {
				return
			}
		}

		// Read acks until client closes. Ack shape confirmed against
		// indigo/cmd/tap types.go: {"type": "ack", "id": <uint>}.
		for {
			var ack map[string]any
			if err := wsjson.Read(ctx, conn, &ack); err != nil {
				return
			}
			if ack["type"] == "ack" {
				if id, ok := ack["id"].(float64); ok {
					f.acks <- uint64(id)
				}
			}
		}
	})
}

func TestWSConsumer_HappyPath(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":1,"type":"record","record":{"live":true,"rev":"r1","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k1","action":"create","cid":"bafy1","record":{"text":"hi"}}}`,
		`{"id":2,"type":"record","record":{"live":true,"rev":"r2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k2","action":"create","cid":"bafy2","record":{"text":"hey"}}}`,
		`{"id":3,"type":"record","record":{"live":false,"rev":"r3","did":"did:plc:b","collection":"app.bsky.feed.post","rkey":"k3","action":"delete","cid":"bafy3"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &fakeIndexer{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      idx,
		AckTimeout:   5 * time.Second,
		ReconnectMax: 1 * time.Second,
		MaxRetries:   5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	// Wait for three events to be indexed.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(idx.Events()) == 3 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	evs := idx.Events()
	if len(evs) != 3 {
		t.Fatalf("indexed %d events, want 3; got %+v", len(evs), evs)
	}

	// Wait for three acks on the server side.
	seenAcks := map[uint64]bool{}
	for i := 0; i < 3; i++ {
		select {
		case id := <-ft.acks:
			seenAcks[id] = true
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for ack #%d; seen so far: %v", i, seenAcks)
		}
	}
	for _, want := range []uint64{1, 2, 3} {
		if !seenAcks[want] {
			t.Errorf("missing ack for id=%d", want)
		}
	}

	// Assert Event field mapping.
	if evs[0].URI != "at://did:plc:a/app.bsky.feed.post/k1" {
		t.Errorf("evs[0].URI = %q", evs[0].URI)
	}
	if evs[0].Action != "create" {
		t.Errorf("evs[0].Action = %q", evs[0].Action)
	}
	if !evs[0].Live {
		t.Errorf("evs[0].Live should be true")
	}
	if string(evs[0].Record) == "" || !json.Valid(evs[0].Record) {
		t.Errorf("evs[0].Record invalid: %q", evs[0].Record)
	}
	if evs[2].Action != "delete" {
		t.Errorf("evs[2].Action = %q", evs[2].Action)
	}

	// Cancel and wait for Run to return.
	cancel()
	select {
	case err := <-done:
		if err != nil && !isContextCanceled(err) {
			t.Errorf("Run returned %v; want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func isContextCanceled(err error) bool {
	return err == context.Canceled || strings.Contains(err.Error(), "context canceled")
}

func TestWSConsumer_IndexerErrorDoesNotAck(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":42,"type":"record","record":{"live":true,"rev":"r","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &fakeIndexer{failOnce: true}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      idx,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   100, // high — we only want to see the first failure
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	// We expect zero acks within 500ms (indexer failed, so no ack sent).
	select {
	case id := <-ft.acks:
		t.Fatalf("unexpected ack for id=%d after indexer error", id)
	case <-time.After(500 * time.Millisecond):
		// good: no ack
	}
}

func TestWSConsumer_ReconnectsOnWSClose(t *testing.T) {
	t.Parallel()

	var connCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&connCount, 1)
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		// Close the connection immediately.
		conn.Close(websocket.StatusInternalError, "simulated failure")
	}))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      &fakeIndexer{},
		AckTimeout:   1 * time.Second,
		ReconnectMax: 200 * time.Millisecond, // tight for fast test
		MaxRetries:   5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	deadline := time.Now().Add(1500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&connCount) >= 3 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&connCount); got < 3 {
		t.Fatalf("connected %d times, expected >=3 reconnects", got)
	}

	// ReconnectAttempt gets reset to 0 on every successful Dial (via
	// setConnected); the server here accepts before closing, so the
	// counter only reflects the most recent pre-connect attempt, not a
	// monotonic total. Just assert it's been bumped at least once.
	st := c.State()
	if st.ReconnectAttempt < 1 {
		t.Errorf("ReconnectAttempt = %d, want >=1", st.ReconnectAttempt)
	}
}

func TestWSConsumer_PoisonPillIsDroppedAfterMaxRetries(t *testing.T) {
	t.Parallel()

	// This is tricky: without redelivery, we only see "id=99" once.
	// So the poison-pill path is exercised only if Tap re-sends. We
	// emulate that by sending the same id 7 times in a row from the
	// fake server. With MaxRetries=5, the first 5 failures are ignored
	// (no ack), the 6th failure triggers the drop-and-ack path. We
	// send one extra frame as a buffer to avoid relying on exact count.
	sameFrame := `{"id":99,"type":"record","record":{"live":true,"rev":"r","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`
	frames := []string{sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &alwaysFailIndexer{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      idx,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	// Expect exactly one ack for id=99 after the 5th-failure -> 6th-attempt drop.
	select {
	case id := <-ft.acks:
		if id != 99 {
			t.Fatalf("ack id=%d, want 99", id)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("timeout waiting for poison-pill ack")
	}
}

type alwaysFailIndexer struct{}

func (alwaysFailIndexer) Handle(ctx context.Context, ev tap.Event) error { return errTest }
