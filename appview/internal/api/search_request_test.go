package api_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

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
		{name: "hashtag", body: `{"type":"hashtag","displayLabel":" #Sock ","payload":{"tag":"#Sock","sort":"chronological"}}`, want: `{"sort":"chronological","tag":"sock"}`},
		{name: "profile", body: `{"type":"profile","displayLabel":"Ali","payload":{"q":" ali "}}`, want: `{"q":"ali"}`},
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

func TestDecodeSaveRecentSearchRequestRejectsInvalidTypedPayloads(t *testing.T) {
	for _, body := range []string{
		`{"type":"hashtag","displayLabel":"Sock","payload":{}}`,
		`{"type":"profile","displayLabel":"Ali","payload":{"q":""}}`,
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

func TestParseProjectSearchRequestFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/search/projects?craftType=Knitting&craftType=Crochet&color=Blue&material=Alpaca&designTag=Cables&projectTag=KAL&patternDifficulty=Intermediate&projectType=Socks", nil)
	parsed, err := api.ParseProjectSearchRequest(req)
	if err != nil {
		t.Fatalf("ParseProjectSearchRequest: %v", err)
	}
	if got := parsed.Filters["craftType"]; len(got) != 2 || got[0] != "knitting" || got[1] != "crochet" {
		t.Fatalf("craftType filters = %#v", got)
	}
	if parsed.Sort != api.SearchSortChronological || parsed.Limit != 25 {
		t.Fatalf("parsed sort/limit = %s/%d", parsed.Sort, parsed.Limit)
	}

	for _, path := range []string{
		"/v1/search/projects?unknown=value",
		"/v1/search/projects?craftType=" + strings.Repeat("a", 257),
		"/v1/search/projects?" + strings.Repeat("color=x&", 11),
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			if _, err := api.ParseProjectSearchRequest(req); err == nil {
				t.Fatalf("ParseProjectSearchRequest(%q) error = nil, want validation error", path)
			}
		})
	}
}
