package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

type recordingPostEffectExecutor struct {
	resolvedGeneration int64
	resolvedTargets    []syntax.DID
	expectedOwners     []ownerlifecycle.ExpectedOwner
	putRequests        []pdseffects.PutRecordRequest
	deleteRequests     []pdseffects.DeleteRecordRequest
	err                error
}

func (executor *recordingPostEffectExecutor) ResolveExpectedOwners(
	_ context.Context,
	generation int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	executor.resolvedGeneration = generation
	executor.resolvedTargets = append([]syntax.DID(nil), targets...)
	if executor.err != nil {
		return nil, executor.err
	}
	return append([]ownerlifecycle.ExpectedOwner(nil), executor.expectedOwners...), nil
}

func (*recordingPostEffectExecutor) ReadRecord(
	context.Context,
	pdseffects.ReadRecordRequest,
	any,
) (syntax.CID, error) {
	panic("unexpected ReadRecord call")
}

func (executor *recordingPostEffectExecutor) PutRecord(
	_ context.Context,
	request pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	executor.putRequests = append(executor.putRequests, request)
	if executor.err != nil {
		return pdseffects.RecordResult{}, executor.err
	}
	return pdseffects.RecordResult{
		URI: syntax.ATURI("at://" + request.Owner.String() + "/" + request.Collection.String() + "/" + request.Rkey.String()),
		CID: syntax.CID("bafy-effect-result"),
	}, nil
}

func (executor *recordingPostEffectExecutor) DeleteRecord(
	_ context.Context,
	request pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	executor.deleteRequests = append(executor.deleteRequests, request)
	if executor.err != nil {
		return pdseffects.RecordResult{}, executor.err
	}
	return pdseffects.RecordResult{}, nil
}

func (*recordingPostEffectExecutor) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	panic("unexpected UploadBlob call")
}

func TestCreatePostUsesPreallocatedDurableEffectIdentityAndOwnerGeneration(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	executor := &recordingPostEffectExecutor{expectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 7}}}
	factoryCalls := 0
	factory := func(_ context.Context, gotOwner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
		factoryCalls++
		if gotOwner != owner || sessionID != "session-alice" {
			t.Fatalf("factory owner/session = %q/%q", gotOwner, sessionID)
		}
		return executor, nil
	}
	handler := api.CreatePostHandler(&fakePostStore{}, factory, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello"}`))
	ctx := middleware.WithDID(request.Context(), owner)
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	ctx = ctxkeys.WithRunID(ctx, "request-123")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if factoryCalls != 1 || len(executor.putRequests) != 1 {
		t.Fatalf("factory/put calls = %d/%d, want 1/1", factoryCalls, len(executor.putRequests))
	}
	put := executor.putRequests[0]
	if put.OwnerGeneration != 7 || len(put.ExpectedOwners) != 1 || put.ExpectedOwners[0].Generation != 7 {
		t.Fatalf("owner generation/fence = %d/%+v", put.OwnerGeneration, put.ExpectedOwners)
	}
	if put.Collection != syntax.NSID("social.craftsky.feed.post") || put.Rkey == "" {
		t.Fatalf("collection/rkey = %q/%q", put.Collection, put.Rkey)
	}
	if _, err := syntax.ParseRecordKey(put.Rkey.String()); err != nil {
		t.Fatalf("preallocated rkey is invalid: %v", err)
	}
	if put.OperationID == "" || put.OperationID != put.MutationKey || !strings.Contains(put.OperationID, "request-123") {
		t.Fatalf("operation/mutation IDs = %q/%q", put.OperationID, put.MutationKey)
	}
	var body api.PostResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Rkey != put.Rkey.String() {
		t.Fatalf("response rkey = %q, durable rkey = %q", body.Rkey, put.Rkey)
	}
}

func TestCreatePostFallbackEffectIdentityIsUniquePerRequest(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	executor := &recordingPostEffectExecutor{expectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}}}
	handler := api.CreatePostHandler(&fakePostStore{}, func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return executor, nil
	}, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	for range 2 {
		request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello"}`))
		ctx := middleware.WithDID(request.Context(), owner)
		ctx = middleware.WithOwnerGeneration(ctx, 2)
		request = request.WithContext(ctx)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	}
	if len(executor.putRequests) != 2 {
		t.Fatalf("put requests = %d, want 2", len(executor.putRequests))
	}
	if executor.putRequests[0].OperationID == executor.putRequests[1].OperationID {
		t.Fatalf("fallback operation ID was reused: %q", executor.putRequests[0].OperationID)
	}
	if executor.putRequests[0].Rkey == executor.putRequests[1].Rkey {
		t.Fatalf("preallocated record key was reused: %q", executor.putRequests[0].Rkey)
	}
}

func TestLikePostCarriesDirectedTargetFence(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	target := syntax.DID("did:plc:bob")
	executor := &recordingPostEffectExecutor{expectedOwners: []ownerlifecycle.ExpectedOwner{
		{Owner: owner, Generation: 7},
		{Owner: target, Generation: 4},
	}}
	store := &fakePostStore{target: &api.PostTargetRef{
		URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost",
	}}
	handler := api.LikePostHandler(store, func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return executor, nil
	}, nilLogger())
	request := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", owner.String())
	ctx := middleware.WithOwnerGeneration(request.Context(), 7)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if executor.resolvedGeneration != 7 || len(executor.resolvedTargets) != 1 || executor.resolvedTargets[0] != target {
		t.Fatalf("resolved generation/targets = %d/%v", executor.resolvedGeneration, executor.resolvedTargets)
	}
	if len(executor.putRequests) != 1 || len(executor.putRequests[0].ExpectedOwners) != 2 {
		t.Fatalf("put expected owners = %+v", executor.putRequests)
	}
}

func TestCreatePostCarriesDirectedOwnersIntoDurableFenceResolution(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	tests := []struct {
		name        string
		body        string
		store       *fakePostStore
		wantTargets []syntax.DID
	}{
		{
			name: "quote",
			body: `{"text":"quote","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/post1","cid":"bafyPost"}}}`,
			store: &fakePostStore{shareTarget: &api.ShareTargetRef{
				URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost",
			}},
			wantTargets: []syntax.DID{"did:plc:bob"},
		},
		{
			name:        "reply parent and root",
			body:        `{"text":"reply","reply":{"root":{"uri":"at://did:plc:carol/social.craftsky.feed.post/root","cid":"bafyRoot"},"parent":{"uri":"at://did:plc:bob/social.craftsky.feed.post/parent","cid":"bafyParent"}}}`,
			store:       &fakePostStore{},
			wantTargets: []syntax.DID{"did:plc:bob", "did:plc:carol"},
		},
		{
			name:        "mention",
			body:        `{"text":"@bob.example","facets":[{"index":{"byteStart":0,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:bob"}]}]}`,
			store:       &fakePostStore{},
			wantTargets: []syntax.DID{"did:plc:bob"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedOwners := []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 9}}
			for index, target := range test.wantTargets {
				expectedOwners = append(expectedOwners, ownerlifecycle.ExpectedOwner{
					Owner: target, Generation: int64(index + 1),
				})
			}
			executor := &recordingPostEffectExecutor{expectedOwners: expectedOwners}
			handler := api.CreatePostHandler(test.store, func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				return executor, nil
			}, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(test.body))
			ctx := middleware.WithDID(request.Context(), owner)
			ctx = middleware.WithOwnerGeneration(ctx, 9)
			request = request.WithContext(ctx)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if len(executor.resolvedTargets) != len(test.wantTargets) {
				t.Fatalf("resolved targets = %v, want %v", executor.resolvedTargets, test.wantTargets)
			}
			for index, want := range test.wantTargets {
				if executor.resolvedTargets[index] != want {
					t.Fatalf("resolved targets = %v, want %v", executor.resolvedTargets, test.wantTargets)
				}
			}
			if len(executor.putRequests) != 1 || len(executor.putRequests[0].ExpectedOwners) != len(expectedOwners) {
				t.Fatalf("put owner fence = %+v, want %+v", executor.putRequests, expectedOwners)
			}
			for index, want := range expectedOwners {
				if executor.putRequests[0].ExpectedOwners[index] != want {
					t.Fatalf("put owner fence = %+v, want %+v", executor.putRequests[0].ExpectedOwners, expectedOwners)
				}
			}
		})
	}
}

func TestUnlikePostCarriesIndexedCIDAsDeletePrecondition(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	executor := &recordingPostEffectExecutor{expectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 3}}}
	store := &fakePostStore{
		target:     &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{Rkey: "like1", CID: "bafyLike", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1"},
	}
	handler := api.UnlikePostHandler(store, func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return executor, nil
	}, nilLogger())
	request := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", owner.String())
	request = request.WithContext(middleware.WithOwnerGeneration(request.Context(), 3))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(executor.deleteRequests) != 1 || executor.deleteRequests[0].ExpectedCID != syntax.CID("bafyLike") {
		t.Fatalf("delete requests = %+v", executor.deleteRequests)
	}
}

func TestUnrepostPostCarriesIndexedCIDAsDeletePrecondition(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	executor := &recordingPostEffectExecutor{expectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 3}}}
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", CID: "bafyRepost", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1"},
	}
	handler := api.UnrepostPostHandler(store, func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return executor, nil
	}, nilLogger())
	request := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", owner.String())
	request = request.WithContext(middleware.WithOwnerGeneration(request.Context(), 3))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if len(executor.deleteRequests) != 1 || executor.deleteRequests[0].ExpectedCID != syntax.CID("bafyRepost") {
		t.Fatalf("delete requests = %+v", executor.deleteRequests)
	}
}

func TestPostPDSMutationHandlersMissingOwnerGenerationFailBeforeEffectFactory(t *testing.T) {
	owner := syntax.DID("did:plc:alice")
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike:   &api.InteractionRow{Rkey: "like1", CID: "bafyLike", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", CID: "bafyRepost", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1"},
	}
	tests := []struct {
		name    string
		handler func(pdseffects.ExecutorFactory) http.Handler
		request func() *http.Request
	}{
		{
			name: "create post",
			handler: func(factory pdseffects.ExecutorFactory) http.Handler {
				return api.CreatePostHandler(store, factory, fakeResolver{}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello"}`))
			},
		},
		{
			name: "delete post",
			handler: func(factory pdseffects.ExecutorFactory) http.Handler {
				return api.DeletePostHandler(factory, nilLogger())
			},
			request: func() *http.Request {
				request := httptest.NewRequest(http.MethodDelete, "/v1/posts/did:plc:alice/post1", nil)
				request.SetPathValue("did", owner.String())
				request.SetPathValue("rkey", "post1")
				return request
			},
		},
		{
			name: "like",
			handler: func(factory pdseffects.ExecutorFactory) http.Handler {
				return api.LikePostHandler(store, factory, nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", owner.String())
			},
		},
		{
			name: "unlike",
			handler: func(factory pdseffects.ExecutorFactory) http.Handler {
				return api.UnlikePostHandler(store, factory, nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", owner.String())
			},
		},
		{
			name: "repost",
			handler: func(factory pdseffects.ExecutorFactory) http.Handler {
				return api.RepostPostHandler(store, factory, nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", owner.String())
			},
		},
		{
			name: "unrepost",
			handler: func(factory pdseffects.ExecutorFactory) http.Handler {
				return api.UnrepostPostHandler(store, factory, nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", owner.String())
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			factoryCalls := 0
			factory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
				factoryCalls++
				return nil, errors.New("must not be called")
			}
			request := test.request()
			request = request.WithContext(middleware.WithDID(request.Context(), owner))
			// authedPostPathReq normally installs a generation; deliberately
			// rebuild the context without carrying it into the handler.
			request = request.WithContext(middleware.WithDID(context.Background(), owner))
			response := httptest.NewRecorder()

			test.handler(factory).ServeHTTP(response, request)

			if response.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if factoryCalls != 0 {
				t.Fatalf("factory calls = %d, want 0", factoryCalls)
			}
		})
	}
}
