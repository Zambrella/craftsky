// appview/internal/app/indexer_wiring_test.go
package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const indexerWiringDDL = `
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
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
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
    did         TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
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
    did         TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
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

type noopPDSClient struct{}

func (noopPDSClient) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", nil
}

func (noopPDSClient) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return nil
}

func (noopPDSClient) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", nil
}

func (noopPDSClient) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return nil
}

func (noopPDSClient) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, nil
}

func TestNewIndexerDispatcherRegistersCraftskyInteractions(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, indexerWiringDDL)
	seedIndexerWiringData(t, pool)
	dispatcher := newIndexerDispatcher(pool, noopPDSClient{}, slog.Default())

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
			if err := dispatcher.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle through dispatcher: %v", err)
			}

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
	dispatcher := newIndexerDispatcher(pool, noopPDSClient{}, slog.Default())

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
	if err := dispatcher.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle through dispatcher: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM atproto_follows").Scan(&count); err != nil {
		t.Fatalf("count follows: %v", err)
	}
	if count != 1 {
		t.Errorf("atproto_follows count = %d; want 1", count)
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
