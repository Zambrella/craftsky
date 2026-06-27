package api_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

func TestParseSearchSuggestionsRequest(t *testing.T) {
	t.Run("trims query defaults to both sections and per-section limit 5", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/search/suggestions?q=%20Sock%20", nil)
		parsed, err := api.ParseSearchSuggestionsRequest(req)
		if err != nil {
			t.Fatalf("ParseSearchSuggestionsRequest valid: %v", err)
		}
		if parsed.Query != "Sock" {
			t.Fatalf("Query = %q, want Sock", parsed.Query)
		}
		wantTypes := map[api.SearchSuggestionType]bool{
			api.SearchSuggestionTypeProfiles: true,
			api.SearchSuggestionTypeHashtags: true,
		}
		if !reflect.DeepEqual(parsed.Types, wantTypes) {
			t.Fatalf("Types = %#v, want %#v", parsed.Types, wantTypes)
		}
		if parsed.ProfileLimit != 5 || parsed.HashtagLimit != 5 {
			t.Fatalf("limits = profile %d hashtag %d, want 5/5", parsed.ProfileLimit, parsed.HashtagLimit)
		}
	})

	t.Run("honors type selection and bounded per-section limits", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/v1/search/suggestions?q=sock&types=profiles&profileLimit=2&hashtagLimit=3", nil)
		parsed, err := api.ParseSearchSuggestionsRequest(req)
		if err != nil {
			t.Fatalf("ParseSearchSuggestionsRequest valid: %v", err)
		}
		wantTypes := map[api.SearchSuggestionType]bool{api.SearchSuggestionTypeProfiles: true}
		if !reflect.DeepEqual(parsed.Types, wantTypes) {
			t.Fatalf("Types = %#v, want %#v", parsed.Types, wantTypes)
		}
		if parsed.ProfileLimit != 2 || parsed.HashtagLimit != 3 {
			t.Fatalf("limits = profile %d hashtag %d, want 2/3", parsed.ProfileLimit, parsed.HashtagLimit)
		}
	})

	for _, path := range []string{
		"/v1/search/suggestions",
		"/v1/search/suggestions?q=+",
		"/v1/search/suggestions?q=sock&types=unknown",
		"/v1/search/suggestions?q=sock&types=profiles,unknown",
		"/v1/search/suggestions?q=sock&profileLimit=0",
		"/v1/search/suggestions?q=sock&hashtagLimit=26",
		"/v1/search/suggestions?q=sock&cursor=opaque",
		"/v1/search/suggestions?q=" + strings.Repeat("a", 257),
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			if _, err := api.ParseSearchSuggestionsRequest(req); err == nil {
				t.Fatalf("ParseSearchSuggestionsRequest(%q) error = nil, want validation error", path)
			}
		})
	}
}

func TestParsePostSearchRequestValidation(t *testing.T) {
	for _, path := range []string{
		"/v1/search/posts",
		"/v1/search/posts?q=+",
		"/v1/search/posts?q=sock&sort=newest",
		"/v1/search/posts?q=sock&limit=101",
		"/v1/search/posts?q=" + strings.Repeat("a", 257),
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			if _, err := api.ParsePostSearchRequest(req); err == nil {
				t.Fatalf("ParsePostSearchRequest(%q) error = nil, want validation error", path)
			}
		})
	}

	req := httptest.NewRequest("GET", "/v1/search/posts?q=%20Sock%20&sort=popular&limit=50&cursor=bad@@", nil)
	if _, err := api.ParsePostSearchRequest(req); err != envelope.ErrInvalidCursor {
		t.Fatalf("invalid cursor error = %v, want envelope.ErrInvalidCursor", err)
	}

	req = httptest.NewRequest("GET", "/v1/search/posts?q=%20Sock%20", nil)
	parsed, err := api.ParsePostSearchRequest(req)
	if err != nil {
		t.Fatalf("ParsePostSearchRequest valid: %v", err)
	}
	if parsed.Query != "Sock" || parsed.Sort != api.SearchSortChronological || parsed.Limit != 25 {
		t.Fatalf("parsed = %+v, want trimmed query, chronological default, limit 25", parsed)
	}
}

func TestDecodeSaveRecentSearchRequestNormalizesTypedPayloads(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "hashtag", body: `{"type":"hashtag","displayLabel":" #Sock ","payload":{"tag":"#Sock"}}`, want: `{"tag":"sock"}`},
		{name: "profile", body: `{"type":"profile","displayLabel":"Ali","payload":{"did":"did:plc:alice","handle":"@Alice.Craftsky.Social","displayName":" Ali "}}`, want: `{"did":"did:plc:alice","displayName":"Ali","handle":"alice.craftsky.social"}`},
		{name: "post popular", body: `{"type":"post","displayLabel":"Alpaca","payload":{"q":" alpaca ","sort":"popular"}}`, want: `{"q":"alpaca","sort":"popular"}`},
		{name: "project filters", body: `{"type":"project","displayLabel":"Projects","payload":{"sort":"chronological","filters":{"projectTag":["KAL","kal"],"craftType":["Knitting"]},"q":" sock "}}`, want: `{"filters":{"craftType":["knitting"],"projectTag":["kal"]},"q":"sock","sort":"chronological"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/search/recent", bytes.NewBufferString(tc.body))
			got, err := api.DecodeSaveRecentSearchRequest(req)
			if err != nil {
				t.Fatalf("DecodeSaveRecentSearchRequest: %v", err)
			}
			var normalized bytes.Buffer
			if err := json.Compact(&normalized, got.NormalizedPayload); err != nil {
				t.Fatalf("compact normalized payload: %v", err)
			}
			if normalized.String() != tc.want {
				t.Fatalf("normalized payload = %s, want %s", normalized.String(), tc.want)
			}
			if got.PayloadHash == "" {
				t.Fatal("PayloadHash empty")
			}
		})
	}
}

func TestDecodeSaveRecentSearchRequestSupportsFutureSearchPayloads(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "query", body: `{"type":"query","displayLabel":" Alpaca socks ","payload":{"q":" alpaca socks "}}`, want: `{"q":"alpaca socks"}`},
		{name: "hashtag tag only", body: `{"type":"hashtag","displayLabel":"#Sock","payload":{"tag":"#Sock"}}`, want: `{"tag":"sock"}`},
		{name: "selected profile identity", body: `{"type":"profile","displayLabel":"Alice","payload":{"did":"did:plc:alice","handle":"Alice.Craftsky.Social","displayName":" Alice ","avatar":"https://cdn.example/alice.jpg"}}`, want: `{"avatar":"https://cdn.example/alice.jpg","did":"did:plc:alice","displayName":"Alice","handle":"alice.craftsky.social"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/search/recent", bytes.NewBufferString(tc.body))
			got, err := api.DecodeSaveRecentSearchRequest(req)
			if err != nil {
				t.Fatalf("DecodeSaveRecentSearchRequest: %v", err)
			}
			var normalized bytes.Buffer
			if err := json.Compact(&normalized, got.NormalizedPayload); err != nil {
				t.Fatalf("compact normalized payload: %v", err)
			}
			if normalized.String() != tc.want {
				t.Fatalf("normalized payload = %s, want %s", normalized.String(), tc.want)
			}
		})
	}
}

func TestDecodeSaveRecentSearchRequestRejectsInvalidTypedPayloads(t *testing.T) {
	for _, body := range []string{
		`{"type":"query","displayLabel":"Query","payload":{"q":""}}`,
		`{"type":"hashtag","displayLabel":"Sock","payload":{}}`,
		`{"type":"hashtag","displayLabel":"Sock","payload":{"tag":"sock","sort":"popular"}}`,
		`{"type":"profile","displayLabel":"Ali","payload":{"q":""}}`,
		`{"type":"profile","displayLabel":"Ali","payload":{"handle":"alice.example"}}`,
		`{"type":"post","displayLabel":"Alpaca","payload":{"q":"alpaca","sort":"newest"}}`,
		`{"type":"project","displayLabel":"Projects","payload":{"filters":{"unknown":["x"]}}}`,
		`{"type":"profile","displayLabel":"Ali","payload":null}`,
	} {
		req := httptest.NewRequest("POST", "/v1/search/recent", bytes.NewBufferString(body))
		if _, err := api.DecodeSaveRecentSearchRequest(req); err == nil {
			t.Fatalf("DecodeSaveRecentSearchRequest(%s) error = nil, want validation", body)
		}
	}
}

func TestParseProfileSearchRequestRejectsSort(t *testing.T) {
	for _, sort := range []string{"popular", "chronological"} {
		req := httptest.NewRequest("GET", "/v1/search/profiles?q=ali&sort="+sort, nil)
		if _, err := api.ParseProfileSearchRequest(req); err == nil {
			t.Fatalf("profile sort %q error = nil, want validation error", sort)
		}
	}

	req := httptest.NewRequest("GET", "/v1/search/profiles?q=%20ali%20&limit=10", nil)
	parsed, err := api.ParseProfileSearchRequest(req)
	if err != nil {
		t.Fatalf("ParseProfileSearchRequest valid: %v", err)
	}
	if parsed.Query != "ali" || parsed.Limit != 10 {
		t.Fatalf("parsed = %+v, want trimmed query and explicit limit", parsed)
	}
}

func TestNormalizeHashtagPathValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "trim lower and remove one hash", raw: " #SockKAL ", want: "sockkal"},
		{name: "canonical sock", raw: "sock", want: "sock"},
		{name: "url decoded hash", raw: "#Sock", want: "sock"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := api.NormalizeHashtagPathValue(tc.raw)
			if err != nil {
				t.Fatalf("NormalizeHashtagPathValue(%q): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeHashtagPathValue(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}

	for _, raw := range []string{"", "#", "##sock", "sock knitting", "sock/knitting", strings.Repeat("a", 129)} {
		t.Run("invalid "+raw, func(t *testing.T) {
			if _, err := api.NormalizeHashtagPathValue(raw); err == nil {
				t.Fatalf("NormalizeHashtagPathValue(%q) error = nil, want validation error", raw)
			}
		})
	}
}

func TestParseExactHashtagPostsRequestNormalizesSafePathTag(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{name: "mixed case without hash", raw: "SockKAL", want: "sockkal"},
		{name: "optional leading hash", raw: "#SockKAL", want: "sockkal"},
		{name: "trimmed tag", raw: " sockkal ", want: "sockkal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/search/hashtags/x/posts", nil)
			req.SetPathValue("tag", tc.raw)
			parsed, err := api.ParseExactHashtagPostsRequest(req)
			if err != nil {
				t.Fatalf("ParseExactHashtagPostsRequest(%q): %v", tc.raw, err)
			}
			if parsed.Tag != tc.want {
				t.Fatalf("Tag = %q, want %q", parsed.Tag, tc.want)
			}
		})
	}

	for _, raw := range []string{"", "#", "##sock", "sock knitting", "sock/knitting", "sock\x00kal"} {
		t.Run("invalid "+raw, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/search/hashtags/x/posts", nil)
			req.SetPathValue("tag", raw)
			if _, err := api.ParseExactHashtagPostsRequest(req); err == nil {
				t.Fatalf("ParseExactHashtagPostsRequest(%q) error = nil, want validation error", raw)
			}
		})
	}
}

func TestParseExactHashtagPostsRequestSortLimitAndCursor(t *testing.T) {
	validCursor, err := envelope.EncodeCursor(map[string]any{"cursor": "opaque"})
	if err != nil {
		t.Fatalf("encode cursor fixture: %v", err)
	}

	for _, tc := range []struct {
		name       string
		query      string
		wantSort   api.SearchSort
		wantLimit  int
		wantCursor string
	}{
		{name: "default sort and limit", query: "", wantSort: api.SearchSortChronological, wantLimit: 25},
		{name: "chronological sort", query: "?sort=chronological&limit=10", wantSort: api.SearchSortChronological, wantLimit: 10},
		{name: "popular sort with cursor", query: "?sort=popular&limit=2&cursor=" + validCursor, wantSort: api.SearchSortPopular, wantLimit: 2, wantCursor: validCursor},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/search/hashtags/sockkal/posts"+tc.query, nil)
			req.SetPathValue("tag", "#SockKAL")
			parsed, err := api.ParseExactHashtagPostsRequest(req)
			if err != nil {
				t.Fatalf("ParseExactHashtagPostsRequest: %v", err)
			}
			if parsed.Tag != "sockkal" || parsed.Sort != tc.wantSort || parsed.Limit != tc.wantLimit || parsed.Cursor != tc.wantCursor {
				t.Fatalf("parsed = %+v, want tag sockkal sort %s limit %d cursor %q", parsed, tc.wantSort, tc.wantLimit, tc.wantCursor)
			}
		})
	}

	for _, path := range []string{
		"/v1/search/hashtags/sockkal/posts?sort=newest",
		"/v1/search/hashtags/sockkal/posts?limit=101",
		"/v1/search/hashtags/sockkal/posts?cursor=bad@@",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			req.SetPathValue("tag", "sockkal")
			if _, err := api.ParseExactHashtagPostsRequest(req); err == nil {
				t.Fatalf("ParseExactHashtagPostsRequest(%q) error = nil, want validation error", path)
			}
		})
	}
}

func TestCanonicalCraftTypes(t *testing.T) {
	const (
		knitting   = "social.craftsky.feed.defs#knitting"
		crochet    = "social.craftsky.feed.defs#crochet"
		sewing     = "social.craftsky.feed.defs#sewing"
		embroidery = "social.craftsky.feed.defs#embroidery"
		quilting   = "social.craftsky.feed.defs#quilting"
	)

	for _, tc := range []struct {
		raw  string
		want string
	}{
		{raw: "knitting", want: knitting},
		{raw: " Knitting ", want: knitting},
		{raw: knitting, want: knitting},
		{raw: "crochet", want: crochet},
		{raw: "sewing", want: sewing},
		{raw: "embroidery", want: embroidery},
		{raw: "quilting", want: quilting},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := api.CanonicalCraftType(tc.raw)
			if err != nil {
				t.Fatalf("CanonicalCraftType(%q): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("CanonicalCraftType(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}

	got, err := api.CanonicalCraftTypes([]string{"knitting", knitting, "Crochet"}, false)
	if err != nil {
		t.Fatalf("CanonicalCraftTypes mixed aliases: %v", err)
	}
	if want := []string{knitting, crochet}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CanonicalCraftTypes mixed aliases = %#v, want %#v", got, want)
	}

	got, err = api.CanonicalCraftTypes(nil, true)
	if err != nil {
		t.Fatalf("CanonicalCraftTypes defaults: %v", err)
	}
	if want := []string{knitting, crochet, sewing, embroidery, quilting}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CanonicalCraftTypes defaults = %#v, want %#v", got, want)
	}

	for _, raw := range []string{"", "spinning", "social.craftsky.feed.defs#spinning"} {
		t.Run("invalid "+raw, func(t *testing.T) {
			if _, err := api.CanonicalCraftType(raw); err == nil {
				t.Fatalf("CanonicalCraftType(%q) error = nil, want validation", raw)
			}
		})
	}
}

func TestCraftTypeRequestParsersUseCanonicalFullTokens(t *testing.T) {
	const (
		knitting   = "social.craftsky.feed.defs#knitting"
		crochet    = "social.craftsky.feed.defs#crochet"
		sewing     = "social.craftsky.feed.defs#sewing"
		embroidery = "social.craftsky.feed.defs#embroidery"
		quilting   = "social.craftsky.feed.defs#quilting"
	)

	projectReq := httptest.NewRequest("GET", "/v1/projects?craftType=knitting&craftType="+knitting+"&craftType=Crochet", nil)
	projectParsed, err := api.ParseProjectListRequest(projectReq)
	if err != nil {
		t.Fatalf("ParseProjectListRequest canonical crafts: %v", err)
	}
	if want := []string{knitting, crochet}; !reflect.DeepEqual(projectParsed.CraftTypes, want) {
		t.Fatalf("project craft types = %#v, want %#v", projectParsed.CraftTypes, want)
	}

	topReq := httptest.NewRequest("GET", "/v1/search/hashtags/top", nil)
	topParsed, err := api.ParseTopHashtagsRequest(topReq)
	if err != nil {
		t.Fatalf("ParseTopHashtagsRequest defaults: %v", err)
	}
	if want := []string{knitting, crochet, sewing, embroidery, quilting}; !reflect.DeepEqual(topParsed.CraftTypes, want) {
		t.Fatalf("top hashtag default craft types = %#v, want %#v", topParsed.CraftTypes, want)
	}

	topReq = httptest.NewRequest("GET", "/v1/search/hashtags/top?craftTypes=quilting&craftTypes="+crochet+"&craftTypes=Quilting", nil)
	topParsed, err = api.ParseTopHashtagsRequest(topReq)
	if err != nil {
		t.Fatalf("ParseTopHashtagsRequest mixed crafts: %v", err)
	}
	if want := []string{quilting, crochet}; !reflect.DeepEqual(topParsed.CraftTypes, want) {
		t.Fatalf("top hashtag craft types = %#v, want %#v", topParsed.CraftTypes, want)
	}

	for _, path := range []string{
		"/v1/projects?craftType=spinning",
		"/v1/search/hashtags/top?craftTypes=social.craftsky.feed.defs%23spinning",
	} {
		t.Run("invalid "+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			var err error
			if strings.HasPrefix(path, "/v1/projects") {
				_, err = api.ParseProjectListRequest(req)
			} else {
				_, err = api.ParseTopHashtagsRequest(req)
			}
			if err == nil {
				t.Fatalf("%s error = nil, want validation", path)
			}
		})
	}
}

func TestProjectBrowseFiltersStayUnderProjectsAPI(t *testing.T) {
	const knitting = "social.craftsky.feed.defs#knitting"

	browseReq := httptest.NewRequest("GET", "/v1/projects?craftType=knitting&color=Blue&material=Alpaca&designTag=Cables&projectTag=KAL&patternDifficulty=Intermediate&projectType=Socks&sort=popular&limit=10", nil)
	browseParsed, err := api.ParseProjectListRequest(browseReq)
	if err != nil {
		t.Fatalf("ParseProjectListRequest browse filters: %v", err)
	}
	if browseParsed.Sort != api.SearchSortPopular || browseParsed.Limit != 10 {
		t.Fatalf("browse sort/limit = %s/%d, want popular/10", browseParsed.Sort, browseParsed.Limit)
	}
	if got := browseParsed.Filters["craftType"]; !reflect.DeepEqual(got, []string{knitting}) {
		t.Fatalf("browse craftType filter = %#v, want [%s]", got, knitting)
	}
	for key, want := range map[string][]string{
		"color":             {"blue"},
		"material":          {"alpaca"},
		"designTag":         {"cables"},
		"projectTag":        {"kal"},
		"patternDifficulty": {"intermediate"},
		"projectType":       {"socks"},
	} {
		if got := browseParsed.Filters[key]; !reflect.DeepEqual(got, want) {
			t.Fatalf("browse %s filter = %#v, want %#v", key, got, want)
		}
	}

	textReq := httptest.NewRequest("GET", "/v1/search/projects?q=%20sock%20&limit=10", nil)
	textParsed, err := api.ParseProjectSearchRequest(textReq)
	if err != nil {
		t.Fatalf("ParseProjectSearchRequest text-only: %v", err)
	}
	if textParsed.Query != "sock" || textParsed.Limit != 10 || len(textParsed.Filters) != 0 {
		t.Fatalf("text project search parsed = %+v, want q sock limit 10 no filters", textParsed)
	}

	for _, path := range []string{
		"/v1/search/projects",
		"/v1/search/projects?q=+",
		"/v1/search/projects?q=sock&sort=popular",
		"/v1/search/projects?q=sock&craftType=knitting",
		"/v1/search/projects?q=sock&material=alpaca",
		"/v1/projects?q=sock",
		"/v1/projects?unknown=value",
	} {
		t.Run("invalid "+path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			var err error
			if strings.HasPrefix(path, "/v1/search/projects") {
				_, err = api.ParseProjectSearchRequest(req)
			} else {
				_, err = api.ParseProjectListRequest(req)
			}
			if err == nil {
				t.Fatalf("%s error = nil, want validation", path)
			}
		})
	}
}

func TestParseProjectSearchRequestTextOnly(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/search/projects?q=%20Sock%20&limit=50", nil)
	parsed, err := api.ParseProjectSearchRequest(req)
	if err != nil {
		t.Fatalf("ParseProjectSearchRequest: %v", err)
	}
	if parsed.Query != "Sock" || parsed.Sort != api.SearchSortChronological || parsed.Limit != 50 || len(parsed.Filters) != 0 {
		t.Fatalf("parsed = %+v, want text-only query Sock limit 50 with no filters", parsed)
	}

	for _, path := range []string{
		"/v1/search/projects",
		"/v1/search/projects?q=+",
		"/v1/search/projects?q=sock&sort=popular",
		"/v1/search/projects?unknown=value",
		"/v1/search/projects?q=sock&craftType=knitting",
		"/v1/search/projects?q=sock&material=alpaca",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			if _, err := api.ParseProjectSearchRequest(req); err == nil {
				t.Fatalf("ParseProjectSearchRequest(%q) error = nil, want validation error", path)
			}
		})
	}
}

func TestParseProjectListRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/projects?craftType=Knitting&craftType=Crochet&sort=popular&limit=50", nil)
	parsed, err := api.ParseProjectListRequest(req)
	if err != nil {
		t.Fatalf("ParseProjectListRequest: %v", err)
	}
	if parsed.Sort != api.SearchSortPopular || parsed.Limit != 50 {
		t.Fatalf("parsed sort/limit = %s/%d, want popular/50", parsed.Sort, parsed.Limit)
	}
	if got := parsed.CraftTypes; len(got) != 2 || got[0] != "social.craftsky.feed.defs#knitting" || got[1] != "social.craftsky.feed.defs#crochet" {
		t.Fatalf("craft types = %#v", got)
	}

	for _, path := range []string{
		"/v1/projects?q=sock",
		"/v1/projects?craftType=",
		"/v1/projects?craftType=" + strings.Repeat("a", 257),
		"/v1/projects?sort=newest",
		"/v1/projects?limit=101",
		"/v1/projects?" + strings.Repeat("craftType=knitting&", 11),
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			if _, err := api.ParseProjectListRequest(req); err == nil {
				t.Fatalf("ParseProjectListRequest(%q) error = nil, want validation error", path)
			}
		})
	}

	req = httptest.NewRequest("GET", "/v1/projects?cursor=bad@@", nil)
	if _, err := api.ParseProjectListRequest(req); err != envelope.ErrInvalidCursor {
		t.Fatalf("invalid cursor error = %v, want envelope.ErrInvalidCursor", err)
	}
}
