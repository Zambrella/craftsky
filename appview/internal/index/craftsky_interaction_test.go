// appview/internal/index/craftsky_interaction_test.go
package index_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const craftskyInteractionsDDL = craftskyPostsDDL + `
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
CREATE INDEX craftsky_likes_active_subject_uri
    ON craftsky_likes (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_active_did_subject_uri
    ON craftsky_likes (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_likes_indexed_at_desc
    ON craftsky_likes (indexed_at DESC);

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
CREATE INDEX craftsky_reposts_active_subject_uri
    ON craftsky_reposts (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_did_subject_uri
    ON craftsky_reposts (did, subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_indexed_at_desc
    ON craftsky_reposts (indexed_at DESC);
`

type interactionIndexerCase struct {
	name       string
	collection string
	table      string
	typeID     string
	newIndexer func(*pgxpool.Pool) index.Indexer
}

func interactionIndexerCases() []interactionIndexerCase {
	return []interactionIndexerCase{
		{
			name:       "like",
			collection: "social.craftsky.feed.like",
			table:      "craftsky_likes",
			typeID:     "social.craftsky.feed.like",
			newIndexer: func(pool *pgxpool.Pool) index.Indexer {
				return index.NewCraftskyLike(pool, testLogger())
			},
		},
		{
			name:       "repost",
			collection: "social.craftsky.feed.repost",
			table:      "craftsky_reposts",
			typeID:     "social.craftsky.feed.repost",
			newIndexer: func(pool *pgxpool.Pool) index.Indexer {
				return index.NewCraftskyRepost(pool, testLogger())
			},
		},
	}
}

func seedCraftskySubjectPost(t *testing.T, pool *pgxpool.Pool, uri string) {
	t.Helper()
	seedCraftskyMember(t, pool, "did:plc:author")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts
			(uri, did, rkey, cid, text, record, created_at)
		VALUES ($1, 'did:plc:author', 'post1', 'subjectcid', 'subject', '{}', $2)
	`, uri, testTime(t)); err != nil {
		t.Fatalf("seed craftsky_posts: %v", err)
	}
}

func interactionEvent(tc interactionIndexerCase, rkey, cid, subjectURI, subjectCID string) tap.Event {
	record := fmt.Sprintf(`{
		"$type": %q,
		"createdAt": %q,
		"subject": {"uri": %q, "cid": %q}
	}`, tc.typeID, fixedCreatedAt, subjectURI, subjectCID)
	return tap.Event{
		URI:        syntax.ATURI("at://did:plc:actor/" + tc.collection + "/" + rkey),
		CID:        syntax.CID(cid),
		DID:        "did:plc:actor",
		Rkey:       syntax.RecordKey(rkey),
		Collection: syntax.NSID(tc.collection),
		Action:     "create",
		Record:     json.RawMessage(record),
	}
}

func TestCraftskyInteraction_CreateAndDuplicateDelivery(t *testing.T) {
	for _, tc := range interactionIndexerCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, craftskyInteractionsDDL)
			seedCraftskyMember(t, pool, "did:plc:actor")
			seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
			idx := tc.newIndexer(pool)

			ev := interactionEvent(tc, "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			if err := idx.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle create: %v", err)
			}
			if err := idx.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle duplicate delivery: %v", err)
			}

			var (
				count      int
				uri        string
				cid        string
				subjectURI string
				subjectCID string
				createdAt  time.Time
				deletedAt  *time.Time
			)
			err := pool.QueryRow(context.Background(), fmt.Sprintf(`
				SELECT count(*), max(uri), max(cid), max(subject_uri), max(subject_cid), max(created_at), max(deleted_at)
				FROM %s
			`, tc.table)).Scan(&count, &uri, &cid, &subjectURI, &subjectCID, &createdAt, &deletedAt)
			if err != nil {
				t.Fatalf("select: %v", err)
			}
			if count != 1 {
				t.Fatalf("count = %d, want 1", count)
			}
			if uri != string(ev.URI) || cid != "bafy1" || subjectURI != "at://did:plc:author/social.craftsky.feed.post/post1" || subjectCID != "subjectcid" {
				t.Errorf("stored row = (%q,%q,%q,%q)", uri, cid, subjectURI, subjectCID)
			}
			if !createdAt.Equal(testTime(t)) {
				t.Errorf("created_at = %v, want %v", createdAt, testTime(t))
			}
			if deletedAt != nil {
				t.Errorf("deleted_at = %v, want nil", deletedAt)
			}
		})
	}
}

func TestCraftskyInteraction_ReplacesOtherActiveRowForSameActorSubject(t *testing.T) {
	for _, tc := range interactionIndexerCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, craftskyInteractionsDDL)
			seedCraftskyMember(t, pool, "did:plc:actor")
			seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
			idx := tc.newIndexer(pool)

			first := interactionEvent(tc, "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			second := interactionEvent(tc, "r2", "bafy2", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			if err := idx.Handle(context.Background(), first); err != nil {
				t.Fatalf("Handle first: %v", err)
			}
			if err := idx.Handle(context.Background(), second); err != nil {
				t.Fatalf("Handle second: %v", err)
			}

			var activeURI string
			if err := pool.QueryRow(context.Background(), fmt.Sprintf(`
				SELECT uri FROM %s WHERE deleted_at IS NULL
			`, tc.table)).Scan(&activeURI); err != nil {
				t.Fatalf("select active: %v", err)
			}
			if activeURI != string(second.URI) {
				t.Errorf("active uri = %q, want %q", activeURI, second.URI)
			}

			var firstDeleted bool
			if err := pool.QueryRow(context.Background(), fmt.Sprintf(`
				SELECT deleted_at IS NOT NULL FROM %s WHERE uri = $1
			`, tc.table), first.URI).Scan(&firstDeleted); err != nil {
				t.Fatalf("select old row: %v", err)
			}
			if !firstDeleted {
				t.Error("first interaction was not soft-deleted")
			}
		})
	}
}

func TestCraftskyInteraction_DeleteSoftDeletes(t *testing.T) {
	for _, tc := range interactionIndexerCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, craftskyInteractionsDDL)
			seedCraftskyMember(t, pool, "did:plc:actor")
			seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
			idx := tc.newIndexer(pool)

			ev := interactionEvent(tc, "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			if err := idx.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle create: %v", err)
			}
			ev.Action = "delete"
			ev.Record = nil
			if err := idx.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle delete: %v", err)
			}

			var deleted bool
			if err := pool.QueryRow(context.Background(), fmt.Sprintf(`
				SELECT deleted_at IS NOT NULL FROM %s WHERE uri = $1
			`, tc.table), ev.URI).Scan(&deleted); err != nil {
				t.Fatalf("select: %v", err)
			}
			if !deleted {
				t.Error("deleted_at is null, want soft-delete timestamp")
			}
		})
	}
}

func TestCraftskyInteraction_NonMemberIgnored(t *testing.T) {
	for _, tc := range interactionIndexerCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, craftskyInteractionsDDL)
			seedCraftskySubjectPost(t, pool, "at://did:plc:author/social.craftsky.feed.post/post1")
			idx := tc.newIndexer(pool)

			ev := interactionEvent(tc, "r1", "bafy1", "at://did:plc:author/social.craftsky.feed.post/post1", "subjectcid")
			if err := idx.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle: %v", err)
			}

			assertInteractionCount(t, pool, tc.table, 0)
		})
	}
}

func TestCraftskyInteraction_MissingSubjectPostIgnored(t *testing.T) {
	for _, tc := range interactionIndexerCases() {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pool := testdb.WithSchema(t, craftskyInteractionsDDL)
			seedCraftskyMember(t, pool, "did:plc:actor")
			idx := tc.newIndexer(pool)

			ev := interactionEvent(tc, "r1", "bafy1", "at://did:plc:missing/social.craftsky.feed.post/post1", "subjectcid")
			if err := idx.Handle(context.Background(), ev); err != nil {
				t.Fatalf("Handle: %v", err)
			}

			assertInteractionCount(t, pool, tc.table, 0)
		})
	}
}

func assertInteractionCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), fmt.Sprintf(`SELECT count(*) FROM %s`, table)).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Errorf("%s count = %d, want %d", table, count, want)
	}
}
