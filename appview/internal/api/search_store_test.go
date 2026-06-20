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

const searchStoreDDL = timelineStoreDDL + `
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
	high := seedSearchProject(t, pool, "did:plc:alice", "older-popular", "popular socks", "knitting", "Popular Socks", now.Add(-48*time.Hour))
	newer := seedSearchProject(t, pool, "did:plc:bob", "newer-quiet", "quiet socks", "knitting", "Quiet Socks", now.Add(-1*time.Hour))
	otherCraft := seedSearchProject(t, pool, "did:plc:carol", "crochet-popular", "popular crochet", "crochet", "Crochet", now.Add(-2*time.Hour))
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

func TestSearchStore_SearchPostsAndProjectsUseFTSFields(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	textMatch := seedPost(t, pool, "did:plc:alice", "text-match", "alpaca mitts", base)
	titleMatch := seedSearchProject(t, pool, "did:plc:alice", "title-match", "quiet", "knitting", "Alpaca Socks", base.Add(-time.Minute))
	materialMatch := seedSearchProject(t, pool, "did:plc:alice", "material-match", "quiet", "knitting", "Hat", base.Add(-2*time.Minute))
	seedProjectDetails(t, pool, materialMatch, []string{"alpaca"}, nil, []string{"cables"}, []string{"kal"})
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply-alpaca", "alpaca reply", textMatch, textMatch, base.Add(time.Minute))

	store := api.NewSearchStore(pool)
	postRows, _, err := store.SearchPosts(ctx, api.PostSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 10}, base)
	if err != nil {
		t.Fatalf("SearchPosts: %v", err)
	}
	if got := searchURIs(postRows); !slices.Equal(got, []string{textMatch, titleMatch, materialMatch}) {
		t.Fatalf("post keyword URIs = %v; reply %s must be absent", got, reply)
	}
	projectRows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 10, Filters: map[string][]string{}}, base)
	if err != nil {
		t.Fatalf("SearchProjects keyword: %v", err)
	}
	if got := searchURIs(projectRows); !slices.Equal(got, []string{titleMatch, materialMatch}) {
		t.Fatalf("project keyword URIs = %v", got)
	}
}

func TestSearchStore_SearchProjectsAppliesFilterSemantics(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, searchStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	socks := seedSearchProject(t, pool, "did:plc:alice", "socks", "", "knitting", "Socks", base)
	shawl := seedSearchProject(t, pool, "did:plc:alice", "shawl", "", "knitting", "Shawl", base.Add(-time.Minute))
	crochet := seedSearchProject(t, pool, "did:plc:alice", "crochet", "", "crochet", "Bag", base.Add(-2*time.Minute))
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
	knit1 := seedSearchProject(t, pool, "did:plc:alice", "knit1", "", "knitting", "", now.Add(-time.Hour))
	knit2 := seedSearchProject(t, pool, "did:plc:alice", "knit2", "", "knitting", "", now.Add(-2*time.Hour))
	crochet := seedSearchProject(t, pool, "did:plc:alice", "crochet1", "", "crochet", "", now.Add(-3*time.Hour))
	old := seedSearchProject(t, pool, "did:plc:alice", "old", "", "knitting", "", now.Add(-29*24*time.Hour))
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at = $2 WHERE uri = $1`, old, now.Add(-29*24*time.Hour)); err != nil {
		t.Fatalf("age old post: %v", err)
	}
	seedPostTags(t, pool, knit1, []string{"sock", "sock"})
	seedPostTags(t, pool, knit2, []string{"sock", "sweater"})
	seedPostTags(t, pool, crochet, []string{"granny"})
	seedPostTags(t, pool, old, []string{"old"})

	groups, err := api.NewSearchStore(pool).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting", "crochet", "quilting"}, Limit: 10}, now)
	if err != nil {
		t.Fatalf("TopHashtags: %v", err)
	}
	if len(groups) != 3 || groups[0].CraftType != "knitting" || groups[1].CraftType != "crochet" || groups[2].CraftType != "quilting" {
		t.Fatalf("groups = %#v", groups)
	}
	if got := groups[0].Items; len(got) != 2 || got[0].Tag != "sock" || got[0].Count != 2 || got[1].Tag != "sweater" || got[1].Count != 1 {
		t.Fatalf("knitting items = %#v", got)
	}
	if got := groups[1].Items; len(got) != 1 || got[0].Tag != "granny" || got[0].Count != 1 {
		t.Fatalf("crochet items = %#v", got)
	}
	if len(groups[2].Items) != 0 {
		t.Fatalf("quilting items = %#v, want empty", groups[2].Items)
	}
}
