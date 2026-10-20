package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

func TestPostCraftsCreatePassThrough(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	executor := &recordingPostEffectExecutor{expectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 7}}}
	handler := api.CreatePostHandler(&fakePostStore{}, func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return executor, nil
	}, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	crafts := `[{"craftType":"social.craftsky.feed.defs#sewing"},{"craftType":"social.craftsky.feed.defs#weaving"}]`
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello","sponsored":false,"crafts":`+crafts+`}`))
	request = request.WithContext(middleware.WithOwnerGeneration(middleware.WithDID(request.Context(), owner), 7))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || len(executor.putRequests) != 1 {
		t.Fatalf("create: status=%d body=%s puts=%d", response.Code, response.Body.String(), len(executor.putRequests))
	}
	var wanted any
	if err := json.Unmarshal([]byte(crafts), &wanted); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]any{"PDS record": executor.putRequests[0].Record, "API response": json.RawMessage(response.Body.Bytes())} {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		var wire map[string]any
		if err := json.Unmarshal(raw, &wire); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(wire["crafts"], wanted) || wire["project"] != nil {
			t.Fatalf("%s changed metadata or introduced a project: %s", name, raw)
		}
	}
}

func TestPostCraftsRequestValidation(t *testing.T) {
	for _, tc := range []struct {
		name, crafts string
		valid        bool
	}{
		{"empty", `[]`, true},
		{"three", `[{"craftType":"a"},{"craftType":"b"},{"craftType":"c"}]`, true},
		{"four", `[{"craftType":"a"},{"craftType":"b"},{"craftType":"c"},{"craftType":"d"}]`, false},
		{"missing type", `[{}]`, false},
		{"empty type", `[{"craftType":""}]`, false},
		{"null item", `[null]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := api.DecodePostCreate(strings.NewReader(`{"text":"hello","sponsored":false,"crafts":` + tc.crafts + `}`))
			if err == nil {
				err = api.ValidatePostCreate(req)
			}
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestPostCraftsReadHydrationAndDiscoveryExclusion(t *testing.T) {
	pool, search, now := searchFixture(t)
	ctx := context.Background()
	uri := searchFixturePost(t, pool, false, "reserved", "A quiet afternoon", nil, now)
	crafts := `[{"craftType":"social.craftsky.feed.defs#embroidery","futureContext":"retained"}]`
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET record=jsonb_build_object('crafts',$2::jsonb) WHERE uri=$1`, uri, crafts); err != nil {
		t.Fatal(err)
	}
	posts := api.NewPostStore(pool, nil)
	row, err := posts.ReadPostByURI(ctx, uri)
	if err != nil {
		t.Fatal(err)
	}
	if row.IsProject || row.Project != nil || row.ProjectCraftType != nil {
		t.Fatal("craft metadata turned an ordinary post into a project")
	}
	for name, response := range map[string]any{
		"full":  api.BuildPostResponse(row, "author.example"),
		"quote": api.BuildQuoteView(&api.QuoteViewRow{State: "visible", Post: row}, "author.example").Post,
	} {
		raw, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), `"futureContext":"retained"`) {
			t.Fatalf("%s lost craft metadata: %s", name, raw)
		}
	}
	rows, _ := submittedSearch(t, search, false, "embroidery", 10, "", now)
	if len(rows) != 0 {
		t.Fatal("reserved craft metadata influenced keyword search")
	}
	rows, _ = submittedSearch(t, search, false, "afternoon", 10, "", now)
	if len(rows) != 1 || len(rows[0].Post.Crafts) == 0 {
		t.Fatal("ordinary text search lost crafts during hydration")
	}
	timeline, _, err := posts.ListTimeline(ctx, "did:plc:author", 10, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 || len(timeline[0].Post.Crafts) == 0 {
		t.Fatal("timeline hydration lost craft metadata")
	}
	projectURI := searchFixturePost(t, pool, true, "project", "A second afternoon", nil, now)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET record=jsonb_build_object('crafts',$2::jsonb),tags=ARRAY['needlecraft'] WHERE uri=$1`, projectURI, crafts); err != nil {
		t.Fatal(err)
	}
	rows, _, err = search.SearchProjects(ctx, api.ProjectSearchRequest{Sort: api.SearchSortChronological, Limit: 10, Filters: map[string][]string{"craftType": {"embroidery"}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatal("reserved craft metadata influenced project craft filtering")
	}
	groups, err := search.TopHashtags(ctx, api.TopHashtagsRequest{CraftTypes: []string{"embroidery"}, Limit: 10}, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range groups {
		if len(group.Items) != 0 {
			t.Fatalf("reserved craft metadata influenced tag discovery: %+v", groups)
		}
	}
}
