package api_test

import (
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestProjectItemDiscoveryUsesKnownBrowseValuesAndCustomKeywords(t *testing.T) {
	pool, store, now := searchFixture(t)
	ctx := t.Context()
	seed := func(craft, key, raw string) string {
		t.Helper()
		uri := searchFixturePost(t, pool, true, key, "", nil, now)
		if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET common_craft_type=$2, raw_details=$3::jsonb WHERE uri=$1`,
			uri, "social.craftsky.feed.defs#"+craft, raw); err != nil {
			t.Fatal(err)
		}
		return uri
	}
	const sharedType = "social.craftsky.project.defs#"
	const sewingSubtype = "social.craftsky.project.sewing.defs#"
	const weavingSubtype = "social.craftsky.project.weaving#"
	bag := seed("sewing", "generic-bag", `{"$type":"social.craftsky.project.sewing#details","projectType":"`+sharedType+`accessory","projectSubtype":"`+sewingSubtype+`bag"}`)
	tote := seed("sewing", "named-tote", `{"$type":"social.craftsky.project.sewing#details","projectType":"`+sharedType+`accessory","projectSubtype":"`+sewingSubtype+`tote"}`)
	backpack := seed("sewing", "named-backpack", `{"$type":"social.craftsky.project.sewing#details","projectType":"`+sharedType+`accessory","projectSubtype":"`+sewingSubtype+`backpack"}`)
	seed("sewing", "custom-bag-text", `{"$type":"social.craftsky.project.sewing#details","projectType":"`+sharedType+`accessory","projectSubtype":"Bag"}`)
	custom := seed("weaving", "custom-item", `{"$type":"social.craftsky.project.weaving#details","projectType":"`+sharedType+`homeGoods","projectSubtype":"Blue lampshade"}`)
	towel := seed("weaving", "named-towel", `{"$type":"social.craftsky.project.weaving#details","projectType":"`+sharedType+`homeGoods","projectSubtype":"`+weavingSubtype+`towel"}`)
	seed("spinning", "unrecognised-item-extension", `{"$type":"social.craftsky.project.spinning#details","projectType":"`+sharedType+`homeGoods","projectSubtype":"Blue lampshade"}`)
	seed("dyeing", "wrong-variant", `{"$type":"social.craftsky.project.weaving#details","projectType":"`+sharedType+`homeGoods","projectSubtype":"Blue lampshade"}`)
	legacyDye := seed("dyeing", "old-dyed-yarn", `{"$type":"social.craftsky.project.dyeing#details","materialForm":"social.craftsky.project.dyeing#spunYarn"}`)
	legacyRoving := seed("dyeing", "old-dyed-roving", `{"$type":"social.craftsky.project.dyeing#details","materialForm":"roving"}`)
	seed("dyeing", "new-dyed-fabric", `{"$type":"social.craftsky.project.dyeing#details","projectType":"social.craftsky.project.dyeing#fabric","materialForm":"social.craftsky.project.dyeing#spunYarn"}`)
	figurine := seed("felting", "old-felted-figurine", `{"$type":"social.craftsky.project.felting#details","projectType":"`+sharedType+`toyHobby","projectSubtype":"social.craftsky.project.felting#figurine"}`)
	sculpture := seed("felting", "new-felted-sculpture", `{"$type":"social.craftsky.project.felting#details","projectType":"`+sharedType+`artDecoration","projectSubtype":"social.craftsky.project.felting#figurine"}`)
	legacySew := seed("sewing", "only-projected-tote", `null`)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET raw_details = NULL,
		sewing_project_type = $2, sewing_project_subtype = $3 WHERE uri = $1`, legacySew, sharedType+"accessory", sewingSubtype+"tote"); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, field, value string
		want               []string
	}{
		{"bag includes descendants", "projectSubtype", sewingSubtype + "bag", []string{bag, tote, backpack, legacySew}},
		{"tote stays specific", "projectSubtype", sewingSubtype + "tote", []string{tote, legacySew}},
		{"selected type includes custom items", "projectType", sharedType + "homeGoods", []string{custom, towel}},
		{"custom text is keyword only", "projectSubtype", "Blue lampshade", nil},
		{"additional craft's known item", "projectSubtype", weavingSubtype + "towel", []string{towel}},
		{"legacy material maps to new dyeing type", "projectType", "social.craftsky.project.dyeing#yarnThread", []string{legacyDye}},
		{"legacy literal material maps to new dyeing type", "projectType", "social.craftsky.project.dyeing#unspunFibre", []string{legacyRoving}},
		{"legacy figurine is discoverable under new parent", "projectType", sharedType + "artDecoration", []string{figurine, sculpture}},
		{"old broad type only matches historical choice", "projectType", sharedType + "toyHobby", []string{figurine}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{
				Limit: 10, Sort: api.SearchSortChronological, Filters: map[string][]string{tc.field: {tc.value}},
			}, now)
			got := searchURIs(rows)
			slices.Sort(got)
			slices.Sort(tc.want)
			if err != nil || !slices.Equal(got, tc.want) {
				t.Fatalf("items=%v, want %v (error: %v)", got, tc.want, err)
			}
		})
	}
	for _, tc := range []struct{ query, uri string }{{"blue lampshade", custom}, {"towel", towel}} {
		rows, _ := submittedSearch(t, store, true, tc.query, 10, "", now)
		if got := searchURIs(rows); !slices.Equal(got, []string{tc.uri}) {
			t.Errorf("keyword %q matched %v, want %s", tc.query, got, tc.uri)
		}
	}

	for _, tc := range []struct {
		name    string
		filters url.Values
		want    []string
	}{
		{"type and subtype are alternatives", url.Values{"projectItem": {sharedType + "homeGoods", sewingSubtype + "tote"}}, []string{custom, towel, tote, legacySew}},
		{"intermediate option includes descendants", url.Values{"projectItem": {sewingSubtype + "bag"}}, []string{bag, tote, backpack, legacySew}},
		{"subtype alone stays specific", url.Values{"projectItem": {sewingSubtype + "tote"}}, []string{tote, legacySew}},
		{"unknown item matches nothing", url.Values{"projectItem": {"unknown"}}, nil},
		{"other filter still narrows", url.Values{"projectItem": {sharedType + "homeGoods", sewingSubtype + "tote"}, "craftType": {"sewing"}}, []string{tote, legacySew}},
		{"legacy fields still intersect", url.Values{"projectType": {sharedType + "accessory"}, "projectSubtype": {sewingSubtype + "tote"}}, []string{tote, legacySew}},
		{"legacy fields also intersect with items", url.Values{"projectItem": {sharedType + "homeGoods", sewingSubtype + "tote"}, "projectType": {sharedType + "homeGoods"}}, []string{custom, towel}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, sort := range []api.SearchSort{api.SearchSortChronological, api.SearchSortPopular} {
				t.Run(string(sort), func(t *testing.T) {
					tc.filters.Set("sort", string(sort))
					tc.filters.Set("limit", "2")
					var got []string
					var cursor string
					for page := 0; page < 3; page++ {
						tc.filters.Set("cursor", cursor)
						path := "/v1/projects?" + tc.filters.Encode()
						parsed, err := api.ParseProjectListRequest(httptest.NewRequest("GET", path, nil))
						if err != nil {
							t.Fatal(err)
						}
						rows, next, err := store.SearchProjects(ctx, api.ProjectSearchRequest{
							Sort: parsed.Sort, Limit: parsed.Limit, Cursor: parsed.Cursor, Filters: parsed.Filters,
						}, now)
						if err != nil {
							t.Fatal(err)
						}
						got = append(got, searchURIs(rows)...)
						if next == "" {
							break
						}
						cursor = next
					}
					slices.Sort(got)
					slices.Sort(tc.want)
					if !slices.Equal(got, tc.want) {
						t.Fatalf("items=%v, want %v", got, tc.want)
					}
				})
			}
		})
	}
}
