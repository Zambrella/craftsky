// appview/internal/app/indexer_wiring_test.go
package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const indexerWiringDDL = `
	CREATE TABLE owner_lifecycles (
	    owner_did TEXT PRIMARY KEY,
	    state TEXT NOT NULL,
	    generation BIGINT NOT NULL,
	    auth_epoch BIGINT NOT NULL,
	    transition_reason TEXT NOT NULL,
	    transitioned_at TIMESTAMPTZ NOT NULL,
	    terminal_at TIMESTAMPTZ,
	    purge_completed_at TIMESTAMPTZ,
	    created_at TIMESTAMPTZ NOT NULL,
	    updated_at TIMESTAMPTZ NOT NULL
	);
	CREATE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
	RETURNS BOOLEAN
	LANGUAGE SQL
	STABLE
	AS $$
	    SELECT COALESCE((
	        SELECT state = 'terminal'
	        FROM owner_lifecycles
	        WHERE owner_did = candidate_did
	    ), false)
	$$;
	CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_follows (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);
CREATE TABLE atproto_blocks (
    uri         TEXT        NOT NULL PRIMARY KEY,
    blocker_did TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_did, rkey)
);
CREATE INDEX atproto_blocks_blocker_subject_idx ON atproto_blocks (blocker_did, subject_did);
CREATE INDEX atproto_blocks_subject_blocker_idx ON atproto_blocks (subject_did, blocker_did);
CREATE TABLE notification_events (
    id UUID PRIMARY KEY,
    recipient_did TEXT NOT NULL,
    actor_did TEXT NOT NULL
);
CREATE TABLE push_deliveries (
    notification_id UUID NOT NULL REFERENCES notification_events(id),
    status TEXT NOT NULL,
    lease_owner TEXT,
    lease_expires_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,
    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,
    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,
    quote_uri        TEXT,
    quote_cid        TEXT,
    tags             TEXT[]      NOT NULL DEFAULT '{}',
    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_likes (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_uri TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE UNIQUE INDEX craftsky_likes_did_subject_uri_active_unique
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;
CREATE TABLE craftsky_reposts (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_uri TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE UNIQUE INDEX craftsky_reposts_did_subject_uri_active_unique
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;
`

func projectIndexerWiringEvent(
	t *testing.T,
	pool *pgxpool.Pool,
	dispatcher *index.TransactionalDispatcher,
	event tap.Event,
) {
	t.Helper()
	ctx := context.Background()
	generation := int64(1)
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',$2,1,'test',now(),now(),now())
		ON CONFLICT (owner_did) DO NOTHING
	`, event.DID, generation); err != nil {
		t.Fatalf("seed projector owner lifecycle: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin projector transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	outcome, err := dispatcher.Project(ctx, tx, ingestion.SourceRecord{
		URI: event.URI, DID: event.DID, Collection: event.Collection,
		Rkey: event.Rkey, SourceEventID: event.ID, Revision: event.Rev,
		CID: event.CID, Action: event.Action, Record: event.Record,
		RecordBytes: len(event.Record), Live: event.Live,
		OwnerGeneration: &generation,
	})
	if err != nil {
		t.Fatalf("Project through transactional dispatcher: %v", err)
	}
	if outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("projector outcome=%+v, want applied", outcome)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit projector transaction: %v", err)
	}
}

func TestNewIndexerDispatcherRegistersCraftskyInteractions(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, indexerWiringDDL)
	seedIndexerWiringData(t, pool)
	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(
		pool, slog.Default(), nil,
		notifications.NoopLifecycle{}, notifications.NewActorDeletionService(pool),
	)

	for _, tc := range []struct {
		name       string
		collection syntax.NSID
		table      string
		rkey       syntax.RecordKey
	}{
		{name: "like", collection: "social.craftsky.feed.like", table: "craftsky_likes", rkey: "like1"},
		{name: "repost", collection: "social.craftsky.feed.repost", table: "craftsky_reposts", rkey: "repost1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev := tap.Event{
				URI:        syntax.ATURI("at://did:plc:actor/" + tc.collection.String() + "/" + tc.rkey.String()),
				CID:        syntax.CID("bafy" + tc.rkey.String()),
				DID:        "did:plc:actor",
				Collection: tc.collection,
				Rkey:       tc.rkey,
				Action:     "create",
				Record: json.RawMessage(`{
					"createdAt": "2026-05-04T12:00:00Z",
					"subject": {"uri": "at://did:plc:author/social.craftsky.feed.post/post1", "cid": "subjectcid"}
				}`),
			}
			projectIndexerWiringEvent(t, pool, dispatcher, ev)

			var count int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+tc.table).Scan(&count); err != nil {
				t.Fatalf("count %s: %v", tc.table, err)
			}
			if count != 1 {
				t.Errorf("%s count = %d, want 1", tc.table, count)
			}
		})
	}
}

func TestNewIndexerDispatcherRegistersBlueskyFollow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, indexerWiringDDL)
	seedIndexerWiringData(t, pool)
	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(
		pool, slog.Default(), nil,
		notifications.NoopLifecycle{}, notifications.NewActorDeletionService(pool),
	)

	ev := tap.Event{
		URI:        "at://did:plc:actor/app.bsky.graph.follow/follow1",
		CID:        "bafyfollow1",
		DID:        "did:plc:actor",
		Collection: "app.bsky.graph.follow",
		Rkey:       "follow1",
		Action:     "create",
		Record: json.RawMessage(`{
			"subject": "did:plc:author",
			"createdAt": "2026-05-04T12:00:00Z"
		}`),
	}
	projectIndexerWiringEvent(t, pool, dispatcher, ev)

	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM atproto_follows").Scan(&count); err != nil {
		t.Fatalf("count follows: %v", err)
	}
	if count != 1 {
		t.Errorf("atproto_follows count = %d; want 1", count)
	}
}

func TestBlockCollectionIsDispatchedAndConfiguredExactlyOnce(t *testing.T) {
	pool := testdb.WithSchema(t, indexerWiringDDL)
	seedIndexerWiringData(t, pool)
	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(
		pool, slog.Default(), nil,
		notifications.NoopLifecycle{}, notifications.NewActorDeletionService(pool),
	)

	ev := tap.Event{
		URI:        "at://did:plc:actor/app.bsky.graph.block/block1",
		CID:        "bafyblock1",
		DID:        "did:plc:actor",
		Collection: "app.bsky.graph.block",
		Rkey:       "block1",
		Action:     "create",
		Record: json.RawMessage(`{
			"$type": "app.bsky.graph.block",
			"subject": "did:plc:author",
			"createdAt": "2026-07-19T12:00:00Z"
		}`),
	}
	projectIndexerWiringEvent(t, pool, dispatcher, ev)
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM atproto_blocks`).Scan(&count); err != nil {
		t.Fatalf("count blocks: %v", err)
	}
	if count != 1 {
		t.Fatalf("atproto_blocks count = %d, want 1", count)
	}

	compose, err := os.ReadFile("../../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read docker-compose.yml: %v", err)
	}
	if got := strings.Count(string(compose), "app.bsky.graph.block"); got != 1 {
		t.Fatalf("docker-compose app.bsky.graph.block occurrences = %d, want exactly 1", got)
	}
	if strings.Contains(string(compose), "social.craftsky.graph.block") {
		t.Fatal("docker-compose configured a local Craftsky block collection")
	}
}

func TestTransactionalIndexerDispatcherRegistersCraftskyProfile(t *testing.T) {
	pool := testdb.WithSchema(t, indexerWiringDDL)
	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(
		pool, slog.Default(), nil,
		notifications.NoopLifecycle{}, notifications.NewActorDeletionService(pool),
	)
	ev := tap.Event{
		URI:        "at://did:plc:joining/social.craftsky.actor.profile/self",
		CID:        "bafy-joining",
		DID:        "did:plc:joining",
		Collection: "social.craftsky.actor.profile",
		Rkey:       "self",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["sewing"]}`),
	}
	projectIndexerWiringEvent(t, pool, dispatcher, ev)
	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM craftsky_profiles WHERE did = $1
	`, ev.DID).Scan(&count); err != nil {
		t.Fatalf("count profile: %v", err)
	}
	if count != 1 {
		t.Fatalf("craftsky profile count = %d, want 1", count)
	}
}

func seedIndexerWiringData(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	createdAt := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:actor', 'actorcid'), ('did:plc:author', 'authorcid')
	`); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, record, created_at)
		VALUES ('at://did:plc:author/social.craftsky.feed.post/post1', 'did:plc:author', 'post1', 'subjectcid', 'subject', '{}', $1)
	`, createdAt); err != nil {
		t.Fatalf("seed post: %v", err)
	}
}
