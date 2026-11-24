package api_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/projectcatalog"
)

func TestBrowseProjectsByCraftDetailFields(t *testing.T) {
	t.Logf("project catalogue release phase %d", projectcatalog.Current.ReleasePhase)
	pool, store, now := searchFixture(t)
	ctx := context.Background()
	seed := func(craft, key, raw string) string {
		t.Helper()
		uri := searchFixturePost(t, pool, true, key, "", nil, now)
		if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET common_craft_type = $2, raw_details = $3::jsonb WHERE uri = $1`,
			uri, "social.craftsky.feed.defs#"+craft, raw); err != nil {
			t.Fatal(err)
		}
		return uri
	}
	sewing := seed("sewing", "sewing-techniques", `{"$type":"social.craftsky.project.sewing#details",
		"techniques":[{"technique":"social.craftsky.project.sewing.defs#frenchSeams"}]}`)
	seed("crochet", "wrong-detail-type", `{"$type":"social.craftsky.project.sewing#details",
		"techniques":[{"technique":"social.craftsky.project.sewing.defs#frenchSeams"}]}`)
	knitting := seed("knitting", "knitting-techniques", `{"$type":"social.craftsky.project.knitting#details",
		"techniques":[{"technique":"colourwork","subtypes":[{"subtype":"fairIsle"}]},
		{"technique":"cables","subtypes":"malformed"}],"needleSizesMm":["4.5mm","7.0mm",3],
		"needleSizeMm":"3.0mm","gauge":{"unit":"cm"}}`)
	legacyKnitting := seed("knitting", "legacy-knitting", `null`)
	if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET raw_details = NULL,
		knitting_needle_size_mm = '9.0mm', knitting_gauge = '{"unit":"in"}' WHERE uri = $1`, legacyKnitting); err != nil {
		t.Fatal(err)
	}
	crochet := seed("crochet", "legacy-crochet", `{"$type":"social.craftsky.project.crochet#details","hookSizeMm":"6.0mm","techniques":[{"technique":"filetCrochet"}]}`)
	seed("crochet", "cleared-hook-sizes", `{"$type":"social.craftsky.project.crochet#details","hookSizeMm":"6.0mm","hookSizesMm":[]}`)
	seed("crochet", "malformed-hook-sizes", `{"$type":"social.craftsky.project.crochet#details","hookSizeMm":"6.0mm","hookSizesMm":{"size":"6.0mm"}}`)
	embroidery := seed("embroidery", "embroidery-methods", `{"$type":"social.craftsky.project.embroidery#details",
		"methods":[{"method":"hand","styles":[{"style":"sashiko"}]},
		{"method":"machine","styles":[{"style":"puff3d"}]},7]}`)
	spinning := seed("spinning", "spinning-details", `{"$type":"social.craftsky.project.spinning#details","tool":"wheel",
		"fibrePreparation":"rolag","draftingTechniques":[{"technique":"longDraw"}],"plyingTechniques":[{"technique":"chainPlying"}],
		"finishedYarn":{"mass":{"unit":"g"},"length":{"unit":"m"}}}`)
	dyeing := seed("dyeing", "dyeing-details", `{"$type":"social.craftsky.project.dyeing#details","materialForm":"spunYarn",
		"dyeType":"acid","applicationMethod":"kettleDyed","colorEffect":"speckled"}`)
	punchNeedle := seed("punchNeedle", "punch-details", `{"$type":"social.craftsky.project.punchNeedle#details","loopHeight":{"unit":"mm"}}`)
	macrame := seed("macrame", "macrame-details", `{"$type":"social.craftsky.project.macrame#details","cords":[{"construction":"singleTwist"}],"knots":[{"knot":"squareKnot"}]}`)
	weaving := seed("weaving", "weaving-details", `{"$type":"social.craftsky.project.weaving#details","loomType":"rigidHeddle",
		"structure":"plainWeave","materials":[{"role":"warp"},{"role":"weft"}],"sett":{"unit":"ends-per-inch"}}`)
	felting := seed("felting", "felting-details", `{"$type":"social.craftsky.project.felting#details","form":"flat","needleGauges":[{"gauge":"36"},{"gauge":38}]}`)

	cases := []struct {
		name, query string
		want        string
		phase       int
	}{
		{"sewing exact technique or equivalent", "technique=flatFelledSeams&technique=social.craftsky.project.sewing.defs%23frenchSeams", sewing, 1},
		{"crochet technique", "technique=filetCrochet", crochet, 1},
		{"knitting parent and child", "technique=colourwork&techniqueSubtype=fairIsle", knitting, 1},
		{"knitting subtype alone", "techniqueSubtype=fairIsle", knitting, 1},
		{"knitting subtype belongs to a different entry", "technique=cables&techniqueSubtype=fairIsle", "", 1},
		{"knitting needle and gauge unit", "needleSizeMm=7.0mm&gaugeUnit=CM", knitting, 1},
		{"knitting legacy projected needle and gauge", "needleSizeMm=9.0mm&gaugeUnit=in", legacyKnitting, 1},
		{"knitting legacy needle overridden", "needleSizeMm=3.0mm", "", 1},
		{"knitting malformed needle size", "needleSizeMm=3", "", 1},
		{"crochet legacy hook", "hookSizeMm=6.0mm", crochet, 1},
		{"embroidery method and style", "embroideryMethod=hand&embroideryStyle=sashiko", embroidery, 1},
		{"embroidery style belongs to a different method", "embroideryMethod=machine&embroideryStyle=sashiko", "", 1},
		{"spinning tool and preparation", "spinningTool=wheel&fibrePreparation=rolag", spinning, 2},
		{"spinning drafting and plying", "draftingTechnique=longDraw&plyingTechnique=chainPlying", spinning, 2},
		{"finished yarn units", "finishedYarnMassUnit=g&finishedYarnLengthUnit=m", spinning, 2},
		{"dyeing material, dye, application and effect", "materialForm=spunYarn&dyeType=acid&applicationMethod=kettleDyed&colorEffect=speckled", dyeing, 2},
		{"punch needle loop height unit", "loopHeightUnit=mm", punchNeedle, 2},
		{"macrame cord and knot", "cordConstruction=singleTwist&knot=squareKnot", macrame, 2},
		{"weaving loom, structure and sett unit", "loomType=rigidHeddle&weaveStructure=plainWeave&settUnit=ends-per-inch", weaving, 2},
		{"weaving material role", "materialRole=warp", weaving, 2},
		{"felting form and gauge", "feltingForm=flat&needleGauge=36", felting, 2},
		{"felting malformed numeric gauge", "needleGauge=38", "", 2},
	}

	for _, sort := range []api.SearchSort{api.SearchSortChronological, api.SearchSortPopular} {
		for _, tc := range cases {
			t.Run(string(sort)+"/"+tc.name, func(t *testing.T) {
				path := "/v1/projects?sort=" + string(sort) + "&" + tc.query
				parsed, err := api.ParseProjectListRequest(httptest.NewRequest("GET", path, nil))
				if projectcatalog.Current.ReleasePhase < tc.phase {
					if err != api.ErrSearchValidation {
						t.Fatalf("future-phase filter error = %v, want validation error", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				rows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Filters: parsed.Filters, Sort: parsed.Sort, Limit: parsed.Limit}, now)
				var want []string
				if tc.want != "" {
					want = []string{tc.want}
				}
				if err != nil || !slices.Equal(searchURIs(rows), want) {
					t.Fatalf("rows=%v err=%v, want %v", searchURIs(rows), err, want)
				}
			})
		}
	}
}

func TestCraftDetailFiltersRespectExistingBrowseLimits(t *testing.T) {
	valid := url.Values{"technique": {" FrenchSeams ", "FLATFELLEDSEAMS", "frenchSeams"}}
	parsed, err := api.ParseProjectListRequest(httptest.NewRequest("GET", "/v1/projects?"+valid.Encode(), nil))
	if err != nil || !slices.Equal(parsed.Filters["technique"], []string{"flatfelledseams", "frenchseams"}) {
		t.Fatalf("normalized technique=%v, err=%v", parsed.Filters, err)
	}
	for i := 0; i < api.ProjectFilterMaxPerFamily; i++ {
		valid.Add("technique", "one-more")
	}
	overBudget := url.Values{}
	for _, family := range []string{"technique", "techniqueSubtype", "embroideryMethod", "needleSizeMm", "hookSizeMm", "gaugeUnit"} {
		for i := 0; i < api.ProjectFilterMaxPerFamily; i++ {
			overBudget.Add(family, "choice"+strconv.Itoa(i))
		}
	}
	for _, path := range []string{
		"/v1/projects?" + valid.Encode(),
		"/v1/projects?" + overBudget.Encode(),
		"/v1/projects?technique=++",
		"/v1/projects?technique=" + strings.Repeat("a", api.SearchMaxQueryLength+1),
	} {
		if _, err := api.ParseProjectListRequest(httptest.NewRequest("GET", path, nil)); err != api.ErrSearchValidation {
			t.Errorf("%q error=%v, want validation error", path, err)
		}
	}
	if _, err := api.ParseProjectSearchRequest(httptest.NewRequest("GET", "/v1/search/projects?q=sew&technique=frenchSeams", nil)); err != api.ErrSearchValidation {
		t.Errorf("keyword search accepted a craft detail filter, error=%v", err)
	}
}

func TestSimplifiedCraftMethodBrowseAndKeywords(t *testing.T) {
	pool, store, now := searchFixture(t)
	ctx := t.Context()
	seed := func(craft, key, fields string) string {
		t.Helper()
		uri := searchFixturePost(t, pool, true, key, "", nil, now)
		raw := `{"$type":"social.craftsky.project.` + craft + `#details",` + fields + `}`
		if _, err := pool.Exec(ctx, `UPDATE craftsky_project_posts SET common_craft_type=$2, raw_details=$3::jsonb WHERE uri=$1`,
			uri, "social.craftsky.feed.defs#"+craft, raw); err != nil {
			t.Fatal(err)
		}
		return uri
	}
	stitch := seed("embroidery", "flat-independent-stitch", `"method":"machine","style":"social.craftsky.project.embroidery.defs#crossStitch","methods":[{"method":"hand","styles":[{"style":"needlepoint"}]}]`)
	legacyStitch := seed("embroidery", "legacy-stitch", `"methods":[{"method":"hand","styles":[{"style":"needlepoint"}]},{"method":"machine","styles":[{"style":"puff3d"}]}]`)
	loom := seed("knitting", "knitting-loom", `"methods":["social.craftsky.project.knitting.defs#loom","hand"]`)
	crochet := seed("crochet", "new-tunisian", `"methods":["standardCrochet","social.craftsky.project.crochet.defs#tunisianCrochet"]`)
	legacyCrochet := seed("crochet", "legacy-tunisian", `"techniques":[{"technique":"tunisianCrochet"}]`)
	seed("crochet", "crochet-clear", `"methods":[],"techniques":[{"technique":"tunisianCrochet"}]`)
	seed("crochet", "crochet-unspecified", `"techniques":[{"technique":"grannySquare"}]`)
	felt := seed("felting", "mixed-felt", `"methods":["social.craftsky.project.felting#wet","needle"]`)
	legacyFelt := seed("felting", "legacy-needle", `"method":"needle"`)
	seed("felting", "felting-clear", `"methods":[],"method":"needle"`)
	weave := seed("weaving", "woven-scarf", `"loomType":"social.craftsky.project.weaving#rigidHeddle"`)

	for _, tc := range []struct {
		name, query string
		want        []string
	}{
		{"flat method and style are independent", "embroideryMethod=machine&embroideryStyle=crossStitch", []string{stitch}},
		{"flat values override grouped values", "embroideryMethod=hand&embroideryStyle=crossStitch", nil},
		{"legacy selections in same method", "embroideryMethod=hand&embroideryStyle=needlepoint", []string{legacyStitch}},
		{"legacy choices across methods stay distinct", "embroideryMethod=machine&embroideryStyle=needlepoint", nil},
		{"loom knitting", "knittingMethod=loom", []string{loom}},
		{"Tunisian field and explicit legacy technique", "crochetMethod=tunisianCrochet", []string{crochet, legacyCrochet}},
		{"standard never inferred", "crochetMethod=standardCrochet", []string{crochet}},
		{"mixed and legacy needle", "feltingMethod=needle", []string{felt, legacyFelt}},
		{"mixed wet", "feltingMethod=wet", []string{felt}},
		{"weaving loom", "loomType=rigidHeddle", []string{weave}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if projectcatalog.Current.ReleasePhase < 2 && (strings.HasPrefix(tc.query, "feltingMethod=") || strings.HasPrefix(tc.query, "loomType=")) {
				t.Skip("craft unavailable in release phase 1")
			}
			parsed, err := api.ParseProjectListRequest(httptest.NewRequest("GET", "/v1/projects?sort=popular&"+tc.query, nil))
			if err != nil {
				t.Fatal(err)
			}
			rows, _, err := store.SearchProjects(ctx, api.ProjectSearchRequest{Filters: parsed.Filters, Sort: parsed.Sort, Limit: parsed.Limit}, now)
			got := searchURIs(rows)
			slices.Sort(got)
			slices.Sort(tc.want)
			if err != nil || !slices.Equal(got, tc.want) {
				t.Fatalf("rows=%v, want %v (error: %v)", got, tc.want, err)
			}
		})
	}
	for _, tc := range []struct{ query, uri string }{{"cross stitch", stitch}, {"wet felting", felt}, {"rigid heddle", weave}} {
		rows, _ := submittedSearch(t, store, true, tc.query, 10, "", now)
		if got := searchURIs(rows); !slices.Equal(got, []string{tc.uri}) {
			t.Errorf("keyword %q matched %v, want %s", tc.query, got, tc.uri)
		}
	}
	rows, _ := submittedSearch(t, store, true, "hand cross stitch", 10, "", now)
	if got := searchURIs(rows); len(got) != 0 {
		t.Errorf("cross-stitch inferred a hand method: %v", got)
	}
}
