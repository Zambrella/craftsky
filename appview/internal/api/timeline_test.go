// appview/internal/api/timeline_test.go
package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

type fakeTimelineStore struct {
	rows                  []*api.TimelineFeedItemRow
	cursor                string
	err                   error
	engagement            map[string]api.EngagementSummary
	engagementErr         error
	quoteViews            map[string]*api.QuoteViewRow
	quoteViewsErr         error
	lastViewerDID         string
	lastLimit             int
	lastCursor            string
	lastEngagementViewer  string
	lastEngagementPostURI []string
	lastQuoteViewRefs     []api.ResponseStrongRef
}

func TestTimelineHandler_HandleResolutionFailureFailsRequest(t *testing.T) {
	row := testPostRow("did:plc:alice", "post", "post", time.Date(2026, 5, 28, 19, 0, 0, 0, time.UTC))
	store := &fakeTimelineStore{rows: []*api.TimelineFeedItemRow{authoredTimelineItem(row)}}
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

func TestTimelineHandler_ClientCancellationDoesNotBecomeServerError(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	store := &fakeTimelineStore{err: context.Canceled}
	handler := api.ListTimelineHandler(store, fakeResolver{handleFor: "viewer.example"}, logger)

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("canceled request wrote status/body %d/%q, want no server response", rec.Code, rec.Body.String())
	}
	if strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatalf("canceled request emitted an error log: %s", logs.String())
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

func (f *fakeTimelineStore) ListTimeline(_ context.Context, viewerDID string, limit int, cursor string) ([]*api.TimelineFeedItemRow, string, error) {
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

func (f *fakeTimelineStore) QuoteViewRows(_ context.Context, refs []api.ResponseStrongRef) (map[string]*api.QuoteViewRow, error) {
	f.lastQuoteViewRefs = append([]api.ResponseStrongRef(nil), refs...)
	if f.quoteViewsErr != nil {
		return nil, f.quoteViewsErr
	}
	out := make(map[string]*api.QuoteViewRow, len(refs))
	for _, ref := range refs {
		out[ref.URI] = &api.QuoteViewRow{State: "unavailable"}
	}
	for uri, view := range f.quoteViews {
		out[uri] = view
	}
	return out, nil
}

func TestTimelineHandler_DoesNotSynthesizeUnindexedPosts(t *testing.T) {
	indexed := testPostRow("did:plc:viewer", "indexed", "indexed only", time.Date(2026, 5, 28, 17, 0, 0, 0, time.UTC))
	store := &fakeTimelineStore{rows: []*api.TimelineFeedItemRow{authoredTimelineItem(indexed)}}
	handler := api.ListTimelineHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Items []struct {
			Post api.PostResponse `json:"post"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Post.URI != indexed.URI {
		t.Fatalf("items = %+v, want exactly indexed store row %s", body.Items, indexed.URI)
	}
	if body.Items[0].Post.URI == "at://did:plc:viewer/social.craftsky.feed.post/unindexed" {
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
		rows:   []*api.TimelineFeedItemRow{authoredTimelineItem(row)},
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
	item := body.Items[0].Post
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

func TestTimelineHandler_AttachesQuoteViewsToTimelinePosts(t *testing.T) {
	visibleQuoteURI := "at://did:plc:carol/social.craftsky.feed.post/quoted"
	visibleQuoteCID := "bafyquoted"
	hiddenQuoteURI := "at://did:plc:eve/social.craftsky.feed.post/hidden"
	hiddenQuoteCID := "bafyhidden"
	quotePost := testPostRow("did:plc:bob", "quote", "bob quote", time.Date(2026, 5, 28, 18, 0, 0, 0, time.UTC))
	quotePost.QuoteURI = &visibleQuoteURI
	quotePost.QuoteCID = &visibleQuoteCID
	hiddenQuotePost := testPostRow("did:plc:dana", "quote-hidden", "dana quote", time.Date(2026, 5, 28, 17, 0, 0, 0, time.UTC))
	hiddenQuotePost.QuoteURI = &hiddenQuoteURI
	hiddenQuotePost.QuoteCID = &hiddenQuoteCID
	quoted := testPostRow("did:plc:carol", "quoted", "carol original", time.Date(2026, 5, 28, 16, 0, 0, 0, time.UTC))
	quoted.URI = visibleQuoteURI
	quoted.CID = visibleQuoteCID
	store := &fakeTimelineStore{
		rows: []*api.TimelineFeedItemRow{
			authoredTimelineItem(quotePost),
			authoredTimelineItem(hiddenQuotePost),
		},
		quoteViews: map[string]*api.QuoteViewRow{
			visibleQuoteURI: {State: "visible", Post: quoted},
			hiddenQuoteURI:  {State: "hidden"},
		},
	}
	handler := api.ListTimelineHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:bob":   "bob.example",
		"did:plc:dana":  "dana.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body api.TimelinePage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid timeline page: %v", err)
	}
	if len(store.lastQuoteViewRefs) != 2 {
		t.Fatalf("quote refs = %+v, want two batched refs", store.lastQuoteViewRefs)
	}
	visible := body.Items[0].Post.QuoteView
	if visible == nil || visible.State != "visible" || visible.Post == nil {
		t.Fatalf("visible quoteView = %+v", visible)
	}
	if visible.Post.URI != visibleQuoteURI || visible.Post.Author.Handle != "carol.example" {
		t.Fatalf("visible quote preview = %+v", visible.Post)
	}
	hidden := body.Items[1].Post.QuoteView
	if hidden == nil || hidden.State != "hidden" || hidden.Post != nil {
		t.Fatalf("hidden quoteView = %+v", hidden)
	}
}

func TestTimelineHandler_ReturnsFeedItemsWithRepostReason(t *testing.T) {
	post := testPostRow("did:plc:carol", "root", "carol post", time.Date(2026, 5, 28, 20, 0, 0, 0, time.UTC))
	displayName := "Bob"
	repostAt := time.Date(2026, 5, 28, 20, 5, 0, 0, time.UTC)
	store := &fakeTimelineStore{
		rows: []*api.TimelineFeedItemRow{{
			ItemKind:   "repost",
			ItemKey:    "repost:at://did:plc:bob/social.craftsky.feed.repost/rp1",
			ActivityAt: repostAt,
			Post:       post,
			Repost: &api.TimelineRepostReasonRow{
				URI:               "at://did:plc:bob/social.craftsky.feed.repost/rp1",
				CID:               "bafyrepost",
				DID:               "did:plc:bob",
				CreatedAt:         repostAt.Add(-time.Minute),
				IndexedAt:         repostAt,
				AuthorDisplayName: &displayName,
			},
		}},
	}
	handler := api.ListTimelineHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:carol": "carol.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/feed/timeline", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var raw struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("body not valid JSON: %v", err)
	}
	if len(raw.Items) != 1 {
		t.Fatalf("raw items len = %d, want 1", len(raw.Items))
	}
	if _, ok := raw.Items[0]["uri"]; ok {
		t.Fatalf("timeline item is a bare post, want feed item: %s", rec.Body.String())
	}

	var body struct {
		Items []struct {
			ItemKey string            `json:"itemKey"`
			Post    *api.PostResponse `json:"post"`
			Reason  *struct {
				Type      string         `json:"type"`
				By        api.PostAuthor `json:"by"`
				URI       string         `json:"uri"`
				CID       string         `json:"cid,omitempty"`
				CreatedAt time.Time      `json:"createdAt"`
				IndexedAt time.Time      `json:"indexedAt"`
			} `json:"reason,omitempty"`
		} `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not valid feed item page: %v", err)
	}
	item := body.Items[0]
	if item.ItemKey != "repost:at://did:plc:bob/social.craftsky.feed.repost/rp1" {
		t.Fatalf("itemKey = %q", item.ItemKey)
	}
	if item.Post == nil || item.Post.URI != post.URI || item.Post.Author.Handle != "carol.example" {
		t.Fatalf("post = %+v, want hydrated Carol post", item.Post)
	}
	if item.Reason == nil || item.Reason.Type != "repost" {
		t.Fatalf("reason = %+v, want repost reason", item.Reason)
	}
	if item.Reason.By.DID != "did:plc:bob" || item.Reason.By.Handle != "bob.example" || item.Reason.By.DisplayName == nil || *item.Reason.By.DisplayName != "Bob" {
		t.Fatalf("reason.by = %+v, want Bob actor summary", item.Reason.By)
	}
	if item.Reason.By.Muted || item.Reason.By.Blocking || item.Reason.By.BlockedBy {
		t.Fatalf("reason.by relationship state = %+v, want known visible state", item.Reason.By)
	}
	if item.Reason.URI != "at://did:plc:bob/social.craftsky.feed.repost/rp1" || item.Reason.CID != "bafyrepost" {
		t.Fatalf("reason identity = %+v", item.Reason)
	}
	if !item.Reason.CreatedAt.Equal(repostAt.Add(-time.Minute)) || !item.Reason.IndexedAt.Equal(repostAt) {
		t.Fatalf("reason timestamps = created %v indexed %v", item.Reason.CreatedAt, item.Reason.IndexedAt)
	}
}

func authoredTimelineItem(row *api.PostRow) *api.TimelineFeedItemRow {
	return &api.TimelineFeedItemRow{
		ItemKind:   "post",
		ItemKey:    "post:" + row.URI,
		ActivityAt: row.IndexedAt,
		Post:       row,
	}
}
