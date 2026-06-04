package api_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestParseFacetSuggestionRequestQueryAndLimitBounds(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		target    string
		wantQuery string
		wantLimit int
		wantErr   error
	}{
		{name: "missing query defaults limit", target: "/v1/facets/mentions", wantQuery: "", wantLimit: 10},
		{name: "empty query", target: "/v1/facets/mentions?q=&limit=10", wantQuery: "", wantLimit: 10},
		{name: "whitespace query trims", target: "/v1/facets/mentions?q=%20%20%20&limit=10", wantQuery: "", wantLimit: 10},
		{name: "64 char query accepted", target: "/v1/facets/hashtags?q=" + strings.Repeat("a", 64) + "&limit=25", wantQuery: strings.Repeat("a", 64), wantLimit: 25},
		{name: "65 char query rejected", target: "/v1/facets/hashtags?q=" + strings.Repeat("a", 65), wantErr: api.ErrFacetValidation},
		{name: "missing limit defaults", target: "/v1/facets/hashtags?q=sock", wantQuery: "sock", wantLimit: 10},
		{name: "limit one accepted", target: "/v1/facets/hashtags?q=sock&limit=1", wantQuery: "sock", wantLimit: 1},
		{name: "limit 10 accepted", target: "/v1/facets/hashtags?q=sock&limit=10", wantQuery: "sock", wantLimit: 10},
		{name: "limit 25 accepted", target: "/v1/facets/hashtags?q=sock&limit=25", wantQuery: "sock", wantLimit: 25},
		{name: "limit 26 rejected", target: "/v1/facets/hashtags?q=sock&limit=26", wantErr: api.ErrFacetValidation},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", tc.target, nil)
			got, err := api.ParseFacetSuggestionRequest(req)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFacetSuggestionRequest: %v", err)
			}
			if got.Query != tc.wantQuery || got.Limit != tc.wantLimit {
				t.Fatalf("request = %+v, want query %q limit %d", got, tc.wantQuery, tc.wantLimit)
			}
		})
	}
}
