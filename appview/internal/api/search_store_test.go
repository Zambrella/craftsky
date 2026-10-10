package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
	"social.craftsky/appview/internal/testlog"
)

const searchStoreBaseDDL = timelineStoreDDL + `
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

// Load migration files only when a search fixture is used. Standalone release
// media probes run this package without a source checkout or database fixtures.
func searchStoreDDL(t *testing.T) string {
	t.Helper()
	contents, err := testdb.ReadMigration("000089_search_matching_helpers.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	return searchStoreBaseDDL + string(contents)
}

func hashtagSearchTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, searchStoreDDL(t))
	if err := testdb.ApplyMigrations(context.Background(), pool, "000090_hashtag_spellings.up.sql"); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestSearchProfilesOmitsBlockedAccountExceptExactHandleManagementShell(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:bob", "did:plc:carol"} {
		seedMember(t, pool, did)
	}
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.example", "Viewer", "viewer bio")
	seedSearchIdentity(t, pool, "did:plc:bob", "bob.example", "Bob Maker", "blocked bio")
	seedSearchIdentity(t, pool, "did:plc:carol", "carol.example", "Carol", "carol bio")
	seedBlockAggregate(t, pool, "did:plc:viewer", "did:plc:bob", time.Now())
	store := api.NewSearchStore(pool, nil)

	ordinary, _, err := store.SearchProfiles(ctx, "did:plc:viewer", api.ProfileSearchRequest{Query: "bob", Limit: 10})
	if err != nil {
		t.Fatalf("ordinary blocked search: %v", err)
	}
	if len(ordinary) != 0 {
		t.Fatalf("ordinary search exposed blocked account: %+v", ordinary)
	}
	exact, _, err := store.SearchProfiles(ctx, "did:plc:viewer", api.ProfileSearchRequest{Query: "bob.example", Limit: 10})
	if err != nil {
		t.Fatalf("exact blocked search: %v", err)
	}
	if len(exact) != 1 || exact[0].DID != "did:plc:bob" || !exact[0].Blocking || exact[0].BlockedBy {
		t.Fatalf("exact management row = %+v", exact)
	}
	raw, err := json.Marshal(api.BuildProfileSearchSummary(exact[0]))
	if err != nil {
		t.Fatal(err)
	}
	var shell map[string]any
	_ = json.Unmarshal(raw, &shell)
	for _, forbidden := range []string{"description", "crafts", "viewerIsFollowing"} {
		if _, ok := shell[forbidden]; ok {
			t.Fatalf("exact blocked shell leaked %q: %s", forbidden, raw)
		}
	}
	hydratedShell, err := api.NewIdentityAccountTypeHydrator(&summaryAccountTypeReader{values: map[syntax.DID]business.AccountType{
		"did:plc:bob": business.AccountTypeBusiness,
	}}).HydrateJSON(ctx, raw)
	if err != nil {
		t.Fatalf("hydrate blocked search shell: %v", err)
	}
	if bytes.Contains(hydratedShell, []byte(`"accountType"`)) || bytes.Contains(hydratedShell, []byte(`"business"`)) {
		t.Fatalf("blocked production search shell exposed business data: %s", hydratedShell)
	}
	searchHandler := hydrateProductionSummaries(api.SearchProfilesHandler(store, nilLogger()), map[syntax.DID]business.AccountType{
		"did:plc:bob": business.AccountTypeBusiness,
	})
	searchRequest := httptest.NewRequest(http.MethodGet, "/v1/search/profiles?q=bob.example", nil)
	searchRequest = searchRequest.WithContext(middleware.WithDID(searchRequest.Context(), syntax.DID("did:plc:viewer")))
	searchResponse := httptest.NewRecorder()
	searchHandler.ServeHTTP(searchResponse, searchRequest)
	if searchResponse.Code != http.StatusOK || bytes.Contains(searchResponse.Body.Bytes(), []byte(`"accountType"`)) || bytes.Contains(searchResponse.Body.Bytes(), []byte(`"business"`)) {
		t.Fatalf("blocked search handler response = %d %s", searchResponse.Code, searchResponse.Body.String())
	}

	carol, _, err := store.SearchProfiles(ctx, "did:plc:carol", api.ProfileSearchRequest{Query: "bob", Limit: 10})
	if err != nil {
		t.Fatalf("unrelated search: %v", err)
	}
	if len(carol) != 1 || carol[0].Description == nil || *carol[0].Description != "blocked bio" {
		t.Fatalf("unrelated search result = %+v", carol)
	}
	visibleRaw, err := json.Marshal(api.BuildProfileSearchSummary(carol[0]))
	if err != nil {
		t.Fatalf("marshal visible search result: %v", err)
	}
	visibleHydrated, err := api.NewIdentityAccountTypeHydrator(&summaryAccountTypeReader{values: map[syntax.DID]business.AccountType{
		"did:plc:bob": business.AccountTypeBusiness,
	}}).HydrateJSON(ctx, visibleRaw)
	if err != nil || !bytes.Contains(visibleHydrated, []byte(`"accountType":"business"`)) {
		t.Fatalf("visible production search accountType missing: %s, error %v", visibleHydrated, err)
	}
}

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
	// Supply public source facets as well as the normalized test projection.
	text := ""
	facets := []any{}
	for _, tag := range tags {
		start := len(text)
		text += "#" + tag + " "
		facets = append(facets, map[string]any{"index": map[string]int{"byteStart": start, "byteEnd": len(text) - 1}, "features": []any{map[string]string{"$type": "app.bsky.richtext.facet#tag", "tag": tag}}})
	}
	normalized := make([]string, len(tags))
	for i, tag := range tags {
		normalized[i] = strings.ToLower(tag)
	}
	source, err := json.Marshal(map[string]any{"text": text, "facets": facets})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `UPDATE craftsky_posts SET tags = $2, record = record || $3::jsonb WHERE uri = $1`, uri, normalized, source); err != nil {
		t.Fatalf("seed tags: %v", err)
	}
}

func hydrateHashtagFixtureSources(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT uri,tags FROM craftsky_posts`)
	if err != nil {
		t.Fatal(err)
	}
	type seed struct {
		uri  string
		tags []string
	}
	seeds := []seed{}
	for rows.Next() {
		var v seed
		if err := rows.Scan(&v.uri, &v.tags); err != nil {
			t.Fatal(err)
		}
		seeds = append(seeds, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	for _, v := range seeds {
		seedPostTags(t, pool, v.uri, v.tags)
	}
}

func searchURIs(rows []api.SearchPostRow) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Post.URI)
	}
	return out
}

func TestSearchStore_ProjectBrowseAppliesLanguageVisibilityBeforePagination(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	now := time.Date(2026, 7, 29, 13, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:viewer", "did:plc:alice"} {
		seedMember(t, pool, did)
	}
	craft := "social.craftsky.feed.defs#knitting"
	english := seedSearchProject(t, pool, "did:plc:alice", "english-project", "English", craft, "English", now.Add(4*time.Minute))
	french := seedSearchProject(t, pool, "did:plc:alice", "french-project", "French", craft, "French", now.Add(3*time.Minute))
	multilingual := seedSearchProject(t, pool, "did:plc:alice", "multi-project", "Multi", craft, "Multi", now.Add(2*time.Minute))
	untagged := seedSearchProject(t, pool, "did:plc:alice", "untagged-project", "Legacy", craft, "Legacy", now.Add(time.Minute))
	ownFrench := seedSearchProject(t, pool, "did:plc:viewer", "own-project", "Mine", craft, "Mine", now)
	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_posts
		SET langs = CASE uri
			WHEN $1 THEN ARRAY['en']::text[]
			WHEN $2 THEN ARRAY['fr']::text[]
			WHEN $3 THEN ARRAY['en', 'fr']::text[]
			WHEN $4 THEN ARRAY['fr']::text[]
			ELSE langs
		END
		WHERE uri IN ($1, $2, $3, $4)
	`, english, french, multilingual, ownFrench); err != nil {
		t.Fatalf("seed languages: %v", err)
	}

	store := api.NewSearchStore(pool, nil)
	request := api.ProjectSearchRequest{
		Sort:    api.SearchSortChronological,
		Limit:   2,
		Filters: map[string][]string{},
	}
	first, cursor, err := store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, request, now)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if got, want := searchURIs(first), []string{english, multilingual}; !slices.Equal(got, want) || cursor == "" {
		t.Fatalf("first page = %v cursor=%q, want %v and cursor", got, cursor, want)
	}
	request.Cursor = cursor
	second, next, err := store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, request, now)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if got, want := searchURIs(second), []string{ownFrench}; !slices.Equal(got, want) || next != "" {
		t.Fatalf("second page = %v cursor=%q, want %v and terminal", got, next, want)
	}
	if slices.Contains(searchURIs(first), french) || slices.Contains(searchURIs(first), untagged) {
		t.Fatal("project browse leaked mismatched or untagged content")
	}
}

func TestSearchStore_PostSearchAppliesLanguageVisibilityBeforePagination(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	base := time.Date(2026, 7, 29, 14, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:viewer", "did:plc:alice"} {
		seedMember(t, pool, did)
	}
	english := seedPost(t, pool, "did:plc:alice", "english-search", "needle English", base.Add(4*time.Minute))
	french := seedPost(t, pool, "did:plc:alice", "french-search", "needle French", base.Add(3*time.Minute))
	multilingual := seedPost(t, pool, "did:plc:alice", "multi-search", "needle Multi", base.Add(2*time.Minute))
	untagged := seedPost(t, pool, "did:plc:alice", "untagged-search", "needle Legacy", base.Add(time.Minute))
	ownFrench := seedPost(t, pool, "did:plc:viewer", "own-search", "needle Mine", base)
	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_posts
		SET langs = CASE uri
			WHEN $1 THEN ARRAY['en']::text[]
			WHEN $2 THEN ARRAY['fr']::text[]
			WHEN $3 THEN ARRAY['en', 'fr']::text[]
			WHEN $4 THEN ARRAY['fr']::text[]
			ELSE langs
		END
		WHERE uri IN ($1, $2, $3, $4)
	`, english, french, multilingual, ownFrench); err != nil {
		t.Fatalf("seed languages: %v", err)
	}

	store := api.NewSearchStore(pool, nil)
	request := api.PostSearchRequest{
		Query: "needle",
		Sort:  api.SearchSortChronological,
		Limit: 2,
	}
	first, cursor, err := store.SearchPostsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, request, base)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if got, want := searchURIs(first), []string{english, multilingual}; !slices.Equal(got, want) || cursor == "" {
		t.Fatalf("first page = %v cursor=%q, want %v and cursor", got, cursor, want)
	}
	request.Cursor = cursor
	second, next, err := store.SearchPostsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, request, base)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if got, want := searchURIs(second), []string{ownFrench}; !slices.Equal(got, want) || next != "" {
		t.Fatalf("second page = %v cursor=%q, want %v and terminal", got, next, want)
	}
	if slices.Contains(searchURIs(first), french) || slices.Contains(searchURIs(first), untagged) {
		t.Fatal("post search leaked mismatched or untagged content")
	}
}

func TestSearchStore_ProjectSearchAppliesLanguageVisibilityBeforePagination(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	base := time.Date(2026, 7, 29, 15, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:viewer", "did:plc:alice"} {
		seedMember(t, pool, did)
	}
	craft := "social.craftsky.feed.defs#knitting"
	english := seedSearchProject(t, pool, "did:plc:alice", "english-project-search", "needle English", craft, "Needle English", base.Add(3*time.Minute))
	french := seedSearchProject(t, pool, "did:plc:alice", "french-project-search", "needle French", craft, "Needle French", base.Add(2*time.Minute))
	multilingual := seedSearchProject(t, pool, "did:plc:alice", "multi-project-search", "needle Multi", craft, "Needle Multi", base.Add(time.Minute))
	ownFrench := seedSearchProject(t, pool, "did:plc:viewer", "own-project-search", "needle Mine", craft, "Needle Mine", base)
	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_posts
		SET langs = CASE uri
			WHEN $1 THEN ARRAY['en']::text[]
			WHEN $2 THEN ARRAY['fr']::text[]
			WHEN $3 THEN ARRAY['en', 'fr']::text[]
			WHEN $4 THEN ARRAY['fr']::text[]
			ELSE langs
		END
		WHERE uri IN ($1, $2, $3, $4)
	`, english, french, multilingual, ownFrench); err != nil {
		t.Fatalf("seed languages: %v", err)
	}

	store := api.NewSearchStore(pool, nil)
	request := api.ProjectSearchRequest{
		Query:   "needle",
		Sort:    api.SearchSortChronological,
		Limit:   2,
		Filters: map[string][]string{},
	}
	first, cursor, err := store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, request, base)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if got, want := searchURIs(first), []string{english, multilingual}; !slices.Equal(got, want) || cursor == "" {
		t.Fatalf("first page = %v cursor=%q, want %v and cursor", got, cursor, want)
	}
	request.Cursor = cursor
	second, next, err := store.SearchProjectsWithLanguages(ctx, "did:plc:viewer", []string{"en"}, request, base)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if got, want := searchURIs(second), []string{ownFrench}; !slices.Equal(got, want) || next != "" {
		t.Fatalf("second page = %v cursor=%q, want %v and terminal", got, next, want)
	}
}

func TestSearchStore_HashtagPostsApplyLanguageVisibilityBeforePagination(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	base := time.Date(2026, 7, 29, 16, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:viewer", "did:plc:alice"} {
		seedMember(t, pool, did)
	}
	english := seedPost(t, pool, "did:plc:alice", "english-hashtag", "English", base.Add(4*time.Minute))
	french := seedPost(t, pool, "did:plc:alice", "french-hashtag", "French", base.Add(3*time.Minute))
	multilingual := seedPost(t, pool, "did:plc:alice", "multi-hashtag", "Multi", base.Add(2*time.Minute))
	untagged := seedPost(t, pool, "did:plc:alice", "untagged-hashtag", "Legacy", base.Add(time.Minute))
	ownFrench := seedPost(t, pool, "did:plc:viewer", "own-hashtag", "Mine", base)
	for _, uri := range []string{english, french, multilingual, untagged, ownFrench} {
		seedPostTags(t, pool, uri, []string{"weaving"})
	}
	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_posts
		SET langs = CASE uri
			WHEN $1 THEN ARRAY['en']::text[]
			WHEN $2 THEN ARRAY['fr']::text[]
			WHEN $3 THEN ARRAY['en', 'fr']::text[]
			WHEN $4 THEN ARRAY['fr']::text[]
			ELSE langs
		END
		WHERE uri IN ($1, $2, $3, $4)
	`, english, french, multilingual, ownFrench); err != nil {
		t.Fatalf("seed languages: %v", err)
	}

	store := api.NewSearchStore(pool, nil)
	first, cursor, err := store.SearchHashtagPostsWithLanguages(
		ctx,
		"did:plc:viewer",
		[]string{"en"},
		"weaving",
		api.SearchSortChronological,
		2,
		"",
		base,
	)
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if got, want := searchURIs(first), []string{english, multilingual}; !slices.Equal(got, want) || cursor == "" {
		t.Fatalf("first page = %v cursor=%q, want %v and cursor", got, cursor, want)
	}
	second, next, err := store.SearchHashtagPostsWithLanguages(
		ctx,
		"did:plc:viewer",
		[]string{"en"},
		"weaving",
		api.SearchSortChronological,
		2,
		cursor,
		base,
	)
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if got, want := searchURIs(second), []string{ownFrench}; !slices.Equal(got, want) || next != "" {
		t.Fatalf("second page = %v cursor=%q, want %v and terminal", got, next, want)
	}
}

func TestSearchStore_SearchProjectsPopularOrdersBrowseAllAndFilteredProjects(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
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

	store := api.NewSearchStore(pool, nil)
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
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	for _, did := range []string{"did:plc:viewer", "did:plc:alice", "did:plc:alicia", "did:plc:mallory"} {
		seedMember(t, pool, did)
	}
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.craftsky.social", "Viewer", "")
	seedSearchIdentity(t, pool, "did:plc:alice", "alice.craftsky.social", "Alice", "")
	seedSearchIdentity(t, pool, "did:plc:alicia", "alicia.craftsky.social", "Alicia", "")
	seedSearchIdentity(t, pool, "did:plc:mallory", "mallory.craftsky.social", "Mallory", "ali bio match")
	seedFollow(t, pool, "did:plc:viewer", "did:plc:mallory", "follow-mallory")

	store := api.NewSearchStore(pool, nil)
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
	pool := hashtagSearchTestPool(t)
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

	searchRows, _, err := api.NewSearchStore(pool, nil).SearchProfiles(ctx, "did:plc:viewer", api.ProfileSearchRequest{Query: "alice", Limit: 10})
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

// IT-001 / AT-001: BR-001, FR-002, RULE-001; AC-001.
func TestHashtagSuggestionsMostUsedSpelling(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	for i := range 9 {
		spelling := "memademay"
		if i >= 4 {
			spelling = "MeMadeMay"
		}
		text := "#" + spelling
		uri := seedSearchProject(t, pool, "did:plc:alice", fmt.Sprintf("spelling-%d", i), text, "social.craftsky.feed.defs#knitting", "", now.Add(-24*time.Hour))
		record, err := json.Marshal(map[string]any{
			"$type":     "social.craftsky.feed.post",
			"text":      text,
			"project":   map[string]any{"common": map[string]string{"craftType": "social.craftsky.feed.defs#knitting"}},
			"createdAt": now.Add(-24 * time.Hour).Format(time.RFC3339),
			"facets": []any{map[string]any{
				"index": map[string]int{"byteStart": 0, "byteEnd": len(text)},
				"features": []any{map[string]string{
					"$type": "app.bsky.richtext.facet#tag", "tag": spelling,
				}},
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET record = $2::jsonb, tags = ARRAY['memademay'] WHERE uri = $1`, uri, record); err != nil {
			t.Fatalf("seed original hashtag spelling: %v", err)
		}
	}
	rows, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "mema", 10, now)
	if err != nil {
		t.Fatalf("SearchHashtagSuggestions: %v", err)
	}
	want := []api.HashtagSuggestionRow{{Tag: "MeMadeMay", PostsLast28Days: 9}}
	if !slices.Equal(rows, want) {
		t.Fatalf("suggestions = %#v, want %#v", rows, want)
	}
	results, _, err := api.NewSearchStore(pool, nil).SearchHashtags(ctx, api.HashtagSearchRequest{Query: "MeMa", Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Tag != "MeMadeMay" || results[0].PostsLast28Days != 9 {
		t.Fatalf("search results=%v", results)
	}

	groups, err := api.NewSearchStore(pool, nil).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting"}, Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0].Items) != 1 || groups[0].Items[0].Tag != "MeMadeMay" || groups[0].Items[0].Count != 9 {
		t.Fatalf("top hashtags=%v", groups)
	}

}

func TestFacetHashtagSuggestionsUseHashtagResultRanking(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
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
	pool := hashtagSearchTestPool(t)
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
	searchRows, _, err := api.NewSearchStore(pool, nil).SearchHashtags(ctx, api.HashtagSearchRequest{Query: "sock", Limit: 10}, now)
	if err != nil {
		t.Fatalf("SearchHashtags: %v", err)
	}

	facetGot := make([]api.HashtagSuggestionRow, 0, len(facetRows))
	for _, row := range facetRows {
		facetGot = append(facetGot, api.HashtagSuggestionRow(row))
	}
	searchGot := make([]api.HashtagSuggestionRow, 0, len(searchRows))
	for _, row := range searchRows {
		searchGot = append(searchGot, api.HashtagSuggestionRow(row))
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
	pool := hashtagSearchTestPool(t)
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

	handler := api.SearchSuggestionsHandler(api.NewSearchStore(pool, nil), testlog.Discard())
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
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	for i, tag := range []string{"sock", "sockkal", "sockkal", "sockkal", "sockmending", "sockmending", "mending-sock", "mending-sock", "mending-sock", "mending-sock"} {
		uri := seedPost(t, pool, "did:plc:alice", "hashtag-page-"+string(rune('a'+i)), "tagged", now.Add(time.Duration(-i)*time.Minute))
		seedPostTags(t, pool, uri, []string{tag})
	}

	store := api.NewSearchStore(pool, nil)
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
	pool := hashtagSearchTestPool(t)
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

	rows, _, err := api.NewSearchStore(pool, nil).SearchHashtagPosts(ctx, "sock", api.SearchSortChronological, 10, "", base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts: %v", err)
	}
	got := searchURIs(rows)
	if !slices.Equal(got, []string{exact, projectExact}) {
		t.Fatalf("hashtag URIs = %v; substring=%s textOnly=%s reply=%s must be absent", got, substring, textOnly, reply)
	}
}

func TestSearchStoreRelationshipFiltersBeforeHashtagPagination(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	base := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:viewer", "did:plc:bob", "did:plc:carol", "did:plc:dave"} {
		seedMember(t, pool, did)
	}
	for i, did := range []string{"did:plc:bob", "did:plc:carol", "did:plc:dave"} {
		uri := seedPost(t, pool, did, fmt.Sprintf("p%d", i), "knit", base.Add(time.Duration(3-i)*time.Minute))
		if _, err := pool.Exec(context.Background(), `UPDATE craftsky_posts SET tags=ARRAY['knit']::text[] WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO actor_mutes(owner_did,subject_did) VALUES('did:plc:viewer','did:plc:bob')
	`); err != nil {
		t.Fatal(err)
	}
	seedBlockAggregate(t, pool, "did:plc:carol", "did:plc:viewer", time.Now())
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	rows, cursor, err := api.NewSearchStore(pool, nil).SearchHashtagPosts(ctx, "knit", api.SearchSortChronological, 1, "", base.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Post.DID != "did:plc:dave" || cursor != "" {
		t.Fatalf("rows=%+v cursor=%q, want only eligible Dave", rows, cursor)
	}
}

func TestSearchStore_SearchHashtagPostsSortsChronologicalAndPopular(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
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
	seedInteraction(t, pool, "repost", "did:plc:fan1", "sock-repost-duplicate", olderPopular, false)
	seedInteraction(t, pool, "repost", "did:plc:fan2", "sock-repost-2", olderPopular, false)
	seedInteraction(t, pool, "like", "did:plc:fan3", "sock-like-1", olderPopular, false)

	store := api.NewSearchStore(pool, nil)
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
	if want := api.PopularityScore(1, 0, 2, base.Add(-48*time.Hour), base); math.Abs(popularRows[0].Score-want) > 1e-6 {
		t.Fatalf("logical popularity score = %f, want %f", popularRows[0].Score, want)
	}
}

func TestSearchStore_SearchPostsAndProjectsUseRelevanceAndDisjointTabs(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	newerWeak := seedPost(t, pool, "did:plc:alice", "newer-weak", "alpaca", base)
	olderStrong := seedPost(t, pool, "did:plc:alice", "older-strong", "alpaca alpaca alpaca socks", base.Add(-time.Hour))
	titleMatch := seedSearchProject(t, pool, "did:plc:alice", "title-match", "quiet", "knitting", "Alpaca Socks", base.Add(-time.Minute))
	materialMatch := seedSearchProject(t, pool, "did:plc:alice", "material-match", "quiet", "knitting", "Hat", base.Add(-2*time.Minute))
	seedProjectDetails(t, pool, materialMatch, []string{"alpaca"}, nil, []string{"cables"}, []string{"kal"})
	reply := seedReplyPost(t, pool, "did:plc:alice", "reply-alpaca", "alpaca reply", newerWeak, newerWeak, base.Add(time.Minute))

	store := api.NewSearchStore(pool, nil)
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

func TestSearchPostsHandlerIncludesAuthenticatedViewerSavedState(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:alice")
	seedSearchIdentity(t, pool, "did:plc:viewer", "viewer.example", "Viewer", "")
	seedSearchIdentity(t, pool, "did:plc:alice", "alice.example", "Alice", "")
	createdAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	postURI := seedPost(t, pool, "did:plc:alice", "saved-search-result", "unique alpaca search result", createdAt)
	folderID := "11111111-1111-1111-1111-111111111111"
	seedSavedPost(t, pool, "did:plc:viewer", postURI, &folderID, createdAt.Add(time.Minute))

	handler := api.SearchPostsHandler(api.NewSearchStore(pool, nil), fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
	}}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/search/posts?q=alpaca&sort=chronological", "", "did:plc:viewer")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("search status = %d body=%s", response.Code, response.Body.String())
	}
	var page api.SearchPostPageResponse
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode search page: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].URI != postURI {
		t.Fatalf("search items = %+v, want saved result", page.Items)
	}
	if !page.Items[0].ViewerHasSaved || page.Items[0].ViewerSavedFolderID == nil || *page.Items[0].ViewerSavedFolderID != folderID {
		t.Fatalf("search saved state = %+v", page.Items[0])
	}

	var authorSummary map[string]api.EngagementSummary
	authorSummary, err := api.NewPostStore(pool).EngagementSummaries(ctx, "did:plc:alice", []string{}, []string{postURI})
	if err != nil {
		t.Fatalf("author summary: %v", err)
	}
	if authorSummary[postURI].ViewerHasSaved || authorSummary[postURI].ViewerSavedFolderID != nil {
		t.Fatalf("search target author received viewer save state: %+v", authorSummary[postURI])
	}
}

func TestSearchStore_SearchPostsEmitsDBOperationTelemetry(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	base := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	seedPost(t, pool, "did:plc:alice", "alpaca-post", "alpaca socks", base)

	transport := &sentry.MockTransport{}
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{
		Env:              "test",
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
		MetricRecorder:   recorder,
	})
	store := api.NewSearchStore(pool, observer)
	traceCtx, root := observer.StartSpan(ctx, observability.SpanContext{Operation: "http.server", Component: "http"})
	if _, _, err := store.SearchPosts(traceCtx, api.PostSearchRequest{Query: "alpaca", Sort: api.SearchSortChronological, Limit: 10}, base); err != nil {
		t.Fatalf("SearchPosts: %v", err)
	}
	root.Finish("success")
	if !observer.Flush(time.Second) {
		t.Fatal("observer Flush returned false")
	}

	var sawDBMetric bool
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_db_operation_duration_seconds" &&
			call.Attributes["operation"] == "search.posts" &&
			call.Attributes["route_pattern"] == "/v1/search/posts" {
			sawDBMetric = true
		}
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
	}
	if !sawDBMetric {
		t.Fatalf("missing search.posts DB metric call: %#v", recorder.Calls())
	}

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d Sentry events, want 1 transaction", len(events))
	}
	if len(events[0].Spans) != 1 {
		t.Fatalf("transaction spans = %d, want 1; event=%#v", len(events[0].Spans), events[0])
	}
	span := events[0].Spans[0]
	if span.Op != "db.search.posts" {
		t.Fatalf("DB span op = %q, want db.search.posts; span=%#v", span.Op, span)
	}
	if span.Data["operation"] != "search.posts" || span.Data["route_pattern"] != "/v1/search/posts" || span.Data["result"] != "success" {
		t.Fatalf("DB span data missing bounded fields: %#v", span.Data)
	}
	for _, forbidden := range []string{"alpaca", "did:plc:alice", "SELECT"} {
		if strings.Contains(span.Op, forbidden) {
			t.Fatalf("DB span op contains forbidden value %q: %#v", forbidden, span)
		}
		for key, value := range span.Data {
			if strings.Contains(key, forbidden) || strings.Contains(valueString(value), forbidden) {
				t.Fatalf("DB span data contains forbidden value %q: %s=%#v", forbidden, key, value)
			}
		}
	}
}

func valueString(value any) string {
	return fmt.Sprint(value)
}

func TestSearchStore_SearchProjectsAppliesFilterSemantics(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
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
	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_project_posts
		SET common_status = 'social.craftsky.feed.defs#finished',
			pattern_difficulty = 'social.craftsky.feed.defs#intermediate',
			pattern_self_drafted = true,
			knitting_project_type = 'social.craftsky.project.defs#accessory',
			knitting_project_subtype = 'social.craftsky.project.knitting.defs#socks',
			knitting_yarn_weight = 'social.craftsky.project.defs#fingering'
		WHERE uri = $1`, socks); err != nil {
		t.Fatalf("seed structured project filters: %v", err)
	}

	store := api.NewSearchStore(pool, nil)
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
	andRows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{
		Sort:        api.SearchSortChronological,
		Limit:       10,
		SelfDrafted: true,
		Filters: map[string][]string{
			"craftType":         {"knitting"},
			"status":            {"social.craftsky.feed.defs#finished"},
			"patternDifficulty": {"social.craftsky.feed.defs#intermediate"},
			"projectType":       {"social.craftsky.project.defs#accessory"},
			"projectSubtype":    {"social.craftsky.project.knitting.defs#socks"},
			"color":             {"blue"},
			"designTag":         {"cables"},
			"yarnWeight":        {"social.craftsky.project.defs#fingering"},
		},
	}, base)
	if err != nil {
		t.Fatalf("SearchProjects AND filters: %v", err)
	}
	if got := searchURIs(andRows); !slices.Equal(got, []string{socks}) {
		t.Fatalf("AND filter URIs = %v", got)
	}
	missingStatusRows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 10, Filters: map[string][]string{"status": {"social.craftsky.feed.defs#wip"}}}, base)
	if err != nil {
		t.Fatalf("SearchProjects missing status: %v", err)
	}
	if len(missingStatusRows) != 0 {
		t.Fatalf("missing status matched active status filter: %v", searchURIs(missingStatusRows))
	}
}

func TestSearchStore_ModerationFiltersBeforeSearchRankingAndLimits(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
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

	rows, _, err := api.NewSearchStore(pool, nil).SearchHashtagPosts(ctx, "sock", api.SearchSortPopular, 1, "", base)
	if err != nil {
		t.Fatalf("SearchHashtagPosts moderated: %v", err)
	}
	if got := searchURIs(rows); !slices.Equal(got, []string{visible}) {
		t.Fatalf("moderated popular URIs = %v, hidden %s must not consume limit", got, hidden)
	}
}

func TestSearchStore_TopHashtagsGroupsDistinctProjectsAndEmptyCrafts(t *testing.T) {
	t.Parallel()
	pool := hashtagSearchTestPool(t)
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

	groups, err := api.NewSearchStore(pool, nil).TopHashtags(ctx, api.TopHashtagsRequest{Limit: 10}, now)
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

	mixedGroups, err := api.NewSearchStore(pool, nil).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting", crochetToken, knitting}, Limit: 10}, now)
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

// UT-001, UT-002, AT-002: RULE-001, RULE-003 / AC-002, AC-009.
func TestHashtagSpellingWinnersAndOverlap(t *testing.T) {
	for _, tc := range []struct {
		name      string
		spellings [][]string
		winner    string
	}{
		{"lowercase majority", [][]string{{"MeMadeMay"}, {"memademay"}, {"memademay"}}, "memademay"},
		{"all caps majority", [][]string{{"MEMADEMAY"}, {"MeMadeMay"}, {"MEMADEMAY"}}, "MEMADEMAY"},
		{"tie", [][]string{{"memademay"}, {"MeMadeMay"}}, "MeMadeMay"},
		{"reverse tie", [][]string{{"MeMadeMay"}, {"memademay"}}, "MeMadeMay"},
		{"accented tie", [][]string{{"été"}, {"Été"}}, "Été"},
		{"uncased", [][]string{{"編み物"}}, "編み物"},
		{"overlap tie", [][]string{{"MeMadeMay", "MeMadeMay", "memademay"}}, "MeMadeMay"},
		{"overlap majority", [][]string{{"MeMadeMay", "MeMadeMay", "memademay"}, {"memademay"}}, "memademay"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := hashtagSearchTestPool(t)
			ctx := context.Background()
			now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
			seedMember(t, pool, "did:plc:alice")
			for i, spellings := range tc.spellings {
				uri := seedSearchProject(t, pool, "did:plc:alice", fmt.Sprintf("winner-%d", i), "", "social.craftsky.feed.defs#knitting", "", now.Add(-time.Hour))
				seedPostTags(t, pool, uri, spellings)
				if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET tags=ARRAY(SELECT DISTINCT lower(tag) FROM unnest(tags) tag) WHERE uri=$1`, uri); err != nil {
					t.Fatal(err)
				}
			}
			for _, q := range []string{strings.ToLower(tc.winner), tc.winner, strings.ToUpper(tc.winner)} {
				facet, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, q, 10, now)
				if err != nil {
					t.Fatal(err)
				}
				search, _, err := api.NewSearchStore(pool, nil).SearchHashtags(ctx, api.HashtagSearchRequest{Query: q, Limit: 10}, now)
				if err != nil {
					t.Fatal(err)
				}
				top, err := api.NewSearchStore(pool, nil).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting"}, Limit: 10}, now)
				if err != nil {
					t.Fatal(err)
				}
				count := len(tc.spellings)
				if len(facet) != 1 || facet[0].Tag != tc.winner || facet[0].PostsLast28Days != count {
					t.Fatalf("facet=%v want %s/%d", facet, tc.winner, count)
				}
				if len(search) != 1 || search[0].Tag != tc.winner || search[0].PostsLast28Days != count {
					t.Fatalf("search=%v", search)
				}
				if len(top) != 1 || len(top[0].Items) != 1 || top[0].Items[0].Tag != tc.winner || top[0].Items[0].Count != count {
					t.Fatalf("top=%v", top)
				}
			}
		})
	}
}

// IT-002, IT-008 / FR-001, NFR-002 / AC-003, AC-011.
func TestHashtagSpellingRankingAndSavedCursor(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	uris := map[string]string{}
	for i, tag := range []string{"Sock", "SockKAL", "SockMending", "WoolSock"} {
		uri := seedPost(t, pool, "did:plc:alice", fmt.Sprintf("cursor-%d", i), "", now.Add(-time.Hour))
		seedPostTags(t, pool, uri, []string{tag})
		uris[tag] = uri
	}
	store := api.NewSearchStore(pool, nil)
	var saved string
	for _, q := range []string{"sock", "Sock", "SOCK"} {
		first, cursor, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: q, Limit: 2}, now)
		if err != nil {
			t.Fatal(err)
		}
		if len(first) != 2 || first[0].Tag != "Sock" || first[1].Tag != "SockKAL" || cursor == "" {
			t.Fatalf("first=%v cursor=%q", first, cursor)
		}
		saved = cursor
	}
	seedPostTags(t, pool, uris["SockMending"], []string{"SOCKMENDING"})
	next, cursor, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: "#sOcK", Limit: 2, Cursor: saved}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || next[0].Tag != "SOCKMENDING" || next[1].Tag != "WoolSock" || cursor != "" {
		t.Fatalf("next=%v cursor=%q", next, cursor)
	}
	for _, q := range []string{"", "absent"} {
		rows, _, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: q, Limit: 2}, now)
		if err != nil || len(rows) != 0 {
			t.Fatalf("empty/no-match=%v %v", rows, err)
		}
	}
}

// IT-003, REG-002 / RULE-004 / AC-010: excluded records cannot vote.
func TestHashtagSpellingEligibilityAndCraft(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:viewer"} {
		seedMember(t, pool, did)
	}
	knit := seedSearchProject(t, pool, "did:plc:alice", "knit", "", "social.craftsky.feed.defs#knitting", "", now.Add(-28*24*time.Hour))
	seedPostTags(t, pool, knit, []string{"MeMadeMay"})
	crochet := seedSearchProject(t, pool, "did:plc:alice", "crochet", "", "social.craftsky.feed.defs#crochet", "", now.Add(-28*24*time.Hour+time.Microsecond))
	seedPostTags(t, pool, crochet, []string{"memademay"})
	for _, kind := range []string{"old", "reply", "quote", "hidden", "blocked", "muted"} {
		did := "did:plc:alice"
		if kind == "blocked" || kind == "muted" {
			did = "did:plc:bob"
		}
		uri := seedSearchProject(t, pool, did, kind, "", "social.craftsky.feed.defs#knitting", "", now.Add(-time.Hour))
		seedPostTags(t, pool, uri, []string{"MEMADEMAY"})
		switch kind {
		case "old":
			if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET created_at=$2 WHERE uri=$1`, uri, now.Add(-28*24*time.Hour-time.Nanosecond)); err != nil {
				t.Fatal(err)
			}
		case "reply":
			if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET reply_root_uri=$2,reply_parent_uri=$2 WHERE uri=$1`, uri, knit); err != nil {
				t.Fatal(err)
			}
		case "quote":
			if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET quote_uri=$2 WHERE uri=$1`, uri, knit); err != nil {
				t.Fatal(err)
			}
		case "hidden":
			seedModerationOutput(t, pool, "post", did, uri, "hide", now)
		case "blocked":
			seedBlockAggregate(t, pool, "did:plc:viewer", "did:plc:bob", now)
		case "muted":
			if _, err := pool.Exec(ctx, `INSERT INTO actor_mutes(owner_did,subject_did,created_at) VALUES('did:plc:viewer','did:plc:bob',$1) ON CONFLICT DO NOTHING`, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	search, _, err := api.NewSearchStore(pool, nil).SearchHashtags(ctx, api.HashtagSearchRequest{Query: "mema", Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(search) != 1 || search[0].Tag != "MeMadeMay" || search[0].PostsLast28Days != 2 {
		t.Fatalf("search=%v", search)
	}
	facet, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "mema", 10, now)
	if err != nil {
		t.Fatal(err)
	}
	// Composer retains its preexisting absence of viewer relationship exclusions.
	if len(facet) != 1 || facet[0].Tag != "MEMADEMAY" || facet[0].PostsLast28Days != 4 {
		t.Fatalf("facet=%v", facet)
	}
	groups, err := api.NewSearchStore(pool, nil).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting", "crochet"}, Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || len(groups[0].Items) != 1 || groups[0].Items[0].Tag != "MeMadeMay" || groups[0].Items[0].Count != 1 || len(groups[1].Items) != 1 || groups[1].Items[0].Tag != "memademay" || groups[1].Items[0].Count != 1 {
		t.Fatalf("crafts=%v", groups)
	}
	aged, _, err := api.NewSearchStore(pool, nil).SearchHashtags(ctx, api.HashtagSearchRequest{Query: "mema", Limit: 10}, now.Add(29*24*time.Hour))
	if err != nil || len(aged) != 0 {
		t.Fatalf("aged=%v %v", aged, err)
	}
}

type fixedHashtagSearchReader struct {
	*api.SearchStore
	now time.Time
}

func (s fixedHashtagSearchReader) SearchHashtags(ctx context.Context, req api.HashtagSearchRequest, _ time.Time) ([]api.HashtagSearchResult, string, error) {
	return s.SearchStore.SearchHashtags(ctx, req, s.now)
}
func (s fixedHashtagSearchReader) TopHashtags(ctx context.Context, req api.TopHashtagsRequest, _ time.Time) ([]api.TopHashtagGroup, error) {
	return s.SearchStore.TopHashtags(ctx, req, s.now)
}

type fixedHashtagFacetReader struct {
	*api.FacetStore
	now time.Time
}

func (s fixedHashtagFacetReader) SearchHashtagSuggestions(ctx context.Context, q string, limit int, _ time.Time) ([]api.HashtagSuggestionRow, error) {
	return s.FacetStore.SearchHashtagSuggestions(ctx, q, limit, s.now)
}

// IT-001 / FR-002 / AC-004: wire JSON retains the selected spelling.
func TestHashtagSpellingHTTPResponses(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	seedMember(t, pool, "did:plc:alice")
	for i := range 9 {
		tag := "memademay"
		if i >= 4 {
			tag = "MeMadeMay"
		}
		uri := seedSearchProject(t, pool, "did:plc:alice", fmt.Sprintf("wire-%d", i), "", "social.craftsky.feed.defs#knitting", "", now.Add(-time.Hour))
		seedPostTags(t, pool, uri, []string{tag})
	}
	search := fixedHashtagSearchReader{api.NewSearchStore(pool, nil), now}
	facet := fixedHashtagFacetReader{api.NewFacetStore(pool), now}
	for _, tc := range []struct {
		path    string
		handler http.Handler
		count   string
	}{
		{"/v1/facets/hashtags?q=mema", api.ListFacetHashtagSuggestionsHandler(facet, nilLogger()), `"postsLast28Days":9`},
		{"/v1/search/hashtags?q=MeMa", api.SearchHashtagsHandler(search, nilLogger()), `"postsLast28Days":9`},
		{"/v1/search/suggestions?q=MEMA&types=hashtags", api.SearchSuggestionsHandler(search, nilLogger()), `"postsLast28Days":9`},
		{"/v1/search/hashtags/top?craftTypes=knitting&limit=10", api.TopHashtagsHandler(search, nilLogger()), `"count":9`},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:alice")))
		response := httptest.NewRecorder()
		tc.handler.ServeHTTP(response, req)
		body := response.Body.String()
		if response.Code != http.StatusOK || !strings.Contains(body, `"tag":"MeMadeMay"`) || !strings.Contains(body, tc.count) {
			t.Fatalf("%s => %d %s", tc.path, response.Code, body)
		}
	}
	// IT-004: case variants address the same post set and pagination.
	store := api.NewSearchStore(pool, nil)
	var baseline []string
	for _, tag := range []string{"memademay", "MeMadeMay", "MEMADEMAY"} {
		rows, _, err := store.SearchHashtagPosts(context.Background(), tag, api.SearchSortChronological, 10, "", now)
		if err != nil {
			t.Fatal(err)
		}
		got := searchURIs(rows)
		if baseline == nil {
			baseline = got
		}
		if len(got) != 9 || !slices.Equal(got, baseline) {
			t.Fatalf("tag feed %s=%v want %v", tag, got, baseline)
		}
	}
}

// REG-002: terminal owners never influence display winners or counts.
func TestHashtagSpellingExcludesTerminalOwners(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:active", "did:plc:terminal"} {
		seedMember(t, pool, did)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,terminal_at,created_at,updated_at) VALUES('did:plc:terminal','terminal',1,1,'test',$1,$1,$1,$1)`, now); err != nil {
		t.Fatal(err)
	}
	for i := range 4 {
		did := "did:plc:terminal"
		tag := "MEMADEMAY"
		if i == 0 {
			did = "did:plc:active"
			tag = "MeMadeMay"
		}
		uri := seedSearchProject(t, pool, did, fmt.Sprintf("terminal-%d", i), "", "social.craftsky.feed.defs#knitting", "", now.Add(-time.Hour))
		seedPostTags(t, pool, uri, []string{tag})
		if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES($1,'post','bafycid','clear')`, uri); err != nil {
			t.Fatal(err)
		}
	}
	facet, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "mema", 10, now)
	if err != nil {
		t.Fatal(err)
	}
	search, _, err := api.NewSearchStore(pool, nil).SearchHashtags(ctx, api.HashtagSearchRequest{Query: "mema", Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := api.NewSearchStore(pool, nil).TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"knitting"}, Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(facet) != 1 || facet[0].Tag != "MeMadeMay" || facet[0].PostsLast28Days != 1 || len(search) != 1 || search[0].Tag != "MeMadeMay" || search[0].PostsLast28Days != 1 || len(groups) != 1 || len(groups[0].Items) != 1 || groups[0].Items[0].Tag != "MeMadeMay" || groups[0].Items[0].Count != 1 {
		t.Fatalf("facet=%v search=%v top=%v", facet, search, groups)
	}
}
