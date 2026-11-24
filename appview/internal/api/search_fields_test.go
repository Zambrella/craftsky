package api_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
)

func TestProjectSearchExpandedPublicFields(t *testing.T) {
	pool, store, now := searchFixture(t)
	uri := searchFixturePost(t, pool, true, "contract", "", nil, now)
	_, err := pool.Exec(context.Background(), `UPDATE craftsky_project_posts SET
	 common_craft_type='social.craftsky.feed.defs#sewing', common_status='social.craftsky.feed.defs#wip',
	 common_duration='weekend',pattern_designer='Mina',pattern_publisher='Luna Studio',
	 pattern_difficulty='social.craftsky.feed.defs#beginner',pattern_self_drafted=true,
	 colors=ARRAY['gray'],design_tags=ARRAY['social.craftsky.project.defs#floral'],
	 sewing_project_type='social.craftsky.project.defs#garment',sewing_project_subtype='social.craftsky.project.sewing.defs#dress',
	 sewing_size_made='twelve',sewing_fit_notes='adjusted darts',
	 raw_project='{"common":{"craftType":"social.craftsky.feed.defs#sewing","additionalCrafts":[{"craftType":"social.craftsky.feed.defs#embroidery"}],"future":"protectedcanary"}}'
	 WHERE uri=$1`, uri)
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"sewing", "embroidery", "work in progress", "weekend", "Mina", "Luna Studio", "beginner", "self drafted", "grey", "floral", "garment", "dress", "twelve", "adjusted darts", "beginner grey dress"} {
		t.Run(query, func(t *testing.T) {
			rows, _ := submittedSearch(t, store, true, query, 10, "", now)
			if !slices.Contains(searchURIs(rows), uri) {
				t.Fatalf("%q missing: %v", query, searchURIs(rows))
			}
		})
	}
	rows, _ := submittedSearch(t, store, true, "protectedcanary", 10, "", now)
	if len(rows) != 0 {
		t.Fatal("unknown extension became searchable")
	}
	for _, query := range []string{"", "Luna Studio"} {
		for _, sort := range []api.SearchSort{api.SearchSortChronological, api.SearchSortPopular} {
			for _, tc := range []struct {
				name   string
				crafts []string
				want   bool
			}{
				{"all crafts", nil, true},
				{"primary craft", []string{"social.craftsky.feed.defs#sewing"}, true},
				{"additional craft", []string{"social.craftsky.feed.defs#embroidery"}, false},
				{"unrelated craft", []string{"social.craftsky.feed.defs#knitting"}, false},
				{"multiple crafts including primary", []string{"social.craftsky.feed.defs#embroidery", "social.craftsky.feed.defs#sewing"}, true},
			} {
				t.Run(query+"/"+string(sort)+"/"+tc.name, func(t *testing.T) {
					filtered, _, err := store.SearchProjectsWithLanguages(context.Background(), "did:plc:viewer", nil, api.ProjectSearchRequest{
						Query: query, Sort: sort, Limit: 10, Filters: map[string][]string{"craftType": tc.crafts},
					}, now)
					if err != nil {
						t.Fatal(err)
					}
					if got := slices.Contains(searchURIs(filtered), uri); got != tc.want {
						t.Fatalf("project included=%v, want %v; rows=%v", got, tc.want, searchURIs(filtered))
					}
				})
			}
		}
	}
}

func TestProjectSearchCraftDetailsAndAncestorTerms(t *testing.T) {
	pool, store, now := searchFixture(t)
	knit := searchFixturePost(t, pool, true, "knit-contract", "", nil, now)
	quilt := searchFixturePost(t, pool, true, "quilt-contract", "", nil, now)
	_, err := pool.Exec(context.Background(), `UPDATE craftsky_project_posts SET
	 knitting_project_subtype='social.craftsky.project.knitting.defs#cardigan',
	 knitting_yarn_weight='social.craftsky.project.defs#dk',knitting_needle_size_mm='4.5mm',
	 knitting_gauge='{"stitches":22,"measurement":10,"unit":"cm"}',knitting_finished_size='oversized'
	 WHERE uri=$1`, knit)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(context.Background(), `UPDATE craftsky_project_posts SET
	 quilting_piecing_technique='social.craftsky.project.quilting.defs#foundationPaperPiecing',
	 quilting_quilting_method='social.craftsky.project.quilting.defs#longarm',quilting_size='queen'
	 WHERE uri=$1`, quilt)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ query, uri string }{{"cardigan", knit}, {"sweater", knit}, {"jumper", knit}, {"dk", knit}, {"4.5mm", knit}, {"22", knit}, {"oversized", knit}, {"foundation paper piecing", quilt}, {"fpp", quilt}, {"longarm", quilt}, {"queen", quilt}} {
		rows, _ := submittedSearch(t, store, true, tc.query, 10, "", now)
		if !slices.Contains(searchURIs(rows), tc.uri) {
			t.Fatalf("%q missing: %v", tc.query, searchURIs(rows))
		}
	}
}

func TestProjectSearchEffectivePatternsAndMethods(t *testing.T) {
	pool, store, now := searchFixture(t)
	ctx := context.Background()
	withPatterns := searchFixturePost(t, pool, true, "multiple-patterns", "", nil, now)
	withMethods := searchFixturePost(t, pool, true, "knitting-methods", "", nil, now)
	cleared := searchFixturePost(t, pool, true, "cleared-project", "", nil, now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET pattern_name='removedcollar',knitting_needle_size_mm='23.0mm',
		raw_project='{"common":{"patterns":[],"pattern":{"name":"removedcollar"}}}',
		common_craft_type='social.craftsky.feed.defs#knitting', raw_details='{"$type":"social.craftsky.project.knitting#details","needleSizesMm":[],"needleSizeMm":"23.0mm","methods":["social.craftsky.project.knitting.defs#hand"]}'
		WHERE uri=$1`, cleared); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET pattern_name='overriddenlegacy',
		raw_project='{"common":{"patterns":[{"name":"collar","difficulty":"beginner","selfDrafted":true},{"name":"sleeves","difficulty":"advanced","designer":"Rare Needle","url":"https://example.org/protectedurl"}]}}'
		WHERE uri=$1`, withPatterns); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET common_craft_type='social.craftsky.feed.defs#knitting',
		knitting_needle_size_mm='19.0mm', raw_details='{"$type":"social.craftsky.project.knitting#details","needleSizesMm":["4.5mm","7.0mm"],"methods":["social.craftsky.project.knitting.defs#hand","social.craftsky.project.knitting.defs#machine"],"techniques":[{"technique":"social.craftsky.project.knitting.defs#colourwork","subtypes":[{"subtype":"social.craftsky.project.knitting.defs#fairIsle"}]}]}'
		WHERE uri=$1`, withMethods); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ query, uri string }{{"Rare Needle", withPatterns}, {"sleeves", withPatterns}, {"4.5mm", withMethods}, {"7.0mm", withMethods}, {"Fair Isle", withMethods}, {"Machine knitting", withMethods}} {
		rows, _ := submittedSearch(t, store, true, tc.query, 10, "", now)
		if !slices.Contains(searchURIs(rows), tc.uri) {
			t.Errorf("%q missing: %v", tc.query, searchURIs(rows))
		}
	}
	for _, query := range []string{"overriddenlegacy", "protectedurl", "19.0mm", "removedcollar", "23.0mm"} {
		rows, _ := submittedSearch(t, store, true, query, 10, "", now)
		if len(rows) != 0 {
			t.Errorf("%q matched hidden field: %v", query, searchURIs(rows))
		}
	}
	for _, tc := range []struct {
		filters     map[string][]string
		selfDrafted bool
		want        []string
	}{
		{map[string][]string{"patternDifficulty": {"beginner"}}, true, []string{withPatterns}},
		{map[string][]string{"patternDifficulty": {"advanced"}}, false, []string{withPatterns}},
		{map[string][]string{"patternDifficulty": {"advanced"}}, true, nil},
		{map[string][]string{"knittingMethod": {"social.craftsky.project.knitting.defs#machine"}}, false, []string{withMethods}},
	} {
		for _, sort := range []api.SearchSort{api.SearchSortChronological, api.SearchSortPopular} {
			rows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Filters: tc.filters, SelfDrafted: tc.selfDrafted, Sort: sort, Limit: 10}, now)
			if err != nil || !slices.Equal(searchURIs(rows), tc.want) {
				t.Errorf("%s %+v=%v err=%v, want %v", sort, tc.filters, searchURIs(rows), err, tc.want)
			}
		}
	}
}

func TestExpandedCraftDiscoveryPreservesPrimaryFeedRules(t *testing.T) {
	const punchNeedle = "social.craftsky.feed.defs#punchNeedle"
	pool, store, now := searchFixture(t)
	ctx := context.Background()
	primary := searchFixturePost(t, pool, true, "punch-needle-primary", "", nil, now)
	secondary := searchFixturePost(t, pool, true, "punch-needle-additional", "", nil, now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET common_craft_type='social.craftsky.feed.defs#punchNeedle',
		raw_project='{"common":{"craftType":"social.craftsky.feed.defs#punchNeedle"}}',
		raw_details='{"$type":"social.craftsky.project.punchNeedle#details","projectType":"social.craftsky.project.punchNeedle#hoopArt"}' WHERE uri=$1`, primary); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET common_craft_type='social.craftsky.feed.defs#knitting',
		raw_project='{"common":{"craftType":"social.craftsky.feed.defs#knitting","additionalCrafts":[{"craftType":"social.craftsky.feed.defs#punchNeedle"}]}}' WHERE uri=$1`, secondary); err != nil {
		t.Fatal(err)
	}
	seedPostTags(t, pool, primary, []string{"sharedhoop"})
	seedPostTags(t, pool, secondary, []string{"sharedhoop"})
	rows, _ := submittedSearch(t, store, true, "Punch Needle", 10, "", now)
	if found := searchURIs(rows); len(found) != 2 || !slices.Contains(found, primary) || !slices.Contains(found, secondary) {
		t.Fatalf("keyword match=%v, want both projects", found)
	}
	rows, _ = submittedSearch(t, store, true, "Hoop art", 10, "", now)
	if found := searchURIs(rows); !slices.Equal(found, []string{primary}) {
		t.Fatalf("known detail search=%v, want %s", found, primary)
	}
	listed, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Filters: map[string][]string{"craftType": {"knitting"}}, Sort: api.SearchSortChronological, Limit: 10}, now)
	if err != nil || !slices.Equal(searchURIs(listed), []string{secondary}) {
		t.Fatalf("knitting feed=%v, err=%v, want %s", searchURIs(listed), err, secondary)
	}
	needleRequest, needleErr := api.ParseProjectListRequest(httptest.NewRequest("GET", "/v1/projects?craftType=punch-needle", nil))
	groups, err := store.TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"punch-needle"}, Limit: 10}, now)
	if !slices.Contains(expectedDiscoveryCrafts(), punchNeedle) {
		if needleErr != api.ErrSearchValidation || err != api.ErrSearchValidation {
			t.Fatalf("unavailable craft: browse err=%v, hashtag err=%v", needleErr, err)
		}
		return
	}
	if err != nil || len(groups) != 1 || groups[0].CraftType != punchNeedle || len(groups[0].Items) != 1 || groups[0].Items[0].Tag != "sharedhoop" || groups[0].Items[0].Count != 2 {
		t.Fatalf("punch needle hashtag group=%+v, err=%v", groups, err)
	}
	if needleErr != nil {
		t.Fatalf("parse punch needle feed: %v", needleErr)
	}
	listed, _, err = store.SearchProjects(ctx, api.ProjectSearchRequest{Filters: needleRequest.Filters, Sort: needleRequest.Sort, Limit: needleRequest.Limit}, now)
	if err != nil || !slices.Equal(searchURIs(listed), []string{primary}) {
		t.Fatalf("punch needle feed=%v, err=%v, want %s", searchURIs(listed), err, primary)
	}
}

func TestHashtagAllTimeCountsAndAdditionalCraftDiscovery(t *testing.T) {
	pool := hashtagSearchTestPool(t)
	ctx := middleware.WithDID(context.Background(), syntax.DID("did:plc:viewer"))
	seedMember(t, pool, "did:plc:viewer")
	seedMember(t, pool, "did:plc:author")
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	uri := seedSearchProject(t, pool, "did:plc:author", "historic-mixed", "", "social.craftsky.feed.defs#sewing", "", now.AddDate(-2, 0, 0))
	seedPostTags(t, pool, uri, []string{"OldPattern", "OldPattern"})
	raw, _ := json.Marshal(map[string]any{"common": map[string]any{"craftType": "social.craftsky.feed.defs#sewing", "additionalCrafts": []any{map[string]string{"craftType": "social.craftsky.feed.defs#embroidery"}}}})
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET raw_project=$2 WHERE uri=$1`, uri, raw); err != nil {
		t.Fatal(err)
	}
	store := api.NewSearchStore(pool, nil)
	rows, _, err := store.SearchHashtags(ctx, api.HashtagSearchRequest{Query: "oldpattern", Limit: 10}, now)
	if err != nil || len(rows) != 1 || rows[0].PostCount != 1 {
		t.Fatalf("history search=%v err=%v", rows, err)
	}
	groups, err := store.TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"embroidery"}, Limit: 10}, now)
	if err != nil || len(groups) != 1 || len(groups[0].Items) != 1 || groups[0].Items[0].Count != 1 {
		t.Fatalf("secondary discovery=%v err=%v", groups, err)
	}
	facet, err := api.NewFacetStore(pool).SearchHashtagSuggestions(ctx, "oldpattern", 10, now)
	if err != nil || len(facet) != 1 || facet[0].PostCount != 1 {
		t.Fatalf("historic autocomplete=%v err=%v", facet, err)
	}
}
