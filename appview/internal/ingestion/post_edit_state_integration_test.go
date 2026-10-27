package ingestion_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/postpublication"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// IT-010 / IT-014: observed time belongs to the current source version and
// survives process reconstruction, duplicate delivery and an older source event.
func TestPostEditIngestionPersistsObservedTimeAcrossRestart(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:post-edit-owner','profile')`); err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	event := tap.Event{ID: 1, URI: "at://did:plc:post-edit-owner/social.craftsky.feed.post/3aaaaaaaaaaa2", DID: "did:plc:post-edit-owner", Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", CID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`)}
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	first := event
	now = now.Add(24 * time.Hour)
	wantTime := now
	event.ID = 2
	event.CID = "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m"
	event.Rev = "3aaaaaaaaaaa3"
	event.Action = "update"
	event.Record = json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"after","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`)
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	assertCurrent := func() {
		t.Helper()
		var cid, revision, source string
		var editedAt time.Time
		var count int
		if err := pool.QueryRow(ctx, `SELECT source_cid,source_revision,edited_at,edit_time_source,(SELECT count(*) FROM craftsky_post_edit_state) FROM craftsky_post_edit_state WHERE uri=$1`, event.URI).Scan(&cid, &revision, &editedAt, &source, &count); err != nil {
			t.Fatal(err)
		}
		if cid != event.CID.String() || revision != event.Rev.String() || !editedAt.Equal(wantTime) || source != "observed" || count != 1 {
			t.Fatalf("current edit = %s %s %s %s rows=%d", cid, revision, editedAt, source, count)
		}
	}
	assertCurrent()
	now = now.Add(time.Hour)
	store, err = ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	first.ID = 3
	if _, err := store.IngestRecord(ctx, first); err != nil {
		t.Fatal(err)
	}
	assertCurrent()
}

// IT-014: authoritative read-only repository reconciliation must use the same
// transition logic as live source ingestion, before replacing the old record.
func TestPostEditReconciliationPersistsObservedTransition(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES('did:plc:post-edit-owner','profile')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:post-edit-owner','active',1,1,'test',$1,$1,$1)`, now); err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	service, err := ingestion.NewService(ingestion.ServiceConfig{Store: store, Lifecycles: lifecycles, ProfileParticipant: func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	event := tap.Event{ID: 1, URI: "at://did:plc:post-edit-owner/social.craftsky.feed.post/3aaaaaaaaaaa2", DID: "did:plc:post-edit-owner", Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", CID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"before","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`)}
	if _, err := service.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	current, err := store.Source(ctx, event.URI)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(24 * time.Hour)
	if _, err := service.ReconcileSource(ctx, ingestion.ReconciledSource{URI: event.URI, DID: event.DID, ExpectedEventID: current.SourceEventID, ExpectedFingerprint: current.SourceFingerprint, Revision: "3aaaaaaaaaaa3", CID: "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m", Present: true, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"after","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"}`)}); err != nil {
		t.Fatal(err)
	}
	var cid, revision string
	var editedAt *time.Time
	var source *string
	if err := pool.QueryRow(ctx, `SELECT source_cid,source_revision,edited_at,edit_time_source FROM craftsky_post_edit_state WHERE uri=$1`, event.URI).Scan(&cid, &revision, &editedAt, &source); err != nil {
		t.Fatal(err)
	}
	if cid != "bafyreif7dhriwzuzrl3gsrxjo3mzxt5jdls7yo3jwoashfx74og6n3ck4m" || revision != "3aaaaaaaaaaa3" || editedAt == nil || !editedAt.Equal(now) || source == nil || *source != "observed" {
		t.Fatalf("reconciled metadata=%s %s %v %v", cid, revision, editedAt, source)
	}
}

// REG-008 / RULE-006: a deleted post retains no current edit state or original
// recipient/reference data; the delivery tombstone cannot reactivate old work.
func TestPostDeletionClearsCurrentEditingFacts(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	event := tap.Event{ID: 20, URI: "at://did:plc:post-edit-owner/social.craftsky.feed.post/3aaaaaaaaaaa2", DID: "did:plc:post-edit-owner", Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", CID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello recipient","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}],"reply":{"root":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/3aaaaaaaaaaa4","cid":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"parent":{"uri":"at://did:plc:recipient/social.craftsky.feed.post/3aaaaaaaaaaa4","cid":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"}}}`)}
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	var initialFacts, initialMentions, initialEditStates int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM post_publication_state WHERE origin_status='established'),(SELECT count(*) FROM post_publication_mentions),(SELECT count(*) FROM craftsky_post_edit_state)`).Scan(&initialFacts, &initialMentions, &initialEditStates); err != nil {
		t.Fatal(err)
	}
	if initialFacts != 1 || initialMentions != 1 || initialEditStates != 1 {
		t.Fatalf("invalid deletion setup: %d/%d/%d", initialFacts, initialMentions, initialEditStates)
	}
	event.ID++
	event.Rev = "3aaaaaaaaaaa3"
	event.Action = "delete"
	event.Record = nil
	event.CID = ""
	if _, err := store.IngestRecord(ctx, event); err != nil {
		t.Fatal(err)
	}
	var editStates, mentions int
	var status string
	var clean bool
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM craftsky_post_edit_state),(SELECT count(*) FROM post_publication_mentions),origin_status,
 original_creation_time IS NULL AND parent_uri IS NULL AND parent_cid IS NULL AND root_uri IS NULL AND root_cid IS NULL AND quoted_uri IS NULL AND quoted_cid IS NULL AND NOT reply_eligible AND NOT quote_eligible
 FROM post_publication_state WHERE uri=$1`, event.URI).Scan(&editStates, &mentions, &status, &clean); err != nil {
		t.Fatal(err)
	}
	if editStates != 0 || mentions != 0 || status != "deleted" || !clean {
		t.Fatalf("deleted post retained editing facts: edits=%d recipients=%d status=%s clean=%v", editStates, mentions, status, clean)
	}
}

// REG-008 / RULE-006: late original-create acceptance cannot restore cleared
// facts after deletion, including when deletion was the first source observed.
func TestLateCreateProofCannotRestoreDeletedPublicationFacts(t *testing.T) {
	for _, seen := range []bool{false, true} {
		name := "delete_first"
		if seen {
			name = "known_post"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			pool := testdb.WithMigratedSchema(t)
			store, err := ingestion.NewStore(pool, time.Now)
			if err != nil {
				t.Fatal(err)
			}
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello recipient","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","facets":[{"index":{"byteStart":6,"byteEnd":15},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:recipient"}]}]}`)
			event := tap.Event{ID: 30, URI: "at://did:plc:post-edit-owner/social.craftsky.feed.post/3aaaaaaaaaaa2", DID: "did:plc:post-edit-owner", Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", CID: "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", Rev: "3aaaaaaaaaaa2", Action: "create", Live: true, Record: raw}
			if seen {
				if _, err := store.IngestRecord(ctx, event); err != nil {
					t.Fatal(err)
				}
			}
			event.ID++
			event.Rev = "3aaaaaaaaaaa3"
			event.Action = "delete"
			event.Record = nil
			event.CID = ""
			if _, err := store.IngestRecord(ctx, event); err != nil {
				t.Fatal(err)
			}
			if err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return postpublication.SeedAcceptedCreate(ctx, tx, event.URI, raw) }); err != nil {
				t.Fatal(err)
			}
			var restored, mentions int
			if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM post_publication_state WHERE origin_status='established' OR original_creation_time IS NOT NULL),(SELECT count(*) FROM post_publication_mentions)`).Scan(&restored, &mentions); err != nil {
				t.Fatal(err)
			}
			if restored != 0 || mentions != 0 {
				t.Fatalf("late create resurrected deleted facts: origins=%d recipients=%d", restored, mentions)
			}
		})
	}
}
