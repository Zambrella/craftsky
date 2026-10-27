package pdscommands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/postpublication"
)

// IT-012 / REG-007: original facts commit with accepted append commands,
// including lost-response recovery, before any Tap source exists.
func TestPostCreateAcceptanceSeedsOriginalFactsBeforeTap(t *testing.T) {
	for _, lost := range []bool{false, true} {
		name := "acknowledged"
		if lost {
			name = "lost_response"
		}
		t.Run(name, func(t *testing.T) {
			now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
			addressed, pds, base := postEditCommandFixture(t, &now)
			service, err := NewAppendCommandService(AppendCommandServiceConfig{Store: addressed.store, Lifecycles: addressed.lifecycles, NewBoundary: addressed.newBoundary, NewRecordKey: func() (syntax.RecordKey, error) { return "3aaaaaaaaaaa4", nil }, Now: func() time.Time { return now }})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			request := AppendCommandRequest{Owner: base.Owner, OwnerGeneration: 1, SessionID: base.SessionID, OperationKind: "post.create", OperationKey: uuid.New(), Collection: "social.craftsky.feed.post", Intent: json.RawMessage(`{"text":"hello recipient"}`),
				BuildRecord: func(created time.Time) (json.RawMessage, error) {
					return json.Marshal(map[string]any{"$type": "social.craftsky.feed.post", "text": "hello recipient", "sponsored": false, "createdAt": created.UTC().Format(time.RFC3339Nano), "facets": []any{map[string]any{"index": map[string]any{"byteStart": 6, "byteEnd": 15}, "features": []any{map[string]any{"$type": "app.bsky.richtext.facet#mention", "did": "did:plc:recipient"}}}}})
				},
				Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
					return TerminalResult{State: CommandAccepted, HTTPStatus: 201, ResponseBody: record.Record}, nil
				}, Rejected: setCommandRejectedResult,
				FinalizeAccepted: func(ctx context.Context, tx pgx.Tx, prepared PreparedCommand, record AuthoritativeRecord) error {
					calls++
					if prepared.ID == uuid.Nil {
						t.Fatal("missing durable create command")
					}
					return postpublication.SeedAcceptedCreate(ctx, tx, record.URI, record.Record)
				},
			}
			pds.loseNextResponse = lost
			result, err := service.Execute(context.Background(), request)
			if lost {
				if err != nil || result.State != CommandAmbiguous || calls != 0 {
					t.Fatalf("lost create=%+v %v calls=%d", result, err, calls)
				}
				result, err = service.Execute(context.Background(), request)
			}
			if err != nil || result.State != CommandAccepted || calls != 1 || pds.applyCalls != 1 {
				t.Fatalf("create acceptance=%+v %v finalizers=%d writes=%d", result, err, calls, pds.applyCalls)
			}
			var origins, mentions, sources int
			if err := addressed.store.pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM post_publication_state WHERE origin_status='established'),(SELECT count(*) FROM post_publication_mentions WHERE recipient_did='did:plc:recipient' AND eligible),(SELECT count(*) FROM tap_source_records)`).Scan(&origins, &mentions, &sources); err != nil {
				t.Fatal(err)
			}
			if origins != 1 || mentions != 1 || sources != 0 {
				t.Fatalf("publication facts=%d mentions=%d sources=%d", origins, mentions, sources)
			}
			replay, err := service.Execute(context.Background(), request)
			if err != nil || replay.State != CommandAccepted || calls != 1 || pds.applyCalls != 1 {
				t.Fatalf("replay repeated create finalization: %+v %v calls=%d writes=%d", replay, err, calls, pds.applyCalls)
			}
		})
	}
}
