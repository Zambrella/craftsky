package ingestion_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

type commitThenDisconnectIngestor struct {
	store     *ingestion.Store
	committed chan struct{}
	dropped   chan struct{}
	calls     atomic.Int32
}

type quarantineCommitObserver struct {
	store *ingestion.Store
	pool  interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}
	committed chan struct{}
}

func (*quarantineCommitObserver) IngestRecord(context.Context, tap.Event) (tap.Outcome, error) {
	return tap.Retryable(tap.ReasonMalformedRecord), fmt.Errorf("invalid record unexpectedly reached source ingestion")
}

func (*quarantineCommitObserver) IngestIdentity(context.Context, tap.IdentityEvent) (tap.Outcome, error) {
	return tap.Retryable(tap.ReasonInvalidIdentity), fmt.Errorf("record unexpectedly reached identity ingestion")
}

func (ingestor *quarantineCommitObserver) Quarantine(ctx context.Context, event tap.InvalidEvent) (tap.Outcome, error) {
	outcome, err := ingestor.store.Quarantine(ctx, event)
	if err != nil {
		return outcome, err
	}
	var replayEnvelope []byte
	if err := ingestor.pool.QueryRow(ctx, `
		SELECT replay_envelope
		FROM tap_quarantined_events
		WHERE tap_event_id=$1
	`, event.ID).Scan(&replayEnvelope); err != nil {
		return tap.Retryable(tap.ReasonStorageUnavailable), fmt.Errorf("observe committed quarantine: %w", err)
	}
	if !bytes.Equal(replayEnvelope, event.Envelope) {
		return tap.Retryable(tap.ReasonStorageUnavailable), fmt.Errorf("committed replay envelope does not match Tap frame")
	}
	select {
	case ingestor.committed <- struct{}{}:
	default:
	}
	return outcome, nil
}

func (ingestor *commitThenDisconnectIngestor) IngestRecord(ctx context.Context, event tap.Event) (tap.Outcome, error) {
	outcome, err := ingestor.store.IngestRecord(ctx, event)
	if err == nil && ingestor.calls.Add(1) == 1 {
		close(ingestor.committed)
		select {
		case <-ingestor.dropped:
		case <-ctx.Done():
			return tap.Retryable(tap.ReasonStorageUnavailable), ctx.Err()
		}
	}
	return outcome, err
}

func (*commitThenDisconnectIngestor) IngestIdentity(context.Context, tap.IdentityEvent) (tap.Outcome, error) {
	return tap.Applied(), nil
}

func (ingestor *commitThenDisconnectIngestor) Quarantine(ctx context.Context, event tap.InvalidEvent) (tap.Outcome, error) {
	return ingestor.store.Quarantine(ctx, event)
}

func TestCommitBeforeAckDisconnectRedeliveryIsIdempotent(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	frame := `{"id":401,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa2","did":"did:plc:crash","collection":"social.craftsky.actor.profile","rkey":"self","action":"create","cid":"bafy-crash","record":{"crafts":["sewing"]}}}`
	committed := make(chan struct{})
	dropped := make(chan struct{})
	acks := make(chan uint64, 1)
	var connections atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		connection, err := websocket.Accept(w, request, nil)
		if err != nil {
			t.Errorf("accept websocket: %v", err)
			return
		}
		connectionNumber := connections.Add(1)
		if err := connection.Write(request.Context(), websocket.MessageText, []byte(frame)); err != nil {
			connection.CloseNow()
			return
		}
		if connectionNumber == 1 {
			<-committed
			connection.CloseNow()
			close(dropped)
			return
		}
		defer connection.CloseNow()
		var ack struct {
			Type string `json:"type"`
			ID   uint64 `json:"id"`
		}
		if err := wsjson.Read(request.Context(), connection, &ack); err == nil && ack.Type == "ack" {
			acks <- ack.ID
		}
	}))
	defer server.Close()

	ingestor := &commitThenDisconnectIngestor{store: store, committed: committed, dropped: dropped}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: 2 * time.Second, ReconnectMax: 20 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	select {
	case id := <-acks:
		if id != 401 {
			t.Fatalf("ack id=%d, want 401", id)
		}
	case <-ctx.Done():
		t.Fatal("Tap event was not acknowledged after redelivery")
	}
	cancel()
	<-done

	if ingestor.calls.Load() != 2 {
		t.Fatalf("ingestion calls=%d, want committed attempt plus redelivery", ingestor.calls.Load())
	}
	var sources, jobs, receipts int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_source_records`).Scan(&sources); err != nil {
		t.Fatalf("count sources: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_projection_jobs`).Scan(&jobs); err != nil {
		t.Fatalf("count projection jobs: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_ingestion_receipts`).Scan(&receipts); err != nil {
		t.Fatalf("count receipts: %v", err)
	}
	if sources != 1 || jobs != 1 || receipts != 1 {
		t.Fatalf("sources/jobs/receipts=%d/%d/%d, want 1/1/1", sources, jobs, receipts)
	}
}

func TestOversizedInvalidEnvelopeCommitsExactReplayPayloadBeforeAck(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	wireEnvelope := append([]byte(`{"id":402,"type":"record","record":{"live":true,"rev":"3aaaaaaaaaaa3","did":"did:plc:ack-quarantine","collection":"invalid collection","rkey":"oversized","action":"create","cid":"bafy-ack-quarantine","record":{"text":"`), bytes.Repeat([]byte("z"), 70_000)...)
	wireEnvelope = append(wireEnvelope, []byte(`"}}}`)...)
	if len(wireEnvelope) <= 64<<10 {
		t.Fatalf("wire envelope bytes=%d, want oversized frame", len(wireEnvelope))
	}

	committed := make(chan struct{}, 1)
	acks := make(chan uint64, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		connection, acceptErr := websocket.Accept(w, request, nil)
		if acceptErr != nil {
			t.Errorf("accept websocket: %v", acceptErr)
			return
		}
		defer connection.CloseNow()
		if writeErr := connection.Write(request.Context(), websocket.MessageText, wireEnvelope); writeErr != nil {
			t.Errorf("write Tap frame: %v", writeErr)
			return
		}
		var ack struct {
			Type string `json:"type"`
			ID   uint64 `json:"id"`
		}
		if readErr := wsjson.Read(request.Context(), connection, &ack); readErr != nil {
			t.Errorf("read Tap ack: %v", readErr)
			return
		}
		select {
		case <-committed:
		default:
			t.Error("Tap ACK arrived before exact quarantine payload committed")
		}
		if ack.Type == "ack" {
			acks <- ack.ID
		}
	}))
	defer server.Close()

	ingestor := &quarantineCommitObserver{store: store, pool: pool, committed: committed}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: strings.Replace(server.URL, "http://", "ws://", 1), Ingestor: ingestor,
		AckTimeout: 2 * time.Second, ReconnectMax: 20 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	select {
	case id := <-acks:
		if id != 402 {
			t.Fatalf("ack id=%d, want 402", id)
		}
	case <-ctx.Done():
		t.Fatal("invalid Tap event was not acknowledged after exact quarantine commit")
	}
	cancel()
	<-done

	items, err := store.ListQuarantine(context.Background(), 1)
	if err != nil || len(items) != 1 {
		t.Fatalf("list quarantine=%+v err=%v", items, err)
	}
	if bytes.Equal(items[0].Envelope, wireEnvelope) || bytes.Contains(items[0].Envelope, bytes.Repeat([]byte("z"), 1024)) {
		t.Fatal("operator listing exposed the exact oversized replay payload")
	}
	if err := store.RequestQuarantineReplay(context.Background(), items[0].Fingerprint); err != nil {
		t.Fatalf("request replay: %v", err)
	}
	claims, err := store.ClaimQuarantine(context.Background(), ingestion.QuarantineClaimRequest{
		Worker: "quarantine-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim quarantine=%+v err=%v", claims, err)
	}
	replayed := false
	err = store.ReplayQuarantine(context.Background(), claims[0], func(operationCtx context.Context, envelope []byte) (tap.Outcome, error) {
		replayed = true
		if !bytes.Equal(envelope, wireEnvelope) {
			t.Fatalf("leased replay bytes=%d, want exact %d-byte Tap frame", len(envelope), len(wireEnvelope))
		}
		return tap.ReplayEnvelope(operationCtx, envelope, &replayDurableIngestor{})
	})
	if err == nil || !replayed {
		t.Fatalf("invalid replay called=%t err=%v, want decoder validation failure", replayed, err)
	}
}
