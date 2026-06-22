package api_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/testdb"
)

const searchStoreDDL = timelineStoreDDL + `
CREATE FUNCTION craftsky_text_array_to_string(arr TEXT[], delimiter TEXT)
RETURNS TEXT
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
RETURNS NULL ON NULL INPUT
AS $$
    SELECT array_to_string(arr, delimiter);
$$;

CREATE TABLE atproto_identity_cache (
    did          TEXT        NOT NULL PRIMARY KEY,
    handle       TEXT        NOT NULL,
    handle_lower TEXT        NOT NULL UNIQUE,
    resolved_at  TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func seedSearchProject(t *testing.T, pool *pgxpool.Pool, did, rkey, text, craftType, title string, createdAt time.Time) string {
	t.Helper()
	uri := seedPost(t, pool, did, rkey, text, createdAt)
	seedProjectMaterialization(t, pool, uri, craftType, title)
	return uri
}

func seedSearchIdentity(t *testing.T, pool *pgxpool.Pool, did, handle, displayName, description string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO atproto_identity_cache (did, handle, handle_lower, resolved_at)
		VALUES ($1, $2, lower($2), $3)`, did, handle, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO bluesky_profiles (did, display_name, description, record_cid)
		VALUES ($1, $2, $3, 'seed')
		ON CONFLICT (did) DO UPDATE SET display_name = excluded.display_name, description = excluded.description`, did, displayName, description); err != nil {
		t.Fatalf("seed bsky profile: %v", err)
	}
}

func seedProjectDetails(t *testing.T, pool *pgxpool.Pool, uri string, materials, colors, designTags, projectTags []string) {
	t.Helper()
	if materials == nil {
		materials = []string{}
	}
	if colors == nil {
		colors = []string{}
	}
	if designTags == nil {
		designTags = []string{}
	}
	if projectTags == nil {
		projectTags = []string{}
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE craftsky_project_posts
		SET materials = $2, colors = $3, design_tags = $4, project_tags = $5
		WHERE uri = $1`, uri, materials, colors, designTags, projectTags); err != nil {
		t.Fatalf("seed project details: %v", err)
	}
}

func seedPostTags(t *testing.T, pool *pgxpool.Pool, uri string, tags []string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_posts SET tags = $2 WHERE uri = $1`, uri, tags); err != nil {
		t.Fatalf("seed tags: %v", err)
	}
}

func searchURIs(rows []api.SearchPostRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Post.URI)
	}
	return out
}

func TestSearchStore_SearchProjectsPopularOrdersBrowseAllAndFilteredProjects(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol", "did:plc:fan1", "did:plc:fan2", "did:plc:fan3"} {
		seedMember(t, pool, did)
	}
	knitting := "social.craftsky.feed.defs#knitting"
	crochet := "social.craftsky.feed.defs#crochet"
	high := seedSearchProject(t, pool, "did:plc:alice", "older-popular", "popular socks", knitting, "Popular Socks", now.Add(-48*time.Hour))
	newer := seedSearchProject(t, pool, "did:plc:bob", "newer-quiet", "quiet socks", knitting, "Quiet Socks", now.Add(-1*time.Hour))
	otherCraft := seedSearchProject(t, pool, "did:plc:carol", "crochet-popular", "popular crochet", crochet, "Crochet", now.Add(-2*time.Hour))
	seedInteraction(t, pool, "like", "did:plc:fan1", "like-high-1", high, false)
	seedInteraction(t, pool, "like", "did:plc:fan2", "like-high-2", high, false)
	seedInteraction(t, pool, "repost", "did:plc:fan3", "repost-high", high, false)
	seedInteraction(t, pool, "like", "did:plc:fan1", "like-other", otherCraft, false)

	store := api.NewSearchStore(pool)
	rows, cursor, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortPopular, Limit: 10, Filters: map[string][]string{}}, now)
	if err != nil {
		t.Fatalf("SearchProjects popular browse: %v", err)
	}
	if cursor != "" {
		t.Fatalf("cursor = %q, want empty", cursor)
	}
	if got := searchURIs(rows); !slices.Equal(got[:2], []string{high, otherCraft}) {
		t.Fatalf("popular browse URIs = %v, want %s then %s before quiet newer %s", got, high, otherCraft, newer)
	}

	filtered, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortPopular, Limit: 10, Filters: map[string][]string{"craftType": {"knitting"}}}, now)
	if err != nil {
		t.Fatalf("SearchProjects popular filtered: %v", err)
	}
	if got := searchURIs(filtered); !slices.Equal(got, []string{high, newer}) {
		t.Fatalf("popular filtered URIs = %v, want [%s %s]", got, high, newer)
	}
}

func TestSearchStore_SearchProfilesPaginatesByRankTuple(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:alice", "did:plc:alicia", "did:plc:mallory"} {
		seedMember(t, pool, did)
	}
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.craftsky.social", "Viewer", "")
	seedSearchIdentity(t, pool, "did:plc:alice", "alice.craftsky.social", "Alice", "")
	seedSearchIdentity(t, pool, "did:plc:alicia", "alicia.craftsky.social", "Alicia", "")
	seedSearchIdentity(t, pool, "did:plc:mallory", "mallory.craftsky.social", "Mallory", "ali bio match")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:mallory", "follow-mallory")

	store := api.NewSearchStore(pool)
	page1, cursor, err := store.SearchProfiles(ctx, "did:plc:viewer", api.ProfileSearchRequest{Query: "ali", Limit: 2})
	if err != nil {
		t.Fatalf("SearchProfiles page1: %v", err)
	}
	if cursor == "" {
		t.Fatal("cursor = empty, want next page")
	}
	if got := []string{page1[0].DID, page1[1].DID}; !slices.Equal(got, []string{"did:plc:mallory", "did:plc:alice"}) {
		t.Fatalf("page1 DIDs = %v", got)
	}
	page2, cursor2, err := store.SearchProfiles(ctx, "did:plc:viewer", api.ProfileSearchRequest{Query: "ali", Limit: 2, Cursor: cursor})
	if err != nil {
		t.Fatalf("SearchProfiles page2: %v", err)
	}
	if cursor2 != "" {
		t.Fatalf("cursor2 = %q, want empty", cursor2)
	}
	if got := []string{page2[0].DID}; !slices.Equal(got, []string{"did:plc:alicia"}) {
		t.Fatalf("page2 DIDs = %v", got)
	}
}

func TestSearchAndFacetProfileSuggestionsShareRankingAndCrafts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:display", "did:plc:description"} {
		seedMember(t, pool, did)
	}
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.craftsky.social", "Viewer", "")
	seedSearchIdentity(t, pool, "did:plc:display", "zzz.craftsky.social", "Alice Maker", "")
	seedSearchIdentity(t, pool, "did:plc:description", "aaa.craftsky.social", "Maker", "Alice in bio")
	if _, err := pool.Exec(ctx, `UPDATE craftsky_profiles SET crafts = ARRAY['social.craftsky.feed.defs#knitting'] WHERE did = 'did:plc:display'`); err != nil {
		t.Fatalf("seed display crafts: %v", err)
	}

	searchRows, _, err := api.NewSearchStore(pool).SearchProfiles(ctx, "did:plc:viewer", api.ProfileSearchRequest{Query: "alice", Limit: 10})
	if err != nil {
		t.Fatalf("SearchProfiles: %v", err)
	}
	facetRows, err := api.NewFacetStore(pool).SearchMentionSuggestions(ctx, syntax.DID("did:plc:viewer"), "alice", 10, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("SearchMentionSuggestions: %v", err)
	}
	if len(searchRows) < 2 {
		t.Fatalf("search rows = %#v, want at least two matches", searchRows)
	}
	if len(facetRows) < 2 {
		t.Fatalf("facet rows = %#v, want at least two matches", facetRows)
	}
	searchDIDs := []string{searchRows[0].DID, searchRows[1].DID}
	facetDIDs := []string{facetRows[0].DID, facetRows[1].DID}
	if !slices.Equal(searchDIDs, []string{"did:plc:display", "did:plc:description"}) {
		t.Fatalf("search DIDs = %v", searchDIDs)
	}
	if !slices.Equal(facetDIDs, searchDIDs) {
		t.Fatalf("facet DIDs = %v, want same overlapping order as search %v", facetDIDs, searchDIDs)
	}
	if got := api.BuildProfileSearchSummary(searchRows[0]).Crafts; !slices.Equal(got, []string{"social.craftsky.feed.defs#knitting"}) {
		t.Fatalf("search summary crafts = %v", got)
	}
}

func TestFacetHashtagSuggestionsUseHashtagResultRanking(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	for i, tag := range []string{"sock", "sockkal", "sockkal", "sockkal", "mending-sock", "mending-sock", "mending-sock", "mending-sock"} {
		uri := seedPost(t, pool, "did:plc:alice", "tag-rank-"+string(rune('a'+i)), "tagged", now.Add(time.Duration(-i)*time.Minute))
		seedPostTags(t, pool, uri, []string{tag})
	}

	rows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "#Sock", 3, now)
	if err != nil {
		t.Fatalf("SearchHashtagSuggestions: %v", err)
	}
	got := make([]api.HashtagSuggestionRow, 0, len(rows))
	for _, row := range rows {
		got = append(got, api.HashtagSuggestionRow{Tag: row.Tag, PostsLast28Days: row.PostsLast28Days})
	}
	want := []api.HashtagSuggestionRow{
		{Tag: "sock", PostsLast28Days: 1},
		{Tag: "sockkal", PostsLast28Days: 3},
		{Tag: "mending-sock", PostsLast28Days: 4},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("hashtag suggestions = %#v, want %#v", got, want)
	}
}

func TestFacetHashtagSuggestionsUseVisibleSearchHashtagCounts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}

	visibleDuplicate := seedPost(t, pool, "did:plc:alice", "visible-duplicate", "tagged", now.Add(-time.Hour))
	visibleSecond := seedPost(t, pool, "did:plc:alice", "visible-second", "tagged", now.Add(-2*time.Hour))
	visibleMending := seedPost(t, pool, "did:plc:alice", "visible-mending", "tagged", now.Add(-3*time.Hour))
	hidden := seedPost(t, pool, "did:plc:bob", "hidden", "hidden", now.Add(-30*time.Minute))
	takedown := seedPost(t, pool, "did:plc:carol", "author-takedown", "hidden author", now.Add(-20*time.Minute))
	old := seedPost(t, pool, "did:plc:alice", "old", "old", now.Add(-29*24*time.Hour))
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply", "reply", visibleDuplicate, visibleDuplicate, now.Add(-10*time.Minute))

	seedPostTags(t, pool, visibleDuplicate, []string{"SockKAL", "sockkal"})
	seedPostTags(t, pool, visibleSecond, []string{"sockkal"})
	seedPostTags(t, pool, visibleMending, []string{"sockmending"})
	seedPostTags(t, pool, hidden, []string{"sockkal"})
	seedPostTags(t, pool, takedown, []string{"sockmending"})
	seedPostTags(t, pool, old, []string{"sockkal"})
	seedPostTags(t, pool, reply, []string{"sockkal"})
	seedModerationOutput(t, pool, "post", "did:plc:bob", hidden, "hide", now)
	seedModerationOutput(t, pool, "account", "did:plc:carol", "", "takedown", now)

	facetRows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "sock", 10, now)
	if err != nil {
		t.Fatalf("SearchHashtagSuggestions: %v", err)
	}
	searchRows, _, err := api.NewSearchStore(pool).SearchHashtags(ctx, api.HashtagSearchRequest{Query: "sock", Limit: 10}, now)
	if err != nil {
		t.Fatalf("SearchHashtags: %v", err)
	}

	facetGot := make([]api.HashtagSuggestionRow, 0, len(facetRows))
	for _, row := range facetRows {
		facetGot = append(facetGot, api.HashtagSuggestionRow{Tag: row.Tag, PostsLast28Days: row.PostsLast28Days})
	}
	searchGot := make([]api.HashtagSuggestionRow, 0, len(searchRows))
	for _, row := range searchRows {
		searchGot = append(searchGot, api.HashtagSuggestionRow{Tag: row.Tag, PostsLast28Days: row.PostsLast28Days})
	}
	want := []api.HashtagSuggestionRow{
		{Tag: "sockkal", PostsLast28Days: 2},
		{Tag: "sockmending", PostsLast28Days: 1},
	}
	if !slices.Equal(searchGot, want) {
		t.Fatalf("search hashtag counts = %#v, want %#v", searchGot, want)
	}
	if !slices.Equal(facetGot, want) {
		t.Fatalf("facet hashtag counts = %#v, want same visible counts %#v", facetGot, want)
	}
	if !slices.Equal(facetGot, searchGot) {
		t.Fatalf("facet hashtag counts = %#v, want same as search path %#v", facetGot, searchGot)
	}
}

func TestSearchSuggestionsHandlerReturnsGroupedTopNSections(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	now := time.Now().UTC().Add(-time.Hour)
	for _, did := range []string{"did:plc:viewer", "did:plc:sock-a", "did:plc:sock-b"} {
		seedMember(t, pool, did)
	}
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.craftsky.social", "Viewer", "")
	seedSearchIdentity(t, pool, "did:plc:sock-a", "sockalpha.craftsky.social", "Sock Alpha", "")
	seedSearchIdentity(t, pool, "did:plc:sock-b", "sockbeta.craftsky.social", "Sock Beta", "")
	if _, err := pool.Exec(ctx, `UPDATE craftsky_profiles SET crafts = ARRAY['social.craftsky.feed.defs#knitting'] WHERE did = 'did:plc:sock-a'`); err != nil {
		t.Fatalf("seed crafts: %v", err)
	}
	for i, tag := range []string{"sock", "sockkal"} {
		uri := seedPost(t, pool, "did:plc:viewer", "suggestion-tag-"+string(rune('a'+i)), "tagged", now.Add(time.Duration(-i)*time.Minute))
		seedPostTags(t, pool, uri, []string{tag})
	}

	handler := api.SearchSuggestionsHandler(api.NewSearchStore(pool), slog.New(slog.NewTextHandler(io.Discard, nil)))
	req := httptest.NewRequest(http.MethodGet, "/v1/search/suggestions?q=sock&types=profiles,hashtags&profileLimit=1&hashtagLimit=1", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:viewer")))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "cursor") {
		t.Fatalf("suggestions response must not include pagination cursor: %s", rr.Body.String())
	}
	var body api.SearchSuggestionsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Profiles.Items) != 1 || !body.Profiles.HasMore || body.Profiles.Items[0].DID.String() != "did:plc:sock-a" {
		t.Fatalf("profiles section = %#v", body.Profiles)
	}
	if got := body.Profiles.Items[0].Crafts; !slices.Equal(got, []string{"social.craftsky.feed.defs#knitting"}) {
		t.Fatalf("profile crafts = %v", got)
	}
	if len(body.Hashtags.Items) != 1 || !body.Hashtags.HasMore || body.Hashtags.Items[0].Tag != "sock" {
		t.Fatalf("hashtags section = %#v", body.Hashtags)
	}
}

func TestSearchStore_SearchHashtagsRanksAndPaginates(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	for i, tag := range []string{"sock", "sockkal", "sockkal", "sockkal", "sockmending", "sockmending", "mending-sock", "mending-sock", "mending-sock", "mending-sock"} {
		uri := seedPost(t, pool, "did:plc:alice", "hashtag-page-"+string(rune('a'+i)), "tagged", now.Add(time.Duration(-i)*time.Minute))
		seedPostTags(t, pool, uri, []string{tag})
	}

	store := api.NewSearchStore(pool)
	page1, cursor, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: "#Sock", Limit: 2}, now)
	if err != nil {
		t.Fatalf("SearchHashtags page1: %v", err)
	}
	wantPage1 := []api.HashtagSearchResult{{Tag: "sock", PostsLast28Days: 1}, {Tag: "sockkal", PostsLast28Days: 3}}
	if !slices.Equal(page1, wantPage1) || cursor == "" {
		t.Fatalf("page1 = %#v cursor=%q, want %#v and cursor", page1, cursor, wantPage1)
	}
	page2, cursor2, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: "sock", Limit: 2, Cursor: cursor}, now)
	if err != nil {
		t.Fatalf("SearchHashtags page2: %v", err)
	}
	wantPage2 := []api.HashtagSearchResult{{Tag: "sockmending", PostsLast28Days: 2}, {Tag: "mending-sock", PostsLast28Days: 4}}
	if !slices.Equal(page2, wantPage2) || cursor2 != "" {
		t.Fatalf("page2 = %#v cursor=%q, want %#v and empty cursor", page2, cursor2, wantPage2)
	}
	if _, _, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: "sock", Limit: 2, Cursor: "bad@@"}, now); err != envelope.ErrInvalidCursor {
		t.Fatalf("invalid cursor error = %v, want envelope.ErrInvalidCursor", err)
	}
}

func TestSearchStore_SearchHashtagPostsUsesStoredTagEqualityOnly(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	exact := seedPost(t, pool, "did:plc:alice", "sock", "tagged sock", base)
	projectExact := seedSearchProject(t, pool, "did:plc:alice", "project-sock", "project", "knitting", "Sock Project", base.Add(-time.Minute))
	substring := seedPost(t, pool, "did:plc:alice", "sockknitting", "tagged sockknitting", base.Add(-2*time.Minute))
	textOnly := seedPost(t, pool, "did:plc:alice", "text-only", "visual #sock", base.Add(-3*time.Minute))
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply-sock", "reply", exact, exact, base.Add(-4*time.Minute))
	seedPostTags(t, pool, exact, []string{"sock"})
	seedPostTags(t, pool, projectExact, []string{"sock"})
	seedPostTags(t, pool, substring, []string{"sockknitting"})
	seedPostTags(t, pool, reply, []string{"sock"})

	rows, _, err := api.NewSearchStore(pool).SearchHashtagPosts(ctx, "sock", api.SearchSortChronological, 10, "", base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts: %v", err)
	}
	got := searchURIs(rows)
	if !slices.Equal(got, []string{exact, projectExact}) {
		t.Fatalf("hashtag URIs = %v; substring=%s textOnly=%s reply=%s must be absent", got, substring, textOnly, reply)
	}
}

func TestSearchStore_SearchHashtagPostsSortsChronologicalAndPopular(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:fan1", "did:plc:fan2", "did:plc:fan3"} {
		seedMember(t, pool, did)
	}
	newestQuiet := seedPost(t, pool, "did:plc:alice", "newest-sock", "newest", base)
	middleProject := seedSearchProject(t, pool, "did:plc:alice", "middle-project-sock", "project", "social.craftsky.feed.defs#knitting", "Middle Sock", base.Add(-time.Hour))
	olderPopular := seedPost(t, pool, "did:plc:alice", "older-popular-sock", "popular", base.Add(-48*time.Hour))
	for _, uri := range []string{newestQuiet, middleProject, olderPopular} {
		seedPostTags(t, pool, uri, []string{"sock"})
	}
	seedInteraction(t, pool, "repost", "did:plc:fan1", "sock-repost-1", olderPopular, false)
	seedInteraction(t, pool, "repost", "did:plc:fan2", "sock-repost-2", olderPopular, false)
	seedInteraction(t, pool, "like", "did:plc:fan3", "sock-like-1", olderPopular, false)

	store := api.NewSearchStore(pool)
	chronPage1, chronCursor, err := store.SearchHashtagPosts(ctx, "sock", api.SearchSortChronological, 2, "", base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts chronological page1: %v", err)
	}
	chronPage2, chronCursor2, err := store.SearchHashtagPosts(ctx, "sock", api.SearchSortChronological, 2, chronCursor, base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts chronological page2: %v", err)
	}
	if !slices.Equal(searchURIs(chronPage1), []string{newestQuiet, middleProject}) || chronCursor == "" || !slices.Equal(searchURIs(chronPage2), []string{olderPopular}) || chronCursor2 != "" {
		t.Fatalf("chronological page1=%v cursor=%q page2=%v cursor2=%q", searchURIs(chronPage1), chronCursor, searchURIs(chronPage2), chronCursor2)
	}
	popularRows, _, err := store.SearchHashtagPosts(ctx, "sock", api.SearchSortPopular, 10, "", base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts popular: %v", err)
	}
	if got := searchURIs(popularRows); !slices.Equal(got, []string{olderPopular, newestQuiet, middleProject}) {
		t.Fatalf("popular URIs = %v", got)
	}
}

func TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	newerWeak := seedPost(t, pool, "did:plc:alice", "newer-weak", "alpaca", base)
	olderStrong := seedPost(t, pool, "did:plc:alice", "older-strong", "alpaca alpaca alpaca socks", base.Add(-time.Hour))
	titleMatch := seedSearchProject(t, pool, "did:plc:alice", "title-match", "quiet", "knitting", "Alpaca Socks", base.Add(-time.Minute))
	materialMatch := seedSearchProject(t, pool, "did:plc:alice", "material-match", "quiet", "knitting", "Hat", base.Add(-2*time.Minute))
	seedProjectDetails(t, pool, materialMatch, []string{"alpaca"}, nil, []string{"cables"}, []string{"kal"})
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply-alpaca", "alpaca reply", newerWeak, newerWeak, base.Add(time.Minute))

	store := api.NewSearchStore(pool)
	postRows, _, err := store.SearchPosts(ctx, api.PostSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 10}, base)
	if err != nil {
		t.Fatalf("SearchPosts: %v", err)
	}
	if got := searchURIs(postRows); !slices.Equal(got, []string{olderStrong, newerWeak}) {
		t.Fatalf("post keyword URIs = %v; projects %s/%s and reply %s must be absent; older stronger match should outrank newer weak match", got, titleMatch, materialMatch, reply)
	}
	postPage1, postCursor, err := store.SearchPosts(ctx, api.PostSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 1}, base)
	if err != nil {
		t.Fatalf("SearchPosts page1: %v", err)
	}
	postPage2, postCursor2, err := store.SearchPosts(ctx, api.PostSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 1, Cursor: postCursor}, base)
	if err != nil {
		t.Fatalf("SearchPosts page2: %v", err)
	}
	if !slices.Equal(searchURIs(postPage1), []string{olderStrong}) || postCursor == "" || !slices.Equal(searchURIs(postPage2), []string{newerWeak}) || postCursor2 != "" {
		t.Fatalf("post pagination page1=%v cursor=%q page2=%v cursor2=%q", searchURIs(postPage1), postCursor, searchURIs(postPage2), postCursor2)
	}
	projectRows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 10, Filters: map[string][]string{}}, base)
	if err != nil {
		t.Fatalf("SearchProjects keyword: %v", err)
	}
	if got := searchURIs(projectRows); !slices.Equal(got, []string{titleMatch, materialMatch}) {
		t.Fatalf("project keyword URIs = %v", got)
	}
	projectPage1, projectCursor, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 1, Filters: map[string][]string{}}, base)
	if err != nil {
		t.Fatalf("SearchProjects page1: %v", err)
	}
	projectPage2, projectCursor2, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 1, Cursor: projectCursor, Filters: map[string][]string{}}, base)
	if err != nil {
		t.Fatalf("SearchProjects page2: %v", err)
	}
	if !slices.Equal(searchURIs(projectPage1), []string{titleMatch}) || projectCursor == "" || !slices.Equal(searchURIs(projectPage2), []string{materialMatch}) || projectCursor2 != "" {
		t.Fatalf("project pagination page1=%v cursor=%q page2=%v cursor2=%q", searchURIs(projectPage1), projectCursor, searchURIs(projectPage2), projectCursor2)
	}
}

func TestSearchStore_SearchProjectsAppliesFilterSemantics(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	knitting := "social.craftsky.feed.defs#knitting"
	crochetToken := "social.craftsky.feed.defs#crochet"
	socks := seedSearchProject(t, pool, "did:plc:alice", "socks", "", knitting, "Socks", base)
	shawl := seedSearchProject(t, pool, "did:plc:alice", "shawl", "", knitting, "Shawl", base.Add(-time.Minute))
	crochet := seedSearchProject(t, pool, "did:plc:alice", "crochet", "", crochetToken, "Bag", base.Add(-2*time.Minute))
	seedProjectDetails(t, pool, socks, []string{"Alpaca"}, []string{"Blue"}, []string{"Cables"}, []string{"KAL"})
	seedProjectDetails(t, pool, shawl, []string{"Wool"}, []string{"Green"}, []string{"Lace"}, []string{"Gift"})
	seedProjectDetails(t, pool, crochet, []string{"Cotton"}, []string{"Blue"}, []string{"Granny"}, []string{"KAL"})

	store := api.NewSearchStore(pool)
	orRows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 10, Filters: map[string][]string{"craftType": {"knitting", "crochet"}}}, base)
	if err != nil {
		t.Fatalf("SearchProjects OR filters: %v", err)
	}
	if got := searchURIs(orRows); !slices.Equal(got, []string{socks, shawl, crochet}) {
		t.Fatalf("OR filter URIs = %v", got)
	}
	page1, cursor, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 1, Filters: map[string][]string{"craftType": {"knitting"}}}, base)
	if err != nil {
		t.Fatalf("SearchProjects page1: %v", err)
	}
	page2, cursor2, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 1, Cursor: cursor, Filters: map[string][]string{"craftType": {"knitting"}}}, base)
	if err != nil {
		t.Fatalf("SearchProjects page2: %v", err)
	}
	if !slices.Equal(searchURIs(page1), []string{socks}) || cursor == "" || !slices.Equal(searchURIs(page2), []string{shawl}) || cursor2 != "" {
		t.Fatalf("pagination page1=%v cursor=%q page2=%v cursor2=%q", searchURIs(page1), cursor, searchURIs(page2), cursor2)
	}
	andRows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 10, Filters: map[string][]string{"craftType": {"knitting"}, "color": {"blue"}, "material": {"alpaca"}}}, base)
	if err != nil {
		t.Fatalf("SearchProjects AND filters: %v", err)
	}
	if got := searchURIs(andRows); !slices.Equal(got, []string{socks}) {
		t.Fatalf("AND filter URIs = %v", got)
	}
}

func TestSearchStore_ModerationFiltersBeforeSearchRankingAndLimits(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:fan"} {
		seedMember(t, pool, did)
	}
	visible := seedPost(t, pool, "did:plc:alice", "visible", "sock", base.Add(-time.Hour))
	hidden := seedPost(t, pool, "did:plc:bob", "hidden", "sock", base)
	seedInteraction(t, pool, "repost", "did:plc:fan", "hidden-repost", hidden, false)
	seedPostTags(t, pool, visible, []string{"sock"})
	seedPostTags(t, pool, hidden, []string{"sock"})
	seedModerationOutput(t, pool, "post", "did:plc:bob", hidden, "hide", base.Add(time.Minute))

	rows, _, err := api.NewSearchStore(pool).SearchHashtagPosts(ctx, "sock", api.SearchSortPopular, 1, "", base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts moderated: %v", err)
	}
	if got := searchURIs(rows); !slices.Equal(got, []string{visible}) {
		t.Fatalf("moderated popular URIs = %v, hidden %s must not consume limit", got, hidden)
	}
}

func TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	knitting := "social.craftsky.feed.defs#knitting"
	crochetToken := "social.craftsky.feed.defs#crochet"
	knit1 := seedSearchProject(t, pool, "did:plc:alice", "knit1", "", knitting, "", now.Add(-time.Hour))
	knit2 := seedSearchProject(t, pool, "did:plc:alice", "knit2", "", knitting, "", now.Add(-2*time.Hour))
	crochet := seedSearchProject(t, pool, "did:plc:alice", "crochet1", "", crochetToken, "", now.Add(-3*time.Hour))
	old := seedSearchProject(t, pool, "did:plc:alice", "old", "", knitting, "", now.Add(-29*24*time.Hour))
	hidden := seedSearchProject(t, pool, "did:plc:alice", "hidden", "", knitting, "", now.Add(-30*time.Minute))
	regular := seedPost(t, pool, "did:plc:alice", "regular-sock", "", now.Add(-15*time.Minute))
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at = $2 WHERE uri = $1`, old, now.Add(-29*24*time.Hour)); err != nil {
		t.Fatalf("age old post: %v", err)
	}
	seedPostTags(t, pool, knit1, []string{"sock", "sock"})
	seedPostTags(t, pool, knit2, []string{"sock", "sweater"})
	seedPostTags(t, pool, crochet, []string{"granny"})
	seedPostTags(t, pool, old, []string{"old"})
	seedPostTags(t, pool, hidden, []string{"sock"})
	seedPostTags(t, pool, regular, []string{"sock"})
	seedModerationOutput(t, pool, "post", "did:plc:alice", hidden, "hide", now)

	groups, err := api.NewSearchStore(pool).TopHashtags(ctx, api.TopHashtagsRequest{Limit: 10}, now)
	if err != nil {
		t.Fatalf("TopHashtags: %v", err)
	}
	wantCrafts := []string{
		"social.craftsky.feed.defs#knitting",
		"social.craftsky.feed.defs#crochet",
		"social.craftsky.feed.defs#sewing",
		"social.craftsky.feed.defs#embroidery",
		"social.craftsky.feed.defs#quilting",
	}
	gotCrafts := make([]string, 0, len(groups))
	for _, group := range groups {
		gotCrafts = append(gotCrafts, group.CraftType)
	}
	if !slices.Equal(gotCrafts, wantCrafts) {
		t.Fatalf("groups = %#v", groups)
	}
	if got := groups[0].Items; len(got) != 2 || got[0].Tag != "sock" || got[0].Count != 2 || got[1].Tag != "sweater" || got[1].Count != 1 {
		t.Fatalf("knitting items = %#v", got)
	}
	if got := groups[1].Items; len(got) != 1 || got[0].Tag != "granny" || got[0].Count != 1 {
		t.Fatalf("crochet items = %#v", got)
	}
	for i := 2; i < len(groups); i++ {
		if len(groups[i].Items) != 0 {
			t.Fatalf("%s items = %#v, want empty", groups[i].CraftType, groups[i].Items)
		}
	}

	mixedGroups, err := api.NewSearchStore(pool).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting", crochetToken, knitting}, Limit: 10}, now)
	if err != nil {
		t.Fatalf("TopHashtags mixed aliases: %v", err)
	}
	mixedCrafts := make([]string, 0, len(mixedGroups))
	for _, group := range mixedGroups {
		mixedCrafts = append(mixedCrafts, group.CraftType)
	}
	if !slices.Equal(mixedCrafts, []string{knitting, crochetToken}) {
		t.Fatalf("mixed groups = %#v", mixedGroups)
	}
}
