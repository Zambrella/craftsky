package tap_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/tap"
)

// fakeIndexer records Handle calls and can be configured to fail.
type fakeIndexer struct {
	mu       sync.Mutex
	events   []tap.Event
	failOnce bool // if true, next Handle returns error (then resets to false)
}

type identityDeletionSpy struct {
	mu   sync.Mutex
	dids []syntax.DID
}

func (s *identityDeletionSpy) HandleIdentityDeleted(_ context.Context, did syntax.DID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dids = append(s.dids, did)
	return nil
}

func (s *identityDeletionSpy) DIDs() []syntax.DID {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]syntax.DID(nil), s.dids...)
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

func TestWSConsumer_ForwardsOnlyTerminalIdentityDeletion(t *testing.T) {
	frames := []string{
		`{"id":1,"type":"identity","identity":{"did":"did:plc:actor","status":"active"}}`,
		`{"id":2,"type":"identity","identity":{"did":"did:plc:actor","status":"deactivated"}}`,
		`{"id":3,"type":"identity","identity":{"did":"did:plc:actor","status":"takendown"}}`,
		`{"id":4,"type":"identity","identity":{"did":"did:plc:actor","status":"deleted"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()
	spy := &identityDeletionSpy{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(srv.URL, "http://", "ws://", 1), Indexer: &fakeIndexer{},
		AckTimeout: time.Second, ReconnectMax: time.Second, MaxRetries: 5, IdentityHandler: spy,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go c.Run(ctx)
	for i := 0; i < len(frames); i++ {
		select {
		case <-ft.acks:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for identity ack")
		}
	}
	got := spy.DIDs()
	if len(got) != 1 || got[0] != "did:plc:actor" {
		t.Fatalf("deleted DIDs=%v", got)
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

type panicIndexer struct{}

func (panicIndexer) Handle(context.Context, tap.Event) error {
	panic("indexer panic")
}

type failOnIDIndexer struct {
	events []tap.Event
}

func (f *failOnIDIndexer) Handle(_ context.Context, ev tap.Event) error {
	if ev.ID == 2 {
		return errTest
	}
	f.events = append(f.events, ev)
	return nil
}

func TestWSConsumer_EmitsTapMetricsAndCapturesIndexerErrors(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":1,"type":"record","record":{"live":true,"rev":"r1","did":"did:plc:a","collection":"social.craftsky.feed.post","rkey":"k1","action":"create","cid":"bafy1","record":{"text":"hi"}}}`,
		`{"id":2,"type":"record","record":{"live":true,"rev":"r2","did":"did:plc:a","collection":"social.craftsky.feed.like","rkey":"k2","action":"create","cid":"bafy2","record":{"subject":"x"}}}`,
		`{"id":3,"type":"record","record":{"live":true,"rev":"r3","did":"did:plc:a","collection":"not-an-nsid!","rkey":"k3","action":"create","cid":"bafy3","record":{"text":"bad"}}}`,
		`{"id":4,"type":"identity","identity":{"did":"did:plc:a"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	transport := &sentry.MockTransport{}
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
		MetricRecorder:  recorder,
	})
	idx := &failOnIDIndexer{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          strings.Replace(srv.URL, "http://", "ws://", 1),
		Indexer:      idx,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   100,
		Observer:     observer,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	seenAcks := map[uint64]bool{}
	deadline := time.After(1500 * time.Millisecond)
	for len(seenAcks) < 3 {
		select {
		case id := <-ft.acks:
			seenAcks[id] = true
		case <-deadline:
			t.Fatalf("timeout waiting for expected acks; seen=%v", seenAcks)
		}
	}
	for _, want := range []uint64{1, 3, 4} {
		if !seenAcks[want] {
			t.Fatalf("missing ack for id=%d; seen=%v", want, seenAcks)
		}
	}
	if seenAcks[2] {
		t.Fatal("indexer-error event id=2 was acked; want retry")
	}

	calls := recorder.Calls()
	for _, want := range []string{
		"craftsky_appview_tap_connected",
		"craftsky_appview_tap_events_received_total",
		"craftsky_appview_tap_events_acknowledged_total",
		"craftsky_appview_tap_indexer_records_total",
	} {
		if !tapMetricCallsContain(calls, want) {
			t.Fatalf("metric calls missing %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}

	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1", len(events))
	}
	if events[0].Tags["component"] != "tap_indexer" || events[0].Tags["nsid"] != "social.craftsky.feed.like" || events[0].Tags["result"] != "error" {
		t.Fatalf("indexer Sentry event missing safe tags: %#v", events[0].Tags)
	}
}

func TestWSConsumer_ExportsSentryConsumeAndIndexerSpans(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":11,"type":"record","record":{"live":true,"rev":"r1","did":"did:plc:alice","collection":"social.craftsky.feed.post","rkey":"post1","action":"create","cid":"bafyPost","record":{"text":"secret body"}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env:                 "test",
		SentryDSN:           "https://public@example.invalid/1",
		SentryTransport:     transport,
		TracingEnabled:      true,
		TracesSampleRate:    1,
		TapTracingEnabled:   true,
		TapTracesSampleRate: 1,
	})
	idx := &fakeIndexer{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          strings.Replace(srv.URL, "http://", "ws://", 1),
		Indexer:      idx,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   5,
		Observer:     observer,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	select {
	case id := <-ft.acks:
		if id != 11 {
			t.Fatalf("ack id=%d, want 11", id)
		}
	case <-time.After(1500 * time.Millisecond):
		cancel()
		t.Fatal("timeout waiting for ack")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && !isContextCanceled(err) {
			t.Fatalf("Run returned %v; want context cancellation", err)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("Run did not return after cancel")
	}

	if !observer.Flush(50 * time.Millisecond) {
		t.Fatal("observer Flush returned false")
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1 transaction", len(events))
	}
	event := events[0]
	if event.Transaction != "tap.consume" {
		t.Fatalf("transaction = %q, want tap.consume; event=%#v", event.Transaction, event)
	}
	if len(event.Spans) < 4 {
		t.Fatalf("transaction spans = %d, want at least 4; event=%#v", len(event.Spans), event)
	}
	spansByOp := map[string]*sentry.Span{}
	for _, span := range event.Spans {
		spansByOp[span.Op] = span
	}
	for _, want := range []string{"tap.receive", "tap.decode", "tap.indexer.handle", "tap.ack"} {
		if spansByOp[want] == nil {
			t.Fatalf("missing Tap child span %q; spans=%#v", want, event.Spans)
		}
	}
	span := spansByOp["tap.indexer.handle"]
	for key, want := range map[string]any{
		"component": "tap_indexer",
		"operation": "tap.indexer.handle",
		"nsid":      "social.craftsky.feed.post",
		"result":    "success",
	} {
		if got := span.Data[key]; got != want {
			t.Fatalf("child span data %q = %#v, want %#v; all data=%#v", key, got, want, span.Data)
		}
	}
	for _, forbidden := range []string{"did:plc:alice", "post1", "bafyPost", "secret body"} {
		if strings.Contains(event.Transaction, forbidden) {
			t.Fatalf("transaction contains forbidden value %q: %#v", forbidden, event)
		}
		for _, span := range event.Spans {
			if strings.Contains(span.Op, forbidden) || strings.Contains(span.Description, forbidden) {
				t.Fatalf("span contains forbidden value %q: %#v", forbidden, span)
			}
			for key, value := range span.Data {
				if strings.Contains(key, forbidden) || strings.Contains(valueAsString(value), forbidden) {
					t.Fatalf("span data contains forbidden value %q: %q=%#v", forbidden, key, value)
				}
			}
		}
	}
}

func TestWSConsumer_IndexerPanicDoesNotCrashConsumer(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":123,"type":"record","record":{"live":true,"rev":"r","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Indexer:      panicIndexer{},
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		MaxRetries:   100,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	select {
	case id := <-ft.acks:
		t.Fatalf("unexpected ack for panicking event id=%d", id)
	case <-time.After(500 * time.Millisecond):
		// good: panic is treated like an indexer error, so Tap can redeliver.
	}
}

func valueAsString(value any) string {
	return fmt.Sprint(value)
}

func tapMetricCallsContain(calls []observability.MetricCall, name string) bool {
	for _, call := range calls {
		if call.Name == name {
			return true
		}
	}
	return false
}

// TestWSConsumer_MalformedIdentifierAcksAndDrops covers the boundary
// validation added when Event fields became typed via indigo syntax types.
// A frame with an invalid NSID can never be successfully indexed, so the
// consumer must ack-and-drop rather than letting Tap redeliver forever.
func TestWSConsumer_MalformedIdentifierAcksAndDrops(t *testing.T) {
	t.Parallel()

	// "x" is a valid DID prefix but "not-an-nsid!" fails syntax.ParseNSID.
	frames := []string{
		`{"id":7,"type":"record","record":{"live":true,"rev":"r","did":"did:plc:a","collection":"not-an-nsid!","rkey":"k","action":"create","cid":"bafy","record":{}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	idx := &fakeIndexer{}
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

	// Expect an ack despite the indexer never being called.
	select {
	case id := <-ft.acks:
		if id != 7 {
			t.Fatalf("ack id=%d, want 7", id)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for ack of malformed event")
	}
	if got := len(idx.Events()); got != 0 {
		t.Errorf("indexer received %d events for malformed envelope; want 0", got)
	}
}
