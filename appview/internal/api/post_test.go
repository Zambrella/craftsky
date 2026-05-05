// appview/internal/api/post_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

// fakePDS records the last call. zero-value methods succeed; populate
// errors to simulate failures.
type fakePDS struct {
	mu             sync.Mutex
	lastCreateColl string
	lastCreateRec  any
	createURI      syntax.ATURI
	createCID      syntax.CID
	createErr      error

	lastDeleteRkey string
	deleteErr      error
}

func (f *fakePDS) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
	return "", nil
}
func (f *fakePDS) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error { return nil }
func (f *fakePDS) CreateRecord(_ context.Context, _ syntax.DID, coll string, rec any) (syntax.ATURI, syntax.CID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCreateColl = coll
	f.lastCreateRec = rec
	if f.createErr != nil {
		return "", "", f.createErr
	}
	if f.createURI == "" {
		f.createURI = syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/rkSrv")
		f.createCID = syntax.CID("bafySrv")
	}
	return f.createURI, f.createCID, nil
}
func (f *fakePDS) DeleteRecord(_ context.Context, _ syntax.DID, _, rkey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastDeleteRkey = rkey
	return f.deleteErr
}

func newPDSFactory(p *fakePDS) auth.PDSClientFactory {
	return func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return p, nil
	}
}

// fakePostStore implements api.PostReader for handler tests.
type fakePostStore struct {
	one        *api.PostRow
	oneErr     error
	listRows   []*api.PostRow
	listCursor string
	listErr    error
	author     *api.PostAuthorRow
	authorErr  error
	lastDID    string
	lastRkey   string
}

func (f *fakePostStore) ReadOne(_ context.Context, did, rkey string) (*api.PostRow, error) {
	f.lastDID = did
	f.lastRkey = rkey
	return f.one, f.oneErr
}
func (f *fakePostStore) ListByAuthor(_ context.Context, _ string, _ int, _ string) ([]*api.PostRow, string, error) {
	return f.listRows, f.listCursor, f.listErr
}
func (f *fakePostStore) ReadAuthor(_ context.Context, _ string) (*api.PostAuthorRow, error) {
	if f.author == nil && f.authorErr == nil {
		return &api.PostAuthorRow{}, nil
	}
	return f.author, f.authorErr
}

func authedReq(method, path, body string, did string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	ctx := middleware.WithDID(r.Context(), syntax.DID(did))
	return r.WithContext(ctx)
}

func TestCreatePost_HappyPath(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	resolver := fakeResolver{handleFor: "alice.example"}
	h := api.CreatePostHandler(store, newPDSFactory(pds), resolver, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hello"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Text != "hello" || resp.URI == "" || resp.CID == "" {
		t.Errorf("resp = %+v", resp)
	}
	if resp.Rkey != "rkSrv" {
		t.Errorf("rkey not derived from PDS uri: %q", resp.Rkey)
	}
	if resp.Author.Handle != "alice.example" {
		t.Errorf("author.handle = %q", resp.Author.Handle)
	}

	body, _ := pds.lastCreateRec.(map[string]any)
	if body["$type"] != "social.craftsky.feed.post" {
		t.Errorf("missing/wrong $type: %v", body["$type"])
	}
	if _, ok := body["createdAt"].(string); !ok {
		t.Errorf("createdAt missing or non-string: %v", body["createdAt"])
	}
}

func TestCreatePost_MalformedBody_400(t *testing.T) {
	t.Parallel()
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(&fakePDS{}), fakeResolver{}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{not json`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreatePost_TextEmpty_422(t *testing.T) {
	t.Parallel()
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(&fakePDS{}), fakeResolver{}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":""}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreatePost_PDSDown_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{createErr: errors.New("pds down")}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestCreatePost_QuoteEmbed_TranslatedToLexiconShape(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	body := `{"text":"hi","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyB"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	if embed["$type"] != "social.craftsky.feed.post#quoteEmbed" {
		t.Errorf("embed $type: %v", embed["$type"])
	}
	r, _ := embed["record"].(map[string]any)
	if r["uri"] != "at://did:plc:bob/social.craftsky.feed.post/r1" {
		t.Errorf("embed.record.uri = %v", r["uri"])
	}
}

func TestCreatePost_TagsExtractedFromFacets(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	body := `{"text":"hi #knit","facets":[{"index":{"byteStart":3,"byteEnd":8},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Knitting"}]}]}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Tags) != 1 || resp.Tags[0] != "knitting" {
		t.Errorf("tags = %v, want [knitting]", resp.Tags)
	}
}

func TestCreatePost_AuthorHydratedFromStore(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	displayName := "Alice"
	avatarCID := "bafyAvatar"
	store := &fakePostStore{
		author: &api.PostAuthorRow{DisplayName: &displayName, AvatarCID: &avatarCID},
	}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Author.DisplayName == nil || *resp.Author.DisplayName != "Alice" {
		t.Errorf("displayName = %v", resp.Author.DisplayName)
	}
	if resp.Author.AvatarCID == nil || *resp.Author.AvatarCID != "bafyAvatar" {
		t.Errorf("avatarCID = %v", resp.Author.AvatarCID)
	}
}

func TestCreatePost_ResolveHandleFails_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{err: errors.New("plc down")}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestGetPost_HappyPath(t *testing.T) {
	t.Parallel()
	row := &api.PostRow{
		URI: "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID: "did:plc:alice", Rkey: "rk1", CID: "bafy", Text: "hi",
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
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Text != "hi" || resp.Author.Handle != "alice.example" {
		t.Errorf("resp = %+v", resp)
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

func TestDeletePost_Self_204_CallsPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRkey != "rk1" {
		t.Errorf("PDS not called: %q", pds.lastDeleteRkey)
	}
}

func TestDeletePost_OtherUser_403_NoPDSCall(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:bob/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:bob")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rr.Code)
	}
	if pds.lastDeleteRkey != "" {
		t.Errorf("PDS should not have been called")
	}
}

func TestDeletePost_RecordAlreadyGone_204_Idempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{deleteErr: auth.ErrRecordNotFound}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeletePost_PDSDown_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{deleteErr: errors.New("pds down")}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeletePost_BadDID_400(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.DeletePostHandler(newPDSFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/not-a-did/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListPosts_HappyPath_PaginatesCorrectly(t *testing.T) {
	t.Parallel()
	rows := []*api.PostRow{
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk2", DID: "did:plc:alice", Rkey: "rk2", Text: "second"},
		{URI: "at://did:plc:alice/social.craftsky.feed.post/rk1", DID: "did:plc:alice", Rkey: "rk1", Text: "first"},
	}
	store := &fakePostStore{listRows: rows, listCursor: "next-cursor-opaque"}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
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
	if resp.Cursor != "next-cursor-opaque" {
		t.Errorf("cursor = %q", resp.Cursor)
	}
}

func TestListPosts_ResolvesHandle(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{listRows: []*api.PostRow{}}
	resolver := fakeResolver{
		didFor:    syntax.DID("did:plc:alice"),
		handleFor: syntax.Handle("alice.example"),
	}
	h := api.ListPostsByAuthorHandler(store, resolver, nilLogger())
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
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
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
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
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
	captured *struct{ limit int }
}

func (f *fakePostStoreCapturing) ListByAuthor(_ context.Context, _ string, limit int, _ string) ([]*api.PostRow, string, error) {
	f.captured.limit = limit
	return f.listRows, f.listCursor, f.listErr
}

func TestListPosts_LimitDefaultAndCap(t *testing.T) {
	t.Parallel()
	captured := struct {
		limit int
	}{}
	store := &fakePostStoreCapturing{
		fakePostStore: fakePostStore{listRows: []*api.PostRow{}},
		captured:      &captured,
	}
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
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
