// appview/internal/api/timeline_store_test.go
package api_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const timelineStoreDDL = postStoreDDL + `
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
`

func seedFollow(t *testing.T, pool *pgxpool.Pool, followerDID, subjectDID, rkey string) string {
	t.Helper()
	uri := "at://" + followerDID + "/app.bsky.graph.follow/" + rkey
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at, indexed_at)
		VALUES ($1, $2, $3, 'bafyfollow' || $3, $4, '{}'::jsonb, $5, $5)`,
		uri, followerDID, rkey, subjectDID, time.Date(2026, 5, 28, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed follow: %v", err)
	}
	return uri
}

func seedQuotePost(t *testing.T, pool *pgxpool.Pool, did, rkey, text, quoteURI, quoteCID string, indexedAt time.Time) string {
	t.Helper()
	uri := "at://" + did + "/social.craftsky.feed.post/" + rkey
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, quote_uri, quote_cid, record, created_at, indexed_at)
		VALUES ($1, $2, $3, 'bafycid-' || $3, $4, $5, $6, '{}'::jsonb, $7, $7)`,
		uri, did, rkey, text, quoteURI, quoteCID, indexedAt); err != nil {
		t.Fatalf("seed quote post: %v", err)
	}
	return uri
}

func timelineURIs(rows []*api.PostRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.URI)
	}
	return out
}

func TestTimelineStore_ListTimeline_ReturnsOwnAndFollowedEligiblePostsOnly(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedMember(t, pool, "did:plc:bob")
	seedBskyProfile(t, pool, "did:plc:viewer", "Viewer", "bafyviewer")
	seedBskyProfile(t, pool, "did:plc:alice", "Alice", "bafyalice")
	seedBskyProfile(t, pool, "did:plc:bob", "Bob", "bafybob")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")

	viewerPost := seedPost(t, pool, "did:plc:viewer", "own", "own post", base.Add(4*time.Minute))
	aliceRoot := seedPost(t, pool, "did:plc:alice", "root", "alice root", base.Add(3*time.Minute))
	aliceProject := seedPost(t, pool, "did:plc:alice", "project", "alice project", base.Add(2*time.Minute))
	aliceQuote := seedQuotePost(t, pool, "did:plc:alice", "quote", "alice quote", viewerPost, "bafyquoted", base.Add(time.Minute))
	bobPost := seedPost(t, pool, "did:plc:bob", "root", "bob root", base.Add(5*time.Minute))

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}
	got := timelineURIs(rows)
	for _, want := range []string{viewerPost, aliceRoot, aliceProject, aliceQuote} {
		if !slices.Contains(got, want) {
			t.Fatalf("timeline URIs = %v, want containing %s", got, want)
		}
	}
	if slices.Contains(got, bobPost) {
		t.Fatalf("timeline URIs = %v, must not contain unfollowed author post %s", got, bobPost)
	}
}

func TestTimelineStore_ListTimeline_ExcludesConversationAndRepostActivity(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 13, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")

	root := seedPost(t, pool, "did:plc:alice", "root", "root post", base.Add(5*time.Minute))
	quote := seedQuotePost(t, pool, "did:plc:alice", "quote", "quote post", root, "bafyroot", base.Add(4*time.Minute))
	comment := seedReplyPost(t, pool, "did:plc:alice", "comment", "comment", root, root, base.Add(3*time.Minute))
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply", "nested reply", root, comment, base.Add(2*time.Minute))
	repost := seedInteraction(t, pool, "repost", "did:plc:alice", "repost-root", root, false)

	store := api.NewPostStore(pool)
	rows, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	got := timelineURIs(rows)
	for _, want := range []string{root, quote} {
		if !slices.Contains(got, want) {
			t.Fatalf("timeline URIs = %v, want containing %s", got, want)
		}
	}
	for _, excluded := range []string{comment, reply, repost} {
		if slices.Contains(got, excluded) {
			t.Fatalf("timeline URIs = %v, must not contain conversation/repost activity %s", got, excluded)
		}
	}
}

func TestTimelineStore_ListTimeline_OrdersByIndexedAtThenURIDesc(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 14, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")

	oldest := seedPost(t, pool, "did:plc:alice", "a-oldest", "oldest", base)
	tiedLowURI := seedPost(t, pool, "did:plc:alice", "b-tie-low", "tie low", base.Add(time.Minute))
	tiedHighURI := seedPost(t, pool, "did:plc:alice", "z-tie-high", "tie high", base.Add(time.Minute))
	newest := seedPost(t, pool, "did:plc:viewer", "newest", "newest", base.Add(2*time.Minute))

	store := api.NewPostStore(pool)
	rows, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	got := timelineURIs(rows)
	want := []string{newest, tiedHighURI, tiedLowURI, oldest}
	if !slices.Equal(got, want) {
		t.Fatalf("timeline URIs = %v, want %v", got, want)
	}
}

func TestTimelineStore_ListTimeline_PaginatesWithOpaqueSeekCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 15, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")

	post1 := seedPost(t, pool, "did:plc:alice", "post-1", "post 1", base.Add(4*time.Minute))
	post2 := seedPost(t, pool, "did:plc:alice", "post-2", "post 2", base.Add(3*time.Minute))
	post3 := seedPost(t, pool, "did:plc:alice", "z-post-3", "post 3", base.Add(2*time.Minute))
	post4 := seedPost(t, pool, "did:plc:alice", "a-post-4", "post 4", base.Add(2*time.Minute))
	post5 := seedPost(t, pool, "did:plc:alice", "post-5", "post 5", base.Add(time.Minute))

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("ListTimeline first: %v", err)
	}
	if cursor == "" {
		t.Fatal("first cursor = empty, want next page cursor")
	}
	if got, want := timelineURIs(first), []string{post1, post2}; !slices.Equal(got, want) {
		t.Fatalf("first page = %v, want %v", got, want)
	}

	second, nextCursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 2, cursor)
	if err != nil {
		t.Fatalf("ListTimeline second: %v", err)
	}
	if nextCursor == "" {
		t.Fatal("second cursor = empty, want final page cursor")
	}
	if got, want := timelineURIs(second), []string{post3, post4}; !slices.Equal(got, want) {
		t.Fatalf("second page = %v, want %v", got, want)
	}

	third, finalCursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 2, nextCursor)
	if err != nil {
		t.Fatalf("ListTimeline third: %v", err)
	}
	if finalCursor != "" {
		t.Fatalf("final cursor = %q, want empty", finalCursor)
	}
	if got, want := timelineURIs(third), []string{post5}; !slices.Equal(got, want) {
		t.Fatalf("third page = %v, want %v", got, want)
	}
}

func TestTimelineStore_ListTimeline_OmitsCursorWhenExactFullFinalPage(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 15, 30, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")

	post1 := seedPost(t, pool, "did:plc:alice", "post-1", "post 1", base.Add(time.Minute))
	post2 := seedPost(t, pool, "did:plc:alice", "post-2", "post 2", base)

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if got, want := timelineURIs(rows), []string{post1, post2}; !slices.Equal(got, want) {
		t.Fatalf("timeline URIs = %v, want %v", got, want)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty when final page exactly fills requested limit", cursor)
	}
}

func TestTimelineStore_ListTimeline_IncludesOwnPostWithoutSelfFollowAndDeduplicatesSelfFollow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 16, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	own := seedPost(t, pool, "did:plc:viewer", "own", "own post", base)

	store := api.NewPostStore(pool)
	withoutSelfFollow, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline without self-follow: %v", err)
	}
	if got, want := timelineURIs(withoutSelfFollow), []string{own}; !slices.Equal(got, want) {
		t.Fatalf("without self-follow timeline = %v, want %v", got, want)
	}

	seedFollow(t, pool, "did:plc:viewer", "did:plc:viewer", "follow-self")
	withSelfFollow, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline with self-follow: %v", err)
	}
	if got, want := timelineURIs(withSelfFollow), []string{own}; !slices.Equal(got, want) {
		t.Fatalf("with self-follow timeline = %v, want exactly one own post %v", got, want)
	}
}

func TestTimelineStore_ListTimeline_UsesCurrentFollowGraphOnEachPage(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 16, 30, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")

	aliceNewest := seedPost(t, pool, "did:plc:alice", "alice-newest", "alice newest", base.Add(3*time.Minute))
	aliceOlder := seedPost(t, pool, "did:plc:alice", "alice-older", "alice older", base.Add(2*time.Minute))
	viewerOwn := seedPost(t, pool, "did:plc:viewer", "own", "own post", base.Add(time.Minute))

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 1, "")
	if err != nil {
		t.Fatalf("ListTimeline first: %v", err)
	}
	if cursor == "" {
		t.Fatal("first cursor = empty, want next page cursor")
	}
	if got, want := timelineURIs(first), []string{aliceNewest}; !slices.Equal(got, want) {
		t.Fatalf("first page = %v, want %v", got, want)
	}

	if _, err := pool.Exec(context.Background(), `DELETE FROM atproto_follows WHERE did = $1 AND subject_did = $2`, "did:plc:viewer", "did:plc:alice"); err != nil {
		t.Fatalf("delete follow: %v", err)
	}

	second, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, cursor)
	if err != nil {
		t.Fatalf("ListTimeline second: %v", err)
	}
	got := timelineURIs(second)
	if slices.Contains(got, aliceOlder) {
		t.Fatalf("second page = %v, must not include formerly-followed author after follow removal", got)
	}
	if got, want := got, []string{viewerOwn}; !slices.Equal(got, want) {
		t.Fatalf("second page = %v, want current-graph eligible rows %v", got, want)
	}
}

func TestTimelineStore_ListTimeline_NonCraftskyFollowsDoNotContributeContent(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)

	seedMember(t, pool, "did:plc:viewer")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:external", "follow-external")

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, want empty for non-Craftsky follow without craftsky_posts rows", timelineURIs(rows))
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}
}
