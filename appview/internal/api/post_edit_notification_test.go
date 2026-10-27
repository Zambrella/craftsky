package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-012: locally accepted removal restricts pending original delivery in the
// command transaction, before the corresponding Tap source update arrives.
func TestEditPostAcceptanceCancelsRemovedMentionBeforeTap(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		name := "tracked"
		if legacy {
			name = "predates-publication-facts"
		}
		t.Run(name, func(t *testing.T) { testEditPostAcceptanceCancelsRemovedMentionBeforeTap(t, legacy) })
	}
}

func testEditPostAcceptanceCancelsRemovedMentionBeforeTap(t *testing.T, legacy bool) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	const uri = syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/3aaaaaaaaaaa2")
	const priorCID = syntax.CID("bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq")
	raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello recipient","sponsored":false,"createdAt":"2026-10-26T12:00:00Z","facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]}`)
	source, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := source.IngestRecord(ctx, tap.Event{ID: 1, URI: uri, DID: "did:plc:alice", Collection: "social.craftsky.feed.post", Rkey: uri.RecordKey(), CID: priorCID, Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: raw}); err != nil {
		t.Fatal(err)
	}
	if legacy {
		if _, err := pool.Exec(ctx, `DELETE FROM post_publication_state WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO push_installations(id,device_id,platform,fcm_token) VALUES('00000000-0000-0000-0000-000000000001','edit-install','ios','edit-token');
 INSERT INTO push_account_subscriptions(id,installation_id,account_did,routing_id) VALUES('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','did:plc:recipient','00000000-0000-0000-0000-000000000003')`); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := notifications.ActivatePublication(ctx, notifications.NewService(), tx, notifications.Activation{RecipientDID: "did:plc:recipient", ActorDID: "did:plc:alice", Category: notifications.Mention, SubjectKey: uri.String(), SourceURI: uri, SourceCID: priorCID, SourceRkey: uri.RecordKey(), ActivityAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	commands := &recordingPostEditCommands{pool: pool, current: pdscommands.AuthoritativeRecord{URI: uri, CID: priorCID, Record: raw}}
	handler := api.CommandEditPostHandler(&fakePostStore{}, commands, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits())
	request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3aaaaaaaaaaa2", strings.NewReader(`{"changes":{"text":"hello everyone","facets":null}}`))
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", uri.RecordKey().String())
	request.Header.Set("If-Match", priorCID.String())
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	requestCtx := middleware.WithDID(ctx, "did:plc:alice")
	requestCtx = middleware.WithOwnerGeneration(requestCtx, 1)
	requestCtx = middleware.WithOAuthSessionID(requestCtx, "session-alice")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(requestCtx))
	if response.Code != 200 {
		t.Fatalf("edit status=%d body=%s", response.Code, response.Body.String())
	}
	var status, sourceCID string
	var eligible bool
	if err := pool.QueryRow(ctx, `SELECT d.status,m.eligible,s.cid FROM push_deliveries d CROSS JOIN post_publication_mentions m JOIN tap_source_records s ON s.uri=m.post_uri`).Scan(&status, &eligible, &sourceCID); err != nil {
		t.Fatal(err)
	}
	if status != "cancelled" || eligible || sourceCID != priorCID.String() {
		t.Fatalf("acceptance waited for Tap: status=%s eligible=%v sourceCID=%s", status, eligible, sourceCID)
	}
}

// IT-012 / REG-007: normal create wiring seeds original eligibility in its
// accepted transaction before any source record can be observed or coalesced.
func TestCreatePostAcceptanceSeedsPublicationBeforeTap(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	commands := &recordingAppendCommands{}
	commands.execute = func(request pdscommands.AppendCommandRequest) (pdscommands.CommandResult, error) {
		raw, err := request.BuildRecord(now)
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		record := pdscommands.AuthoritativeRecord{URI: "at://did:plc:alice/social.craftsky.feed.post/3aaaaaaaaaaa4", CID: "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m", Record: raw}
		terminal, err := request.Accepted(record)
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		if request.FinalizeAccepted != nil {
			err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
				return request.FinalizeAccepted(ctx, tx, pdscommands.PreparedCommand{CreatedAt: now}, record)
			})
		}
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	handler := api.CreatePostHandler(&fakePostStore{}, nil, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger(), api.CreatePostHandlerOptions{Commands: commands})
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello recipient","sponsored":false,"facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]}`))
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	requestCtx := middleware.WithDID(ctx, "did:plc:alice")
	requestCtx = middleware.WithOwnerGeneration(requestCtx, 1)
	requestCtx = middleware.WithOAuthSessionID(requestCtx, "session-alice")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(requestCtx))
	if response.Code != 201 {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body.String())
	}
	var facts, mentions, sources int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM post_publication_state WHERE origin_status='established'),(SELECT count(*) FROM post_publication_mentions WHERE recipient_did='did:plc:recipient' AND eligible),(SELECT count(*) FROM tap_source_records)`).Scan(&facts, &mentions, &sources); err != nil {
		t.Fatal(err)
	}
	if facts != 1 || mentions != 1 || sources != 0 {
		t.Fatalf("create waited for Tap: facts=%d mentions=%d sources=%d", facts, mentions, sources)
	}
}
