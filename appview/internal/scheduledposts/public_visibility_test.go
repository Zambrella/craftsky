package scheduledposts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/scheduledposts"
)

// IT-003 protects the public read and search boundary for pending project details.
func TestScheduledProjectDetailPublicOnlyAfterPublication(t *testing.T) {
	const marker = "amberacornfitproof"
	project := json.RawMessage(`{
		"common":{"craftType":"social.craftsky.feed.defs#knitting"},
		"details":{
			"$type":"social.craftsky.project.knitting#details",
			"finishedSize":"` + marker + `"
		}
	}`)
	scheduledposts.CheckProjectPublicationForTest(t, project, func(pool *pgxpool.Pool, owner syntax.DID, key syntax.RecordKey, published bool) {
		resolver := scheduledProjectResolver{}
		store := api.NewPublicPageStore(pool, api.NewPostStore(pool), resolver, business.NewStore(pool))
		mux := http.NewServeMux()
		mux.Handle("GET /v1/public/posts/{did}/{rkey}", api.PublicPagesHandler(store, resolver, "post"))
		mux.Handle("GET /v1/public/search/projects", api.PublicBrowsingHandler(store, resolver, "search/projects"))
		read := httptest.NewRecorder()
		mux.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/v1/public/posts/"+owner.String()+"/"+key.String(), nil))
		search := httptest.NewRecorder()
		mux.ServeHTTP(search, httptest.NewRequest(http.MethodGet, "/v1/public/search/projects?q="+url.QueryEscape(marker), nil))
		if search.Code != http.StatusOK {
			t.Fatalf("search status = %d, body = %s", search.Code, search.Body)
		}
		var results api.PublicPostPage
		if err := json.Unmarshal(search.Body.Bytes(), &results); err != nil {
			t.Fatal(err)
		}
		if !published {
			if read.Code != http.StatusNotFound || bytes.Contains(read.Body.Bytes(), []byte(marker)) {
				t.Fatalf("pending project detail read status = %d, body = %s", read.Code, read.Body)
			}
			if len(results.Items) != 0 || results.Cursor != "" || bytes.Contains(search.Body.Bytes(), []byte(marker)) {
				t.Fatalf("pending project detail appeared in search: %s", search.Body)
			}
			return
		}
		if read.Code != http.StatusOK {
			t.Fatalf("published project detail read status = %d, body = %s", read.Code, read.Body)
		}
		var post api.PublicPost
		if err := json.Unmarshal(read.Body.Bytes(), &post); err != nil {
			t.Fatal(err)
		}
		wantURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + key.String())
		assertPublishedProjectDetail(t, post, wantURI, marker)
		if len(results.Items) != 1 || results.Cursor != "" {
			t.Fatalf("expected exactly one published project result: %s", search.Body)
		}
		assertPublishedProjectDetail(t, results.Items[0], wantURI, marker)
	})
}

func assertPublishedProjectDetail(t *testing.T, post api.PublicPost, uri syntax.ATURI, marker string) {
	t.Helper()
	if post.URI != uri || post.Project == nil {
		t.Fatalf("published result = %s, want project %s", post.URI, uri)
	}
	var details struct {
		FinishedSize string `json:"finishedSize"`
	}
	if err := json.Unmarshal(post.Project.Details, &details); err != nil || details.FinishedSize != marker {
		t.Fatalf("published project details = %s, want finished size %q: %v", post.Project.Details, marker, err)
	}
}

type scheduledProjectResolver struct{}

func (scheduledProjectResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return "alice.test", nil
}

func (scheduledProjectResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "did:plc:alice", nil
}
