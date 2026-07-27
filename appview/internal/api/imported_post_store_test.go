package api_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/testdb"
)

func TestPostStoreListByAuthorAcceptsLegacyIndexedAtCursor(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	olderAt := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	newerAt := olderAt.Add(time.Hour)
	olderURI := seedPost(t, pool, "did:plc:alice", "older", "older", olderAt)
	newerURI := seedPost(t, pool, "did:plc:alice", "newer", "newer", newerAt)

	legacyCursor, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": newerAt.Format(time.RFC3339Nano),
		"uri":       newerURI,
	})
	if err != nil {
		t.Fatalf("encode legacy cursor: %v", err)
	}

	rows, next, err := api.NewPostStore(pool).ListByAuthor(
		context.Background(),
		"did:plc:alice",
		10,
		legacyCursor,
	)
	if err != nil {
		t.Fatalf("ListByAuthor with legacy indexedAt cursor: %v", err)
	}
	if len(rows) != 1 || rows[0].URI != olderURI {
		t.Fatalf("rows = %+v, want only %s", rows, olderURI)
	}
	if next != "" {
		t.Fatalf("next cursor = %q, want empty final cursor", next)
	}

	_, generatedCursor, err := api.NewPostStore(pool).ListByAuthor(
		context.Background(),
		"did:plc:alice",
		1,
		"",
	)
	if err != nil {
		t.Fatalf("ListByAuthor first page: %v", err)
	}
	payload, err := envelope.DecodeCursor(generatedCursor)
	if err != nil {
		t.Fatalf("decode generated cursor: %v", err)
	}
	if _, ok := payload["indexedAt"]; !ok {
		t.Fatalf("generated cursor payload = %v, want indexedAt key", payload)
	}
	if _, ok := payload["profileSortAt"]; ok {
		t.Fatalf("generated cursor payload = %v, profileSortAt must remain internal", payload)
	}
}

func TestPostStoreListProjectsByAuthorAcceptsLegacyIndexedAtCursor(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	olderAt := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	newerAt := olderAt.Add(time.Hour)
	olderURI := seedPost(t, pool, "did:plc:alice", "older-project", "older", olderAt)
	seedProjectMaterialization(t, pool, olderURI, "social.craftsky.feed.defs#knitting", "Older")
	newerURI := seedPost(t, pool, "did:plc:alice", "newer-project", "newer", newerAt)
	seedProjectMaterialization(t, pool, newerURI, "social.craftsky.feed.defs#knitting", "Newer")

	legacyCursor, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": newerAt.Format(time.RFC3339Nano),
		"uri":       newerURI,
	})
	if err != nil {
		t.Fatalf("encode legacy cursor: %v", err)
	}

	store := api.NewPostStore(pool)
	rows, next, err := store.ListProjectsByAuthor(
		context.Background(),
		"did:plc:alice",
		10,
		legacyCursor,
	)
	if err != nil {
		t.Fatalf("ListProjectsByAuthor with legacy indexedAt cursor: %v", err)
	}
	if len(rows) != 1 || rows[0].URI != olderURI {
		t.Fatalf("rows = %+v, want only %s", rows, olderURI)
	}
	if next != "" {
		t.Fatalf("next cursor = %q, want empty final cursor", next)
	}

	_, generatedCursor, err := store.ListProjectsByAuthor(
		context.Background(),
		"did:plc:alice",
		1,
		"",
	)
	if err != nil {
		t.Fatalf("ListProjectsByAuthor first page: %v", err)
	}
	payload, err := envelope.DecodeCursor(generatedCursor)
	if err != nil {
		t.Fatalf("decode generated cursor: %v", err)
	}
	if _, ok := payload["indexedAt"]; !ok {
		t.Fatalf("generated cursor payload = %v, want indexedAt key", payload)
	}
	if _, ok := payload["profileSortAt"]; ok {
		t.Fatalf("generated cursor payload = %v, profileSortAt must remain internal", payload)
	}
}

func TestPostStoreListCommentsByAuthorAcceptsLegacyIndexedAtCursor(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	rootAt := time.Date(2026, 7, 21, 10, 0, 0, 0, time.UTC)
	olderAt := rootAt.Add(time.Hour)
	newerAt := olderAt.Add(time.Hour)
	rootURI := seedPost(t, pool, "did:plc:alice", "root", "root", rootAt)
	olderURI := seedReplyPost(
		t,
		pool,
		"did:plc:alice",
		"older-comment",
		"older",
		rootURI,
		rootURI,
		olderAt,
	)
	newerURI := seedReplyPost(
		t,
		pool,
		"did:plc:alice",
		"newer-comment",
		"newer",
		rootURI,
		rootURI,
		newerAt,
	)

	legacyCursor, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": newerAt.Format(time.RFC3339Nano),
		"uri":       newerURI,
	})
	if err != nil {
		t.Fatalf("encode legacy cursor: %v", err)
	}

	store := api.NewPostStore(pool)
	rows, next, err := store.ListCommentsByAuthor(
		context.Background(),
		"did:plc:alice",
		10,
		legacyCursor,
	)
	if err != nil {
		t.Fatalf("ListCommentsByAuthor with legacy indexedAt cursor: %v", err)
	}
	if len(rows) != 1 || rows[0].URI != olderURI {
		t.Fatalf("rows = %+v, want only %s", rows, olderURI)
	}
	if next != "" {
		t.Fatalf("next cursor = %q, want empty final cursor", next)
	}

	_, generatedCursor, err := store.ListCommentsByAuthor(
		context.Background(),
		"did:plc:alice",
		1,
		"",
	)
	if err != nil {
		t.Fatalf("ListCommentsByAuthor first page: %v", err)
	}
	payload, err := envelope.DecodeCursor(generatedCursor)
	if err != nil {
		t.Fatalf("decode generated cursor: %v", err)
	}
	if _, ok := payload["indexedAt"]; !ok {
		t.Fatalf("generated cursor payload = %v, want indexedAt key", payload)
	}
	if _, ok := payload["profileSortAt"]; ok {
		t.Fatalf("generated cursor payload = %v, profileSortAt must remain internal", payload)
	}
}

// AT-008, IT-011, REG-002: imported profile rows use original chronology.
func TestPostStoreListByAuthorUsesHistoricalProfileChronology(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	ordinaryOld := seedPost(t, pool, "did:plc:alice", "ordinary-old", "ordinary old", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	imported := seedPost(t, pool, "did:plc:alice", "imported", "historical import", time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC))
	if _, err := pool.Exec(context.Background(), `
		UPDATE craftsky_posts
		SET external_import_source = 'instagram',
		    created_at = '2021-01-01T00:00:00Z',
		    profile_sort_at = '2021-01-01T00:00:00Z'
		WHERE uri = $1
	`, imported); err != nil {
		t.Fatalf("classify imported row: %v", err)
	}
	ordinaryNew := seedPost(t, pool, "did:plc:alice", "ordinary-new", "ordinary new", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))

	store := api.NewPostStore(pool)
	var got []string
	cursor := ""
	for {
		page, next, err := store.ListByAuthor(context.Background(), "did:plc:alice", 1, cursor)
		if err != nil {
			t.Fatalf("ListByAuthor cursor=%q: %v", cursor, err)
		}
		for _, row := range page {
			got = append(got, row.URI)
			if row.URI == imported && (row.ExternalImportSource == nil || *row.ExternalImportSource != "instagram") {
				t.Fatalf("imported row provenance = %v", row.ExternalImportSource)
			}
		}
		if next == "" {
			break
		}
		cursor = next
	}
	want := []string{ordinaryNew, imported, ordinaryOld}
	if !slices.Equal(got, want) {
		t.Fatalf("profile order = %v, want %v", got, want)
	}
}

// IT-011, REG-002: tied chronology pages have no duplicates or gaps.
func TestPostStoreListByAuthorPaginatesAcrossProfileSortTiesWithoutGaps(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	tiedAt := time.Date(2020, 1, 1, 0, 0, 0, 123456000, time.UTC)
	want := make([]string, 0, 5)
	for _, rkey := range []string{"tie-e", "tie-d", "tie-c", "tie-b", "tie-a"} {
		uri := seedPost(t, pool, "did:plc:alice", rkey, "same timestamp", tiedAt)
		if rkey == "tie-d" || rkey == "tie-b" {
			if _, err := pool.Exec(context.Background(), `
				UPDATE craftsky_posts
				SET external_import_source = 'instagram'
				WHERE uri = $1
			`, uri); err != nil {
				t.Fatalf("classify %s as imported: %v", rkey, err)
			}
		}
		want = append(want, uri)
	}

	store := api.NewPostStore(pool)
	seen := make(map[string]bool, len(want))
	got := make([]string, 0, len(want))
	cursor := ""
	for {
		page, next, err := store.ListByAuthor(context.Background(), "did:plc:alice", 2, cursor)
		if err != nil {
			t.Fatalf("ListByAuthor cursor=%q: %v", cursor, err)
		}
		for _, row := range page {
			if seen[row.URI] {
				t.Fatalf("duplicate row %q across profile pages", row.URI)
			}
			seen[row.URI] = true
			got = append(got, row.URI)
		}
		if next == "" {
			break
		}
		if next == cursor {
			t.Fatalf("cursor did not advance: %q", cursor)
		}
		cursor = next
	}

	if !slices.Equal(got, want) {
		t.Fatalf("tied profile pages = %v, want %v", got, want)
	}
}

// IT-011: the production pagination predicate uses its matching index.
func TestPostStoreProfilePaginationQueryUsesProfileSortIndex(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, postStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	for i, rkey := range []string{"indexed-a", "indexed-b", "indexed-c"} {
		seedPost(t, pool, "did:plc:alice", rkey, "indexed", time.Date(2020+i, 1, 1, 0, 0, 0, 0, time.UTC))
	}
	if _, err := pool.Exec(context.Background(), `
		CREATE INDEX craftsky_posts_profile_posts_sort_idx
		ON craftsky_posts (did, profile_sort_at DESC, uri DESC)
		WHERE is_project = false
		  AND reply_root_uri IS NULL
		  AND reply_parent_uri IS NULL
	`); err != nil {
		t.Fatalf("create profile sort index: %v", err)
	}

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("acquire explain connection: %v", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(context.Background(), `SET enable_seqscan = off`); err != nil {
		t.Fatalf("disable sequential scans for deterministic plan assertion: %v", err)
	}

	var plan string
	if err := conn.QueryRow(context.Background(), `
		EXPLAIN (FORMAT JSON)
		SELECT p.uri
		FROM craftsky_posts p
		WHERE p.did = $1
		  AND p.is_project = false
		  AND p.reply_root_uri IS NULL
		  AND p.reply_parent_uri IS NULL
		  AND (p.profile_sort_at, p.uri) < ($2::timestamptz, $3::text)
		ORDER BY p.profile_sort_at DESC, p.uri DESC
		LIMIT $4
	`, "did:plc:alice", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		"at://did:plc:alice/social.craftsky.feed.post/indexed-z", 2).Scan(&plan); err != nil {
		t.Fatalf("explain profile pagination: %v", err)
	}
	if !strings.Contains(plan, "craftsky_posts_profile_posts_sort_idx") {
		t.Fatalf("profile pagination plan did not use profile-sort index: %s", plan)
	}
}

// AT-008, IT-012, REG-003: original backfill is excluded while deliberate
// later shares retain ordinary home-timeline behavior.
func TestTimelineExcludesImportedOriginalButIncludesLaterRepostAndQuote(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, timelineStoreDDL)
	for _, did := range []string{"did:plc:viewer", "did:plc:author", "did:plc:sharer"} {
		seedMember(t, pool, did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:author", "follow-author")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:sharer", "follow-sharer")

	base := time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC)
	imported := seedPost(t, pool, "did:plc:author", "imported", "historical import", base)
	if _, err := pool.Exec(context.Background(), `
		UPDATE craftsky_posts
		SET external_import_source = 'instagram',
		    created_at = '2019-01-01T00:00:00Z',
		    profile_sort_at = '2019-01-01T00:00:00Z'
		WHERE uri = $1
	`, imported); err != nil {
		t.Fatalf("classify import: %v", err)
	}
	repost := seedInteraction(t, pool, "repost", "did:plc:sharer", "share-import", imported, false)
	quote := seedQuotePost(t, pool, "did:plc:sharer", "quote-import", "remember this", imported, "bafycid", base.Add(2*time.Minute))

	store := api.NewPostStore(pool)
	items, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	var keys []string
	for _, item := range items {
		keys = append(keys, item.ItemKey)
	}
	if slices.Contains(keys, "post:"+imported) {
		t.Fatalf("timeline contains imported original: %v", keys)
	}
	for _, want := range []string{"repost:" + repost, "post:" + quote} {
		if !slices.Contains(keys, want) {
			t.Fatalf("timeline keys = %v, want %s", keys, want)
		}
	}
}

// IT-014, REG-005: imported posts remain eligible for ordinary search.
func TestSearchPostsIncludesImportedPostsWithOriginalChronologyAndProvenance(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, searchStoreDDL)
	seedMember(t, pool, "did:plc:alice")
	imported := seedPost(t, pool, "did:plc:alice", "imported-search", "heritage cardigan", time.Date(2026, 7, 23, 12, 0, 0, 0, time.UTC))
	if _, err := pool.Exec(context.Background(), `
		UPDATE craftsky_posts
		SET external_import_source = 'instagram',
		    created_at = '2019-01-01T00:00:00Z',
		    profile_sort_at = '2019-01-01T00:00:00Z'
		WHERE uri = $1
	`, imported); err != nil {
		t.Fatalf("classify search import: %v", err)
	}
	ordinary := seedPost(t, pool, "did:plc:alice", "ordinary-search", "heritage jumper", time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC))

	store := api.NewSearchStore(pool, nil)
	rows, cursor, err := store.SearchPosts(
		context.Background(),
		api.PostSearchRequest{Query: "heritage", Sort: api.SearchSortChronological, Limit: 10},
		time.Date(2026, 7, 23, 13, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("SearchPosts: %v", err)
	}
	if cursor != "" || len(rows) != 2 {
		t.Fatalf("search rows=%d cursor=%q, want 2/terminal", len(rows), cursor)
	}
	if got := []string{rows[0].Post.URI, rows[1].Post.URI}; !slices.Equal(got, []string{ordinary, imported}) {
		t.Fatalf("search order = %v, want ordinary then historical import", got)
	}
	if rows[1].Post.ExternalImportSource == nil || *rows[1].Post.ExternalImportSource != "instagram" {
		t.Fatalf("search import provenance = %v", rows[1].Post.ExternalImportSource)
	}
}

// IT-014, REG-005: hashtag and keyword pagination preserve original chronology.
func TestSearchImportedHashtagsAndKeywordsPaginateWithOriginalChronology(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, searchStoreDDL)
	seedMember(t, pool, "did:plc:alice")

	type fixture struct {
		rkey     string
		created  time.Time
		imported bool
	}
	fixtures := []fixture{
		{rkey: "ordinary-new", created: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)},
		{rkey: "imported-z", created: time.Date(2020, 1, 1, 0, 0, 0, 123456000, time.UTC), imported: true},
		{rkey: "imported-a", created: time.Date(2020, 1, 1, 0, 0, 0, 123456000, time.UTC), imported: true},
		{rkey: "ordinary-old", created: time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	want := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		uri := seedPost(t, pool, "did:plc:alice", fixture.rkey, "heritage cardigan", fixture.created)
		seedPostTags(t, pool, uri, []string{"heritage"})
		if fixture.imported {
			if _, err := pool.Exec(context.Background(), `
				UPDATE craftsky_posts
				SET external_import_source = 'instagram',
				    profile_sort_at = created_at,
				    indexed_at = '2026-07-23T12:00:00Z'
				WHERE uri = $1
			`, uri); err != nil {
				t.Fatalf("classify %s as imported: %v", fixture.rkey, err)
			}
		}
		want = append(want, uri)
	}

	store := api.NewSearchStore(pool, nil)
	var hashtagURIs []string
	var hashtagSources []*string
	cursor := ""
	for {
		page, next, err := store.SearchHashtagPosts(
			context.Background(), "heritage", api.SearchSortChronological, 1, cursor,
			time.Date(2026, 7, 23, 13, 0, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("SearchHashtagPosts cursor=%q: %v", cursor, err)
		}
		for _, row := range page {
			hashtagURIs = append(hashtagURIs, row.Post.URI)
			hashtagSources = append(hashtagSources, row.Post.ExternalImportSource)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if !slices.Equal(hashtagURIs, want) {
		t.Fatalf("hashtag pages = %v, want %v", hashtagURIs, want)
	}
	for i, source := range hashtagSources {
		wantImported := fixtures[i].imported
		if wantImported && (source == nil || *source != "instagram") {
			t.Fatalf("hashtag row %d provenance = %v, want instagram", i, source)
		}
		if !wantImported && source != nil {
			t.Fatalf("ordinary hashtag row %d provenance = %v, want nil", i, source)
		}
	}

	var keywordURIs []string
	cursor = ""
	for {
		page, next, err := store.SearchPosts(
			context.Background(),
			api.PostSearchRequest{Query: "heritage cardigan", Sort: api.SearchSortChronological, Limit: 1, Cursor: cursor},
			time.Date(2026, 7, 23, 13, 0, 0, 0, time.UTC),
		)
		if err != nil {
			t.Fatalf("SearchPosts cursor=%q: %v", cursor, err)
		}
		for _, row := range page {
			keywordURIs = append(keywordURIs, row.Post.URI)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if !slices.Equal(keywordURIs, want) {
		t.Fatalf("keyword pages = %v, want %v", keywordURIs, want)
	}
}
