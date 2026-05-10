// appview/internal/api/post_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

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
	lastCreateRepo syntax.DID
	lastCreateColl string
	lastCreateRec  any
	createURI      syntax.ATURI
	createCID      syntax.CID
	createErr      error

	lastDeleteRepo syntax.DID
	lastDeleteColl string
	lastDeleteRkey string
	deleteCalls    int
	deleteErr      error
}

func (f *fakePDS) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
	return "", nil
}
func (f *fakePDS) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error { return nil }
func (f *fakePDS) CreateRecord(_ context.Context, repo syntax.DID, coll string, rec any) (syntax.ATURI, syntax.CID, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastCreateRepo = repo
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
func (f *fakePDS) DeleteRecord(_ context.Context, repo syntax.DID, coll string, rkey string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastDeleteRepo = repo
	f.lastDeleteColl = coll
	f.lastDeleteRkey = rkey
	f.deleteCalls++
	return f.deleteErr
}

func newPDSFactory(p *fakePDS) auth.PDSClientFactory {
	return func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return p, nil
	}
}

func failingPDSFactory(err error) auth.PDSClientFactory {
	return func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return nil, err
	}
}

// fakePostStore implements api.PostReader for handler tests.
type fakePostStore struct {
	one                   *api.PostRow
	oneErr                error
	listRows              []*api.PostRow
	listCursor            string
	listErr               error
	replyRows             []*api.PostRow
	replyCursor           string
	replyErr              error
	ancestorRows          []*api.PostRow
	ancestorErr           error
	threadRows            []*api.PostRow
	threadErr             error
	author                *api.PostAuthorRow
	authorErr             error
	engagement            map[string]api.EngagementSummary
	engagementErr         error
	target                *api.PostTargetRef
	targetErr             error
	activeLike            *api.InteractionRow
	activeLikeErr         error
	activeRepost          *api.InteractionRow
	activeRepostErr       error
	lastDID               string
	lastRkey              string
	lastEngagementViewer  string
	lastEngagementURIs    []string
	engagementCalls       int
	lastTargetDID         string
	lastTargetRkey        string
	lastActiveLikeDID     string
	lastActiveLikeURI     string
	lastActiveRepostDID   string
	lastActiveRepostURI   string
	lastReplyParentURI    string
	lastReplyLimit        int
	lastReplyCursor       string
	lastAncestorRootURI   string
	lastAncestorParentURI string
	lastAncestorTargetURI string
	lastAncestorLimit     int
	lastThreadRootURI     string
	lastThreadTargetURI   string
	lastThreadLimit       int
}

func (f *fakePostStore) ReadOne(_ context.Context, did, rkey string) (*api.PostRow, error) {
	f.lastDID = did
	f.lastRkey = rkey
	return f.one, f.oneErr
}
func (f *fakePostStore) ListByAuthor(_ context.Context, _ string, _ int, _ string) ([]*api.PostRow, string, error) {
	return f.listRows, f.listCursor, f.listErr
}
func (f *fakePostStore) ListDirectReplies(_ context.Context, parentURI string, limit int, cursor string) ([]*api.PostRow, string, error) {
	f.lastReplyParentURI = parentURI
	f.lastReplyLimit = limit
	f.lastReplyCursor = cursor
	return f.replyRows, f.replyCursor, f.replyErr
}
func (f *fakePostStore) LoadThreadCandidates(_ context.Context, rootURI, targetURI string, limit int) ([]*api.PostRow, error) {
	f.lastThreadRootURI = rootURI
	f.lastThreadTargetURI = targetURI
	f.lastThreadLimit = limit
	return f.threadRows, f.threadErr
}
func (f *fakePostStore) LoadThreadAncestors(_ context.Context, rootURI, parentURI, targetURI string, limit int) ([]*api.PostRow, error) {
	f.lastAncestorRootURI = rootURI
	f.lastAncestorParentURI = parentURI
	f.lastAncestorTargetURI = targetURI
	f.lastAncestorLimit = limit
	return f.ancestorRows, f.ancestorErr
}
func (f *fakePostStore) ReadAuthor(_ context.Context, _ string) (*api.PostAuthorRow, error) {
	if f.author == nil && f.authorErr == nil {
		return &api.PostAuthorRow{}, nil
	}
	return f.author, f.authorErr
}

func (f *fakePostStore) ResolvePostTarget(_ context.Context, did, rkey string) (*api.PostTargetRef, error) {
	f.lastTargetDID = did
	f.lastTargetRkey = rkey
	if f.targetErr != nil {
		return nil, f.targetErr
	}
	return f.target, nil
}
func (f *fakePostStore) FindActiveLike(_ context.Context, did, subjectURI string) (*api.InteractionRow, error) {
	f.lastActiveLikeDID = did
	f.lastActiveLikeURI = subjectURI
	if f.activeLikeErr != nil {
		return nil, f.activeLikeErr
	}
	if f.activeLike == nil {
		return nil, api.ErrInteractionNotFound
	}
	return f.activeLike, nil
}
func (f *fakePostStore) FindActiveRepost(_ context.Context, did, subjectURI string) (*api.InteractionRow, error) {
	f.lastActiveRepostDID = did
	f.lastActiveRepostURI = subjectURI
	if f.activeRepostErr != nil {
		return nil, f.activeRepostErr
	}
	if f.activeRepost == nil {
		return nil, api.ErrInteractionNotFound
	}
	return f.activeRepost, nil
}

func (f *fakePostStore) EngagementSummaries(_ context.Context, viewerDID string, postURIs []string) (map[string]api.EngagementSummary, error) {
	f.engagementCalls++
	f.lastEngagementViewer = viewerDID
	f.lastEngagementURIs = append([]string(nil), postURIs...)
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

func authedPostPathReq(method, urlPath, body string, did string) *http.Request {
	req := authedReq(method, urlPath, body, did)
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	req.SetPathValue("did", parts[2])
	req.SetPathValue("rkey", parts[3])
	return req
}

func testPostRow(did, rkey, text string, createdAt time.Time) *api.PostRow {
	return &api.PostRow{
		URI:       "at://" + did + "/social.craftsky.feed.post/" + rkey,
		DID:       did,
		Rkey:      rkey,
		CID:       "bafy" + rkey,
		Text:      text,
		CreatedAt: createdAt,
		IndexedAt: createdAt,
	}
}

func testReplyRow(did, rkey, text, rootURI, parentURI string, createdAt time.Time) *api.PostRow {
	row := testPostRow(did, rkey, text, createdAt)
	rootCID := "bafyroot"
	parentCID := "bafyparent"
	row.ReplyRootURI = &rootURI
	row.ReplyRootCID = &rootCID
	row.ReplyParentURI = &parentURI
	row.ReplyParentCID = &parentCID
	return row
}

func TestLikePost_CreatesPDSLikeRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{
		createURI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/likeSrv"),
		createCID: syntax.CID("bafyLike"),
	}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "   ", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateRepo != "did:plc:alice" || pds.lastCreateColl != "social.craftsky.feed.like" {
		t.Fatalf("CreateRecord repo/coll = %q/%q", pds.lastCreateRepo, pds.lastCreateColl)
	}
	rec, ok := pds.lastCreateRec.(map[string]any)
	if !ok {
		t.Fatalf("record type = %T", pds.lastCreateRec)
	}
	if rec["$type"] != "social.craftsky.feed.like" {
		t.Errorf("$type = %v", rec["$type"])
	}
	subject := rec["subject"].(map[string]any)
	if subject["uri"] != store.target.URI || subject["cid"] != store.target.CID {
		t.Errorf("subject = %+v", subject)
	}
	if _, err := time.Parse(time.RFC3339, rec["createdAt"].(string)); err != nil {
		t.Errorf("createdAt is not RFC3339: %v", err)
	}
	var resp api.InteractionWriteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.URI != string(pds.createURI) || resp.CID != string(pds.createCID) || resp.Rkey != "likeSrv" {
		t.Errorf("resp identity = %+v", resp)
	}
	if resp.Subject.URI != store.target.URI || resp.Subject.CID != store.target.CID {
		t.Errorf("resp subject = %+v", resp.Subject)
	}
	if store.lastTargetDID != "did:plc:bob" || store.lastTargetRkey != "post1" {
		t.Errorf("target lookup = %q/%q", store.lastTargetDID, store.lastTargetRkey)
	}
	if store.lastActiveLikeDID != "did:plc:alice" || store.lastActiveLikeURI != store.target.URI {
		t.Errorf("active lookup = %q/%q", store.lastActiveLikeDID, store.lastActiveLikeURI)
	}
}

func TestLikePost_AlreadyLikedReturnsExistingIdentity(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	created := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store := &fakePostStore{
		target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{
			URI: "at://did:plc:alice/social.craftsky.feed.like/existing", DID: "did:plc:alice", Rkey: "existing", CID: "bafyExisting",
			SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost", CreatedAt: created,
		},
	}
	h := api.LikePostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateColl != "" {
		t.Fatalf("CreateRecord called for already-liked path: %q", pds.lastCreateColl)
	}
	var resp api.InteractionWriteResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.URI != store.activeLike.URI || resp.CID != store.activeLike.CID || resp.Rkey != store.activeLike.Rkey {
		t.Errorf("resp = %+v", resp)
	}
}

func TestLikePost_RejectsNonEmptyBody(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", `{"foo":true}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "unexpected_field" {
		t.Errorf("error = %q", body.Error)
	}
	if pdsLookup := store.lastTargetDID; pdsLookup != "" {
		t.Errorf("target lookup should not run, got %q", pdsLookup)
	}
}

func TestLikePost_MissingSubjectReturns404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: api.ErrPostNotFound}
	h := api.LikePostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/missing/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "post_not_found" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestLikePost_PDSCreateFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSFactory(&fakePDS{createErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_write_failed" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnlikePost_ExistingDeletesPDSRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{
		target:     &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.like/like1", DID: "did:plc:alice", Rkey: "like1", CID: "bafyLike", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnlikePostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRepo != "did:plc:alice" || pds.lastDeleteColl != "social.craftsky.feed.like" || pds.lastDeleteRkey != "like1" {
		t.Errorf("DeleteRecord = repo %q coll %q rkey %q", pds.lastDeleteRepo, pds.lastDeleteColl, pds.lastDeleteRkey)
	}
	if pds.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", pds.deleteCalls)
	}
}

func TestUnlikePost_AbsentActiveLikeIsIdempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.UnlikePostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", pds.deleteCalls)
	}
}

func TestUnlikePost_PDSDeleteFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:     &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{Rkey: "like1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnlikePostHandler(store, newPDSFactory(&fakePDS{deleteErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_CreatesPDSRepostRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{
		createURI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.repost/repostSrv"),
		createCID: syntax.CID("bafyRepost"),
	}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "   ", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateRepo != "did:plc:alice" || pds.lastCreateColl != "social.craftsky.feed.repost" {
		t.Fatalf("CreateRecord repo/coll = %q/%q", pds.lastCreateRepo, pds.lastCreateColl)
	}
	rec, ok := pds.lastCreateRec.(map[string]any)
	if !ok {
		t.Fatalf("record type = %T", pds.lastCreateRec)
	}
	if rec["$type"] != "social.craftsky.feed.repost" {
		t.Errorf("$type = %v", rec["$type"])
	}
	subject := rec["subject"].(map[string]any)
	if subject["uri"] != store.target.URI || subject["cid"] != store.target.CID {
		t.Errorf("subject = %+v", subject)
	}
	if _, hasText := rec["text"]; hasText {
		t.Errorf("repost record unexpectedly had text: %+v", rec)
	}
	if _, hasEmbed := rec["embed"]; hasEmbed {
		t.Errorf("repost record unexpectedly had embed: %+v", rec)
	}
	if _, err := time.Parse(time.RFC3339, rec["createdAt"].(string)); err != nil {
		t.Errorf("createdAt is not RFC3339: %v", err)
	}
	var resp api.InteractionWriteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.URI != string(pds.createURI) || resp.CID != string(pds.createCID) || resp.Rkey != "repostSrv" {
		t.Errorf("resp identity = %+v", resp)
	}
	if resp.Subject.URI != store.target.URI || resp.Subject.CID != store.target.CID {
		t.Errorf("resp subject = %+v", resp.Subject)
	}
	if store.lastTargetDID != "did:plc:bob" || store.lastTargetRkey != "post1" {
		t.Errorf("target lookup = %q/%q", store.lastTargetDID, store.lastTargetRkey)
	}
	if store.lastActiveRepostDID != "did:plc:alice" || store.lastActiveRepostURI != store.target.URI {
		t.Errorf("active lookup = %q/%q", store.lastActiveRepostDID, store.lastActiveRepostURI)
	}
}

func TestRepostPost_AlreadyRepostedReturnsExistingIdentity(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	created := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store := &fakePostStore{
		target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{
			URI: "at://did:plc:alice/social.craftsky.feed.repost/existing", DID: "did:plc:alice", Rkey: "existing", CID: "bafyExisting",
			SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost", CreatedAt: created,
		},
	}
	h := api.RepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateColl != "" {
		t.Fatalf("CreateRecord called for already-reposted path: %q", pds.lastCreateColl)
	}
	var resp api.InteractionWriteResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.URI != store.activeRepost.URI || resp.CID != store.activeRepost.CID || resp.Rkey != store.activeRepost.Rkey {
		t.Errorf("resp = %+v", resp)
	}
}

func TestRepostPost_RejectsNonEmptyBody(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", `{"text":"quote-like body","embed":{"foo":true}}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "unexpected_field" {
		t.Errorf("error = %q", body.Error)
	}
	if store.lastTargetDID != "" {
		t.Errorf("target lookup should not run, got %q", store.lastTargetDID)
	}
	if pds.lastCreateColl != "" {
		t.Errorf("CreateRecord should not run, got coll %q", pds.lastCreateColl)
	}
}

func TestRepostPost_MissingSubjectReturns404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: api.ErrPostNotFound}
	h := api.RepostPostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/missing/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "post_not_found" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_PDSCreateFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, newPDSFactory(&fakePDS{createErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_write_failed" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_NewPDSFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, failingPDSFactory(errors.New("session missing")), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_TargetLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: errors.New("database unavailable")}
	h := api.RepostPostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_ActiveLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:          &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepostErr: errors.New("database unavailable"),
	}
	h := api.RepostPostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_ExistingDeletesPDSRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.repost/repost1", DID: "did:plc:alice", Rkey: "repost1", CID: "bafyRepost", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRepo != "did:plc:alice" || pds.lastDeleteColl != "social.craftsky.feed.repost" || pds.lastDeleteRkey != "repost1" {
		t.Errorf("DeleteRecord = repo %q coll %q rkey %q", pds.lastDeleteRepo, pds.lastDeleteColl, pds.lastDeleteRkey)
	}
	if pds.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", pds.deleteCalls)
	}
}

func TestUnrepostPost_AbsentActiveRepostIsIdempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.UnrepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", pds.deleteCalls)
	}
}

func TestUnrepostPost_NewPDSFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, failingPDSFactory(errors.New("session missing")), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_TargetLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: errors.New("database unavailable")}
	h := api.UnrepostPostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_ActiveLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:          &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepostErr: errors.New("database unavailable"),
	}
	h := api.UnrepostPostHandler(store, newPDSFactory(&fakePDS{}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_PDSRecordAlreadyGoneIsIdempotent(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSFactory(&fakePDS{deleteErr: auth.ErrRecordNotFound}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestUnrepostPost_PDSDeleteFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSFactory(&fakePDS{deleteErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
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
	if resp.LikeCount != 0 || resp.RepostCount != 0 || resp.ReplyCount != 0 || resp.ViewerHasLiked || resp.ViewerHasReposted {
		t.Errorf("engagement defaults = %+v", resp)
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

func TestCreatePost_PDSWriteFailed_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{createErr: errors.New("pds rejected the create")}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	// The response envelope should distinguish write-failed from unavailable.
	body := rr.Body.String()
	if !strings.Contains(body, "pds_write_failed") {
		t.Errorf("expected pds_write_failed in body, got: %s", body)
	}
}

func TestCreatePost_PDSUnavailable_502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{}
	// Factory itself fails — the PDS RPC layer is never reached.
	failingFactory := func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return nil, errors.New("session lookup failed")
	}
	h := api.CreatePostHandler(store, failingFactory, fakeResolver{handleFor: "a.example"}, nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "pds_unavailable") {
		t.Errorf("expected pds_unavailable in body, got: %s", body)
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
	store := &fakePostStore{
		one: row,
		engagement: map[string]api.EngagementSummary{
			row.URI: {LikeCount: 3, RepostCount: 1, ReplyCount: 2, ViewerHasLiked: true, ViewerHasReposted: false},
		},
	}
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
	if resp.LikeCount != 3 || resp.RepostCount != 1 || resp.ReplyCount != 2 || !resp.ViewerHasLiked || resp.ViewerHasReposted {
		t.Errorf("engagement = %+v", resp)
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 1 || store.lastEngagementURIs[0] != row.URI || store.lastEngagementViewer != "did:plc:alice" {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
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

func TestListDirectReplies_HappyPath_PaginatesEngagementAndAuthorHandles(t *testing.T) {
	t.Parallel()
	parentURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	replies := []*api.PostRow{
		{URI: "at://did:plc:bob/social.craftsky.feed.post/reply1", DID: "did:plc:bob", Rkey: "reply1", CID: "bafy1", Text: "first"},
		{URI: "at://did:plc:carol/social.craftsky.feed.post/reply2", DID: "did:plc:carol", Rkey: "reply2", CID: "bafy2", Text: "second"},
	}
	store := &fakePostStore{
		target:      &api.PostTargetRef{URI: parentURI, CID: "bafyRoot"},
		replyRows:   replies,
		replyCursor: "next-replies",
		engagement: map[string]api.EngagementSummary{
			replies[0].URI: {LikeCount: 2, RepostCount: 1, ReplyCount: 4, ViewerHasLiked: true},
			replies[1].URI: {LikeCount: 1, RepostCount: 0, ReplyCount: 0, ViewerHasReposted: true},
		},
	}
	h := api.ListDirectRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/root/replies?limit=2&cursor=opaque", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "root")
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
	if store.lastTargetDID != "did:plc:alice" || store.lastTargetRkey != "root" {
		t.Fatalf("target lookup = %s/%s", store.lastTargetDID, store.lastTargetRkey)
	}
	if store.lastReplyParentURI != parentURI || store.lastReplyLimit != 2 || store.lastReplyCursor != "opaque" {
		t.Fatalf("reply lookup = uri:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyLimit, store.lastReplyCursor)
	}
	if len(resp.Items) != 2 || resp.Items[0].Author.Handle != "bob.example" || resp.Items[1].Author.Handle != "carol.example" {
		t.Fatalf("items = %+v", resp.Items)
	}
	if resp.Items[0].LikeCount != 2 || resp.Items[0].RepostCount != 1 || resp.Items[0].ReplyCount != 4 || !resp.Items[0].ViewerHasLiked {
		t.Errorf("item0 engagement = %+v", resp.Items[0])
	}
	if !resp.Items[1].ViewerHasReposted {
		t.Errorf("item1 engagement = %+v", resp.Items[1])
	}
	if store.engagementCalls != 1 || store.lastEngagementViewer != "did:plc:viewer" || len(store.lastEngagementURIs) != 2 {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
	}
	if resp.Cursor != "next-replies" {
		t.Errorf("cursor = %q", resp.Cursor)
	}
}

func TestGetPostThread_TargetWithNoRepliesReturnsEmptyArrays(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	store := &fakePostStore{
		one:        root,
		target:     &api.PostTargetRef{URI: root.URI, CID: root.CID},
		threadRows: []*api.PostRow{root},
	}
	h := api.GetPostThreadHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/thread", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"replies":[]`) {
		t.Fatalf("body should include empty replies array: %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"ancestors":[]`) {
		t.Fatalf("body should include empty ancestors array: %s", rr.Body.String())
	}
	var resp api.ThreadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "root" || len(resp.Ancestors) != 0 || len(resp.Replies) != 0 || resp.Truncated {
		t.Fatalf("thread response = %+v", resp)
	}
	if store.lastThreadRootURI != root.URI || store.lastThreadTargetURI != root.URI || store.lastThreadLimit != 501 {
		t.Fatalf("thread lookup = root:%q target:%q limit:%d", store.lastThreadRootURI, store.lastThreadTargetURI, store.lastThreadLimit)
	}
}

func TestGetPostThread_NestedTreeDescendantsOnlyOldestFirst(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	replyA := testReplyRow("did:plc:bob", "replyA", "a", root.URI, root.URI, base.Add(2*time.Minute))
	replyB := testReplyRow("did:plc:carol", "replyB", "b", root.URI, replyA.URI, base.Add(3*time.Minute))
	replyC := testReplyRow("did:plc:dave", "replyC", "c", root.URI, root.URI, base.Add(time.Minute))
	orphan := testReplyRow("did:plc:erin", "orphan", "orphan", root.URI, "at://did:plc:missing/social.craftsky.feed.post/gone", base.Add(4*time.Minute))
	store := &fakePostStore{
		one:        root,
		target:     &api.PostTargetRef{URI: root.URI, CID: root.CID},
		threadRows: []*api.PostRow{replyA, replyB, replyC, orphan, root},
		engagement: map[string]api.EngagementSummary{
			root.URI:   {ReplyCount: 2},
			replyA.URI: {LikeCount: 2, ViewerHasLiked: true},
			replyB.URI: {RepostCount: 1},
		},
	}
	h := api.GetPostThreadHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/thread", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ThreadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "root" || resp.Post.Author.Handle != "alice.example" || resp.Post.ReplyCount != 2 {
		t.Fatalf("root = %+v", resp.Post)
	}
	if len(resp.Replies) != 2 || resp.Replies[0].Post.Rkey != "replyC" || resp.Replies[1].Post.Rkey != "replyA" {
		t.Fatalf("top-level replies = %+v", resp.Replies)
	}
	if len(resp.Replies[1].Replies) != 1 || resp.Replies[1].Replies[0].Post.Rkey != "replyB" {
		t.Fatalf("nested reply = %+v", resp.Replies[1].Replies)
	}
	if !resp.Replies[1].Post.ViewerHasLiked || resp.Replies[1].Post.LikeCount != 2 || resp.Replies[1].Replies[0].Post.RepostCount != 1 {
		t.Fatalf("engagement = %+v", resp)
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 4 || store.lastEngagementViewer != "did:plc:viewer" {
		t.Fatalf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
	}
}

func TestGetPostThread_TargetIsReplyReturnsDescendantsOnly(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	replyA := testReplyRow("did:plc:bob", "replyA", "a", root.URI, root.URI, base.Add(time.Minute))
	replyB := testReplyRow("did:plc:carol", "replyB", "b", root.URI, replyA.URI, base.Add(2*time.Minute))
	replyC := testReplyRow("did:plc:dave", "replyC", "c", root.URI, root.URI, base.Add(3*time.Minute))
	store := &fakePostStore{
		one:          replyA,
		target:       &api.PostTargetRef{URI: replyA.URI, CID: replyA.CID},
		ancestorRows: []*api.PostRow{root},
		threadRows:   []*api.PostRow{root, replyA, replyB, replyC},
	}
	h := api.GetPostThreadHandler(store, fakeResolver{handleFor: "handle.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:bob/replyA/thread", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ThreadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "replyA" || len(resp.Replies) != 1 || resp.Replies[0].Post.Rkey != "replyB" {
		t.Fatalf("thread should be replyA subtree only: %+v", resp)
	}
	if len(resp.Ancestors) != 1 || resp.Ancestors[0].Rkey != "root" {
		t.Fatalf("ancestors = %+v", resp.Ancestors)
	}
	if store.lastThreadRootURI != root.URI || store.lastThreadTargetURI != replyA.URI {
		t.Fatalf("thread lookup = root:%q target:%q", store.lastThreadRootURI, store.lastThreadTargetURI)
	}
	if store.lastAncestorRootURI != root.URI || store.lastAncestorParentURI != root.URI || store.lastAncestorTargetURI != replyA.URI || store.lastAncestorLimit != 7 {
		t.Fatalf("ancestor lookup = root:%q parent:%q target:%q limit:%d", store.lastAncestorRootURI, store.lastAncestorParentURI, store.lastAncestorTargetURI, store.lastAncestorLimit)
	}
}

func TestGetPostThread_ReplyTargetReturnsAncestorsRootToParent(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	replyA := testReplyRow("did:plc:bob", "replyA", "a", root.URI, root.URI, base.Add(time.Minute))
	replyB := testReplyRow("did:plc:carol", "replyB", "b", root.URI, replyA.URI, base.Add(2*time.Minute))
	replyC := testReplyRow("did:plc:dave", "replyC", "c", root.URI, replyB.URI, base.Add(3*time.Minute))
	store := &fakePostStore{
		one:          replyB,
		target:       &api.PostTargetRef{URI: replyB.URI, CID: replyB.CID},
		ancestorRows: []*api.PostRow{root, replyA},
		threadRows:   []*api.PostRow{replyB, replyC},
		engagement: map[string]api.EngagementSummary{
			root.URI:   {LikeCount: 5},
			replyA.URI: {ReplyCount: 1},
		},
	}
	h := api.GetPostThreadHandler(store, fakeResolver{handleFor: "handle.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:carol/replyB/thread", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ThreadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "replyB" || len(resp.Replies) != 1 || resp.Replies[0].Post.Rkey != "replyC" {
		t.Fatalf("thread subtree = %+v", resp)
	}
	if len(resp.Ancestors) != 2 || resp.Ancestors[0].Rkey != "root" || resp.Ancestors[1].Rkey != "replyA" {
		t.Fatalf("ancestors = %+v", resp.Ancestors)
	}
	if resp.Ancestors[0].LikeCount != 5 || resp.Ancestors[1].ReplyCount != 1 {
		t.Fatalf("ancestor engagement = %+v", resp.Ancestors)
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 4 {
		t.Fatalf("engagement lookup = calls:%d uris:%v", store.engagementCalls, store.lastEngagementURIs)
	}
}

func TestGetPostThread_MissingParentKeepsIndexedRootAncestor(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	missingParentURI := "at://did:plc:missing/social.craftsky.feed.post/gone"
	reply := testReplyRow("did:plc:bob", "reply", "reply", root.URI, missingParentURI, base.Add(time.Minute))
	store := &fakePostStore{
		one:          reply,
		target:       &api.PostTargetRef{URI: reply.URI, CID: reply.CID},
		ancestorRows: []*api.PostRow{root},
		threadRows:   []*api.PostRow{reply},
	}
	h := api.GetPostThreadHandler(store, fakeResolver{handleFor: "handle.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:bob/reply/thread", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ThreadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "reply" || len(resp.Replies) != 0 {
		t.Fatalf("thread response = %+v", resp)
	}
	if len(resp.Ancestors) != 1 || resp.Ancestors[0].Rkey != "root" || resp.Ancestors[0].URI == missingParentURI {
		t.Fatalf("ancestors should omit missing parent and keep indexed root: %+v", resp.Ancestors)
	}
	if store.lastAncestorRootURI != root.URI || store.lastAncestorParentURI != missingParentURI {
		t.Fatalf("ancestor lookup = root:%q parent:%q", store.lastAncestorRootURI, store.lastAncestorParentURI)
	}
}

func TestGetPostThread_TruncatesAtDepthAndTotalCap(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	rows := []*api.PostRow{root}
	parentURI := root.URI
	for i := 1; i <= 7; i++ {
		row := testReplyRow("did:plc:alice", "depth"+strconv.Itoa(i), "d", root.URI, parentURI, base.Add(time.Duration(i)*time.Minute))
		rows = append(rows, row)
		parentURI = row.URI
	}
	for i := 0; i < 493; i++ {
		rows = append(rows, testReplyRow("did:plc:alice", "sibling"+strconv.Itoa(i), "s", root.URI, root.URI, base.Add(time.Duration(i+10)*time.Minute)))
	}
	store := &fakePostStore{
		one:        root,
		target:     &api.PostTargetRef{URI: root.URI, CID: root.CID},
		threadRows: rows,
	}
	h := api.GetPostThreadHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/thread", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ThreadResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Truncated {
		t.Fatal("want truncated true")
	}
	node := resp.Replies[0]
	for depth := 1; depth < 6; depth++ {
		if len(node.Replies) == 0 {
			t.Fatalf("missing depth %d child", depth+1)
		}
		node = node.Replies[0]
	}
	if len(node.Replies) != 0 {
		t.Fatalf("depth cap should omit children beyond depth 6: %+v", node.Replies)
	}
}

func TestGetPostThread_MissingAndInvalidTargets(t *testing.T) {
	t.Parallel()
	h := api.GetPostThreadHandler(&fakePostStore{targetErr: api.ErrPostNotFound}, fakeResolver{}, nilLogger())
	missingReq := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/thread", "", "did:plc:viewer")
	missingRec := httptest.NewRecorder()
	h.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, body = %s", missingRec.Code, missingRec.Body.String())
	}

	invalidReq := authedReq(http.MethodGet, "/v1/posts/not-a-did/root/thread", "", "did:plc:viewer")
	invalidReq.SetPathValue("did", "not-a-did")
	invalidReq.SetPathValue("rkey", "root")
	invalidRec := httptest.NewRecorder()
	h.ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, body = %s", invalidRec.Code, invalidRec.Body.String())
	}
}

func TestListDirectReplies_DefaultLimitAndMissingTarget(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/root", CID: "bafyRoot"}}
	h := api.ListDirectRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/root/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "root")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 50 {
		t.Fatalf("default limit = %d, want 50", store.lastReplyLimit)
	}
}

func TestListDirectReplies_LimitCapsAt100(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/root", CID: "bafyRoot"}}
	h := api.ListDirectRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/root/replies?limit=500", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "root")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 100 {
		t.Fatalf("capped limit = %d, want 100", store.lastReplyLimit)
	}
}

func TestListDirectReplies_MissingTarget_404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: api.ErrPostNotFound}
	h := api.ListDirectRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/missing/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "missing")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 0 {
		t.Fatalf("ListDirectReplies should not run for missing target")
	}
}

func TestListDirectReplies_BadDID_400(t *testing.T) {
	t.Parallel()
	h := api.ListDirectRepliesHandler(&fakePostStore{}, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/not-a-did/root/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "root")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListDirectReplies_InvalidCursor_400(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:   &api.PostTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/root", CID: "bafyRoot"},
		replyErr: envelope.ErrInvalidCursor,
	}
	h := api.ListDirectRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/root/replies?cursor=bad", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "root")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "invalid_cursor" {
		t.Fatalf("error = %q", body.Error)
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
	store := &fakePostStore{
		listRows:   rows,
		listCursor: "next-cursor-opaque",
		engagement: map[string]api.EngagementSummary{
			rows[0].URI: {LikeCount: 5, RepostCount: 4, ReplyCount: 3, ViewerHasLiked: true, ViewerHasReposted: true},
			rows[1].URI: {LikeCount: 1, RepostCount: 0, ReplyCount: 2, ViewerHasLiked: false, ViewerHasReposted: false},
		},
	}
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
	if resp.Items[0].LikeCount != 5 || !resp.Items[0].ViewerHasLiked || !resp.Items[0].ViewerHasReposted {
		t.Errorf("item0 engagement = %+v", resp.Items[0])
	}
	if resp.Items[1].LikeCount != 1 || resp.Items[1].RepostCount != 0 || resp.Items[1].ReplyCount != 2 {
		t.Errorf("item1 engagement = %+v", resp.Items[1])
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 2 || store.lastEngagementViewer != "did:plc:alice" {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
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

func TestCreatePost_WithReply_PassesThroughToPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, nilLogger())
	body := `{"text":"replying","reply":{"root":{"uri":"at://did:plc:bob/social.craftsky.feed.post/root1","cid":"bafyR1"},"parent":{"uri":"at://did:plc:bob/social.craftsky.feed.post/par1","cid":"bafyP1"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	reply, _ := rec["reply"].(map[string]any)
	if reply == nil {
		t.Fatalf("expected reply in PDS body, got: %+v", rec)
	}
	root, _ := reply["root"].(map[string]any)
	if root["uri"] != "at://did:plc:bob/social.craftsky.feed.post/root1" {
		t.Errorf("reply.root.uri = %v", root["uri"])
	}
	if root["cid"] != "bafyR1" {
		t.Errorf("reply.root.cid = %v", root["cid"])
	}
	parent, _ := reply["parent"].(map[string]any)
	if parent["uri"] != "at://did:plc:bob/social.craftsky.feed.post/par1" {
		t.Errorf("reply.parent.uri = %v", parent["uri"])
	}
	if parent["cid"] != "bafyP1" {
		t.Errorf("reply.parent.cid = %v", parent["cid"])
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
	h := api.ListPostsByAuthorHandler(store, fakeResolver{err: errors.New("plc down")}, nilLogger())
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
