package push

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-012: final dispatch sees a committed removal even when notification
// projection has not yet processed the newer authoritative source.
func TestPostEditRemovalBeforeProjectionSuppressesPush(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Now().UTC()
	seedDelivery(t, pool, "pending", now.Add(time.Hour))
	source, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	event := tap.Event{ID: 900, URI: "at://did:plc:actor/social.craftsky.feed.post/3aaaaaaaaaaa2", DID: "did:plc:actor", Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", CID: "cid-original", Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello viewer","sponsored":false,"createdAt":"2026-10-26T12:00:00Z","facets":[{"index":{"byteStart":6,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:viewer"}]}]}`)}
	if _, err := source.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE notification_events SET category='mention',subject_key=$1,source_uri=$1,source_cid=$2,source_rkey=$3`, event.URI, event.CID, event.Rkey); err != nil {
		t.Fatal(err)
	}
	event.ID++
	event.Action = "update"
	event.Rev = "3aaaaaaaaaaa3"
	event.CID = "cid-edited"
	event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello everyone","sponsored":false,"createdAt":"2026-10-26T12:00:00Z"}`)
	if _, err := source.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	var committedStatus string
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&committedStatus); err != nil {
		t.Fatal(err)
	}
	if committedStatus != "cancelled" {
		t.Fatalf("source commit left removed reason pending: %s", committedStatus)
	}
	now = time.Now().UTC()
	sender := &scriptedSender{}
	dispatcher := newTestDispatcher(t, pool, sender, DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1})
	if n, err := dispatcher.ProcessBatch(ctx, "worker"); err != nil || n != 0 {
		t.Fatalf("dispatch n=%d error=%v", n, err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 0 || status != "cancelled" {
		t.Fatalf("removed mention sent: requests=%d status=%s", len(sender.requests), status)
	}
}

// IT-012: a committed dispatch decision wins over a later removal; removal
// commits after its bounded provider call, rather than racing through it.
func TestPostEditRemovalWaitsForInFlightPublicationPush(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool := testdb.WithMigratedSchema(t)
	now := time.Now().UTC()
	seedDelivery(t, pool, "pending", now.Add(time.Hour))
	source, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	event := tap.Event{ID: 910, URI: "at://did:plc:actor/social.craftsky.feed.post/3aaaaaaaaaaa2", DID: "did:plc:actor", Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", CID: "cid-original", Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello viewer","sponsored":false,"createdAt":"2026-10-26T12:00:00Z","facets":[{"index":{"byteStart":6,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:viewer"}]}]}`)}
	if _, err := source.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE notification_events SET category='mention',subject_key=$1,source_uri=$1,source_cid=$2,source_rkey=$3`, event.URI, event.CID, event.Rkey); err != nil {
		t.Fatal(err)
	}
	sender := &blockingSender{started: make(chan int, 1), release: make(chan struct{}, 1)}
	defer func() {
		select {
		case sender.release <- struct{}{}:
		default:
		}
	}()
	dispatcher := newTestDispatcher(t, pool, sender, DispatcherOptions{Now: time.Now, BatchSize: 1})
	sent := make(chan error, 1)
	go func() { _, err := dispatcher.ProcessBatch(ctx, "worker"); sent <- err }()
	select {
	case <-sender.started:
	case <-ctx.Done():
		t.Fatal("provider did not start")
	}
	event.ID++
	event.Action = "update"
	event.Rev = "3aaaaaaaaaaa3"
	event.CID = "cid-edited"
	event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello everyone","sponsored":false,"createdAt":"2026-10-26T12:00:00Z"}`)
	removed := make(chan error, 1)
	go func() { _, err := source.IngestRecord(ctx, event); removed <- err }()
	select {
	case err := <-removed:
		t.Fatalf("removal raced past in-flight provider: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	sender.release <- struct{}{}
	if err := <-sent; err != nil {
		t.Fatal(err)
	}
	if err := <-removed; err != nil {
		t.Fatal(err)
	}
	if n, err := dispatcher.ProcessBatch(ctx, "worker"); err != nil || n != 0 {
		t.Fatalf("duplicate dispatch n=%d error=%v", n, err)
	}
}
