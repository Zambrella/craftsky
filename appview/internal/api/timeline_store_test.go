// appview/internal/api/timeline_store_test.go
package api_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
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

func timelineURIs(items []*api.TimelineFeedItemRow) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.Post.URI)
	}
	return out
}

func TestTimelineFiltersMuteBlockAndRepostAttributionBeforePagination(t *testing.T) {
	pool := testdb.WithSchema(t, timelineStoreDDL)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:bob", "did:plc:carol", "did:plc:dana", "did:plc:erin"} {
		seedMember(t, pool, did)
		if did != "did:plc:viewer" {
			seedFollow(t, pool, "did:plc:viewer", did, "follow-"+did[len("did:plc:"):])
		}
	}
	base := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	bobNewest := seedPost(t, pool, "did:plc:bob", "bob-newest", "hidden muted", base.Add(10*time.Minute))
	carol := seedPost(t, pool, "did:plc:carol", "carol", "eligible", base.Add(9*time.Minute))
	repost := seedInteraction(t, pool, "repost", "did:plc:carol", "repost-bob", bobNewest, false)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_reposts SET created_at = $1, indexed_at = $1 WHERE uri = $2`, base.Add(8*time.Minute), repost); err != nil {
		t.Fatalf("date repost: %v", err)
	}
	seedPost(t, pool, "did:plc:dana", "dana", "hidden blocked", base.Add(7*time.Minute))
	erin := seedPost(t, pool, "did:plc:erin", "erin", "eligible", base.Add(6*time.Minute))
	viewer := seedPost(t, pool, "did:plc:viewer", "viewer", "own", base.Add(5*time.Minute))
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did) VALUES ('did:plc:viewer', 'did:plc:bob');
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://did:plc:dana/app.bsky.graph.block/viewer', 'did:plc:dana', 'viewer', 'block-cid', 'did:plc:viewer', '{}', now());
	`); err != nil {
		t.Fatalf("seed relationships: %v", err)
	}

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListTimeline(ctx, "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if got, want := timelineURIs(first), []string{carol, erin}; !slices.Equal(got, want) || cursor == "" {
		t.Fatalf("first page = %v cursor=%q, want %v and cursor", got, cursor, want)
	}
	second, next, err := store.ListTimeline(ctx, "did:plc:viewer", 2, cursor)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if got, want := timelineURIs(second), []string{viewer}; !slices.Equal(got, want) || next != "" {
		t.Fatalf("second page = %v cursor=%q, want %v and terminal", got, next, want)
	}
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

func TestTimelineStore_ListTimeline_FiltersHiddenPostsAndAuthors(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	for _, did := range []string{"did:plc:viewer", "did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
		seedFollow(t, pool, "did:plc:viewer", did, "follow-"+did[len("did:plc:"):])
	}
	visible := seedPost(t, pool, "did:plc:alice", "visible", "visible", base.Add(4*time.Minute))
	hiddenPost := seedPost(t, pool, "did:plc:bob", "hidden-post", "hidden post", base.Add(3*time.Minute))
	hiddenAuthorPost := seedPost(t, pool, "did:plc:carol", "hidden-author", "hidden author", base.Add(2*time.Minute))
	seedModerationOutput(t, pool, "post", "did:plc:bob", hiddenPost, "hide", base.Add(time.Minute))
	seedModerationOutput(t, pool, "account", "did:plc:carol", "", "takedown", base.Add(time.Minute))

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}
	got := timelineURIs(rows)
	if !slices.Contains(got, visible) {
		t.Fatalf("timeline URIs = %v, want visible %s", got, visible)
	}
	if slices.Contains(got, hiddenPost) || slices.Contains(got, hiddenAuthorPost) {
		t.Fatalf("timeline URIs = %v, leaked hidden rows", got)
	}
}

func TestTimelineStore_ListTimeline_OmitsRepostsOfHiddenSubjects(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 12, 15, 0, 0, time.UTC)

	for _, did := range []string{"did:plc:viewer", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:bob", "follow-bob")
	hiddenSubject := seedPost(t, pool, "did:plc:carol", "hidden", "hidden subject", base)
	visibleSubject := seedPost(t, pool, "did:plc:carol", "visible", "visible subject", base.Add(time.Minute))
	hiddenRepost := seedInteraction(t, pool, "repost", "did:plc:bob", "hidden-repost", hiddenSubject, false)
	visibleRepost := seedInteraction(t, pool, "repost", "did:plc:bob", "visible-repost", visibleSubject, false)
	seedModerationOutput(t, pool, "post", "did:plc:carol", hiddenSubject, "hide", base.Add(2*time.Minute))

	store := api.NewPostStore(pool)
	items, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	gotKeys := make([]string, 0, len(items))
	for _, item := range items {
		gotKeys = append(gotKeys, item.ItemKey)
	}
	if slices.Contains(gotKeys, "repost:"+hiddenRepost) {
		t.Fatalf("timeline item keys = %v, leaked hidden subject repost %s", gotKeys, hiddenRepost)
	}
	if !slices.Contains(gotKeys, "repost:"+visibleRepost) {
		t.Fatalf("timeline item keys = %v, want visible subject repost %s", gotKeys, visibleRepost)
	}
}

func TestTimelineStore_ListTimeline_FiltersBeforeLimitForDeterministicPagination(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC)

	for _, did := range []string{"did:plc:viewer", "did:plc:alice"} {
		seedMember(t, pool, did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:alice", "follow-alice")
	hiddenNewest := seedPost(t, pool, "did:plc:alice", "hidden-newest", "hidden newest", base.Add(5*time.Minute))
	visibleFirst := seedPost(t, pool, "did:plc:alice", "visible-first", "visible first", base.Add(4*time.Minute))
	visibleSecond := seedPost(t, pool, "did:plc:alice", "visible-second", "visible second", base.Add(3*time.Minute))
	visibleThird := seedPost(t, pool, "did:plc:alice", "visible-third", "visible third", base.Add(2*time.Minute))
	seedModerationOutput(t, pool, "post", "did:plc:alice", hiddenNewest, "hide", base.Add(time.Minute))

	store := api.NewPostStore(pool)
	rows, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	if got := timelineURIs(rows); !slices.Equal(got, []string{visibleFirst, visibleSecond}) {
		t.Fatalf("page1 URIs = %v, want first two visible rows", got)
	}
	if cursor == "" {
		t.Fatal("cursor = empty, want next page cursor")
	}
	page2, cursor2, err := store.ListTimeline(context.Background(), "did:plc:viewer", 2, cursor)
	if err != nil {
		t.Fatalf("ListTimeline page2: %v", err)
	}
	if got := timelineURIs(page2); !slices.Equal(got, []string{visibleThird}) {
		t.Fatalf("page2 URIs = %v, want final visible row", got)
	}
	if cursor2 != "" {
		t.Fatalf("cursor2 = %q, want empty", cursor2)
	}
}

func TestTimelineStore_ListTimeline_IncludesFollowedRepostActivityWithReasonAndExcludesReplies(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	base := time.Date(2026, 5, 28, 13, 0, 0, 0, time.UTC)

	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:bob")
	seedMember(t, pool, "did:plc:carol")
	seedBskyProfile(t, pool, "did:plc:bob", "Bob", "bafybob")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:bob", "follow-bob")

	root := seedPost(t, pool, "did:plc:carol", "root", "root post", base.Add(2*time.Minute))
	quote := seedQuotePost(t, pool, "did:plc:bob", "quote", "quote post", root, "bafyroot", base.Add(4*time.Minute))
	comment := seedReplyPost(t, pool, "did:plc:bob", "comment", "comment", root, root, base.Add(3*time.Minute))
	reply := seedReplyPost(t, pool, "did:plc:bob", "reply", "nested reply", root, comment, base.Add(time.Minute))
	repost := seedInteraction(t, pool, "repost", "did:plc:bob", "repost-root", root, false)
	repostAt := base.Add(5 * time.Minute)
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_reposts SET created_at = $1, indexed_at = $1 WHERE uri = $2`, repostAt, repost); err != nil {
		t.Fatalf("update repost time: %v", err)
	}

	store := api.NewPostStore(pool)
	items, _, err := store.ListTimeline(context.Background(), "did:plc:viewer", 20, "")
	if err != nil {
		t.Fatalf("ListTimeline: %v", err)
	}
	got := timelineURIs(items)
	for _, want := range []string{root, quote} {
		if !slices.Contains(got, want) {
			t.Fatalf("timeline URIs = %v, want containing %s", got, want)
		}
	}
	for _, excluded := range []string{comment, reply} {
		if slices.Contains(got, excluded) {
			t.Fatalf("timeline URIs = %v, must not contain conversation/repost activity %s", got, excluded)
		}
	}
	if len(items) == 0 || items[0].ItemKind != "repost" {
		t.Fatalf("first item = %+v, want repost item", items)
	}
	repostItem := items[0]
	if repostItem.ItemKey != "repost:"+repost {
		t.Fatalf("repost item key = %q, want %q", repostItem.ItemKey, "repost:"+repost)
	}
	if repostItem.Post == nil || repostItem.Post.URI != root {
		t.Fatalf("repost item post = %+v, want original %s", repostItem.Post, root)
	}
	if repostItem.Repost == nil {
		t.Fatal("repost item reason = nil, want reposter metadata")
	}
	if repostItem.Repost.URI != repost || repostItem.Repost.DID != "did:plc:bob" || repostItem.Repost.CID != "bafyrepost-root" {
		t.Fatalf("repost reason = %+v, want Bob repost identity", repostItem.Repost)
	}
	if repostItem.Repost.AuthorDisplayName == nil || *repostItem.Repost.AuthorDisplayName != "Bob" {
		t.Fatalf("repost reason displayName = %v, want Bob", repostItem.Repost.AuthorDisplayName)
	}
	if !repostItem.Repost.IndexedAt.Equal(repostAt) || !repostItem.ActivityAt.Equal(repostAt) {
		t.Fatalf("repost timestamps = reason %v activity %v, want %v", repostItem.Repost.IndexedAt, repostItem.ActivityAt, repostAt)
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

func TestTimelineStore_ListTimeline_PaginatesMixedPostsAndRepostsWithFeedItemCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, timelineStoreDDL)
	tied := time.Date(2026, 5, 28, 15, 15, 0, 0, time.UTC)

	for _, did := range []string{"did:plc:viewer", "did:plc:bob", "did:plc:carol", "did:plc:dana"} {
		seedMember(t, pool, did)
	}
	seedFollow(t, pool, "did:plc:viewer", "did:plc:bob", "follow-bob")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:dana", "follow-dana")

	root := seedPost(t, pool, "did:plc:carol", "root", "root post", tied.Add(-time.Hour))
	repost := seedInteraction(t, pool, "repost", "did:plc:bob", "repost-root", root, false)
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_reposts SET created_at = $1, indexed_at = $1 WHERE uri = $2`, tied, repost); err != nil {
		t.Fatalf("update repost time: %v", err)
	}
	danaPost := seedPost(t, pool, "did:plc:dana", "post", "dana post", tied)

	store := api.NewPostStore(pool)
	first, cursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 1, "")
	if err != nil {
		t.Fatalf("ListTimeline first: %v", err)
	}
	if cursor == "" {
		t.Fatal("first cursor = empty, want next page cursor")
	}
	if len(first) != 1 || first[0].ItemKey != "repost:"+repost {
		t.Fatalf("first page = %+v, want repost item %s", first, repost)
	}
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if payload["itemKey"] != first[0].ItemKey {
		t.Fatalf("cursor itemKey = %v, want %s", payload["itemKey"], first[0].ItemKey)
	}
	if _, ok := payload["uri"]; ok {
		t.Fatalf("cursor leaked old uri tie-breaker key: %v", payload)
	}

	second, nextCursor, err := store.ListTimeline(context.Background(), "did:plc:viewer", 1, cursor)
	if err != nil {
		t.Fatalf("ListTimeline second: %v", err)
	}
	if len(second) != 1 || second[0].ItemKey != "post:"+danaPost {
		t.Fatalf("second page = %+v, want authored post %s", second, danaPost)
	}
	if second[0].ItemKey == first[0].ItemKey {
		t.Fatalf("duplicate item across pages: first=%s second=%s", first[0].ItemKey, second[0].ItemKey)
	}

	if nextCursor != "" {
		t.Fatalf("second cursor = %q, want exhausted timeline", nextCursor)
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
