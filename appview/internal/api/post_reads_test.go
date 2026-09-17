package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"slices"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/relationships"
	"strings"
	"testing"
	"time"
)

func TestGetPostBlockedPairReturnsGenericPayloadWithoutHydration(t *testing.T) {
	t.Parallel()
	row := testPostRow("did:plc:bob", "post1", "protected text sentinel", time.Now())
	row.Images = json.RawMessage(`[{"cid":"protected-image","mime":"image/jpeg","alt":"secret"}]`)
	store := &fakePostStore{
		one: row,
		relationshipStates: map[syntax.DID]relationships.State{
			"did:plc:bob": {BlockedBy: true},
		},
	}
	h := hydrateProductionSummaries(
		api.GetPostHandler(store, fakeResolver{handleFor: "bob.example"}, nilLogger()),
		map[syntax.DID]business.AccountType{"did:plc:bob": business.AccountTypeBusiness},
	)
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:bob/post1", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "protected") || store.engagementCalls != 0 {
		t.Fatalf("unsafe response or hydration: body=%s engagementCalls=%d", rr.Body.String(), store.engagementCalls)
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body["availability"] != "blocked" || body["text"] != nil || body["author"] != nil || body["uri"] != nil {
		t.Fatalf("body = %+v", body)
	}
	if strings.Contains(rr.Body.String(), "accountType") || strings.Contains(rr.Body.String(), "business") {
		t.Fatalf("blocked production post exposed business fields: %s", rr.Body.String())
	}
}

func TestGetPostThirdPartyQuoteCannotBridgeBlockedAuthors(t *testing.T) {
	t.Parallel()
	quotedURI := "at://did:plc:bob/social.craftsky.feed.post/quoted"
	quotedCID := "bafyQuoted"
	row := testPostRow("did:plc:carol", "quote", "Carol context", time.Now())
	row.QuoteURI = &quotedURI
	row.QuoteCID = &quotedCID
	quoted := testPostRow("did:plc:bob", "quoted", "protected quote sentinel", time.Now())
	store := &fakePostStore{
		one: row,
		quoteViews: map[string]*api.QuoteViewRow{
			quotedURI: {State: "visible", Post: quoted},
		},
		blockedPairs: map[api.RelationshipPair]bool{
			{First: "did:plc:carol", Second: "did:plc:bob"}: true,
		},
	}
	h := hydrateProductionSummaries(api.GetPostHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:carol": "carol.example", "did:plc:bob": "bob.example",
	}}, nilLogger()), map[syntax.DID]business.AccountType{
		"did:plc:carol": business.AccountTypeRegular,
		"did:plc:bob":   business.AccountTypeBusiness,
	})
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:carol/quote", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || strings.Contains(rr.Body.String(), "protected quote sentinel") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	quoteView := body["quoteView"].(map[string]any)
	if quoteView["state"] != "blocked" || quoteView["post"] != nil {
		t.Fatalf("quoteView = %+v", quoteView)
	}
	if _, exists := quoteView["accountType"]; exists {
		t.Fatalf("blocked quote exposed accountType: %+v", quoteView)
	}
	if _, exists := quoteView["business"]; exists {
		t.Fatalf("blocked quote exposed business data: %+v", quoteView)
	}
}

func TestGetPost_HappyPath(t *testing.T) {
	t.Parallel()
	savedFolderID := "00000000-0000-4000-8000-000000000001"
	row := &api.PostRow{
		URI: "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID: "did:plc:alice", Rkey: "rk1", CID: "bafy", Text: "hi",
		Langs: []string{"fr-CA"},
	}
	store := &fakePostStore{
		one: row,
		engagement: map[string]api.EngagementSummary{
			row.URI: {LikeCount: 3, RepostCount: 1, ReplyCount: 2, ViewerHasLiked: true, ViewerHasReposted: false, ViewerHasReplied: true, ViewerHasSaved: true, ViewerSavedFolderID: &savedFolderID},
		},
	}
	h := hydrateProductionSummaries(
		api.GetPostHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger()),
		map[syntax.DID]business.AccountType{"did:plc:alice": business.AccountTypeBusiness},
	)
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var rawBody struct {
		Author struct {
			AccountType string `json:"accountType"`
		} `json:"author"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &rawBody); err != nil || rawBody.Author.AccountType != "business" {
		t.Fatalf("post author accountType = %q, error %v", rawBody.Author.AccountType, err)
	}
	var resp api.PostResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Text != "hi" || resp.Author.Handle != "alice.example" {
		t.Errorf("resp = %+v", resp)
	}
	if !slices.Equal(resp.Langs, []string{"fr-CA"}) {
		t.Fatalf("IT-020 direct post langs = %v, want [fr-CA]", resp.Langs)
	}
	if resp.LikeCount != 3 || resp.RepostCount != 1 || resp.ReplyCount != 2 || !resp.ViewerHasLiked || resp.ViewerHasReposted || !resp.ViewerHasReplied || !resp.ViewerHasSaved || resp.ViewerSavedFolderID == nil || *resp.ViewerSavedFolderID != savedFolderID {
		t.Errorf("engagement = %+v", resp)
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 1 || store.lastEngagementURIs[0] != row.URI || store.lastEngagementViewer != "did:plc:viewer" {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
	}
}

func TestGetPostReadsDirectPostForAuthenticatedViewer(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{
		URI: "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID: "did:plc:alice", Rkey: "rk1", CID: "bafy", Text: "hidden",
	}
	store := &fakePostStore{one: row}
	h := api.GetPostHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastViewerDID != "did:plc:alice" {
		t.Fatalf("viewer DID = %q, want did:plc:alice", store.lastViewerDID)
	}
}

func TestGetPost_IR002UsesAuthoritativeContentLanguagesForCounts(t *testing.T) {
	row := &api.PostRow{
		URI: "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID: "did:plc:alice", Rkey: "rk1", CID: "bafy", Text: "hi",
	}
	store := &fakePostStore{one: row}
	preferences := &fakeLanguagePreferenceReader{preferences: languages.Preferences{
		PrimaryLanguage:  "fr",
		ContentLanguages: []string{"en", "de"},
	}}
	h := api.GetPostHandler(
		store,
		fakeResolver{handleFor: "alice.example"},
		nilLogger(),
		preferences,
	)
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if got, want := store.lastEngagementLanguages, []string{"en", "de"}; !slices.Equal(got, want) {
		t.Fatalf("content languages = %v, want %v", got, want)
	}
}

func TestGetPostIncludesAuthorViewerRelationshipState(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{
		URI: "at://did:plc:bob/social.craftsky.feed.post/rk1",
		DID: "did:plc:bob", Rkey: "rk1", CID: "bafy", Text: "hi",
	}
	store := &fakePostStore{
		one: row,
		relationshipStates: map[syntax.DID]relationships.State{
			syntax.DID("did:plc:bob"): {Muted: true},
		},
	}
	h := api.GetPostHandler(store, fakeResolver{handleFor: "bob.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:bob/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:bob")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	author := body["author"].(map[string]any)
	if author["muted"] != true || author["blocking"] != false || author["blockedBy"] != false {
		t.Fatalf("author viewer state = %#v", author)
	}
}

func TestGetPost_WithImages_ReturnsRenderReadyMetadata(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{
		URI:    "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID:    "did:plc:alice",
		Rkey:   "rk1",
		CID:    "bafycid",
		Text:   "hi",
		Images: json.RawMessage(`[{"cid":"bafkimage","mime":"image/jpeg","size":253496,"alt":"project photo","aspectRatio":{"width":919,"height":2000}}]`),
	}
	store := &fakePostStore{one: row}
	h := api.GetPostHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Images) != 1 {
		t.Fatalf("len(images) = %d", len(resp.Images))
	}
	img := resp.Images[0]
	if img.CID != "bafkimage" || img.MIME != "image/jpeg" || img.Size != 253496 || img.Alt != "project photo" {
		t.Fatalf("image = %+v", img)
	}
	if img.AspectRatio == nil || img.AspectRatio.Width != 919 || img.AspectRatio.Height != 2000 {
		t.Fatalf("aspectRatio = %+v", img.AspectRatio)
	}
	if img.Thumb == "" || img.Fullsize == "" {
		t.Fatalf("urls missing: %+v", img)
	}
}

func TestGetPost_WithQuote_AttachesCompactQuoteView(t *testing.T) {
	t.Parallel()
	quoteURI := "at://did:plc:carol/social.craftsky.feed.post/root"
	quoteCID := "bafyroot"
	row := &api.PostRow{
		URI:      "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID:      "did:plc:alice",
		Rkey:     "rk1",
		CID:      "bafycid",
		Text:     "quote commentary",
		QuoteURI: &quoteURI,
		QuoteCID: &quoteCID,
	}
	quoted := &api.PostRow{
		URI:  quoteURI,
		DID:  "did:plc:carol",
		Rkey: "root",
		CID:  quoteCID,
		Text: "original text",
	}
	store := &fakePostStore{
		one: row,
		quoteViews: map[string]*api.QuoteViewRow{
			quoteURI: {State: "visible", Post: quoted},
		},
	}
	h := hydrateProductionSummaries(api.GetPostHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger()), map[syntax.DID]business.AccountType{"did:plc:alice": business.AccountTypeRegular, "did:plc:carol": business.AccountTypeBusiness})
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var rawBody struct {
		Author struct {
			AccountType string `json:"accountType"`
		} `json:"author"`
		QuoteView struct {
			Post struct {
				Author struct {
					AccountType string `json:"accountType"`
				} `json:"author"`
			} `json:"post"`
		} `json:"quoteView"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &rawBody); err != nil || rawBody.Author.AccountType != "regular" || rawBody.QuoteView.Post.Author.AccountType != "business" {
		t.Fatalf("quote account types = %+v, error %v", rawBody, err)
	}
	var resp api.PostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(store.lastQuoteViewRefs) != 1 || store.lastQuoteViewRefs[0].URI != quoteURI || store.lastQuoteViewRefs[0].CID != quoteCID {
		t.Fatalf("quote view refs = %+v", store.lastQuoteViewRefs)
	}
	if resp.QuoteView == nil || resp.QuoteView.State != "visible" || resp.QuoteView.Post == nil {
		t.Fatalf("quoteView = %+v, want visible preview", resp.QuoteView)
	}
	if resp.QuoteView.Post.URI != quoteURI || resp.QuoteView.Post.Author.Handle != "carol.example" {
		t.Fatalf("quoteView.post = %+v", resp.QuoteView.Post)
	}
}

func TestGetPost_NotFound_404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{oneErr: api.ErrPostNotFound}
	h := api.GetPostHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetPost_BadDID_400(t *testing.T) {
	t.Parallel()
	h := api.GetPostHandler(&fakePostStore{}, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/not-a-did/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestGetPost_HandleResolutionFailure_502(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{DID: "did:plc:alice", Rkey: "rk1"}
	store := &fakePostStore{one: row}
	h := api.GetPostHandler(store, fakeResolver{err: errors.New("plc down")}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListPosts_HappyPath_PaginatesCorrectly(t *testing.T) {
	t.Parallel()
	savedFolderID := "00000000-0000-4000-8000-000000000001"
	rows := []*api.PostRow{
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk2", DID: "did:plc:alice", Rkey: "rk2", Text: "second"},
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk1", DID: "did:plc:alice", Rkey: "rk1", Text: "first"},
	}
	store := &fakePostStore{
		listRows:   rows,
		listCursor: "next-cursor-opaque",
		engagement: map[string]api.EngagementSummary{
			rows[0].URI: {LikeCount: 5, RepostCount: 4, ReplyCount: 3, ViewerHasLiked: true, ViewerHasReposted: true, ViewerHasSaved: true, ViewerSavedFolderID: &savedFolderID},
			rows[1].URI: {LikeCount: 1, RepostCount: 0, ReplyCount: 2, ViewerHasLiked: false, ViewerHasReposted: false},
		},
	}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts?limit=2", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items  []api.PostResponse `json:"items"`
		Cursor string             `json:"cursor,omitempty"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Items) != 2 {
		t.Fatalf("items len = %d", len(resp.Items))
	}
	if resp.Items[0].Rkey != "rk2" {
		t.Errorf("ordering wrong: %q", resp.Items[0].Rkey)
	}
	if resp.Items[0].LikeCount != 5 || !resp.Items[0].ViewerHasLiked || !resp.Items[0].ViewerHasReposted || !resp.Items[0].ViewerHasSaved || resp.Items[0].ViewerSavedFolderID == nil || *resp.Items[0].ViewerSavedFolderID != savedFolderID {
		t.Errorf("item0 engagement = %+v", resp.Items[0])
	}
	if resp.Items[1].LikeCount != 1 || resp.Items[1].RepostCount != 0 || resp.Items[1].ReplyCount != 2 {
		t.Errorf("item1 engagement = %+v", resp.Items[1])
	}
	if resp.Items[1].ViewerHasSaved || resp.Items[1].ViewerSavedFolderID != nil {
		t.Errorf("item1 saved state = %+v, want unsaved", resp.Items[1])
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 2 || store.lastEngagementViewer != "did:plc:alice" {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
	}
	if resp.Cursor != "next-cursor-opaque" {
		t.Errorf("cursor = %q", resp.Cursor)
	}
}

func TestListPosts_AttachesQuoteViewsToAuthoredQuotePosts(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	quoteURI := "at://did:plc:carol/social.craftsky.feed.post/root"
	quoteCID := "bafyroot"
	row := testPostRow("did:plc:alice", "quote1", "quote commentary", base)
	row.QuoteURI = &quoteURI
	row.QuoteCID = &quoteCID
	quoted := testPostRow("did:plc:carol", "root", "quoted original", base.Add(-time.Minute))
	store := &fakePostStore{
		listRows: []*api.PostRow{row},
		quoteViews: map[string]*api.QuoteViewRow{
			quoteURI: {State: "visible", Post: quoted},
		},
	}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts", "", "did:plc:viewer")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []api.PostResponse `json:"items"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(store.lastQuoteViewRefs) != 1 || store.lastQuoteViewRefs[0].URI != quoteURI || store.lastQuoteViewRefs[0].CID != quoteCID {
		t.Fatalf("quote view refs = %+v", store.lastQuoteViewRefs)
	}
	if len(resp.Items) != 1 || resp.Items[0].QuoteView == nil || resp.Items[0].QuoteView.State != "visible" || resp.Items[0].QuoteView.Post == nil {
		t.Fatalf("items = %+v, want visible quoteView", resp.Items)
	}
	if resp.Items[0].QuoteView.Post.Author.Handle != "carol.example" {
		t.Fatalf("quote preview author = %+v", resp.Items[0].QuoteView.Post.Author)
	}
}

func TestListPosts_WithImages_ReturnsRenderReadyMetadata(t *testing.T) {
	t.Parallel()
	rows := []*api.PostRow{
		{
			URI:    "at://did:plc:alice/social.craftsky.feed.post/rk1",
			DID:    "did:plc:alice",
			Rkey:   "rk1",
			CID:    "bafycid",
			Text:   "post",
			Images: json.RawMessage(`[{"cid":"bafkimage","mime":"image/jpeg","size":253496,"alt":"project photo"}]`),
		},
	}
	store := &fakePostStore{listRows: rows}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items []api.PostResponse `json:"items"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 || len(resp.Items[0].Images) != 1 {
		t.Fatalf("items/images = %+v", resp.Items)
	}
	img := resp.Items[0].Images[0]
	if img.CID != "bafkimage" || img.Thumb == "" || img.Fullsize == "" {
		t.Fatalf("image = %+v", img)
	}
}

func TestListProjectsByAuthor_HappyPath(t *testing.T) {
	t.Parallel()
	title := "Hitchhiker Shawl"
	rows := []*api.PostRow{
		{
			URI:     "at://did:plc:alice/social.craftsky.feed.post/project1",
			DID:     "did:plc:alice",
			Rkey:    "project1",
			CID:     "bafyproject",
			Text:    "project",
			Project: &api.Project{Common: api.ProjectCommon{CraftType: "social.craftsky.feed.defs#knitting", Title: &title}},
		},
	}
	store := &fakePostStore{
		projectListRows:   rows,
		projectListCursor: "next-projects",
		engagement:        map[string]api.EngagementSummary{rows[0].URI: {LikeCount: 2, ViewerHasSaved: true}},
	}
	h := api.ListProjectsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/projects?limit=2", "", "did:plc:viewer")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items  []api.PostResponse `json:"items"`
		Cursor string             `json:"cursor,omitempty"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Project == nil || resp.Items[0].Project.Common.Title == nil || *resp.Items[0].Project.Common.Title != title {
		t.Fatalf("items = %+v", resp.Items)
	}
	if resp.Items[0].LikeCount != 2 || !resp.Items[0].ViewerHasSaved || resp.Items[0].ViewerSavedFolderID != nil || resp.Cursor != "next-projects" {
		t.Fatalf("engagement/cursor = %+v cursor=%q", resp.Items[0], resp.Cursor)
	}
	if store.lastListProjectsDID != "did:plc:alice" || store.lastListProjectsLimit != 2 || store.lastListProjectsCursor != "" {
		t.Fatalf("project list call did=%q limit=%d cursor=%q", store.lastListProjectsDID, store.lastListProjectsLimit, store.lastListProjectsCursor)
	}
}

func TestListCommentsByAuthor_HappyPath(t *testing.T) {
	t.Parallel()
	rows := []*api.PostRow{
		testReplyRow("did:plc:alice", "reply", "reply", "at://did:plc:bob/social.craftsky.feed.post/root", "at://did:plc:bob/social.craftsky.feed.post/comment", time.Now()),
		testReplyRow("did:plc:alice", "comment", "comment", "at://did:plc:bob/social.craftsky.feed.post/root", "at://did:plc:bob/social.craftsky.feed.post/root", time.Now()),
	}
	store := &fakePostStore{
		commentListRows:   rows,
		commentListCursor: "next-comments",
		engagement: map[string]api.EngagementSummary{
			rows[0].URI: {ReplyCount: 1, ViewerHasSaved: true},
			rows[1].URI: {ReplyCount: 2},
		},
	}
	h := api.ListCommentsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/comments?limit=2", "", "did:plc:viewer")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Items  []api.PostResponse `json:"items"`
		Cursor string             `json:"cursor,omitempty"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 || resp.Items[0].Rkey != "reply" || resp.Items[1].Rkey != "comment" {
		t.Fatalf("items = %+v", resp.Items)
	}
	if !resp.Items[0].ViewerHasSaved || resp.Items[0].ViewerSavedFolderID != nil || resp.Items[1].ViewerHasSaved || resp.Items[1].ViewerSavedFolderID != nil {
		t.Fatalf("comment/reply saved states = %+v", resp.Items)
	}
	if resp.Cursor != "next-comments" {
		t.Fatalf("cursor = %q", resp.Cursor)
	}
	if store.lastListCommentsDID != "did:plc:alice" || store.lastListCommentsLimit != 2 || store.lastListCommentsCursor != "" {
		t.Fatalf("list call did=%q limit=%d cursor=%q", store.lastListCommentsDID, store.lastListCommentsLimit, store.lastListCommentsCursor)
	}
	if store.engagementCalls != 1 || store.lastEngagementViewer != "did:plc:viewer" {
		t.Fatalf("engagement calls=%d viewer=%q", store.engagementCalls, store.lastEngagementViewer)
	}
}

func TestListCommentsByAuthor_BadCursor_400(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{commentListErr: envelope.ErrInvalidCursor}
	h := api.ListCommentsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/comments?cursor=garbage", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListPosts_ResolvesHandle(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listRows: []*api.PostRow{}}
	resolver := fakeResolver{
		didFor:    syntax.DID("did:plc:alice"),
		handleFor: syntax.Handle("alice.example"),
	}
	h := api.ListPostsByAuthorHandler(store, resolver, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@alice.example/posts", "", "did:plc:bob")
	req.SetPathValue("handleOrDid", "alice.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestListPosts_BadCursor_400(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listErr: envelope.ErrInvalidCursor}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts?cursor=garbage", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListPosts_FinalPage_OmitsCursorField(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listRows: []*api.PostRow{}, listCursor: ""}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), `"cursor"`) {
		t.Errorf("cursor field should be omitted, got body: %s", rr.Body.String())
	}
}

type fakePostStoreCapturing struct {
	fakePostStore
	captured *struct {
		limit            int
		viewerDID        string
		authorDID        string
		contentLanguages []string
	}
}

func (f *fakePostStoreCapturing) ListByAuthor(_ context.Context, _ string, limit int, _ string) ([]*api.PostRow, string, error) {
	f.captured.limit = limit
	return f.listRows, f.listCursor, f.listErr
}

func (f *fakePostStoreCapturing) ListByAuthorWithLanguages(
	_ context.Context,
	viewerDID string,
	authorDID string,
	contentLanguages []string,
	limit int,
	_ string,
) ([]*api.PostRow, string, error) {
	f.captured.limit = limit
	f.captured.viewerDID = viewerDID
	f.captured.authorDID = authorDID
	f.captured.contentLanguages = append([]string(nil), contentLanguages...)
	return f.listRows, f.listCursor, f.listErr
}

func TestListPosts_LoadsAuthoritativeContentLanguages(t *testing.T) {
	captured := struct {
		limit            int
		viewerDID        string
		authorDID        string
		contentLanguages []string
	}{}
	store := &fakePostStoreCapturing{
		fakePostStore: fakePostStore{listRows: []*api.PostRow{}},
		captured:      &captured,
	}
	preferences := &fakeLanguagePreferenceReader{
		preferences: languages.Preferences{
			PrimaryLanguage:  "fr",
			ContentLanguages: []string{"en", "de"},
		},
	}
	h := api.ListPostsByAuthorHandler(
		store,
		fakeResolver{handleFor: "alice.example"},
		nilLogger(),
		nil,
		preferences,
	)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts", "", "did:plc:viewer")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if captured.viewerDID != "did:plc:viewer" || captured.authorDID != "did:plc:alice" {
		t.Fatalf("viewer/author = %q/%q", captured.viewerDID, captured.authorDID)
	}
	if got, want := captured.contentLanguages, []string{"en", "de"}; !slices.Equal(got, want) {
		t.Fatalf("content languages = %v, want %v", got, want)
	}
}

func TestListPosts_LimitDefaultAndCap(t *testing.T) {
	t.Parallel()
	captured := struct {
		limit            int
		viewerDID        string
		authorDID        string
		contentLanguages []string
	}{}
	store := &fakePostStoreCapturing{
		fakePostStore: fakePostStore{listRows: []*api.PostRow{}},
		captured:      &captured,
	}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts?limit=500", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if captured.limit != 100 {
		t.Errorf("limit = %d, want capped at 100", captured.limit)
	}
}

func TestListPosts_HandleResolutionFails_502(t *testing.T) {
	t.Parallel()
	rows := []*api.PostRow{
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk1", DID: "did:plc:alice", Rkey: "rk1", Text: "hi"},
	}
	store := &fakePostStore{listRows: rows}
	// Resolver fails. With non-empty rows the handler MUST resolve the
	// handle to render the response — and on failure must return 502.
	h := api.ListPostsByAuthorHandler(store, fakeResolver{err: errors.New("plc down")}, nilLogger(), nil)
	req := authedReq(http.MethodGet, "/v1/profiles/@did:plc:alice/posts", "", "did:plc:alice")
	req.SetPathValue("handleOrDid", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "identity_unavailable") {
		t.Errorf("expected identity_unavailable in body, got: %s", body)
	}
}
