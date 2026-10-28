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
func TestPostEditRemovalKeepsOriginalPendingPush(t *testing.T) {
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
	if committedStatus != "pending" {
		t.Fatalf("edit changed original delivery: %s", committedStatus)
	}
	now = time.Now().UTC()
	sender := &scriptedSender{}
	dispatcher := newTestDispatcher(t, pool, sender, DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1})
	if n, err := dispatcher.ProcessBatch(ctx, "worker"); err != nil || n != 1 {
		t.Fatalf("dispatch n=%d error=%v", n, err)
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if len(sender.requests) != 1 || status != "succeeded" {
		t.Fatalf("original delivery lost: requests=%d status=%s", len(sender.requests), status)
	}

}

// IT-012/IT-013/IT-014 / IR-008: recipient retention does not bypass source validity.
func TestPostEditInvalidCurrentSourceCannotDispatchOriginalPush(t *testing.T) {
	for _, kind := range []string{"malformed", "semanticInvalid", "uncertain", "deleted", "missing"} {
		t.Run(kind, func(t *testing.T) {
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
			event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello everyone","sponsored":false,"createdAt":"2026-10-26T12:00:00Z","editedAt":"not-a-timestamp"}`)
			if kind != "malformed" {
				event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello everyone","sponsored":false,"createdAt":"2026-10-26T12:00:00Z"}`)
			}
			if _, err := source.IngestRecord(ctx, event); err != nil {
				t.Fatal(err)
			}
			if kind != "malformed" {
				var query string
				switch kind {
				case "semanticInvalid":
					query = `UPDATE tap_source_records SET semantic_validation_status='invalid' WHERE uri=$1`
				case "uncertain":
					query = `UPDATE tap_source_records SET ordering_status='uncertain' WHERE uri=$1`
				case "deleted":
					query = `UPDATE tap_source_records SET action='delete',record=NULL,record_bytes=0 WHERE uri=$1`
				case "missing":
					query = `DELETE FROM tap_source_records WHERE uri=$1`
				}
				if _, err := pool.Exec(ctx, query, event.URI); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "malformed" {
				var structural, semantic string
				if err := pool.QueryRow(ctx, `SELECT structural_validation_status,semantic_validation_status FROM tap_source_records WHERE uri=$1`, event.URI).Scan(&structural, &semantic); err != nil {
					t.Fatal(err)
				}
				if structural != "invalid" || semantic != "invalid" {
					t.Fatalf("reproduction requires an invalid current source: %s/%s", structural, semantic)
				}
			}
			var committedStatus string
			if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&committedStatus); err != nil {
				t.Fatal(err)
			}
			if committedStatus != "pending" {
				t.Fatalf("edit changed original delivery: %s", committedStatus)
			}
			now = time.Now().UTC()
			sender := &scriptedSender{}
			dispatcher := newTestDispatcher(t, pool, sender, DispatcherOptions{Now: func() time.Time { return now }, BatchSize: 1})
			if n, err := dispatcher.ProcessBatch(ctx, "worker"); err != nil || n != 1 {
				t.Fatalf("dispatch n=%d error=%v", n, err)
			}
			var status string
			if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
				t.Fatal(err)
			}
			if len(sender.requests) != 0 || status != "cancelled" {
				t.Fatalf("invalid current source was dispatched: requests=%d status=%s", len(sender.requests), status)
			}

		})
	}
}
