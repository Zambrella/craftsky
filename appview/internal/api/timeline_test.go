// appview/internal/api/timeline_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

type fakeTimelineStore struct {
	rows                  []*api.PostRow
	cursor                string
	err                   error
	engagement            map[string]api.EngagementSummary
	engagementErr         error
	lastViewerDID         string
	lastLimit             int
	lastCursor            string
	lastEngagementViewer  string
	lastEngagementPostURI []string
}

func TestTimelineHandler_HandleResolutionFailureFailsRequest(t *testing.T) {
	row := testPostRow("did:plc:alice", "post", "post", time.Date(2026, 5, 28, 19, 0, 0, 0, time.UTC))
	store := &fakeTimelineStore{rows: []*api.PostRow{row}}
	handler := api.ListTimelineHandler(store, fakeResolver{err: errors.New("resolver unavailable")}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body=%s", rec.Code, rec.Body.String())
	}
	var env envelope.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body not valid error envelope: %v", err)
	}
	if env.Error != "identity_unavailable" {
		t.Fatalf("error = %q, want identity_unavailable", env.Error)
	}
}

func TestTimelineHandler_InvalidCursorUsesStandardErrorEnvelope(t *testing.T) {
	store := &fakeTimelineStore{err: envelope.ErrInvalidCursor}
	handler := api.ListTimelineHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline?cursor=not-a-valid-cursor", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var env envelope.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body not valid error envelope: %v", err)
	}
	if env.Error != "invalid_cursor" || env.Message == "" {
		t.Fatalf("envelope = %+v, want invalid_cursor with message", env)
	}
}

func TestTimelineHandler_IgnoresUnknownParamsAndUsesTimelineLimits(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantLimit  int
		wantCursor string
	}{
		{name: "default", path: "/v1/feed/timeline?craftType=weaving&tag=loom", wantLimit: 20},
		{name: "cap", path: "/v1/feed/timeline?limit=999&cursor=opaque&authorList=ignored", wantLimit: 50, wantCursor: "opaque"},
		{name: "invalid", path: "/v1/feed/timeline?limit=not-an-int", wantLimit: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeTimelineStore{}
			handler := api.ListTimelineHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
			req := authedReq(http.MethodGet, tt.path, "", "did:plc:viewer")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if store.lastLimit != tt.wantLimit || store.lastCursor != tt.wantCursor {
				t.Fatalf("store limit/cursor = %d/%q, want %d/%q", store.lastLimit, store.lastCursor, tt.wantLimit, tt.wantCursor)
			}
		})
	}
}

func TestTimelineHandler_EmptyTimelineReturnsEmptyItemsAndOmitsCursor(t *testing.T) {
	store := &fakeTimelineStore{}
	handler := api.ListTimelineHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if _, ok := raw["cursor"]; ok {
		t.Fatalf("empty timeline body contains cursor: %s", rec.Body.String())
	}
	if _, ok := raw["suggestions"]; ok {
		t.Fatalf("empty timeline body contains suggestions: %s", rec.Body.String())
	}
	var body api.TimelinePage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid timeline page: %v", err)
	}
	if len(body.Items) != 0 {
		t.Fatalf("items = %+v, want empty", body.Items)
	}
}

func (f *fakeTimelineStore) ListTimeline(_ context.Context, viewerDID string, limit int, cursor string) ([]*api.PostRow, string, error) {
	f.lastViewerDID = viewerDID
	f.lastLimit = limit
	f.lastCursor = cursor
	return f.rows, f.cursor, f.err
}

func (f *fakeTimelineStore) EngagementSummaries(_ context.Context, viewerDID string, postURIs []string) (map[string]api.EngagementSummary, error) {
	f.lastEngagementViewer = viewerDID
	f.lastEngagementPostURI = append([]string(nil), postURIs...)
	if f.engagementErr != nil {
		return nil, f.engagementErr
	}
	out := make(map[string]api.EngagementSummary, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = api.EngagementSummary{}
	}
	for uri, summary := range f.engagement {
		out[uri] = summary
	}
	return out, nil
}

func TestTimelineHandler_DoesNotSynthesizeUnindexedPosts(t *testing.T) {
	indexed := testPostRow("did:plc:viewer", "indexed", "indexed only", time.Date(2026, 5, 28, 17, 0, 0, 0, time.UTC))
	store := &fakeTimelineStore{rows: []*api.PostRow{indexed}}
	handler := api.ListTimelineHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []api.PostResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].URI != indexed.URI {
		t.Fatalf("items = %+v, want exactly indexed store row %s", body.Items, indexed.URI)
	}
	if body.Items[0].URI == "at://did:plc:viewer/social.craftsky.feed.post/unindexed" {
		t.Fatal("timeline synthesized an unindexed post")
	}
}

func TestTimelineHandler_ReturnsPostResponseItemsWithEngagementAndNoTotalCount(t *testing.T) {
	quoteURI := "at://did:plc:other/social.craftsky.feed.post/quoted"
	quoteCID := "bafyquoted"
	displayName := "Alice"
	avatarCID := "bafyavatar"
	row := testPostRow("did:plc:alice", "quote", "quote post", time.Date(2026, 5, 28, 18, 0, 0, 0, time.UTC))
	row.QuoteURI = &quoteURI
	row.QuoteCID = &quoteCID
	row.Tags = []string{"weaving"}
	row.AuthorDisplayName = &displayName
	row.AuthorAvatarCID = &avatarCID
	row.Images = json.RawMessage(`[{
		"cid":"bafyimage",
		"mime":"image/jpeg",
		"size":123,
		"alt":"loom",
		"aspectRatio":{"width":4,"height":3}
	}]`)
	store := &fakeTimelineStore{
		rows:   []*api.PostRow{row},
		cursor: "next-cursor",
		engagement: map[string]api.EngagementSummary{
			row.URI: {
				LikeCount:         2,
				RepostCount:       1,
				ReplyCount:        3,
				ViewerHasLiked:    true,
				ViewerHasReposted: true,
				ViewerHasReplied:  true,
			},
		},
	}
	handler := api.ListTimelineHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{"did:plc:alice": "alice.example"}}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline?limit=1&cursor=opaque", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if _, ok := raw["totalCount"]; ok {
		t.Fatalf("body contains totalCount: %s", rec.Body.String())
	}
	var body api.TimelinePage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid timeline page: %v", err)
	}
	if body.Cursor != "next-cursor" {
		t.Fatalf("cursor = %q, want next-cursor", body.Cursor)
	}
	if len(body.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(body.Items))
	}
	item := body.Items[0]
	if item.URI != row.URI || item.Author.Handle != "alice.example" || item.Author.DisplayName == nil || *item.Author.DisplayName != "Alice" {
		t.Fatalf("item response mismatch: %+v", item)
	}
	if item.Quote == nil || item.Quote.URI != quoteURI || item.Quote.CID != quoteCID {
		t.Fatalf("quote = %+v, want strong ref only", item.Quote)
	}
	if item.LikeCount != 2 || item.RepostCount != 1 || item.ReplyCount != 3 || !item.ViewerHasLiked || !item.ViewerHasReposted || !item.ViewerHasReplied {
		t.Fatalf("engagement fields not applied: %+v", item)
	}
	if len(item.Images) != 1 || item.Images[0].CID != "bafyimage" || item.Images[0].Thumb == "" || item.Images[0].Fullsize == "" {
		t.Fatalf("images not rendered: %+v", item.Images)
	}
	if got := store.lastEngagementPostURI; len(got) != 1 || got[0] != row.URI {
		t.Fatalf("engagement lookup URIs = %v, want [%s]", got, row.URI)
	}
}
