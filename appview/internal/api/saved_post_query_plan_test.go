package api

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const savedPostQueryPlanPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT NOT NULL PRIMARY KEY,
    record_cid  TEXT NOT NULL
);
CREATE TABLE craftsky_posts (
    uri         TEXT NOT NULL PRIMARY KEY,
    did         TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    cid         TEXT NOT NULL
);
`

func TestSavedPostQueryPlansUseOwnerScopedIndexes(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostQueryPlanPreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO saved_post_folders (id, owner_did, name, created_at, updated_at)
		VALUES ('00000000-0000-4000-8000-000000000001', 'did:plc:alice', 'Ideas', now(), now());
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		SELECT
			'at://did:plc:bob/social.craftsky.feed.post/' || n,
			'did:plc:bob', n::text, 'cid-' || n
		FROM generate_series(1, 1200) AS n;
		INSERT INTO saved_posts (owner_did, post_uri, folder_id, saved_at)
		SELECT
			'did:plc:alice', uri,
			CASE WHEN rkey::integer % 3 = 0 THEN '00000000-0000-4000-8000-000000000001'::uuid ELSE NULL END,
			'2026-07-20T00:00:00Z'::timestamptz + rkey::integer * interval '1 second'
		FROM craftsky_posts;
		ANALYZE saved_post_folders;
		ANALYZE saved_posts;
		SET enable_seqscan = off;
	`); err != nil {
		t.Fatalf("insert representative cardinality: %v", err)
	}

	tests := []struct {
		name          string
		scope         SavedPostScope
		sort          SavedPostSort
		cursorSavedAt any
		cursorURI     any
		wantIndex     string
		wantDirection string
	}{
		{name: "all newest first page", scope: SavedPostScopeAll, sort: SavedPostSortNewest, wantIndex: "saved_posts_owner_saved_at_idx", wantDirection: "Forward"},
		{name: "all newest cursor page", scope: SavedPostScopeAll, sort: SavedPostSortNewest, cursorSavedAt: time.Date(2026, 7, 20, 0, 10, 0, 0, time.UTC), cursorURI: "at://did:plc:bob/social.craftsky.feed.post/600", wantIndex: "saved_posts_owner_saved_at_idx", wantDirection: "Forward"},
		{name: "all oldest first page", scope: SavedPostScopeAll, sort: SavedPostSortOldest, wantIndex: "saved_posts_owner_saved_at_idx", wantDirection: "Backward"},
		{name: "all oldest cursor page", scope: SavedPostScopeAll, sort: SavedPostSortOldest, cursorSavedAt: time.Date(2026, 7, 20, 0, 10, 0, 0, time.UTC), cursorURI: "at://did:plc:bob/social.craftsky.feed.post/600", wantIndex: "saved_posts_owner_saved_at_idx", wantDirection: "Backward"},
		{name: "folder newest first page", scope: SavedPostScopeFolder, sort: SavedPostSortNewest, wantIndex: "saved_posts_owner_folder_saved_at_idx", wantDirection: "Forward"},
		{name: "folder newest cursor page", scope: SavedPostScopeFolder, sort: SavedPostSortNewest, cursorSavedAt: time.Date(2026, 7, 20, 0, 10, 0, 0, time.UTC), cursorURI: "at://did:plc:bob/social.craftsky.feed.post/600", wantIndex: "saved_posts_owner_folder_saved_at_idx", wantDirection: "Forward"},
		{name: "folder oldest first page", scope: SavedPostScopeFolder, sort: SavedPostSortOldest, wantIndex: "saved_posts_owner_folder_saved_at_idx", wantDirection: "Backward"},
		{name: "folder oldest cursor page", scope: SavedPostScopeFolder, sort: SavedPostSortOldest, cursorSavedAt: time.Date(2026, 7, 20, 0, 10, 0, 0, time.UTC), cursorURI: "at://did:plc:bob/social.craftsky.feed.post/600", wantIndex: "saved_posts_owner_folder_saved_at_idx", wantDirection: "Backward"},
		{name: "unfiled newest first page", scope: SavedPostScopeUnfiled, sort: SavedPostSortNewest, wantIndex: "saved_posts_owner_unfiled_saved_at_idx", wantDirection: "Forward"},
		{name: "unfiled newest cursor page", scope: SavedPostScopeUnfiled, sort: SavedPostSortNewest, cursorSavedAt: time.Date(2026, 7, 20, 0, 10, 0, 0, time.UTC), cursorURI: "at://did:plc:bob/social.craftsky.feed.post/600", wantIndex: "saved_posts_owner_unfiled_saved_at_idx", wantDirection: "Forward"},
		{name: "unfiled oldest first page", scope: SavedPostScopeUnfiled, sort: SavedPostSortOldest, wantIndex: "saved_posts_owner_unfiled_saved_at_idx", wantDirection: "Backward"},
		{name: "unfiled oldest cursor page", scope: SavedPostScopeUnfiled, sort: SavedPostSortOldest, cursorSavedAt: time.Date(2026, 7, 20, 0, 10, 0, 0, time.UTC), cursorURI: "at://did:plc:bob/social.craftsky.feed.post/600", wantIndex: "saved_posts_owner_unfiled_saved_at_idx", wantDirection: "Backward"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			folderID := any(nil)
			if test.scope == SavedPostScopeFolder {
				folderID = "00000000-0000-4000-8000-000000000001"
			}
			plan := explainSavedPostQuery(t, pool, savedPostListQuery(test.scope, test.sort),
				"did:plc:alice", string(test.scope), folderID, test.cursorSavedAt, test.cursorURI, 51)
			if !strings.Contains(plan, test.wantIndex) {
				t.Fatalf("query plan does not use %s:\n%s", test.wantIndex, plan)
			}
			if !strings.Contains(plan, `"Scan Direction": "`+test.wantDirection+`"`) {
				t.Fatalf("query plan does not use %s index direction:\n%s", test.wantDirection, plan)
			}
			if strings.Contains(plan, "Seq Scan on saved_posts") {
				t.Fatalf("query plan uses a sequential saved-state scan:\n%s", plan)
			}
		})
	}

	otherQueries := []struct {
		name      string
		query     string
		wantIndex string
	}{
		{
			name: "folder list ordered page",
			query: `
				SELECT id::text, name, created_at, updated_at
				FROM saved_post_folders
				WHERE owner_did = 'did:plc:alice'
				ORDER BY lower(name), id
				LIMIT 51
			`,
			wantIndex: "saved_post_folders_owner_name_idx",
		},
		{
			name: "shared viewer state batch",
			query: `
				SELECT post_uri, folder_id::text
				FROM saved_posts
				WHERE owner_did = 'did:plc:alice'
				  AND post_uri = ANY(ARRAY[
					'at://did:plc:bob/social.craftsky.feed.post/1',
					'at://did:plc:bob/social.craftsky.feed.post/600',
					'at://did:plc:bob/social.craftsky.feed.post/1200'
				]::text[])
			`,
			wantIndex: "saved_posts_pkey",
		},
	}

	for _, test := range otherQueries {
		t.Run(test.name, func(t *testing.T) {
			plan := explainSavedPostQuery(t, pool, test.query)
			if !strings.Contains(plan, test.wantIndex) {
				t.Fatalf("query plan does not use %s:\n%s", test.wantIndex, plan)
			}
			if strings.Contains(plan, "Seq Scan on saved_posts") || strings.Contains(plan, "Seq Scan on saved_post_folders") {
				t.Fatalf("query plan uses a sequential saved-state scan:\n%s", plan)
			}
		})
	}
}

func explainSavedPostQuery(t *testing.T, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()
	var plan []byte
	if err := pool.QueryRow(context.Background(), "EXPLAIN (FORMAT JSON, COSTS OFF) "+query, args...).Scan(&plan); err != nil {
		t.Fatalf("explain saved-post query: %v", err)
	}
	return string(plan)
}
