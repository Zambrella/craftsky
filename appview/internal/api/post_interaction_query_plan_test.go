package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const postInteractionQueryPlanIndexesDDL = `
CREATE INDEX craftsky_likes_active_subject_uri
    ON craftsky_likes (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_reposts_active_subject_uri
    ON craftsky_reposts (subject_uri) WHERE deleted_at IS NULL;
CREATE INDEX craftsky_posts_quote_uri
    ON craftsky_posts (quote_uri) WHERE quote_uri IS NOT NULL;
`

// IT-008: NFR-001; AC-019.
func TestPostInteractionListQueriesUseSubjectAndQuoteIndexes(t *testing.T) {
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL+postInteractionQueryPlanIndexesDDL)
	ctx := context.Background()
	const targetURI = "at://did:plc:owner/social.craftsky.feed.post/target"

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:viewer', 'viewer-cid'), ('did:plc:owner', 'owner-cid');
		INSERT INTO craftsky_profiles (did, record_cid)
		SELECT 'did:plc:actor' || n, 'actor-cid-' || n
		FROM generate_series(1, 1200) AS n;
		INSERT INTO atproto_identity_cache (did, handle, handle_lower)
		SELECT 'did:plc:actor' || n, 'actor' || n || '.test', 'actor' || n || '.test'
		FROM generate_series(1, 1200) AS n;
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at)
		VALUES ('at://did:plc:owner/social.craftsky.feed.post/target', 'did:plc:owner', 'target', 'target-cid', 'target', '{}', '2026-09-06T00:00:00Z');
		INSERT INTO craftsky_likes (uri, did, rkey, cid, subject_uri, subject_cid, record, created_at)
		SELECT 'at://did:plc:actor' || n || '/social.craftsky.feed.like/' || n,
		       'did:plc:actor' || n, n::text, 'like-cid-' || n, 'at://did:plc:owner/social.craftsky.feed.post/target', 'target-cid', '{}',
		       '2026-09-06T00:00:00Z'::timestamptz + n * interval '1 second'
		FROM generate_series(1, 1200) AS n;
		INSERT INTO craftsky_reposts (uri, did, rkey, cid, subject_uri, subject_cid, record, created_at)
		SELECT 'at://did:plc:actor' || n || '/social.craftsky.feed.repost/' || n,
		       'did:plc:actor' || n, n::text, 'repost-cid-' || n, 'at://did:plc:owner/social.craftsky.feed.post/target', 'target-cid', '{}',
		       '2026-09-06T00:00:00Z'::timestamptz + n * interval '1 second'
		FROM generate_series(1, 1200) AS n;
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, quote_uri, quote_cid, record, created_at)
		SELECT 'at://did:plc:actor' || n || '/social.craftsky.feed.post/quote-' || n,
		       'did:plc:actor' || n, 'quote-' || n, 'quote-cid-' || n, 'quote', 'at://did:plc:owner/social.craftsky.feed.post/target', 'target-cid', '{}',
		       '2026-09-06T00:00:00Z'::timestamptz + n * interval '1 second'
		FROM generate_series(1, 1200) AS n;
		ANALYZE craftsky_profiles;
		ANALYZE atproto_identity_cache;
		ANALYZE craftsky_likes;
		ANALYZE craftsky_reposts;
		ANALYZE craftsky_posts;
		SET enable_seqscan = off;
	`); err != nil {
		t.Fatalf("seed representative interaction cardinality: %v", err)
	}

	for _, tc := range []struct {
		kind        api.PostInteractionKind
		wantIndexes []string
		wantTable   string
	}{
		{
			kind: api.PostInteractionLikes,
			wantIndexes: []string{
				"craftsky_likes_active_subject_uri", "craftsky_profiles_pkey", "atproto_identity_cache_pkey",
			},
			wantTable: "craftsky_likes",
		},
		{
			kind: api.PostInteractionReposts,
			wantIndexes: []string{
				"craftsky_reposts_active_subject_uri", "craftsky_profiles_pkey", "atproto_identity_cache_pkey",
			},
			wantTable: "craftsky_reposts",
		},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			query, err := api.PostInteractionAccountListQuery(tc.kind)
			if err != nil {
				t.Fatalf("build %s query: %v", tc.kind, err)
			}
			plan := explainPostInteractionQuery(t, pool, query,
				targetURI, "did:plc:viewer", "did:plc:owner",
				time.Date(2026, 9, 6, 0, 10, 0, 0, time.UTC),
				syntax.ATURI(fmt.Sprintf("at://did:plc:actor600/social.craftsky.feed.%s/600", strings.TrimSuffix(string(tc.kind), "s"))),
				51,
			)
			for _, index := range tc.wantIndexes {
				if !strings.Contains(plan, index) {
					t.Fatalf("query plan does not use %s:\n%s", index, plan)
				}
			}
			if queryPlanHasSequentialScan(t, plan, tc.wantTable) {
				t.Fatalf("query plan uses a full %s scan:\n%s", tc.wantTable, plan)
			}
		})
	}

	t.Run("quotes", func(t *testing.T) {
		plan := explainPostInteractionQuery(t, pool, api.PostQuoteListQuery(),
			targetURI, "did:plc:viewer", []string{},
			time.Date(2026, 9, 6, 0, 10, 0, 0, time.UTC),
			syntax.ATURI("at://did:plc:actor600/social.craftsky.feed.post/quote-600"),
			51,
		)
		for _, index := range []string{
			"craftsky_posts_quote_uri", "craftsky_project_posts_pkey", "bluesky_profiles_pkey",
		} {
			if !strings.Contains(plan, index) {
				t.Fatalf("query plan does not use %s:\n%s", index, plan)
			}
		}
		if queryPlanHasSequentialScan(t, plan, "craftsky_posts") {
			t.Fatalf("query plan uses a full craftsky_posts scan:\n%s", plan)
		}
	})
}

func explainPostInteractionQuery(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var plan []byte
	if err := pool.QueryRow(context.Background(), "EXPLAIN (FORMAT JSON, COSTS OFF) "+query, args...).Scan(&plan); err != nil {
		t.Fatalf("explain post-interaction query: %v", err)
	}
	return string(plan)
}

func queryPlanHasSequentialScan(t *testing.T, plan string, relation string) bool {
	t.Helper()
	var decoded any
	if err := json.Unmarshal([]byte(plan), &decoded); err != nil {
		t.Fatalf("decode query plan: %v", err)
	}
	var visit func(any) bool
	visit = func(value any) bool {
		switch value := value.(type) {
		case []any:
			for _, child := range value {
				if visit(child) {
					return true
				}
			}
		case map[string]any:
			if value["Node Type"] == "Seq Scan" && value["Relation Name"] == relation {
				return true
			}
			for _, child := range value {
				if visit(child) {
					return true
				}
			}
		}
		return false
	}
	return visit(decoded)
}
