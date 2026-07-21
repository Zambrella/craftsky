// appview/internal/api/post_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
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
	"social.craftsky/appview/internal/relationships"
)

// fakePDS records the last call. zero-value methods succeed; populate
// errors to simulate failures.
type fakePDS struct {
	mu             sync.Mutex
	lastCreateRepo syntax.DID
	lastCreateColl string
	lastCreateRec  any
	createCalls    int
	createURI      syntax.ATURI
	createCID      syntax.CID
	createErr      error

	lastDeleteRepo syntax.DID
	lastDeleteColl string
	lastDeleteRkey string
	deleteCalls    int
	deleteErr      error

	uploadCalls    int
	lastUploadMIME string
	lastUploadBody []byte
	uploadResp     *auth.UploadedBlob
	uploadErr      error
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
	f.createCalls++
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
func (f *fakePDS) UploadBlob(_ context.Context, contentType string, body []byte) (*auth.UploadedBlob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploadCalls++
	f.lastUploadMIME = contentType
	f.lastUploadBody = append([]byte(nil), body...)
	if f.uploadErr != nil {
		return nil, f.uploadErr
	}
	if f.uploadResp != nil {
		return f.uploadResp, nil
	}
	return &auth.UploadedBlob{
		Raw:  map[string]any{"$type": "blob", "ref": map[string]any{"$link": "bafk-default"}, "mimeType": "image/jpeg", "size": float64(1)},
		CID:  "bafk-default",
		MIME: "image/jpeg",
		Size: 1,
	}, nil
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
	authorizationErr       error
	authorizationCalls     []authorizationCall
	relationshipStates     map[syntax.DID]relationships.State
	relationshipStateErr   error
	blockedPairs           map[api.RelationshipPair]bool
	one                    *api.PostRow
	oneErr                 error
	listRows               []*api.PostRow
	listCursor             string
	listErr                error
	projectListRows        []*api.PostRow
	projectListCursor      string
	projectListErr         error
	commentListRows        []*api.PostRow
	commentListCursor      string
	commentListErr         error
	commentRows            []*api.PostRow
	commentCursor          string
	commentErr             error
	postByURI              *api.PostRow
	postsByURI             map[string]*api.PostRow
	postByURIErr           error
	replyRows              []*api.PostRow
	replyCursor            string
	replyErr               error
	aroundReplyRows        []*api.PostRow
	aroundReplyCursor      string
	aroundReplyErr         error
	author                 *api.PostAuthorRow
	authorErr              error
	engagement             map[string]api.EngagementSummary
	engagementErr          error
	quoteViews             map[string]*api.QuoteViewRow
	quoteViewsErr          error
	target                 *api.PostTargetRef
	targetErr              error
	shareTarget            *api.ShareTargetRef
	shareTargetErr         error
	activeLike             *api.InteractionRow
	activeLikeErr          error
	activeRepost           *api.InteractionRow
	activeRepostErr        error
	lastDID                string
	lastRkey               string
	lastListCommentsDID    string
	lastListCommentsLimit  int
	lastListCommentsCursor string
	lastListProjectsDID    string
	lastListProjectsLimit  int
	lastListProjectsCursor string
	lastEngagementViewer   string
	lastEngagementURIs     []string
	lastQuoteViewRefs      []api.ResponseStrongRef
	engagementCalls        int
	lastTargetDID          string
	lastTargetRkey         string
	lastShareTargetDID     string
	lastShareTargetRkey    string
	lastActiveLikeDID      string
	lastActiveLikeURI      string
	lastActiveRepostDID    string
	lastActiveRepostURI    string
	lastCommentRootURI     string
	lastCommentViewerDID   string
	lastCommentSort        string
	lastCommentLimit       int
	lastCommentCursor      string
	lastReplyParentURI     string
	lastReplyRootURI       string
	lastReplyLimit         int
	lastReplyCursor        string
	lastReplyFocusURI      string
}

type authorizationCall struct {
	Actor     syntax.DID
	Subject   syntax.DID
	Operation relationships.Operation
}

func (f *fakePostStore) AuthorizeDirectedInteraction(_ context.Context, actor, subject syntax.DID, operation relationships.Operation) error {
	f.authorizationCalls = append(f.authorizationCalls, authorizationCall{Actor: actor, Subject: subject, Operation: operation})
	return f.authorizationErr
}

func (f *fakePostStore) RelationshipState(_ context.Context, _ syntax.DID, subject syntax.DID) (relationships.State, error) {
	if f.relationshipStateErr != nil {
		return relationships.State{}, f.relationshipStateErr
	}
	return f.relationshipStates[subject], nil
}

func (f *fakePostStore) RelationshipStates(_ context.Context, _ syntax.DID, subjects []syntax.DID) (map[syntax.DID]relationships.State, error) {
	if f.relationshipStateErr != nil {
		return nil, f.relationshipStateErr
	}
	out := make(map[syntax.DID]relationships.State, len(subjects))
	for _, subject := range subjects {
		out[subject] = f.relationshipStates[subject]
	}
	return out, nil
}

func (f *fakePostStore) BlockedPairs(_ context.Context, pairs []api.RelationshipPair) (map[api.RelationshipPair]bool, error) {
	out := make(map[api.RelationshipPair]bool, len(pairs))
	for _, pair := range pairs {
		out[pair] = f.blockedPairs[pair]
	}
	return out, nil
}

func (f *fakePostStore) ReadOne(_ context.Context, did, rkey string) (*api.PostRow, error) {
	f.lastDID = did
	f.lastRkey = rkey
	return f.one, f.oneErr
}
func (f *fakePostStore) ListByAuthor(_ context.Context, _ string, _ int, _ string) ([]*api.PostRow, string, error) {
	return f.listRows, f.listCursor, f.listErr
}
func (f *fakePostStore) ListProjectsByAuthor(_ context.Context, did string, limit int, cursor string) ([]*api.PostRow, string, error) {
	f.lastListProjectsDID = did
	f.lastListProjectsLimit = limit
	f.lastListProjectsCursor = cursor
	return f.projectListRows, f.projectListCursor, f.projectListErr
}
func (f *fakePostStore) ListCommentsByAuthor(_ context.Context, did string, limit int, cursor string) ([]*api.PostRow, string, error) {
	f.lastListCommentsDID = did
	f.lastListCommentsLimit = limit
	f.lastListCommentsCursor = cursor
	return f.commentListRows, f.commentListCursor, f.commentListErr
}
func (f *fakePostStore) ListRootComments(_ context.Context, rootURI, viewerDID, sort string, limit int, cursor string) ([]*api.PostRow, string, error) {
	f.lastCommentRootURI = rootURI
	f.lastCommentViewerDID = viewerDID
	f.lastCommentSort = sort
	f.lastCommentLimit = limit
	f.lastCommentCursor = cursor
	return f.commentRows, f.commentCursor, f.commentErr
}
func (f *fakePostStore) ReadPostByURI(_ context.Context, uri string) (*api.PostRow, error) {
	if f.postByURIErr != nil {
		return nil, f.postByURIErr
	}
	if f.postByURI != nil && f.postByURI.URI == uri {
		return f.postByURI, nil
	}
	if f.postsByURI != nil {
		if row := f.postsByURI[uri]; row != nil {
			return row, nil
		}
	}
	return nil, api.ErrPostNotFound
}
func (f *fakePostStore) ListCommentBranchReplies(_ context.Context, commentURI, rootURI string, limit int, cursor string) ([]*api.PostRow, string, error) {
	f.lastReplyParentURI = commentURI
	f.lastReplyRootURI = rootURI
	f.lastReplyLimit = limit
	f.lastReplyCursor = cursor
	return f.replyRows, f.replyCursor, f.replyErr
}
func (f *fakePostStore) ListCommentBranchRepliesAround(_ context.Context, commentURI, rootURI, focusURI string, limit int) ([]*api.PostRow, string, error) {
	f.lastReplyParentURI = commentURI
	f.lastReplyRootURI = rootURI
	f.lastReplyFocusURI = focusURI
	f.lastReplyLimit = limit
	f.lastReplyCursor = ""
	if f.aroundReplyRows != nil || f.aroundReplyCursor != "" || f.aroundReplyErr != nil {
		return f.aroundReplyRows, f.aroundReplyCursor, f.aroundReplyErr
	}
	return f.replyRows, f.replyCursor, f.replyErr
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

func (f *fakePostStore) ResolveShareTarget(_ context.Context, did, rkey string) (*api.ShareTargetRef, error) {
	f.lastShareTargetDID = did
	f.lastShareTargetRkey = rkey
	if f.shareTargetErr != nil {
		return nil, f.shareTargetErr
	}
	if f.shareTarget != nil {
		return f.shareTarget, nil
	}
	if f.target != nil {
		return &api.ShareTargetRef{URI: f.target.URI, CID: f.target.CID}, nil
	}
	return nil, api.ErrPostNotFound
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

func (f *fakePostStore) QuoteViewRows(_ context.Context, refs []api.ResponseStrongRef) (map[string]*api.QuoteViewRow, error) {
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

func replyItemsContainURI(items []api.ReplyItem, uri string) bool {
	for _, item := range items {
		if item.Post.URI == uri {
			return true
		}
	}
	return false
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
	h := api.GetPostHandler(store, fakeResolver{handleFor: "bob.example"}, nilLogger())
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
	h := api.GetPostHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:carol": "carol.example", "did:plc:bob": "bob.example",
	}}, nilLogger())
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
}

func TestListCommentRepliesShapesMutedBranchAndOmitsBlockedBranch(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:root/social.craftsky.feed.post/root"
	commentURI := "at://did:plc:commenter/social.craftsky.feed.post/comment"
	target := testReplyRow("did:plc:commenter", "comment", "comment", rootURI, rootURI, time.Now())
	muted := testReplyRow("did:plc:bob", "muted", "protected muted text", rootURI, commentURI, time.Now().Add(time.Second))
	child := testReplyRow("did:plc:carol", "child", "descendant text", rootURI, muted.URI, time.Now().Add(2*time.Second))

	for _, test := range []struct {
		name      string
		state     relationships.State
		wantItems int
		wantState string
	}{
		{name: "mute collapses full descendant branch", state: relationships.State{Muted: true}, wantItems: 1, wantState: "muted"},
		{name: "block omits full descendant branch", state: relationships.State{Blocking: true}, wantItems: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &fakePostStore{
				one:       target,
				replyRows: []*api.PostRow{muted, child},
				relationshipStates: map[syntax.DID]relationships.State{
					"did:plc:bob": test.state,
				},
			}
			h := api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
				"did:plc:bob": "bob.example", "did:plc:carol": "carol.example",
			}}, nilLogger())
			req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:commenter/comment/replies", "", "did:plc:alice")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			var body map[string]any
			_ = json.Unmarshal(rr.Body.Bytes(), &body)
			items := body["items"].([]any)
			if len(items) != test.wantItems || strings.Contains(rr.Body.String(), "protected muted text") || strings.Contains(rr.Body.String(), "descendant text") {
				t.Fatalf("body = %s", rr.Body.String())
			}
			if test.wantState != "" {
				post := items[0].(map[string]any)["post"].(map[string]any)
				if post["availability"] != test.wantState {
					t.Fatalf("post = %+v", post)
				}
			}
		})
	}
}

func TestDirectedPostCreatesRejectBlockedPairBeforePDSWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation relationships.Operation
		handler   func(*fakePostStore, *fakePDS) http.Handler
		request   func() *http.Request
	}{
		{
			name:      "like",
			operation: relationships.OperationLikeCreate,
			handler: func(store *fakePostStore, pds *fakePDS) http.Handler {
				return api.LikePostHandler(store, newPDSFactory(pds), nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
			},
		},
		{
			name:      "repost",
			operation: relationships.OperationRepostCreate,
			handler: func(store *fakePostStore, pds *fakePDS) http.Handler {
				return api.RepostPostHandler(store, newPDSFactory(pds), nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
			},
		},
		{
			name:      "reply",
			operation: relationships.OperationReplyCreate,
			handler: func(store *fakePostStore, pds *fakePDS) http.Handler {
				return api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				body := `{"text":"reply","reply":{"root":{"uri":"at://did:plc:bob/social.craftsky.feed.post/root","cid":"bafyRoot"},"parent":{"uri":"at://did:plc:bob/social.craftsky.feed.post/post1","cid":"bafyPost"}}}`
				return authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
			},
		},
		{
			name:      "quote",
			operation: relationships.OperationQuoteCreate,
			handler: func(store *fakePostStore, pds *fakePDS) http.Handler {
				return api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				body := `{"text":"quote","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/post1","cid":"bafyPost"}}}`
				return authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
			},
		},
		{
			name:      "mention",
			operation: relationships.OperationMentionCreate,
			handler: func(store *fakePostStore, pds *fakePDS) http.Handler {
				return api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				body := `{"text":"@bob.example","facets":[{"index":{"byteStart":0,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:bob"}]}]}`
				return authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pds := &fakePDS{}
			store := &fakePostStore{
				authorizationErr: api.ErrInteractionBlocked,
				target:           &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
				shareTarget:      &api.ShareTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
			}
			rr := httptest.NewRecorder()
			test.handler(store, pds).ServeHTTP(rr, test.request())

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			var env envelope.Error
			_ = json.NewDecoder(rr.Body).Decode(&env)
			if env.Error != "interaction_blocked" {
				t.Fatalf("error = %q, want interaction_blocked", env.Error)
			}
			if pds.createCalls != 0 {
				t.Fatalf("PDS CreateRecord calls = %d, want 0", pds.createCalls)
			}
			if len(store.authorizationCalls) != 1 || store.authorizationCalls[0] != (authorizationCall{
				Actor: "did:plc:alice", Subject: "did:plc:bob", Operation: test.operation,
			}) {
				t.Fatalf("authorization calls = %+v", store.authorizationCalls)
			}
		})
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

func TestLikePost_PDSSessionExpiredReturns401(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSFactory(&fakePDS{createErr: auth.ErrPDSSessionExpired}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_session_expired" {
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
	if store.lastShareTargetDID != "did:plc:bob" || store.lastShareTargetRkey != "post1" {
		t.Errorf("share target lookup = %q/%q", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	if store.lastActiveRepostDID != "did:plc:alice" || store.lastActiveRepostURI != store.target.URI {
		t.Errorf("active lookup = %q/%q", store.lastActiveRepostDID, store.lastActiveRepostURI)
	}
}

func TestRepostPost_AllowsSelfRepost(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{
		createURI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.repost/selfRepost"),
		createCID: syntax.CID("bafySelfRepost"),
	}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/own", CID: "bafyOwn"}}
	h := api.RepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:alice/own/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastShareTargetDID != "did:plc:alice" || store.lastShareTargetRkey != "own" {
		t.Fatalf("share target lookup = %s/%s", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	if pds.lastCreateRepo != "did:plc:alice" || pds.lastCreateColl != "social.craftsky.feed.repost" {
		t.Fatalf("CreateRecord repo/coll = %q/%q", pds.lastCreateRepo, pds.lastCreateColl)
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

func TestRepostPost_RejectsReplyTargetBeforePDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI:     "at://did:plc:bob/social.craftsky.feed.post/reply",
			CID:     "bafyReply",
			IsReply: true,
		},
	}
	h := api.RepostPostHandler(store, newPDSFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/reply/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("CreateRecord calls = %d, want 0", pds.createCalls)
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", body.Error)
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
	store := &fakePostStore{shareTargetErr: api.ErrPostNotFound}
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
	store := &fakePostStore{shareTargetErr: errors.New("database unavailable")}
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

func TestUnrepostPost_WithAuthoredQuoteDeletesOnlyStraightRepost(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	quoteRkey := "quote1"
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
	if pds.deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d, want 1", pds.deleteCalls)
	}
	if pds.lastDeleteColl != "social.craftsky.feed.repost" || pds.lastDeleteRkey != "repost1" {
		t.Fatalf("DeleteRecord = coll %q rkey %q, want straight repost", pds.lastDeleteColl, pds.lastDeleteRkey)
	}
	if pds.lastDeleteColl == "social.craftsky.feed.post" || pds.lastDeleteRkey == quoteRkey {
		t.Fatalf("DeleteRecord targeted quote post coll %q rkey %q", pds.lastDeleteColl, pds.lastDeleteRkey)
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
	h := api.CreatePostHandler(store, newPDSFactory(pds), resolver, api.DefaultMediaLimits(), nilLogger())
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
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(&fakePDS{}), fakeResolver{}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{not json`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreatePost_TextEmpty_422(t *testing.T) {
	t.Parallel()
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(&fakePDS{}), fakeResolver{}, api.DefaultMediaLimits(), nilLogger())
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
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
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

func TestCreatePost_PDSWriteFailure_LogsExcludeRequestTextAndToken(t *testing.T) {
	t.Parallel()
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	const sentinelText = "SENSITIVE_POST_TEXT"
	const sentinelToken = "SENSITIVE_TOKEN"
	pds := &fakePDS{createErr: errors.New("pds rejected create")}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), logger)
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"`+sentinelText+`"}`, "did:plc:alice")
	req.Header.Set("Authorization", "Bearer "+sentinelToken)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	out := logs.String()
	if strings.Contains(out, sentinelText) {
		t.Fatalf("logs leaked request text: %s", out)
	}
	if strings.Contains(out, sentinelToken) {
		t.Fatalf("logs leaked token: %s", out)
	}
}

func TestCreatePost_PDSUnavailable_502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{}
	// Factory itself fails — the PDS RPC layer is never reached.
	failingFactory := func(_ context.Context, _ syntax.DID, _ string) (auth.PDSClient, error) {
		return nil, errors.New("session lookup failed")
	}
	h := api.CreatePostHandler(store, failingFactory, fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
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

func TestCreatePost_PDSSessionExpiredReturns401(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(&fakePDS{createErr: auth.ErrPDSSessionExpired}), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_session_expired" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestCreatePost_QuoteEmbed_TranslatedToLexiconShape(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/r1", CID: "bafyB"}}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
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

func TestCreatePost_QuoteEmbed_UsesResolvedTargetCID(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI: "at://did:plc:bob/social.craftsky.feed.post/r1",
			CID: "bafyActual",
		},
	}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyStale"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	record, _ := embed["record"].(map[string]any)
	if record["uri"] != "at://did:plc:bob/social.craftsky.feed.post/r1" || record["cid"] != "bafyActual" {
		t.Fatalf("quote record = %+v, want resolved target strongRef", record)
	}
	var resp api.PostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if resp.Quote == nil || resp.Quote.CID != "bafyActual" {
		t.Fatalf("response quote = %+v, want resolved target CID", resp.Quote)
	}
}

func TestCreatePost_QuoteEmbed_AttachesCompactQuoteView(t *testing.T) {
	t.Parallel()
	quoteURI := "at://did:plc:bob/social.craftsky.feed.post/r1"
	quoteCID := "bafyActual"
	pds := &fakePDS{}
	quoted := testPostRow("did:plc:bob", "r1", "quoted text", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	quoted.CID = quoteCID
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI: quoteURI,
			CID: quoteCID,
		},
		quoteViews: map[string]*api.QuoteViewRow{
			quoteURI: {State: "visible", Post: quoted},
		},
	}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyStale"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if len(store.lastQuoteViewRefs) != 1 || store.lastQuoteViewRefs[0].URI != quoteURI || store.lastQuoteViewRefs[0].CID != quoteCID {
		t.Fatalf("quote view refs = %+v", store.lastQuoteViewRefs)
	}
	var resp api.PostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if resp.Quote == nil || resp.Quote.CID != quoteCID {
		t.Fatalf("response quote = %+v, want resolved target CID", resp.Quote)
	}
	if resp.QuoteView == nil || resp.QuoteView.State != "visible" || resp.QuoteView.Post == nil {
		t.Fatalf("quoteView = %+v, want visible preview", resp.QuoteView)
	}
	if resp.QuoteView.Post.URI != quoteURI || resp.QuoteView.Post.Text != "quoted text" || resp.QuoteView.Post.Author.Handle != "bob.example" {
		t.Fatalf("quoteView.post = %+v", resp.QuoteView.Post)
	}
}

func TestCreatePost_AllowsMultipleQuotePostsForSameSubject(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/r1", CID: "bafyB"}}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"another angle","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyB"}}}`

	for i := 0; i < 2; i++ {
		req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, body = %s", i+1, rr.Code, rr.Body.String())
		}
	}
	if pds.createCalls != 2 {
		t.Fatalf("createCalls = %d, want 2 independent quote post writes", pds.createCalls)
	}
	if store.lastShareTargetDID != "did:plc:bob" || store.lastShareTargetRkey != "r1" {
		t.Fatalf("last share target = %s/%s", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
}

func TestCreatePost_AllowsSelfQuote(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/own", CID: "bafyOwn"}}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"adding more context","embed":{"quote":{"uri":"at://did:plc:alice/social.craftsky.feed.post/own","cid":"bafyOwn"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastShareTargetDID != "did:plc:alice" || store.lastShareTargetRkey != "own" {
		t.Fatalf("share target lookup = %s/%s", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	record, _ := embed["record"].(map[string]any)
	if record["uri"] != "at://did:plc:alice/social.craftsky.feed.post/own" {
		t.Fatalf("quote target uri = %v", record["uri"])
	}
}

func TestCreatePost_QuoteRejectsReplyTargetBeforePDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI:     "at://did:plc:bob/social.craftsky.feed.post/reply",
			CID:     "bafyReply",
			IsReply: true,
		},
	}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/reply","cid":"bafyReply"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("CreateRecord calls = %d, want 0", pds.createCalls)
	}
	var env envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&env)
	if env.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", env.Error)
	}
	if store.lastShareTargetDID != "did:plc:bob" || store.lastShareTargetRkey != "reply" {
		t.Fatalf("share target lookup = %q/%q", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
}

func TestCreatePost_ProjectQuoteRejectedBeforePDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"project quote",
		"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}},
		"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/target","cid":"bafyTarget"}}
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("CreateRecord calls = %d, want 0", pds.createCalls)
	}
	if store.lastShareTargetDID != "" || store.lastShareTargetRkey != "" {
		t.Fatalf("share target lookup = %q/%q, want none", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	var env envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&env)
	if env.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", env.Error)
	}
}

func TestCreatePost_TagsExtractedFromFacets(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
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

func TestCreatePost_WithProject_WritesProjectToPDSAndResponse(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"finished shawl #FairIsle",
		"facets":[{"index":{"byteStart":15,"byteEnd":24},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"FairIsle"}]}],
		"project":{
			"common":{
				"craftType":"social.craftsky.feed.defs#knitting",
				"title":"Hitchhiker Shawl",
				"materials":[{"text":"3m of @alice.craftsky.social #viscose fabric","facets":[{"index":{"byteStart":6,"byteEnd":28},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]},{"index":{"byteStart":29,"byteEnd":37},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Viscose"}]}]}],
				"tags":[" fairisle ", "WIP"],
				"pattern":{
					"name":"#hitchhiker",
					"nameFacets":[{"index":{"byteStart":0,"byteEnd":11},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Hitchhiker"}]}],
					"designer":"@alice.craftsky.social",
					"designerFacets":[{"index":{"byteStart":0,"byteEnd":22},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]}]
				}
			},
			"details":{"$type":"social.craftsky.project.knitting#details","projectType":"shawl"}
		}
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	rec, _ := pds.lastCreateRec.(map[string]any)
	project, _ := rec["project"].(*api.Project)
	if project == nil || project.Common.CraftType != "social.craftsky.feed.defs#knitting" || project.Common.Title == nil || *project.Common.Title != "Hitchhiker Shawl" {
		t.Fatalf("PDS project = %#v", rec["project"])
	}
	if project.Common.Pattern == nil || project.Common.Pattern.Name == nil || *project.Common.Pattern.Name != "#hitchhiker" {
		t.Fatalf("PDS project pattern = %#v", project.Common.Pattern)
	}
	if len(project.Common.Materials) != 1 || project.Common.Materials[0].Text != "3m of @alice.craftsky.social #viscose fabric" {
		t.Fatalf("PDS project materials = %#v", project.Common.Materials)
	}
	if got := facetArrayLength(t, project.Common.Materials[0].Facets); got != 2 {
		t.Fatalf("PDS material facets len = %d, want 2", got)
	}
	if got := facetArrayLength(t, project.Common.Pattern.NameFacets); got != 1 {
		t.Fatalf("PDS pattern name facets len = %d, want 1", got)
	}
	if got := facetArrayLength(t, project.Common.Pattern.DesignerFacets); got != 1 {
		t.Fatalf("PDS pattern designer facets len = %d, want 1", got)
	}
	if _, ok := rec["createdAt"].(string); !ok {
		t.Fatalf("createdAt missing from PDS record: %#v", rec)
	}

	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Project == nil || resp.Project.Common.Title == nil || *resp.Project.Common.Title != "Hitchhiker Shawl" {
		t.Fatalf("response project = %+v", resp.Project)
	}
	wantTags := []string{"fairisle", "wip", "hitchhiker", "viscose"}
	if !reflect.DeepEqual(resp.Tags, wantTags) {
		t.Fatalf("response tags = %v, want %v", resp.Tags, wantTags)
	}
}

func TestCreatePost_InvalidProjectDoesNotWritePDS(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.CreatePostHandler(&fakePostStore{}, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi","project":{"common":{}}}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("PDS create calls = %d, want 0", pds.createCalls)
	}
}

func TestCreatePost_AuthorHydratedFromStore(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	displayName := "Alice"
	avatarCID := "bafyAvatar"
	avatarMime := "image/jpeg"
	store := &fakePostStore{
		author: &api.PostAuthorRow{DisplayName: &displayName, AvatarCID: &avatarCID, AvatarMime: &avatarMime},
	}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
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
	if resp.Author.Avatar == nil || *resp.Author.Avatar != "https://cdn.bsky.app/img/avatar/plain/did:plc:alice/bafyAvatar@jpeg" {
		t.Errorf("avatar = %v", resp.Author.Avatar)
	}
}

func TestCreatePost_ResolveHandleFails_502(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{err: errors.New("plc down")}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi"}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestGetPost_HappyPath(t *testing.T) {
	t.Parallel()
	savedFolderID := "00000000-0000-4000-8000-000000000001"
	row := &api.PostRow{
		URI: "at://did:plc:alice/social.craftsky.feed.post/rk1",
		DID: "did:plc:alice", Rkey: "rk1", CID: "bafy", Text: "hi",
	}
	store := &fakePostStore{
		one: row,
		engagement: map[string]api.EngagementSummary{
			row.URI: {LikeCount: 3, RepostCount: 1, ReplyCount: 2, ViewerHasLiked: true, ViewerHasReposted: false, ViewerHasReplied: true, ViewerHasSaved: true, ViewerSavedFolderID: &savedFolderID},
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
	if resp.LikeCount != 3 || resp.RepostCount != 1 || resp.ReplyCount != 2 || !resp.ViewerHasLiked || resp.ViewerHasReposted || !resp.ViewerHasReplied || !resp.ViewerHasSaved || resp.ViewerSavedFolderID == nil || *resp.ViewerSavedFolderID != savedFolderID {
		t.Errorf("engagement = %+v", resp)
	}
	if store.engagementCalls != 1 || len(store.lastEngagementURIs) != 1 || store.lastEngagementURIs[0] != row.URI || store.lastEngagementViewer != "did:plc:alice" {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
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
	h := api.GetPostHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
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

func TestListCommentReplies_HappyPath_PaginatesEngagementAndAuthorHandles(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	parentURI := "at://did:plc:alice/social.craftsky.feed.post/comment"
	comment := testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())
	replies := []*api.PostRow{
		{URI: "at://did:plc:bob/social.craftsky.feed.post/reply1", DID: "did:plc:bob", Rkey: "reply1", CID: "bafy1", Text: "first"},
		{URI: "at://did:plc:carol/social.craftsky.feed.post/reply2", DID: "did:plc:carol", Rkey: "reply2", CID: "bafy2", Text: "second"},
	}
	store := &fakePostStore{
		one:         comment,
		replyRows:   replies,
		replyCursor: "next-replies",
		engagement: map[string]api.EngagementSummary{
			replies[0].URI: {LikeCount: 2, RepostCount: 1, ReplyCount: 4, ViewerHasLiked: true, ViewerHasReplied: true},
			replies[1].URI: {LikeCount: 1, RepostCount: 0, ReplyCount: 0, ViewerHasReposted: true},
		},
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?limit=2&cursor=opaque", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ReplyPage
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if store.lastDID != "did:plc:alice" || store.lastRkey != "comment" {
		t.Fatalf("target lookup = %s/%s", store.lastDID, store.lastRkey)
	}
	if store.lastReplyParentURI != parentURI || store.lastReplyLimit != 2 || store.lastReplyCursor != "opaque" {
		t.Fatalf("reply lookup = uri:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyLimit, store.lastReplyCursor)
	}
	if !resp.Loaded {
		t.Fatal("loaded = false, want true")
	}
	if len(resp.Items) != 2 || resp.Items[0].Post.Author.Handle != "bob.example" || resp.Items[1].Post.Author.Handle != "carol.example" {
		t.Fatalf("items = %+v", resp.Items)
	}
	if resp.Items[0].Flattened || resp.Items[0].ReplyingTo != nil {
		t.Fatalf("direct reply should not be flattened: %+v", resp.Items[0])
	}
	if resp.Items[0].Post.LikeCount != 2 || resp.Items[0].Post.RepostCount != 1 || resp.Items[0].Post.ReplyCount != 4 || !resp.Items[0].Post.ViewerHasLiked || !resp.Items[0].Post.ViewerHasReplied {
		t.Errorf("item0 engagement = %+v", resp.Items[0])
	}
	if !resp.Items[1].Post.ViewerHasReposted {
		t.Errorf("item1 engagement = %+v", resp.Items[1])
	}
	if store.engagementCalls != 1 || store.lastEngagementViewer != "did:plc:viewer" || len(store.lastEngagementURIs) != 2 {
		t.Errorf("engagement lookup = calls:%d viewer:%q uris:%v", store.engagementCalls, store.lastEngagementViewer, store.lastEngagementURIs)
	}
	if resp.Cursor != "next-replies" {
		t.Errorf("cursor = %q", resp.Cursor)
	}
}

func TestListCommentReplies_NestedBranchReplyIncludesFlattenedMetadata(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	parentReply := testReplyRow("did:plc:carol", "reply", "reply", root.URI, comment.URI, base.Add(2*time.Minute))
	nestedReply := testReplyRow("did:plc:dave", "nested", "nested", root.URI, parentReply.URI, base.Add(3*time.Minute))
	displayName := "Carol"
	parentReply.AuthorDisplayName = &displayName
	store := &fakePostStore{
		one:       comment,
		replyRows: []*api.PostRow{parentReply, nestedReply},
		postsByURI: map[string]*api.PostRow{
			parentReply.URI: parentReply,
		},
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:bob/comment/replies", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.ReplyPage
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Fatalf("items = %+v", resp.Items)
	}
	if resp.Items[0].Flattened {
		t.Fatalf("direct reply flattened = true: %+v", resp.Items[0])
	}
	nested := resp.Items[1]
	if !nested.Flattened || nested.ReplyingTo == nil {
		t.Fatalf("nested reply missing flattened metadata: %+v", nested)
	}
	if nested.ReplyingTo.URI != parentReply.URI || nested.ReplyingTo.DID != parentReply.DID || nested.ReplyingTo.Handle != "carol.example" {
		t.Fatalf("replyingTo = %+v", nested.ReplyingTo)
	}
	if nested.ReplyingTo.DisplayName == nil || *nested.ReplyingTo.DisplayName != "Carol" {
		t.Fatalf("displayName = %+v", nested.ReplyingTo.DisplayName)
	}
}

func TestListCommentReplies_CapsPageSizeAtTen(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{
		one:       testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now()),
		replyRows: []*api.PostRow{},
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?limit=20", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 10 {
		t.Fatalf("reply limit = %d, want capped 10", store.lastReplyLimit)
	}
}

func TestGetPostComments_ReturnsRootAndCommentsOnly(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	reply := testReplyRow("did:plc:carol", "reply1", "reply", root.URI, comment.URI, base.Add(2*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{comment},
		engagement: map[string]api.EngagementSummary{
			root.URI:    {ReplyCount: 1},
			comment.URI: {ReplyCount: 1},
			reply.URI:   {ReplyCount: 0},
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Post.Rkey != "root" || resp.Post.Author.Handle != "alice.example" {
		t.Fatalf("root post = %+v", resp.Post)
	}
	if resp.Sort != "oldest" {
		t.Fatalf("sort = %q, want oldest", resp.Sort)
	}
	if len(resp.Comments.Items) != 1 {
		t.Fatalf("comments len = %d, want 1: %+v", len(resp.Comments.Items), resp.Comments.Items)
	}
	if resp.Comments.Items[0].Post.Rkey != "comment1" {
		t.Fatalf("comment item = %+v", resp.Comments.Items[0])
	}
	if resp.Comments.Items[0].Post.Rkey == reply.Rkey {
		t.Fatalf("nested reply was returned as top-level comment: %+v", resp.Comments.Items[0])
	}
	if len(resp.Comments.Items[0].Replies.Items) != 0 || resp.Comments.Items[0].Replies.Loaded {
		t.Fatalf("replies should not be expanded by default: %+v", resp.Comments.Items[0].Replies)
	}
	if store.lastCommentRootURI != root.URI || store.lastCommentLimit != 10 || store.lastCommentCursor != "" || store.lastCommentSort != "oldest" || store.lastCommentViewerDID != "did:plc:viewer" {
		t.Fatalf("comment lookup = root:%q limit:%d cursor:%q sort:%q viewer:%q", store.lastCommentRootURI, store.lastCommentLimit, store.lastCommentCursor, store.lastCommentSort, store.lastCommentViewerDID)
	}
}

func TestGetPostComments_AttachesQuoteViewsToPostShapedResponses(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	rootQuoteURI := "at://did:plc:carol/social.craftsky.feed.post/root-target"
	rootQuoteCID := "bafyRootTarget"
	commentQuoteURI := "at://did:plc:dave/social.craftsky.feed.post/hidden-target"
	commentQuoteCID := "bafyHiddenTarget"
	root := testPostRow("did:plc:alice", "root", "root quote commentary", base)
	root.QuoteURI = &rootQuoteURI
	root.QuoteCID = &rootQuoteCID
	comment := testReplyRow("did:plc:bob", "comment1", "comment quote commentary", root.URI, root.URI, base.Add(time.Minute))
	comment.QuoteURI = &commentQuoteURI
	comment.QuoteCID = &commentQuoteCID
	quotedRoot := testPostRow("did:plc:carol", "root-target", "quoted root target", base.Add(-time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{comment},
		quoteViews: map[string]*api.QuoteViewRow{
			rootQuoteURI:    {State: "visible", Post: quotedRoot},
			commentQuoteURI: {State: "hidden"},
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(store.lastQuoteViewRefs) != 2 {
		t.Fatalf("quote view refs = %+v, want 2 refs", store.lastQuoteViewRefs)
	}
	if resp.Post.QuoteView == nil || resp.Post.QuoteView.State != "visible" || resp.Post.QuoteView.Post == nil {
		t.Fatalf("root quoteView = %+v, want visible preview", resp.Post.QuoteView)
	}
	if resp.Post.QuoteView.Post.Author.Handle != "carol.example" {
		t.Fatalf("root quote preview author = %+v", resp.Post.QuoteView.Post.Author)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Post.QuoteView == nil || resp.Comments.Items[0].Post.QuoteView.State != "hidden" {
		t.Fatalf("comment item = %+v, want hidden quoteView", resp.Comments.Items)
	}
}

func TestGetPostComments_CommentItemsIncludePlacement(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	store := &fakePostStore{one: root, commentRows: []*api.PostRow{comment}}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Comments.Items) != 1 {
		t.Fatalf("items = %+v", resp.Comments.Items)
	}
	if resp.Comments.Items[0].Placement != "normal" {
		t.Fatalf("placement = %q, want normal", resp.Comments.Items[0].Placement)
	}
}

func TestGetPostComments_CommentItemsAlwaysIncludeRepliesObject(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	store := &fakePostStore{one: root, commentRows: []*api.PostRow{comment}}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var raw struct {
		Comments struct {
			Items []struct {
				Replies *struct {
					Loaded bool            `json:"loaded"`
					Items  json.RawMessage `json:"items"`
					Cursor *string         `json:"cursor,omitempty"`
				} `json:"replies"`
			} `json:"items"`
		} `json:"comments"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw.Comments.Items) != 1 {
		t.Fatalf("items len = %d", len(raw.Comments.Items))
	}
	if raw.Comments.Items[0].Replies == nil {
		t.Fatal("replies object is missing")
	}
	if raw.Comments.Items[0].Replies.Loaded {
		t.Fatal("replies.loaded = true, want false before expansion")
	}
	if string(raw.Comments.Items[0].Replies.Items) != "[]" {
		t.Fatalf("replies.items = %s, want []", raw.Comments.Items[0].Replies.Items)
	}
	if raw.Comments.Items[0].Replies.Cursor != nil {
		t.Fatalf("replies.cursor should be omitted when not loaded, got %q", *raw.Comments.Items[0].Replies.Cursor)
	}
}

func TestGetPostComments_FocusQueryIdentifiesIncludedComment(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment1", "comment", root.URI, root.URI, base.Add(time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{comment},
		postByURI:   comment,
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(comment.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil {
		t.Fatal("focus metadata missing")
	}
	if resp.Focus.Status != "included" || resp.Focus.URI != comment.URI || resp.Focus.Kind != "comment" {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Placement != "focused" || resp.Comments.Items[0].Post.URI != comment.URI {
		t.Fatalf("focused comment item = %+v", resp.Comments.Items)
	}
}

func TestGetPostComments_FocusedCommentOutsidePageIsIncludedFirst(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	pageComment := testReplyRow("did:plc:bob", "page-comment", "page", root.URI, root.URI, base.Add(time.Minute))
	focusedComment := testReplyRow("did:plc:carol", "focused-comment", "focused", root.URI, root.URI, base.Add(20*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{pageComment},
		postByURI:   focusedComment,
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedComment.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "comment" {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if got := len(resp.Comments.Items); got != 2 {
		t.Fatalf("comments len = %d, want focused extra plus page item: %+v", got, resp.Comments.Items)
	}
	if resp.Comments.Items[0].Post.URI != focusedComment.URI || resp.Comments.Items[0].Placement != "focused" {
		t.Fatalf("first item = %+v", resp.Comments.Items[0])
	}
	if resp.Comments.Items[1].Post.URI != pageComment.URI || resp.Comments.Items[1].Placement != "normal" {
		t.Fatalf("second item = %+v", resp.Comments.Items[1])
	}
}

func TestGetPostComments_FocusedReplyExpandsCommentBranch(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	pageComment := testReplyRow("did:plc:bob", "page-comment", "page", root.URI, root.URI, base.Add(time.Minute))
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(20*time.Minute))
	firstReply := testReplyRow("did:plc:erin", "first-reply", "first", root.URI, comment.URI, base.Add(20*time.Minute+30*time.Second))
	focusedReply := testReplyRow("did:plc:dave", "focused-reply", "reply", root.URI, comment.URI, base.Add(21*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{pageComment},
		replyRows:   []*api.PostRow{firstReply, focusedReply},
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if got := len(resp.Comments.Items); got != 2 {
		t.Fatalf("comments len = %d, want focused branch plus page item: %+v", got, resp.Comments.Items)
	}
	focusedBranch := resp.Comments.Items[0]
	if focusedBranch.Post.URI != comment.URI || focusedBranch.Placement != "focused" {
		t.Fatalf("focused branch = %+v", focusedBranch)
	}
	if !focusedBranch.Replies.Loaded || len(focusedBranch.Replies.Items) != 2 {
		t.Fatalf("focused branch replies = %+v", focusedBranch.Replies)
	}
	if focusedBranch.Replies.Items[0].Post.URI != firstReply.URI || focusedBranch.Replies.Items[0].Flattened {
		t.Fatalf("first reply item = %+v", focusedBranch.Replies.Items[0])
	}
	if focusedBranch.Replies.Items[1].Post.URI != focusedReply.URI || focusedBranch.Replies.Items[1].Flattened {
		t.Fatalf("focused reply item = %+v", focusedBranch.Replies.Items[1])
	}
	if store.lastReplyParentURI != comment.URI || store.lastReplyRootURI != root.URI || store.lastReplyLimit != 10 || store.lastReplyCursor != "" {
		t.Fatalf("branch reply lookup = parent:%q root:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyRootURI, store.lastReplyLimit, store.lastReplyCursor)
	}
}

func TestGetPostComments_FocusedReplyBranchUsesBoundedPage(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	focusedReply := testReplyRow("did:plc:dave", "focused-reply", "reply", root.URI, comment.URI, base.Add(30*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		replyRows:   []*api.PostRow{focusedReply},
		replyCursor: "next-replies",
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Comments.Items) != 1 {
		t.Fatalf("comments = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies
	if !replies.Loaded {
		t.Fatalf("replies not loaded: %+v", replies)
	}
	if len(replies.Items) != 1 {
		t.Fatalf("focused reply branch len = %d, want bounded branch page", len(replies.Items))
	}
	if replies.Items[0].Post.URI != focusedReply.URI {
		t.Fatalf("focused reply item = %+v", replies.Items[0])
	}
	if replies.Cursor != "next-replies" {
		t.Fatalf("reply cursor = %q", replies.Cursor)
	}
	if store.lastReplyParentURI != comment.URI || store.lastReplyRootURI != root.URI || store.lastReplyLimit != 10 || store.lastReplyCursor != "" {
		t.Fatalf("branch reply lookup = parent:%q root:%q limit:%d cursor:%q", store.lastReplyParentURI, store.lastReplyRootURI, store.lastReplyLimit, store.lastReplyCursor)
	}
}

func TestGetPostComments_FocusedReplyAfterFirstBranchPageIsIncluded(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	firstPage := make([]*api.PostRow, 0, 10)
	for i := 0; i < 10; i++ {
		firstPage = append(firstPage, testReplyRow("did:plc:dave", "reply-before-"+strconv.Itoa(i), "before", root.URI, comment.URI, base.Add(time.Duration(i+2)*time.Minute)))
	}
	focusedReply := testReplyRow("did:plc:erin", "focused-reply", "target", root.URI, comment.URI, base.Add(20*time.Minute))
	store := &fakePostStore{
		one:               root,
		commentRows:       []*api.PostRow{},
		replyRows:         firstPage,
		replyCursor:       "first-page-cursor",
		aroundReplyRows:   append(firstPage[1:], focusedReply),
		aroundReplyCursor: "after-focus-cursor",
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
		"did:plc:erin":  "erin.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || !resp.Comments.Items[0].Replies.Loaded {
		t.Fatalf("focused branch = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies.Items
	if len(replies) > 10 {
		t.Fatalf("focused branch loaded %d replies, want bounded page", len(replies))
	}
	if !replyItemsContainURI(replies, focusedReply.URI) {
		t.Fatalf("focused reply missing from bounded branch page: %+v", replies)
	}
}

func TestGetPostComments_FocusStatusContract(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	otherRoot := testPostRow("did:plc:alice", "other", "other", base)
	mismatched := testReplyRow("did:plc:bob", "mismatched", "mismatch", otherRoot.URI, otherRoot.URI, base.Add(time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		postsByURI: map[string]*api.PostRow{
			mismatched.URI: mismatched,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, nilLogger())

	t.Run("malformed", func(t *testing.T) {
		req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus=not-an-at-uri", "", "did:plc:viewer")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var body envelope.Error
		if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
			t.Fatalf("decode error: %v", err)
		}
		if body.Error != "invalid_focus" {
			t.Fatalf("error = %q", body.Error)
		}
	})

	t.Run("not found", func(t *testing.T) {
		missingURI := "at://did:plc:missing/social.craftsky.feed.post/missing"
		req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(missingURI), "", "did:plc:viewer")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var resp api.CommentSectionResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Focus == nil || resp.Focus.URI != missingURI || resp.Focus.Status != "notFound" || resp.Focus.Kind != "" {
			t.Fatalf("focus = %+v", resp.Focus)
		}
	})

	t.Run("mismatched root", func(t *testing.T) {
		req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(mismatched.URI), "", "did:plc:viewer")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var resp api.CommentSectionResponse
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Focus == nil || resp.Focus.URI != mismatched.URI || resp.Focus.Status != "mismatchedRoot" || resp.Focus.Kind != "" {
			t.Fatalf("focus = %+v", resp.Focus)
		}
	})
}

func TestGetPostComments_InvalidCursorUsesStandardEnvelope(t *testing.T) {
	t.Parallel()
	root := testPostRow("did:plc:alice", "root", "root", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	store := &fakePostStore{one: root, commentErr: envelope.ErrInvalidCursor}
	h := api.GetPostCommentsHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?cursor=bad", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Error != "invalid_cursor" || body.Message == "" {
		t.Fatalf("error envelope = %+v", body)
	}
}

func TestGetPostComments_RejectsNonRootPost(t *testing.T) {
	t.Parallel()
	root := testPostRow("did:plc:alice", "root", "root", time.Now())
	comment := testReplyRow("did:plc:alice", "comment", "comment", root.URI, root.URI, time.Now())
	store := &fakePostStore{one: comment}
	h := api.GetPostCommentsHandler(store, fakeResolver{}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/comments", "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "invalid_post_role" {
		t.Fatalf("error = %q", body.Error)
	}
	if store.lastCommentRootURI != "" {
		t.Fatalf("ListRootComments should not run for non-root target")
	}
}

func TestGetPostComments_DeeperFocusedReplyIncludesFlattenedMetadata(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	parentReply := testReplyRow("did:plc:dave", "parent-reply", "parent", root.URI, comment.URI, base.Add(2*time.Minute))
	deeperReply := testReplyRow("did:plc:erin", "deeper-reply", "deep", root.URI, parentReply.URI, base.Add(3*time.Minute))
	displayName := "Dave"
	parentReply.AuthorDisplayName = &displayName
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		replyRows:   []*api.PostRow{parentReply, deeperReply},
		postsByURI: map[string]*api.PostRow{
			deeperReply.URI: deeperReply,
			parentReply.URI: parentReply,
			comment.URI:     comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
		"did:plc:erin":  "erin.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(deeperReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Post.URI != comment.URI {
		t.Fatalf("comments = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies.Items
	if len(replies) != 2 {
		t.Fatalf("replies = %+v", replies)
	}
	if replies[0].Post.URI != parentReply.URI || replies[0].Flattened {
		t.Fatalf("parent reply = %+v", replies[0])
	}
	got := replies[1]
	if got.Post.URI != deeperReply.URI || !got.Flattened {
		t.Fatalf("flattened reply = %+v", got)
	}
	if got.ReplyingTo == nil || got.ReplyingTo.URI != parentReply.URI || got.ReplyingTo.DID != parentReply.DID || got.ReplyingTo.Handle != "dave.example" {
		t.Fatalf("replyingTo = %+v", got.ReplyingTo)
	}
	if got.ReplyingTo.DisplayName == nil || *got.ReplyingTo.DisplayName != "Dave" {
		t.Fatalf("replyingTo displayName = %+v", got.ReplyingTo.DisplayName)
	}
}

func TestGetPostComments_DeepFocusedReplyResolvesCommentAncestor(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:carol", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	reply1 := testReplyRow("did:plc:dave", "reply-1", "one", root.URI, comment.URI, base.Add(2*time.Minute))
	reply2 := testReplyRow("did:plc:erin", "reply-2", "two", root.URI, reply1.URI, base.Add(3*time.Minute))
	focusedReply := testReplyRow("did:plc:frank", "reply-3", "three", root.URI, reply2.URI, base.Add(4*time.Minute))
	store := &fakePostStore{
		one:         root,
		commentRows: []*api.PostRow{},
		replyRows:   []*api.PostRow{reply1, reply2, focusedReply},
		postsByURI: map[string]*api.PostRow{
			focusedReply.URI: focusedReply,
			reply2.URI:       reply2,
			reply1.URI:       reply1,
			comment.URI:      comment,
		},
	}
	h := api.GetPostCommentsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:carol": "carol.example",
		"did:plc:dave":  "dave.example",
		"did:plc:erin":  "erin.example",
		"did:plc:frank": "frank.example",
	}}, nilLogger())
	req := authedPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/root/comments?focus="+url.QueryEscape(focusedReply.URI), "", "did:plc:viewer")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.CommentSectionResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Focus == nil || resp.Focus.Status != "included" || resp.Focus.Kind != "reply" || resp.Focus.CommentURI != comment.URI {
		t.Fatalf("focus = %+v", resp.Focus)
	}
	if len(resp.Comments.Items) != 1 || resp.Comments.Items[0].Post.URI != comment.URI || resp.Comments.Items[0].Placement != "focused" {
		t.Fatalf("comments = %+v", resp.Comments.Items)
	}
	replies := resp.Comments.Items[0].Replies.Items
	if len(replies) != 3 {
		t.Fatalf("replies = %+v", replies)
	}
	got := replies[2]
	if got.Post.URI != focusedReply.URI || !got.Flattened || got.ReplyingTo == nil || got.ReplyingTo.URI != reply2.URI {
		t.Fatalf("focused reply = %+v", got)
	}
}

func TestListCommentReplies_DefaultLimit(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{one: testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 10 {
		t.Fatalf("default limit = %d, want 10", store.lastReplyLimit)
	}
}

func TestListCommentReplies_LimitCapsAt10(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{one: testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now())}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?limit=500", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 10 {
		t.Fatalf("capped limit = %d, want 10", store.lastReplyLimit)
	}
}

func TestListCommentReplies_MissingTarget_404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{oneErr: api.ErrPostNotFound}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/missing/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "missing")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastReplyLimit != 0 {
		t.Fatalf("ListCommentBranchReplies should not run for missing target")
	}
}

func TestListCommentReplies_RejectsNonCommentTarget(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "root", "root", base)
	comment := testReplyRow("did:plc:bob", "comment", "comment", root.URI, root.URI, base.Add(time.Minute))
	reply := testReplyRow("did:plc:carol", "reply", "reply", root.URI, comment.URI, base.Add(2*time.Minute))

	tests := []struct {
		name string
		row  *api.PostRow
		path string
	}{
		{name: "root", row: root, path: "/v1/posts/did:plc:alice/root/replies"},
		{name: "nested reply", row: reply, path: "/v1/posts/did:plc:carol/reply/replies"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := &fakePostStore{one: tc.row}
			h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
			req := authedPostPathReq(http.MethodGet, tc.path, "", "did:plc:viewer")
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			var body envelope.Error
			_ = json.NewDecoder(rr.Body).Decode(&body)
			if body.Error != "invalid_post_role" {
				t.Fatalf("error = %q", body.Error)
			}
			if store.lastReplyParentURI != "" {
				t.Fatalf("ListCommentBranchReplies should not run for non-comment target")
			}
		})
	}
}

func TestListCommentReplies_BadDID_400(t *testing.T) {
	t.Parallel()
	h := api.ListCommentRepliesHandler(&fakePostStore{}, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/not-a-did/root/replies", "", "did:plc:viewer")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "root")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestListCommentReplies_InvalidCursor_400(t *testing.T) {
	t.Parallel()
	rootURI := "at://did:plc:alice/social.craftsky.feed.post/root"
	store := &fakePostStore{
		one:      testReplyRow("did:plc:alice", "comment", "comment", rootURI, rootURI, time.Now()),
		replyErr: envelope.ErrInvalidCursor,
	}
	h := api.ListCommentRepliesHandler(store, fakeResolver{}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/posts/did:plc:alice/comment/replies?cursor=bad", "", "did:plc:viewer")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "comment")
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
	}}, nilLogger())
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
	h := api.ListPostsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
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
	h := api.ListProjectsByAuthorHandler(store, fakeResolver{handleFor: "alice.example"}, nilLogger())
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
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
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

func TestCreatePost_WithImages_WritesTopLevelImagesToPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"image post",
		"images":[
			{
				"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496},
				"alt":"project photo",
				"aspectRatio":{"width":919,"height":2000}
			}
		]
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	recJSON, _ := json.Marshal(rec)
	var normalized map[string]any
	if err := json.Unmarshal(recJSON, &normalized); err != nil {
		t.Fatalf("decode rec: %v", err)
	}
	imagesRaw, ok := normalized["images"].([]any)
	if !ok || len(imagesRaw) != 1 {
		t.Fatalf("images = %T %v, want single image array", normalized["images"], normalized["images"])
	}
	img, _ := imagesRaw[0].(map[string]any)
	if img["alt"] != "project photo" {
		t.Fatalf("alt = %v", img["alt"])
	}
	if _, ok := img["image"].(map[string]any); !ok {
		t.Fatalf("image blob missing: %+v", img)
	}
	aspect, _ := img["aspectRatio"].(map[string]any)
	if aspect["width"] != float64(919) || aspect["height"] != float64(2000) {
		t.Fatalf("aspectRatio = %+v", aspect)
	}
	if _, hasEmbed := normalized["embed"]; hasEmbed {
		t.Fatalf("embed must be absent for plain image post; got %+v", normalized["embed"])
	}

	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Images) != 1 {
		t.Fatalf("response images = %d, want 1", len(resp.Images))
	}
	view := resp.Images[0]
	if view.CID != "bafkimage" || view.MIME != "image/jpeg" || view.Size != 253496 || view.Alt != "project photo" {
		t.Fatalf("response image = %+v", view)
	}
	if view.AspectRatio == nil || view.AspectRatio.Width != 919 || view.AspectRatio.Height != 2000 {
		t.Fatalf("response aspectRatio = %+v", view.AspectRatio)
	}
	if view.Thumb == "" || view.Fullsize == "" {
		t.Fatalf("response image urls missing: %+v", view)
	}
}

func TestCreatePost_WithImageMissingAlt_OmitsAltInPDSRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"image post",
		"images":[
			{
				"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496}
			}
		]
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	imagesRaw, ok := rec["images"].([]map[string]any)
	if !ok || len(imagesRaw) != 1 {
		t.Fatalf("images = %T %v, want single image array", rec["images"], rec["images"])
	}
	if _, ok := imagesRaw[0]["alt"]; ok {
		t.Fatalf("alt should be absent when not provided: %+v", imagesRaw[0])
	}
}

func TestCreatePost_WithMoreThanFourImages_422WithoutPDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"too many","images":[
		{"image":{"$type":"blob","ref":{"$link":"b1"},"mimeType":"image/jpeg","size":1},"alt":"1"},
		{"image":{"$type":"blob","ref":{"$link":"b2"},"mimeType":"image/jpeg","size":1},"alt":"2"},
		{"image":{"$type":"blob","ref":{"$link":"b3"},"mimeType":"image/jpeg","size":1},"alt":"3"},
		{"image":{"$type":"blob","ref":{"$link":"b4"},"mimeType":"image/jpeg","size":1},"alt":"4"},
		{"image":{"$type":"blob","ref":{"$link":"b5"},"mimeType":"image/jpeg","size":1},"alt":"5"}
	]}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", pds.createCalls)
	}
}

func TestCreatePost_WithInvalidImageBlobMetadata_422WithoutPDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"bad image","images":[{"image":{"foo":"bar"},"alt":"ok"}]}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("createCalls = %d, want 0", pds.createCalls)
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

func facetArrayLength(t *testing.T, raw json.RawMessage) int {
	t.Helper()
	var facets []map[string]any
	if err := json.Unmarshal(raw, &facets); err != nil {
		t.Fatalf("unmarshal facets %s: %v", raw, err)
	}
	return len(facets)
}
