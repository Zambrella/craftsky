package tap_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"

	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/tap"
)

func TestWSConsumerConfigExposesOnlyDurableIngestionBoundary(t *testing.T) {
	t.Parallel()

	configType := reflect.TypeOf(tap.WSConsumerConfig{})
	want := map[string]bool{
		"URL": true, "Ingestor": true, "AckTimeout": true,
		"ReconnectMax": true, "Logger": true, "Observer": true,
	}
	if configType.NumField() != len(want) {
		t.Fatalf("WSConsumerConfig has %d fields, want %d durable-only fields", configType.NumField(), len(want))
	}
	for i := range configType.NumField() {
		field := configType.Field(i).Name
		if !want[field] {
			t.Fatalf("WSConsumerConfig exposes non-durable compatibility field %q", field)
		}
		delete(want, field)
	}
	if len(want) != 0 {
		t.Fatalf("WSConsumerConfig is missing durable fields %v", want)
	}
}

type durableIngestorSpy struct {
	mu              sync.Mutex
	records         []tap.Event
	identities      []tap.IdentityEvent
	invalid         []tap.InvalidEvent
	recordFunc      func(context.Context, tap.Event) (tap.Outcome, error)
	identityFunc    func(context.Context, tap.IdentityEvent) (tap.Outcome, error)
	recordErr       error
	identityErr     error
	quarantineErr   error
	quarantineFunc  func(context.Context, tap.InvalidEvent) (tap.Outcome, error)
	recordOutcome   tap.Outcome
	identityOutcome tap.Outcome
	recordObserved  chan tap.Event
}

type retryThenSucceedIngestor struct {
	remainingFailures atomic.Int32
	calls             atomic.Int32
}

func newRetryThenSucceedIngestor(failures int32) *retryThenSucceedIngestor {
	ingestor := &retryThenSucceedIngestor{}
	ingestor.remainingFailures.Store(failures)
	return ingestor
}

func (ingestor *retryThenSucceedIngestor) IngestRecord(context.Context, tap.Event) (tap.Outcome, error) {
	ingestor.calls.Add(1)
	if ingestor.remainingFailures.Add(-1) >= 0 {
		return tap.Retryable(tap.ReasonStorageUnavailable), errTest
	}
	return tap.Applied(), nil
}

func (*retryThenSucceedIngestor) IngestIdentity(context.Context, tap.IdentityEvent) (tap.Outcome, error) {
	return tap.Applied(), nil
}

func (*retryThenSucceedIngestor) Quarantine(context.Context, tap.InvalidEvent) (tap.Outcome, error) {
	return tap.PermanentInvalid(tap.ReasonInvalidEnvelope), nil
}

func (s *durableIngestorSpy) IngestRecord(ctx context.Context, event tap.Event) (tap.Outcome, error) {
	s.mu.Lock()
	s.records = append(s.records, event)
	fn := s.recordFunc
	outcome := s.recordOutcome
	err := s.recordErr
	s.mu.Unlock()
	if s.recordObserved != nil {
		s.recordObserved <- event
	}
	if fn != nil {
		return fn(ctx, event)
	}
	if outcome.Kind == "" {
		outcome = tap.Applied()
	}
	return outcome, err
}

func (s *durableIngestorSpy) IngestIdentity(ctx context.Context, event tap.IdentityEvent) (tap.Outcome, error) {
	s.mu.Lock()
	s.identities = append(s.identities, event)
	fn := s.identityFunc
	outcome := s.identityOutcome
	err := s.identityErr
	s.mu.Unlock()
	if fn != nil {
		return fn(ctx, event)
	}
	if outcome.Kind == "" {
		outcome = tap.Applied()
	}
	return outcome, err
}

func TestWSConsumer_IdentityOverProcessingDeadlineIsNeverAcked(t *testing.T) {
	frames := []string{
		`{"id":910,"type":"identity","identity":{"did":"did:plc:in-budget","status":"deleted"}}`,
		`{"id":911,"type":"identity","identity":{"did":"did:plc:over-budget","status":"deleted"}}`,
	}
	ft := newFakeTap(frames)
	server := httptest.NewServer(ft.handler(t))
	defer server.Close()
	ingestor := &durableIngestorSpy{identityFunc: func(ctx context.Context, event tap.IdentityEvent) (tap.Outcome, error) {
		if event.ID == 910 {
			return tap.Applied(), nil
		}
		<-ctx.Done()
		return tap.Retryable(tap.ReasonStorageUnavailable), ctx.Err()
	}}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: 40 * time.Millisecond, ReconnectMax: time.Second,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go consumer.Run(ctx)

	select {
	case id := <-ft.acks:
		if id != 910 {
			t.Fatalf("first ACK id=%d, want in-budget identity 910", id)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("in-budget identity was not acknowledged")
	}
	select {
	case id := <-ft.acks:
		t.Fatalf("over-budget identity was acknowledged: id=%d", id)
	case <-time.After(120 * time.Millisecond):
	}
	if got := ingestor.identityEvents(); len(got) != 2 {
		t.Fatalf("identity attempts=%v, want both in-budget and over-budget events", got)
	}
}

func (s *durableIngestorSpy) Quarantine(ctx context.Context, event tap.InvalidEvent) (tap.Outcome, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalid = append(s.invalid, event)
	if s.quarantineFunc != nil {
		return s.quarantineFunc(ctx, event)
	}
	if s.quarantineErr != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), s.quarantineErr
	}
	return tap.PermanentInvalid(event.Reason), nil
}

func (s *durableIngestorSpy) invalidEvents() []tap.InvalidEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tap.InvalidEvent(nil), s.invalid...)
}

func (s *durableIngestorSpy) recordEvents() []tap.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tap.Event(nil), s.records...)
}

func (s *durableIngestorSpy) identityEvents() []tap.IdentityEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]tap.IdentityEvent(nil), s.identities...)
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
		`{"id":1,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k1","action":"create","cid":"bafy1","record":{"text":"hi"}}}`,
		`{"id":2,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa3","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k2","action":"create","cid":"bafy2","record":{"text":"hey"}}}`,
		`{"id":3,"type":"record","record":{"live":false,"rev":"3aaaaaaaaaaa4","did":"did:plc:b","collection":"app.bsky.feed.post","rkey":"k3","action":"delete","cid":"bafy3"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	ingestor := &durableIngestorSpy{recordObserved: make(chan tap.Event, len(frames))}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Ingestor:     ingestor,
		AckTimeout:   5 * time.Second,
		ReconnectMax: 1 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

	// Each signal is emitted at the durable ingestion boundary.
	for range frames {
		select {
		case <-ingestor.recordObserved:
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for durable ingestion")
		}
	}
	evs := ingestor.recordEvents()
	if len(evs) != 3 {
		t.Fatalf("ingested %d events, want 3; got %+v", len(evs), evs)
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

func TestWSConsumer_DurablyIngestsEverySupportedIdentityEventBeforeAck(t *testing.T) {
	frames := []string{
		`{"id":1,"type":"identity","identity":{"did":"did:plc:actor","status":"active"}}`,
		`{"id":2,"type":"identity","identity":{"did":"did:plc:actor","status":"deactivated"}}`,
		`{"id":3,"type":"identity","identity":{"did":"did:plc:actor","status":"suspended"}}`,
		`{"id":4,"type":"identity","identity":{"did":"did:plc:actor","status":"takendown"}}`,
		`{"id":5,"type":"identity","identity":{"did":"did:plc:actor","status":"deleted"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()
	ingestor := &durableIngestorSpy{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:        strings.Replace(srv.URL, "http://", "ws://", 1),
		Ingestor:   ingestor,
		AckTimeout: time.Second, ReconnectMax: time.Second,
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
	got := ingestor.identityEvents()
	if len(got) != len(frames) {
		t.Fatalf("identity events=%v, want %d durable events", got, len(frames))
	}
	for i, wantStatus := range []string{"active", "deactivated", "suspended", "takendown", "deleted"} {
		if got[i].DID != "did:plc:actor" || got[i].Status != wantStatus {
			t.Fatalf("identity event %d = %+v, want DID did:plc:actor and status %q", i, got[i], wantStatus)
		}
	}
}

func isContextCanceled(err error) bool {
	return err == context.Canceled || strings.Contains(err.Error(), "context canceled")
}

func TestWSConsumer_RetryableIngestionErrorDoesNotAck(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":42,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	ingestor := &durableIngestorSpy{
		recordOutcome: tap.Retryable(tap.ReasonStorageUnavailable),
		recordErr:     errTest,
	}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Ingestor:     ingestor,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	// We expect zero acks within 500ms because no durable outcome committed.
	select {
	case id := <-ft.acks:
		t.Fatalf("unexpected ack for id=%d after retryable ingestion error", id)
	case <-time.After(500 * time.Millisecond):
		// good: no ack
	}
}

func TestWSConsumer_ReconnectsOnWSClose(t *testing.T) {
	t.Parallel()

	var connCount int32
	connected := make(chan struct{}, 3)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&connCount, 1)
		connected <- struct{}{}
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
		Ingestor:     &durableIngestorSpy{},
		AckTimeout:   1 * time.Second,
		ReconnectMax: 200 * time.Millisecond, // tight for fast test
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	for range 3 {
		select {
		case <-connected:
		case <-time.After(1500 * time.Millisecond):
			t.Fatalf("connected %d times, expected >=3 reconnects", atomic.LoadInt32(&connCount))
		}
	}
	if got := atomic.LoadInt32(&connCount); got < 3 {
		t.Fatalf("connected %d times, expected >=3 reconnects", got)
	}
}

func TestWSConsumer_RetryableFailureNeverBecomesTerminalAck(t *testing.T) {
	t.Parallel()

	// Tap redelivers the same event id until AppView acks a durable outcome.
	// Delivery count is not consumer state: even a long infrastructure outage
	// must never turn a retryable failure into a terminal acknowledgement.
	sameFrame := `{"id":99,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`
	frames := []string{sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	ingestor := &durableIngestorSpy{
		recordOutcome: tap.Retryable(tap.ReasonStorageUnavailable),
		recordErr:     errTest,
	}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Ingestor:     ingestor,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	// More than the former six-delivery threshold still produces no ack.
	select {
	case id := <-ft.acks:
		t.Fatalf("retryable event was terminally acked after repeated delivery: id=%d", id)
	case <-time.After(750 * time.Millisecond):
		// Correct: Tap retains the event for a later retry.
	}
	if got := len(ingestor.recordEvents()); got != len(frames) {
		t.Fatalf("durable ingestion attempts=%d, want one attempt for each of %d redeliveries", got, len(frames))
	}
}

func TestWSConsumer_RetryableFailureBeyondFormerLimitEventuallyAcksDurableSuccess(t *testing.T) {
	t.Parallel()

	sameFrame := `{"id":100,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`
	frames := []string{sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame, sameFrame}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	ingestor := newRetryThenSucceedIngestor(7)
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: time.Second, ReconnectMax: 500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go consumer.Run(ctx)

	select {
	case id := <-ft.acks:
		if id != 100 {
			t.Fatalf("ack id=%d, want 100", id)
		}
	case <-time.After(time.Second):
		t.Fatal("durable success after seven retryable failures was not acknowledged")
	}
	if got := ingestor.calls.Load(); got != 8 {
		t.Fatalf("ingestion calls=%d, want seven retryable attempts plus one durable success", got)
	}
	select {
	case id := <-ft.acks:
		t.Fatalf("event was acknowledged more than once: id=%d", id)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestWSConsumer_ReconnectsAfterQuarantiningEnvelopeWithoutAckID(t *testing.T) {
	var connections atomic.Int32
	connected := make(chan struct{}, 2)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connections.Add(1)
		connected <- struct{}{}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Errorf("accept: %v", err)
			return
		}
		defer conn.CloseNow()
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"id":0,"type":"unsupported"}`))
		var ack map[string]any
		_ = wsjson.Read(r.Context(), conn, &ack)
	}))
	defer srv.Close()

	ingestor := &durableIngestorSpy{}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: time.Second, ReconnectMax: 20 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go consumer.Run(ctx)
	for range 2 {
		select {
		case <-connected:
		case <-time.After(350 * time.Millisecond):
			t.Fatalf("connections=%d, want reconnect after durable quarantine without an ack id", connections.Load())
		}
	}
	if got := connections.Load(); got < 2 {
		t.Fatalf("connections=%d, want reconnect after durable quarantine without an ack id", got)
	}
	invalid := ingestor.invalidEvents()
	if len(invalid) == 0 || invalid[0].Reason != tap.ReasonInvalidEnvelope {
		t.Fatalf("quarantined events=%+v, want invalid envelope", invalid)
	}
}

func TestWSConsumer_EmitsTapMetricsAndCapturesIndexerErrors(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":1,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"social.craftsky.feed.post","rkey":"k1","action":"create","cid":"bafy1","record":{"text":"hi"}}}`,
		`{"id":2,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa3","did":"did:plc:a","collection":"social.craftsky.feed.like","rkey":"k2","action":"create","cid":"bafy2","record":{"subject":"x"}}}`,
		`{"id":3,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa4","did":"did:plc:a","collection":"not-an-nsid!","rkey":"k3","action":"create","cid":"bafy3","record":{"text":"bad"}}}`,
		`{"id":4,"type":"identity","identity":{"did":"did:plc:a"}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	transport := &sentry.MockTransport{}
	recorder := observability.NewInMemoryMetricRecorder()
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	observer := observability.New(observability.Config{
		Env:             "test",
		SentryDSN:       "https://public@example.invalid/1",
		SentryTransport: transport,
		MetricRecorder:  recorder,
		Logger:          logger,
	})
	ingestor := &durableIngestorSpy{
		recordFunc: func(_ context.Context, event tap.Event) (tap.Outcome, error) {
			if event.ID == 2 {
				return tap.Retryable(tap.ReasonProjectionFailure), &pgconn.PgError{Code: "40001", Message: "opaque private row canary"}
			}
			return tap.Applied(), nil
		},
	}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          strings.Replace(srv.URL, "http://", "ws://", 1),
		Ingestor:     ingestor,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
		Observer:     observer,
		Logger:       logger,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()

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

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop")
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
	if len(events) != 2 {
		t.Fatalf("terminal input quarantine issues=%d, want2", len(events))
	}
	for _, event := range events {
		if event.Contexts["diagnostic"]["event_id"] == "2" {
			t.Fatal("expected record retry created an issue")
		}
	}
	for _, want := range []string{"pgconn.PgError", "40001", "at://did:plc:a/social.craftsky.feed.like/k2", "bafy2", "retry", "not_acknowledged"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("missing %s: %s", want, local.String())
		}
	}
	if strings.Contains(local.String(), "opaque private row canary") {
		t.Fatal("row leaked")
	}
}

func TestWSConsumer_ExportsSentryConsumeAndIndexerSpans(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":11,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:alice","collection":"social.craftsky.feed.post","rkey":"post1","action":"create","cid":"bafyPost","record":{"text":"secret body"}}}`,
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
	ingestor := &durableIngestorSpy{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          strings.Replace(srv.URL, "http://", "ws://", 1),
		Ingestor:     ingestor,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
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

func TestWSConsumer_IngestorPanicDoesNotCrashConsumer(t *testing.T) {
	t.Parallel()

	frames := []string{
		`{"id":123,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)
	ingestor := &durableIngestorSpy{
		recordFunc: func(context.Context, tap.Event) (tap.Outcome, error) {
			panic("ingestor panic")
		},
	}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Ingestor:     ingestor,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go c.Run(ctx)

	select {
	case id := <-ft.acks:
		t.Fatalf("unexpected ack for panicking event id=%d", id)
	case <-time.After(500 * time.Millisecond):
		// Correct: a panic is retryable, so Tap can redeliver.
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

// TestWSConsumer_MalformedIdentifierQuarantinesBeforeAck covers boundary
// validation for typed atproto identifiers. A frame with an invalid NSID can
// never be projected, so the consumer must durably quarantine it before ACK.
func TestWSConsumer_MalformedIdentifierQuarantinesBeforeAck(t *testing.T) {
	t.Parallel()

	// "x" is a valid DID prefix but "not-an-nsid!" fails syntax.ParseNSID.
	frames := []string{
		`{"id":7,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"not-an-nsid!","rkey":"k","action":"create","cid":"bafy","record":{}}}`,
	}
	ft := newFakeTap(frames)
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()

	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1)

	var local bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: slog.New(slog.NewJSONHandler(&local, nil)), SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	ingestor := &durableIngestorSpy{}
	c := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          wsURL,
		Observer:     observer,
		Logger:       slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil))),
		Ingestor:     ingestor,
		AckTimeout:   1 * time.Second,
		ReconnectMax: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { c.Run(ctx); close(done) }()

	// Expect an ack only after durable quarantine; source ingestion is skipped.
	select {
	case id := <-ft.acks:
		if id != 7 {
			t.Fatalf("ack id=%d, want 7", id)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for ack of malformed event")
	}
	if got := len(ingestor.recordEvents()); got != 0 {
		t.Errorf("source ingestor received %d events for malformed envelope; want 0", got)
	}
	cancel()
	<-done
	observer.Flush(time.Second)
	data, _ := json.Marshal(transport.Events())
	for _, want := range []string{"quarantine", "durably_committed", "invalid_collection", "did:plc:a"} {
		if !strings.Contains(string(data), want) || !strings.Contains(local.String(), want) {
			t.Fatalf("missing %s: %s %s", want, local.String(), data)
		}
	}
	if len(transport.Events()) != 1 {
		t.Fatalf("terminal events=%d", len(transport.Events()))
	}

}

func TestWSConsumer_MalformedIdentifierRequiresDurableQuarantineBeforeAck(t *testing.T) {
	frame := `{"id":71,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"not-an-nsid!","rkey":"k","action":"create","cid":"bafy","record":{}}}`

	t.Run("quarantine committed", func(t *testing.T) {
		ft := newFakeTap([]string{frame})
		srv := httptest.NewServer(ft.handler(t))
		defer srv.Close()
		ingestor := &durableIngestorSpy{}
		consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
			URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
			AckTimeout: time.Second, ReconnectMax: 500 * time.Millisecond,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		go consumer.Run(ctx)

		select {
		case id := <-ft.acks:
			if id != 71 {
				t.Fatalf("ack id=%d, want 71", id)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for ack after durable quarantine")
		}
		invalid := ingestor.invalidEvents()
		if len(invalid) != 1 || invalid[0].Reason != tap.ReasonInvalidCollection || invalid[0].ID != 71 {
			t.Fatalf("quarantined events=%+v", invalid)
		}
	})

	t.Run("quarantine storage failure", func(t *testing.T) {
		ft := newFakeTap([]string{frame})
		srv := httptest.NewServer(ft.handler(t))
		defer srv.Close()
		ingestor := &durableIngestorSpy{quarantineErr: errTest}
		consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
			URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
			AckTimeout: time.Second, ReconnectMax: 500 * time.Millisecond,
		})
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		go consumer.Run(ctx)

		select {
		case id := <-ft.acks:
			t.Fatalf("quarantine failure was acked: id=%d", id)
		case <-time.After(400 * time.Millisecond):
			// Correct: Tap retains the event until quarantine can commit.
		}
	})
}

func TestWSConsumer_DeterministicRecordEnvelopeDefectsAreQuarantined(t *testing.T) {
	tests := []struct {
		name   string
		frame  string
		reason tap.ReasonCode
	}{
		{
			name:   "missing repository revision",
			frame:  `{"id":81,"type":"record","record":{"live":true,"did":"did:plc:a","collection":"social.craftsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{}}}`,
			reason: tap.ReasonInvalidEnvelope,
		},
		{
			name:   "non TID repository revision",
			frame:  `{"id":83,"type":"record","record":{"live":true,"rev":"zzzzzzzzzzzzz","did":"did:plc:a","collection":"social.craftsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{}}}`,
			reason: tap.ReasonInvalidEnvelope,
		},
		{
			name:   "missing create CID",
			frame:  `{"id":82,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"social.craftsky.feed.post","rkey":"k","action":"create","record":{}}}`,
			reason: tap.ReasonMalformedRecord,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ft := newFakeTap([]string{test.frame})
			srv := httptest.NewServer(ft.handler(t))
			defer srv.Close()
			ingestor := &durableIngestorSpy{}
			consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
				URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
				AckTimeout: time.Second, ReconnectMax: 500 * time.Millisecond,
			})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			go consumer.Run(ctx)
			select {
			case <-ft.acks:
			case <-time.After(time.Second):
				t.Fatal("timeout waiting for ACK after durable quarantine")
			}
			invalid := ingestor.invalidEvents()
			if len(invalid) != 1 || invalid[0].Reason != test.reason {
				t.Fatalf("quarantined events=%+v, want reason %s", invalid, test.reason)
			}
			if len(ingestor.recordEvents()) != 0 {
				t.Fatal("deterministically invalid record reached source ingestion")
			}
		})
	}
}

func TestWSConsumer_AllowsBoundedRecordFramesLargerThanLibraryDefault(t *testing.T) {
	payload := strings.Repeat("x", 40<<10)
	frameBytes, err := json.Marshal(map[string]any{
		"id": 91, "type": "record",
		"record": map[string]any{
			"live": true, "rev": "3aaaaaaaaaaa2", "did": "did:plc:a",
			"collection": "social.craftsky.feed.post", "rkey": "large",
			"action": "create", "cid": "bafy-large",
			"record": map[string]any{"text": payload, "createdAt": "2026-08-14T12:00:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	ft := newFakeTap([]string{string(frameBytes)})
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()
	ingestor := &durableIngestorSpy{}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: time.Second, ReconnectMax: 500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go consumer.Run(ctx)
	select {
	case id := <-ft.acks:
		if id != 91 {
			t.Fatalf("ack id=%d, want 91", id)
		}
	case <-time.After(time.Second):
		t.Fatal("large but bounded Tap frame was not consumed")
	}
}

func TestWSConsumer_QuarantinesRecordAboveDurableSourceLimit(t *testing.T) {
	payload := strings.Repeat("x", (1<<20)+1)
	frameBytes, err := json.Marshal(map[string]any{
		"id": 92, "type": "record",
		"record": map[string]any{
			"live": true, "rev": "3aaaaaaaaaaa3", "did": "did:plc:a",
			"collection": "social.craftsky.feed.post", "rkey": "too-large",
			"action": "create", "cid": "bafy-too-large",
			"record": map[string]any{"text": payload, "createdAt": "2026-08-14T12:00:00Z"},
		},
	})
	if err != nil {
		t.Fatalf("encode frame: %v", err)
	}
	ft := newFakeTap([]string{string(frameBytes)})
	srv := httptest.NewServer(ft.handler(t))
	defer srv.Close()
	ingestor := &durableIngestorSpy{}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(srv.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: 2 * time.Second, ReconnectMax: 500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	go consumer.Run(ctx)
	select {
	case id := <-ft.acks:
		if id != 92 {
			t.Fatalf("ack id=%d, want 92", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("oversized record was not ACKed after durable quarantine")
	}
	invalid := ingestor.invalidEvents()
	if len(invalid) != 1 || invalid[0].Reason != tap.ReasonRecordTooLarge {
		t.Fatalf("quarantined events=%+v", invalid)
	}
	if len(ingestor.recordEvents()) != 0 {
		t.Fatal("oversized record reached durable source ingestion")
	}
}

// AT-002 / BR-001, FR-001 / AC-002: real WS ingestion recovery emits
// useful local details without acknowledging a retryable panicking event.
func TestWSConsumerPanicHasLocalCauseAndRecoveryFrame(t *testing.T) {
	ft := newFakeTap([]string{`{"id":123,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"app.bsky.feed.post","rkey":"k","action":"create","cid":"bafy","record":{"text":"x"}}}`})
	server := httptest.NewServer(ft.handler(t))
	defer server.Close()
	entered := make(chan struct{})
	ingestor := &durableIngestorSpy{recordFunc: func(context.Context, tap.Event) (tap.Outcome, error) {
		defer close(entered)
		panicAtTapDiagnosticBoundary()
		return tap.Outcome{}, nil
	}}
	var logs bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor, Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Observer: observer})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { consumer.Run(ctx); close(done) }()
	select {
	case <-entered:
		cancel()
	case <-ctx.Done():
		t.Fatal("ingestion never reached panic")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer failed to stop")
	}
	observer.Flush(time.Second)
	for _, want := range []string{"runtime.TypeAssertionError", "int, not string", "panicAtTapDiagnosticBoundary"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("local Tap panic missing %s", want)
		}
	}
	events := transport.Events()
	if len(events) != 1 || events[0].Exception[0].Stacktrace == nil {
		t.Fatalf("panic issue/stack missing: %#v", events)
	}
	select {
	case id := <-ft.acks:
		t.Fatalf("panicking event acknowledged: %d", id)
	default:
	}
	if strings.Contains(logs.String(), "/Users/") {
		t.Fatal("local user path leaked")
	}
}

func panicAtTapDiagnosticBoundary() { var value any = 1; _ = value.(string) }

// IT-006: durable terminal record rejection retains selected published context;
// retry/ACK behavior belongs to the existing durable boundary.
func TestTapTerminalRejectionRetainsPublishedFailureExcerpt(t *testing.T) {
	frame := `{"id":1,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:public","collection":"social.craftsky.feed.post","rkey":"post1","action":"create","cid":"bafyPublic","record":{"text":"published knit explanation access_token=credential-canary"}}}`
	ft := newFakeTap([]string{frame})
	server := httptest.NewServer(ft.handler(t))
	defer server.Close()
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	ingestor := &durableIngestorSpy{recordFunc: func(context.Context, tap.Event) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor, Observer: observer, Logger: logger})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	select {
	case id := <-ft.acks:
		if id != 1 {
			t.Fatal("wrong ack")
		}
	case <-time.After(time.Second):
		t.Fatal("missing durable ack")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop")
	}
	observer.Flush(time.Second)
	if len(transport.Events()) != 1 {
		t.Fatalf("terminal issues=%d", len(transport.Events()))
	}
	data, _ := json.Marshal(transport.Events())
	for _, want := range []string{"published knit explanation", "did:plc:public", "bafyPublic", "post1", "quarantine", "durably_committed"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("missing %s: %s", want, data)
		}
	}
	if strings.Contains(string(data)+local.String(), "credential-canary") {
		t.Fatal("credential leaked")
	}
	if os.Getenv("CRAFTSKY_DIAGNOSTIC_EVIDENCE") == "1" {
		t.Logf("Tap terminal local diagnostics:\n%s\nmock issue envelope:\n%s", local.String(), data)
	}
}

func TestWSConsumerIdentityPanicHasLocalCauseAndRecoveryFrame(t *testing.T) {
	ft := newFakeTap([]string{`{"id":123,"type":"identity","identity":{"did":"did:plc:a","status":"active"}}`})
	server := httptest.NewServer(ft.handler(t))
	defer server.Close()
	entered := make(chan struct{})
	ingestor := &durableIngestorSpy{identityFunc: func(context.Context, tap.IdentityEvent) (tap.Outcome, error) {
		defer close(entered)
		panicAtTapDiagnosticBoundary()
		return tap.Outcome{}, nil
	}}
	var logs bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor, Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Observer: observer})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { consumer.Run(ctx); close(done) }()
	select {
	case <-entered:
		cancel()
	case <-ctx.Done():
		t.Fatal("ingestion never reached panic")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer failed to stop")
	}
	observer.Flush(time.Second)
	for _, want := range []string{"runtime.TypeAssertionError", "int, not string", "panicAtTapDiagnosticBoundary"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("local Tap panic missing %s", want)
		}
	}
	events := transport.Events()
	if len(events) != 1 || events[0].Exception[0].Stacktrace == nil {
		t.Fatalf("panic issue/stack missing: %#v", events)
	}
	select {
	case id := <-ft.acks:
		t.Fatalf("panicking event acknowledged: %d", id)
	default:
	}
	if strings.Contains(logs.String(), "/Users/") {
		t.Fatal("local user path leaked")
	}
}

func TestIT016TapRetryBurstKeepsEveryAttemptAndTerminalSummary(t *testing.T) {
	frame := `{"id":1,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:public","collection":"social.craftsky.feed.post","rkey":"post1","action":"create","cid":"bafyPublic","record":{"text":"published explanation"}}}`
	frames := make([]string, 10)
	for i := range frames {
		frames[i] = frame
	}
	ft := newFakeTap(frames)
	server := httptest.NewServer(ft.handler(t))
	defer server.Close()
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	metrics := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, LogsEnabled: true, MetricRecorder: metrics})
	attempts := 0
	ingestor := &durableIngestorSpy{recordFunc: func(context.Context, tap.Event) (tap.Outcome, error) {
		attempts++
		if attempts < 10 {
			return tap.Retryable(tap.ReasonProjectionFailure), &pgconn.PgError{Code: "40001", Detail: "PRIVATE_ROW"}
		}
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor, Observer: observer, Logger: logger})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	select {
	case id := <-ft.acks:
		if id != 1 {
			t.Fatal("wrong terminal ACK")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("missing terminal ACK")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("consumer did not stop")
	}
	observer.Flush(time.Second)
	if attempts != 10 {
		t.Fatalf("attempts=%d", attempts)
	}
	failures := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(local.String()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record["operation"] == "tap.indexer.handle" && record["causes"] != nil {
			failures = append(failures, record)
		}
	}
	if len(failures) != 2 {
		t.Fatalf("selected failures=%d", len(failures))
	}
	if failures[1]["suppressed_count"] != float64(8) {
		t.Fatalf("terminal summary=%#v", failures[1])
	}
	issues := 0
	remote := 0
	for _, event := range transport.Events() {
		if len(event.Exception) > 0 {
			issues++
			payload, _ := json.Marshal(event)
			if !strings.Contains(string(payload), "published explanation") || !strings.Contains(string(payload), "quarantine") {
				t.Fatal("terminal cause lost")
			}
		}
		for _, log := range event.Logs {
			if log.Body == "indexer handle failed" {
				t.Error("expected retry retained duplicate ERROR noise")
			}
			if log.Attributes["operation"].AsString() == "tap.indexer.handle" {
				remote++
			}
		}
	}
	if issues != 1 || remote != 2 {
		t.Fatalf("issues=%d remote=%d", issues, remote)
	}
	if strings.Contains(local.String(), "PRIVATE_ROW") {
		t.Fatal("private row leaked")
	}
	total := 0.0
	for _, call := range metrics.Calls() {
		if call.Name == "craftsky_appview_tap_indexer_records_total" {
			total += call.Value
		}
	}
	if total != 10 {
		t.Fatalf("metric attempts=%v", total)
	}
}
func TestWSConsumerQuarantinePanicHasLocalCauseAndRecoveryFrame(t *testing.T) {
	ft := newFakeTap([]string{`{"id":123,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:a","collection":"not-an-nsid!","rkey":"k","action":"create","cid":"bafy","record":{}}}`})
	server := httptest.NewServer(ft.handler(t))
	defer server.Close()
	entered := make(chan struct{})
	ingestor := &durableIngestorSpy{quarantineFunc: func(context.Context, tap.InvalidEvent) (tap.Outcome, error) {
		defer close(entered)
		panicAtTapDiagnosticBoundary()
		return tap.Outcome{}, nil
	}}
	var logs bytes.Buffer
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor, Logger: slog.New(slog.NewJSONHandler(&logs, nil)), Observer: observer})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { consumer.Run(ctx); close(done) }()
	select {
	case <-entered:
		cancel()
	case <-ctx.Done():
		t.Fatal("ingestion never reached panic")
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("consumer failed to stop")
	}
	observer.Flush(time.Second)
	for _, want := range []string{"runtime.TypeAssertionError", "int, not string", "panicAtTapDiagnosticBoundary"} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("local Tap panic missing %s", want)
		}
	}
	events := transport.Events()
	if len(events) != 1 || events[0].Exception[0].Stacktrace == nil {
		t.Fatalf("panic issue/stack missing: %#v", events)
	}
	select {
	case id := <-ft.acks:
		t.Fatalf("panicking event acknowledged: %d", id)
	default:
	}
	if strings.Contains(logs.String(), "/Users/") {
		t.Fatal("local user path leaked")
	}
}
