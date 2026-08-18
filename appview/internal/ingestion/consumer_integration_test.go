package ingestion_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

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
	frame := `{"id":401,"type":"record","record":{"live":true,"rev":"3m00000000401","did":"did:plc:crash","collection":"social.craftsky.actor.profile","rkey":"self","action":"create","cid":"bafy-crash","record":{"crafts":["sewing"]}}}`
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
