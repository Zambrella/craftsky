package api_test

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/video"
	"sync"
	"time"
)

// fakePostEffectState records durable effect calls. Zero-value methods
// succeed; populate errors to simulate failures.
type fakePostEffectState struct {
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
}

type fakeVideoCompletionVerifier struct {
	verified video.Blob
	err      error
	calls    int
	owner    syntax.DID
	jobID    string
	blob     video.Blob
}

func (f *fakeVideoCompletionVerifier) Verify(_ context.Context, owner syntax.DID, jobID string, blob video.Blob) (video.Blob, error) {
	f.calls++
	f.owner = owner
	f.jobID = jobID
	f.blob = blob
	if f.err != nil {
		return video.Blob{}, f.err
	}
	if f.verified.CID != "" {
		return f.verified, nil
	}
	return blob, nil
}

// fakePostEffects preserves the handler-level assertions in this file while
// exercising only the durable EffectExecutor capability accepted by the post
// handlers. It is intentionally not a production compatibility adapter.
type fakePostEffects struct {
	pds   *fakePostEffectState
	owner syntax.DID
}

func (effects *fakePostEffects) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	owners := []ownerlifecycle.ExpectedOwner{{Owner: effects.owner, Generation: ownerGeneration}}
	for _, target := range targets {
		if target != owners[0].Owner {
			owners = append(owners, ownerlifecycle.ExpectedOwner{Owner: target, Generation: 1})
		}
	}
	return owners, nil
}

func (*fakePostEffects) ReadRecord(context.Context, pdseffects.ReadRecordRequest, any) (syntax.CID, error) {
	panic("unexpected ReadRecord call")
}

func (effects *fakePostEffects) PutRecord(
	_ context.Context,
	request pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	pds := effects.pds
	pds.mu.Lock()
	defer pds.mu.Unlock()
	pds.lastCreateRepo = request.Owner
	pds.lastCreateColl = request.Collection.String()
	pds.lastCreateRec = request.Record
	pds.createCalls++
	if pds.createErr != nil {
		return pdseffects.RecordResult{}, pds.createErr
	}
	if pds.createURI == "" {
		pds.createURI = syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/rkSrv")
		pds.createCID = syntax.CID("bafySrv")
	}
	return pdseffects.RecordResult{URI: pds.createURI, CID: pds.createCID}, nil
}

func (effects *fakePostEffects) DeleteRecord(
	_ context.Context,
	request pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	pds := effects.pds
	pds.mu.Lock()
	defer pds.mu.Unlock()
	pds.lastDeleteRepo = request.Owner
	pds.lastDeleteColl = request.Collection.String()
	pds.lastDeleteRkey = request.Rkey.String()
	pds.deleteCalls++
	if errors.Is(pds.deleteErr, auth.ErrRecordNotFound) {
		return pdseffects.RecordResult{}, nil
	}
	return pdseffects.RecordResult{}, pds.deleteErr
}

func (*fakePostEffects) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	panic("unexpected UploadBlob call")
}

func newPDSEffectsFactory(pds *fakePostEffectState) pdseffects.ExecutorFactory {
	return func(_ context.Context, owner syntax.DID, _ string) (pdseffects.EffectExecutor, error) {
		return &fakePostEffects{pds: pds, owner: owner}, nil
	}
}

func failingPDSEffectsFactory(err error) pdseffects.ExecutorFactory {
	return func(_ context.Context, _ syntax.DID, _ string) (pdseffects.EffectExecutor, error) {
		return nil, err
	}
}

// fakePostStore implements the post handler capability interfaces used here.
type fakePostStore struct {
	authorizationErr        error
	authorizationCalls      []authorizationCall
	relationshipStates      map[syntax.DID]relationships.State
	relationshipStateErr    error
	blockedPairs            map[api.RelationshipPair]bool
	one                     *api.PostRow
	oneErr                  error
	listRows                []*api.PostRow
	listCursor              string
	listErr                 error
	projectListRows         []*api.PostRow
	projectListCursor       string
	projectListErr          error
	commentListRows         []*api.PostRow
	commentListCursor       string
	commentListErr          error
	commentRows             []*api.PostRow
	commentCursor           string
	commentErr              error
	postByURI               *api.PostRow
	postsByURI              map[string]*api.PostRow
	postByURIErr            error
	replyRows               []*api.PostRow
	replyCursor             string
	replyErr                error
	aroundReplyRows         []*api.PostRow
	aroundReplyCursor       string
	aroundReplyErr          error
	author                  *api.PostAuthorRow
	authorErr               error
	engagement              map[string]api.EngagementSummary
	engagementErr           error
	quoteViews              map[string]*api.QuoteViewRow
	quoteViewsErr           error
	target                  *api.PostTargetRef
	targetErr               error
	shareTarget             *api.ShareTargetRef
	shareTargetErr          error
	activeLike              *api.InteractionRow
	activeLikeErr           error
	activeRepost            *api.InteractionRow
	activeRepostErr         error
	lastDID                 string
	lastRkey                string
	lastViewerDID           string
	lastListCommentsDID     string
	lastListCommentsLimit   int
	lastListCommentsCursor  string
	lastListProjectsDID     string
	lastListProjectsLimit   int
	lastListProjectsCursor  string
	lastEngagementViewer    string
	lastEngagementURIs      []string
	lastEngagementLanguages []string
	lastQuoteViewRefs       []api.ResponseStrongRef
	engagementCalls         int
	lastTargetDID           string
	lastTargetRkey          string
	lastShareTargetDID      string
	lastShareTargetRkey     string
	lastActiveLikeDID       string
	lastActiveLikeURI       string
	lastActiveRepostDID     string
	lastActiveRepostURI     string
	lastCommentRootURI      string
	lastCommentViewerDID    string
	lastCommentSort         string
	lastCommentLimit        int
	lastCommentCursor       string
	lastReplyParentURI      string
	lastReplyRootURI        string
	lastReplyLimit          int
	lastReplyCursor         string
	lastReplyFocusURI       string
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

func (f *fakePostStore) ReadOneForViewer(_ context.Context, did, rkey, viewerDID string) (*api.PostRow, error) {
	f.lastDID = did
	f.lastRkey = rkey
	f.lastViewerDID = viewerDID
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

func (f *fakePostStore) EngagementSummaries(_ context.Context, viewerDID string, contentLanguages []string, postURIs []string) (map[string]api.EngagementSummary, error) {
	f.engagementCalls++
	f.lastEngagementViewer = viewerDID
	f.lastEngagementLanguages = append([]string(nil), contentLanguages...)
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
