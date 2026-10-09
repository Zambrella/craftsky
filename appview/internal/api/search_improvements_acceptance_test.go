package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
)

type searchCorpusRecord struct {
	Key         string
	Caption     string
	Title       string
	PatternName string
	Materials   []string
	Tags        []string
	Alt         string
	QuoteOnly   bool
	PrivateOnly bool
}
type searchCorpusCase struct {
	Query     string
	Project   bool
	Language  string
	Rationale string
	Useful    searchCorpusRecord
	Negatives []searchCorpusRecord
}

func seedCorpusRecord(t *testing.T, pool *pgxpool.Pool, tc searchCorpusCase, record searchCorpusRecord, now time.Time) string {
	t.Helper()
	title := record.Caption
	if tc.Project {
		title = record.Title
	}
	uri := searchFixturePost(t, pool, tc.Project, record.Key, title, []string{tc.Language}, now)
	ctx := context.Background()
	if tc.Project {
		if _, err := pool.Exec(ctx, `WITH caption AS (UPDATE craftsky_posts SET text=$2 WHERE uri=$1) UPDATE craftsky_project_posts SET pattern_name=$3 WHERE uri=$1`, uri, record.Caption, record.PatternName); err != nil {
			t.Fatal(err)
		}
		seedProjectDetails(t, pool, uri, record.Materials, nil, nil, nil)
	}
	if len(record.Tags) > 0 {
		seedPostTags(t, pool, uri, record.Tags)
	}
	if record.Alt != "" {
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET images=jsonb_build_array(jsonb_build_object('alt',$2::text)) WHERE uri=$1`, uri, record.Alt); err != nil {
			t.Fatal(err)
		}
	}
	if record.QuoteOnly {
		target := searchFixturePost(t, pool, false, "quote-target", "shawl", []string{"en"}, now)
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET quote_uri=$2,quote_cid='bafycid' WHERE uri=$1`, uri, target); err != nil {
			t.Fatal(err)
		}
	}
	if record.PrivateOnly {
		snapshot, _ := json.Marshal(map[string]string{"text": "lace"})
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot) VALUES('11111111-1111-4111-8111-111111111111','private','account','did:plc:author','did:plc:author',$1)`, snapshot); err != nil {
			t.Fatal(err)
		}
	}
	return uri
}

// AT-001 / BR-001 / AC-001: independent reviewed cases with concrete negatives.
func TestSearchImprovementsRelevanceCorpus(t *testing.T) {
	raw, err := os.ReadFile("testdata/search_improvements.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []searchCorpusCase
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 31 {
		t.Fatalf("expected all 31 explicit query/tab cases, got %d", len(cases))
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%t/%s", tc.Project, tc.Query), func(t *testing.T) {
			if tc.Rationale == "" {
				t.Fatal("missing usefulness rationale")
			}
			pool, store, now := searchFixture(t)
			useful := seedCorpusRecord(t, pool, tc, tc.Useful, now)
			negatives := []string{}
			for _, negative := range tc.Negatives {
				negatives = append(negatives, seedCorpusRecord(t, pool, tc, negative, now))
			}
			handler := api.SearchPostsHandler(store, fakeResolver{}, nilLogger())
			route := "/v1/search/posts"
			if tc.Project {
				route = "/v1/search/projects"
				handler = api.SearchProjectsHandler(store, fakeResolver{}, nilLogger())
			}
			all := []string{}
			cursor := ""
			for pageNumber := 0; pageNumber < 20; pageNumber++ {
				response := httptest.NewRecorder()
				request := authedReq("GET", route+"?q="+url.QueryEscape(tc.Query)+"&limit=5&cursor="+url.QueryEscape(cursor), "", "did:plc:viewer")
				handler.ServeHTTP(response, request)
				if response.Code != 200 {
					t.Fatalf("handler status=%d body=%s", response.Code, response.Body.String())
				}
				var page api.SearchPostPageResponse
				if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
					t.Fatal(err)
				}
				pageURIs := []string{}
				for _, post := range page.Items {
					pageURIs = append(pageURIs, post.URI)
				}
				if pageNumber == 0 && !slices.Contains(pageURIs, useful) {
					t.Fatalf("useful URI absent from top five: %v; %s", pageURIs, tc.Rationale)
				}
				all = append(all, pageURIs...)
				cursor = page.Cursor
				if cursor == "" {
					break
				}
			}
			for _, negative := range negatives {
				if slices.Contains(all, negative) {
					t.Fatalf("designated negative retrieved: %s", negative)
				}
			}
		})
	}
	t.Run("corrected_after_six_reliable", func(t *testing.T) {
		pool, store, now := searchFixture(t)
		for i := range 6 {
			searchFixturePost(t, pool, false, fmt.Sprint(i), "crochett blanket", nil, now)
		}
		corrected := searchFixturePost(t, pool, false, "corrected", "crochet blanket", nil, now)
		all := []string{}
		cursor := ""
		for page := 0; page < 10; page++ {
			rows, next := submittedSearch(t, store, false, "crochett blanket", 2, cursor, now)
			all = append(all, searchURIs(rows)...)
			cursor = next
			if cursor == "" {
				break
			}
		}
		if len(all) != 7 || all[6] != corrected {
			t.Fatalf("tier exception=%v", all)
		}
	})
}
