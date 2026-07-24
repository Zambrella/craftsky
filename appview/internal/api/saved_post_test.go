package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type fakeSavedPostTargetResolver struct {
	uri syntax.ATURI
	err error
}

func (f *fakeSavedPostTargetResolver) ResolveSavedPostTarget(context.Context, syntax.DID, syntax.DID, syntax.RecordKey) (syntax.ATURI, error) {
	return f.uri, f.err
}

type fakeSavedPostStore struct {
	saveResult api.SaveMutationResult
	saveErr    error
	savedOwner syntax.DID
	savedURI   syntax.ATURI
	assignment api.FolderAssignment

	unsavedOwner syntax.DID
	unsavedURI   syntax.ATURI
	unsaveErr    error

	createdName string
	folder      api.SavedPostFolder
	folderErr   error
	renamedID   string
	deletedID   string
	deletedMode api.SavedPostFolderDeleteMode

	listedOwner  syntax.DID
	listedLimit  int
	listedCursor string
	folders      []api.SavedPostFolder
	nextCursor   string
}

func (f *fakeSavedPostStore) Save(_ context.Context, owner syntax.DID, uri syntax.ATURI, assignment api.FolderAssignment) (api.SaveMutationResult, error) {
	f.savedOwner, f.savedURI, f.assignment = owner, uri, assignment
	return f.saveResult, f.saveErr
}

func (f *fakeSavedPostStore) Unsave(_ context.Context, owner syntax.DID, uri syntax.ATURI) error {
	f.unsavedOwner, f.unsavedURI = owner, uri
	return f.unsaveErr
}

func (f *fakeSavedPostStore) CreateFolder(_ context.Context, _ syntax.DID, name string) (api.SavedPostFolder, error) {
	f.createdName = name
	return f.folder, f.folderErr
}

func (f *fakeSavedPostStore) RenameFolder(_ context.Context, _ syntax.DID, id, name string) (api.SavedPostFolder, error) {
	f.renamedID, f.createdName = id, name
	return f.folder, f.folderErr
}

func (f *fakeSavedPostStore) DeleteFolder(_ context.Context, _ syntax.DID, id string, mode api.SavedPostFolderDeleteMode) error {
	f.deletedID, f.deletedMode = id, mode
	return f.folderErr
}

func (f *fakeSavedPostStore) ListFolders(_ context.Context, owner syntax.DID, limit int, cursor string) ([]api.SavedPostFolder, string, error) {
	f.listedOwner, f.listedLimit, f.listedCursor = owner, limit, cursor
	return f.folders, f.nextCursor, f.folderErr
}

type fakeSavedPostListService struct {
	owner  syntax.DID
	filter api.SavedPostListFilter
	page   api.SavedPostPage
	err    error
}

type fakeSavedPostRefStore struct {
	refs   []api.SavedPostRef
	cursor string
	err    error
}

func (f *fakeSavedPostRefStore) ListSavedRefs(context.Context, syntax.DID, api.SavedPostListFilter) ([]api.SavedPostRef, string, error) {
	return f.refs, f.cursor, f.err
}

type fakeSavedPostHydrator struct {
	rows      map[syntax.ATURI]*api.PostRow
	summaries map[string]api.EngagementSummary
	quoteRows map[string]*api.QuoteViewRow
	states    map[syntax.DID]relationships.State
	contexts  map[syntax.ATURI]bool
}

func (f *fakeSavedPostHydrator) ReadEligiblePostsByURI(context.Context, syntax.DID, []syntax.ATURI) (map[syntax.ATURI]*api.PostRow, error) {
	return f.rows, nil
}

func (f *fakeSavedPostHydrator) EngagementSummaries(context.Context, string, []string) (map[string]api.EngagementSummary, error) {
	return f.summaries, nil
}

func (f *fakeSavedPostHydrator) QuoteViewRows(context.Context, []api.ResponseStrongRef) (map[string]*api.QuoteViewRow, error) {
	return f.quoteRows, nil
}

func (f *fakeSavedPostHydrator) RelationshipStates(_ context.Context, _ syntax.DID, subjects []syntax.DID) (map[syntax.DID]relationships.State, error) {
	out := make(map[syntax.DID]relationships.State, len(subjects))
	for _, subject := range subjects {
		out[subject] = f.states[subject]
	}
	return out, nil
}

func (f *fakeSavedPostHydrator) RequiredContextStates(_ context.Context, _ syntax.DID, uris []syntax.ATURI) (map[syntax.ATURI]bool, error) {
	out := make(map[syntax.ATURI]bool, len(uris))
	for _, uri := range uris {
		valid, specified := f.contexts[uri]
		out[uri] = valid || !specified
	}
	return out, nil
}

type fakeSavedPostHandleResolver struct{}

func (fakeSavedPostHandleResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	return syntax.ParseHandle(strings.TrimPrefix(did.String(), "did:plc:") + ".test")
}

func (fakeSavedPostHandleResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "", errors.New("not used")
}

func (f *fakeSavedPostListService) ListSavedPosts(_ context.Context, owner syntax.DID, filter api.SavedPostListFilter) (api.SavedPostPage, error) {
	f.owner, f.filter = owner, filter
	return f.page, f.err
}

func TestSavedPostHandlersExposeMutationAndListContracts(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	folderID := "opaque-folder"
	postURI := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/one")
	t.Run("save created", func(t *testing.T) {
		store := &fakeSavedPostStore{saveResult: api.SaveMutationResult{Created: true, State: api.SavedPostState{SavedAt: now, FolderID: &folderID}}}
		targets := &fakeSavedPostTargetResolver{uri: postURI}
		req := savedPostRequest(http.MethodPost, "/v1/posts/did:plc:bob/one/saves", `{"folderId":"opaque-folder"}`)
		req.SetPathValue("did", "did:plc:bob")
		req.SetPathValue("rkey", "one")
		resp := httptest.NewRecorder()
		api.SavePostHandler(targets, store).ServeHTTP(resp, req)
		if resp.Code != http.StatusCreated {
			t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
		}
		if store.savedOwner != "did:plc:alice" || store.savedURI != postURI || !store.assignment.Present || store.assignment.ID == nil || *store.assignment.ID != folderID {
			t.Fatalf("save call = %s/%s/%+v", store.savedOwner, store.savedURI, store.assignment)
		}
		var state api.SavedPostState
		if err := json.Unmarshal(resp.Body.Bytes(), &state); err != nil || !state.SavedAt.Equal(now) || state.FolderID == nil || *state.FolderID != folderID {
			t.Fatalf("response state = %+v err=%v", state, err)
		}
	})

	t.Run("save existing", func(t *testing.T) {
		store := &fakeSavedPostStore{saveResult: api.SaveMutationResult{State: api.SavedPostState{SavedAt: now}}}
		req := savedPostRequest(http.MethodPost, "/v1/posts/did:plc:bob/one/saves", "")
		req.SetPathValue("did", "did:plc:bob")
		req.SetPathValue("rkey", "one")
		resp := httptest.NewRecorder()
		api.SavePostHandler(&fakeSavedPostTargetResolver{uri: postURI}, store).ServeHTTP(resp, req)
		if resp.Code != http.StatusOK || store.assignment.Present {
			t.Fatalf("status/assignment = %d/%+v body=%s", resp.Code, store.assignment, resp.Body.String())
		}
	})

	t.Run("unsave constructs canonical uri without target lookup", func(t *testing.T) {
		store := &fakeSavedPostStore{}
		req := savedPostRequest(http.MethodDelete, "/v1/posts/did:plc:bob/one/saves", "")
		req.SetPathValue("did", "did:plc:bob")
		req.SetPathValue("rkey", "one")
		resp := httptest.NewRecorder()
		api.UnsavePostHandler(store).ServeHTTP(resp, req)
		if resp.Code != http.StatusNoContent || store.unsavedOwner != "did:plc:alice" || store.unsavedURI != postURI {
			t.Fatalf("unsave = %d/%s/%s", resp.Code, store.unsavedOwner, store.unsavedURI)
		}
	})

	t.Run("folder create trims", func(t *testing.T) {
		store := &fakeSavedPostStore{folder: api.SavedPostFolder{ID: folderID, Name: "Ideas", CreatedAt: now, UpdatedAt: now}}
		resp := httptest.NewRecorder()
		api.CreateSavedPostFolderHandler(store).ServeHTTP(resp, savedPostRequest(http.MethodPost, "/v1/saved-post-folders", `{"name":" Ideas "}`))
		if resp.Code != http.StatusCreated || store.createdName != "Ideas" {
			t.Fatalf("create = %d/%q body=%s", resp.Code, store.createdName, resp.Body.String())
		}
	})

	t.Run("folder rename not found", func(t *testing.T) {
		store := &fakeSavedPostStore{folderErr: api.ErrSavedPostFolderNotFound}
		req := savedPostRequest(http.MethodPatch, "/v1/saved-post-folders/missing", `{"name":"Ideas"}`)
		req.SetPathValue("folderId", "missing")
		resp := httptest.NewRecorder()
		api.RenameSavedPostFolderHandler(store).ServeHTTP(resp, req)
		assertSavedPostError(t, resp, http.StatusNotFound, "saved_post_folder_not_found")
	})

	t.Run("folder delete selects strict modes", func(t *testing.T) {
		for _, tt := range []struct {
			name       string
			query      string
			wantStatus int
			wantMode   api.SavedPostFolderDeleteMode
		}{
			{name: "absent", wantStatus: http.StatusNoContent, wantMode: api.SavedPostFolderPreserveSaves},
			{name: "false", query: "?deleteSaves=false", wantStatus: http.StatusNoContent, wantMode: api.SavedPostFolderPreserveSaves},
			{name: "true", query: "?deleteSaves=true", wantStatus: http.StatusNoContent, wantMode: api.SavedPostFolderRemoveSaves},
			{name: "invalid", query: "?deleteSaves=yes", wantStatus: http.StatusUnprocessableEntity},
			{name: "unknown", query: "?other=true", wantStatus: http.StatusUnprocessableEntity},
		} {
			t.Run(tt.name, func(t *testing.T) {
				store := &fakeSavedPostStore{}
				req := savedPostRequest(http.MethodDelete, "/v1/saved-post-folders/missing"+tt.query, "")
				req.SetPathValue("folderId", "missing")
				resp := httptest.NewRecorder()
				api.DeleteSavedPostFolderHandler(store).ServeHTTP(resp, req)
				if resp.Code != tt.wantStatus {
					t.Fatalf("status = %d, want %d body=%s", resp.Code, tt.wantStatus, resp.Body.String())
				}
				if tt.wantStatus == http.StatusNoContent && (store.deletedID != "missing" || store.deletedMode != tt.wantMode) {
					t.Fatalf("delete = %q/%v, want missing/%v", store.deletedID, store.deletedMode, tt.wantMode)
				}
				if tt.wantStatus == http.StatusUnprocessableEntity && store.deletedID != "" {
					t.Fatalf("invalid query reached store: %q", store.deletedID)
				}
			})
		}
	})

	t.Run("folder list defaults and cursor", func(t *testing.T) {
		store := &fakeSavedPostStore{folders: []api.SavedPostFolder{{ID: folderID, Name: "Ideas", CreatedAt: now, UpdatedAt: now}}, nextCursor: "next"}
		resp := httptest.NewRecorder()
		api.ListSavedPostFoldersHandler(store).ServeHTTP(resp, savedPostRequest(http.MethodGet, "/v1/saved-post-folders?cursor=prior", ""))
		if resp.Code != http.StatusOK || store.listedLimit != 50 || store.listedCursor != "prior" || store.listedOwner != "did:plc:alice" {
			t.Fatalf("list call = %d/%d/%q/%s body=%s", resp.Code, store.listedLimit, store.listedCursor, store.listedOwner, resp.Body.String())
		}
		var page struct {
			Items  []api.SavedPostFolder `json:"items"`
			Cursor string                `json:"cursor"`
		}
		if err := json.Unmarshal(resp.Body.Bytes(), &page); err != nil || len(page.Items) != 1 || page.Cursor != "next" {
			t.Fatalf("page = %+v err=%v", page, err)
		}
	})

	t.Run("saved list filter", func(t *testing.T) {
		service := &fakeSavedPostListService{page: api.SavedPostPage{Items: []api.SavedPostItem{}, Cursor: "next"}}
		resp := httptest.NewRecorder()
		api.ListSavedPostsHandler(service).ServeHTTP(resp, savedPostRequest(http.MethodGet, "/v1/saved-posts?folderId=opaque-folder&sort=oldest&limit=25&cursor=prior", ""))
		if resp.Code != http.StatusOK || service.owner != "did:plc:alice" || service.filter.Scope != api.SavedPostScopeFolder || service.filter.FolderID != folderID || service.filter.Sort != api.SavedPostSortOldest || service.filter.Limit != 25 || service.filter.Cursor != "prior" {
			t.Fatalf("saved list call = %d/%s/%+v body=%s", resp.Code, service.owner, service.filter, resp.Body.String())
		}
	})
}

func TestSavedPostHandlersRejectInvalidRequests(t *testing.T) {
	for name, req := range map[string]*http.Request{
		"invalid did": func() *http.Request {
			r := savedPostRequest(http.MethodPost, "/v1/posts/nope/one/saves", "")
			r.SetPathValue("did", "nope")
			r.SetPathValue("rkey", "one")
			return r
		}(),
		"invalid rkey": func() *http.Request {
			r := savedPostRequest(http.MethodPost, "/v1/posts/did:plc:bob/bad%20key/saves", "")
			r.SetPathValue("did", "did:plc:bob")
			r.SetPathValue("rkey", "bad key")
			return r
		}(),
	} {
		t.Run(name, func(t *testing.T) {
			resp := httptest.NewRecorder()
			api.SavePostHandler(&fakeSavedPostTargetResolver{}, &fakeSavedPostStore{}).ServeHTTP(resp, req)
			assertSavedPostError(t, resp, http.StatusBadRequest, "invalid_identifier")
		})
	}

	resp := httptest.NewRecorder()
	api.CreateSavedPostFolderHandler(&fakeSavedPostStore{}).ServeHTTP(resp, savedPostRequest(http.MethodPost, "/v1/saved-post-folders", `{"name":"bad/name","unknown":true}`))
	assertSavedPostError(t, resp, http.StatusBadRequest, "unexpected_field")

	resp = httptest.NewRecorder()
	api.ListSavedPostsHandler(&fakeSavedPostListService{}).ServeHTTP(resp, savedPostRequest(http.MethodGet, "/v1/saved-posts?folderId=x&unfiled=true", ""))
	assertSavedPostError(t, resp, http.StatusBadRequest, "validation_failed")
}

func TestSavedPostServiceHydratesEveryExactPostType(t *testing.T) {
	base := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	rootURI := "at://did:plc:bob/social.craftsky.feed.post/root"
	commentURI := "at://did:plc:bob/social.craftsky.feed.post/comment"
	parentURI := "at://did:plc:bob/social.craftsky.feed.post/parent"
	quotedURI := "at://did:plc:carol/social.craftsky.feed.post/quoted"
	rows := []*api.PostRow{
		{URI: "at://did:plc:bob/social.craftsky.feed.post/ordinary", DID: "did:plc:bob", Rkey: "ordinary", CID: "ordinary-cid", Text: "ordinary", CreatedAt: base, IndexedAt: base},
		{URI: "at://did:plc:bob/social.craftsky.feed.post/project", DID: "did:plc:bob", Rkey: "project", CID: "project-cid", Text: "project", CreatedAt: base, IndexedAt: base, IsProject: true, Project: &api.Project{}},
		{URI: "at://did:plc:bob/social.craftsky.feed.post/quote", DID: "did:plc:bob", Rkey: "quote", CID: "quote-cid", Text: "quote", CreatedAt: base, IndexedAt: base, QuoteURI: &quotedURI, QuoteCID: stringPointer("quoted-cid")},
		{URI: commentURI, DID: "did:plc:bob", Rkey: "comment", CID: "comment-cid", Text: "comment", CreatedAt: base, IndexedAt: base, ReplyRootURI: &rootURI, ReplyRootCID: stringPointer("root-cid"), ReplyParentURI: &rootURI, ReplyParentCID: stringPointer("root-cid")},
		{URI: "at://did:plc:bob/social.craftsky.feed.post/reply", DID: "did:plc:bob", Rkey: "reply", CID: "reply-cid", Text: "reply", CreatedAt: base, IndexedAt: base, ReplyRootURI: &rootURI, ReplyRootCID: stringPointer("root-cid"), ReplyParentURI: &parentURI, ReplyParentCID: stringPointer("parent-cid")},
	}
	refs := make([]api.SavedPostRef, 0, len(rows))
	rowMap := make(map[syntax.ATURI]*api.PostRow, len(rows))
	summaries := make(map[string]api.EngagementSummary, len(rows))
	folderID := "opaque-folder"
	for i, row := range rows {
		uri := syntax.ATURI(row.URI)
		refs = append(refs, api.SavedPostRef{PostURI: uri, SavedAt: base.Add(time.Duration(i) * time.Minute), FolderID: &folderID})
		rowMap[uri] = row
		summaries[row.URI] = api.EngagementSummary{ViewerHasSaved: true, ViewerSavedFolderID: &folderID}
	}
	hydrator := &fakeSavedPostHydrator{
		rows:      rowMap,
		summaries: summaries,
		quoteRows: map[string]*api.QuoteViewRow{quotedURI: {State: "visible", Post: &api.PostRow{URI: quotedURI, DID: "did:plc:carol", CID: "quoted-cid", Text: "quoted", CreatedAt: base, IndexedAt: base}}},
	}
	service := api.NewSavedPostService(&fakeSavedPostRefStore{refs: refs, cursor: "next"}, hydrator, fakeSavedPostHandleResolver{})
	page, err := service.ListSavedPosts(context.Background(), syntax.DID("did:plc:alice"), api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 50})
	if err != nil {
		t.Fatalf("ListSavedPosts: %v", err)
	}
	if len(page.Items) != len(refs) || page.Cursor != "next" {
		t.Fatalf("page len/cursor = %d/%q", len(page.Items), page.Cursor)
	}
	for i, item := range page.Items {
		if item.Post == nil || item.Post.URI != refs[i].PostURI.String() || !item.SavedAt.Equal(refs[i].SavedAt) || item.FolderID == nil || *item.FolderID != folderID {
			t.Fatalf("item[%d] = %+v, ref=%+v", i, item, refs[i])
		}
		if !item.Post.ViewerHasSaved || item.Post.ViewerSavedFolderID == nil || *item.Post.ViewerSavedFolderID != folderID {
			t.Fatalf("item[%d] viewer state = %+v", i, item.Post)
		}
	}
	if page.Items[1].Post.Project == nil {
		t.Fatal("project materialization was lost")
	}
	if page.Items[2].Post.Quote == nil || page.Items[2].Post.Quote.URI != quotedURI || page.Items[2].Post.QuoteView == nil || page.Items[2].Post.QuoteView.Post == nil {
		t.Fatalf("quote hydration = %+v", page.Items[2].Post)
	}
	if page.Items[3].Post.Reply == nil || page.Items[3].Post.Reply.Root.URI != rootURI || page.Items[3].Post.Reply.Parent.URI != rootURI {
		t.Fatalf("comment refs = %+v", page.Items[3].Post.Reply)
	}
	if page.Items[4].Post.Reply == nil || page.Items[4].Post.Reply.Root.URI != rootURI || page.Items[4].Post.Reply.Parent.URI != parentURI {
		t.Fatalf("nested reply refs = %+v", page.Items[4].Post.Reply)
	}
}

func TestSavedPostServiceAppliesDirectPolicyAndRetainsSuppressedReferences(t *testing.T) {
	base := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	folderID := "opaque-folder"
	rootURI := "at://did:plc:root/social.craftsky.feed.post/root"
	targets := []struct {
		name string
		uri  syntax.ATURI
		did  syntax.DID
	}{
		{name: "eligible", uri: "at://did:plc:eligible/social.craftsky.feed.post/one", did: "did:plc:eligible"},
		{name: "muted", uri: "at://did:plc:muted/social.craftsky.feed.post/two", did: "did:plc:muted"},
		{name: "blocked", uri: "at://did:plc:blocked/social.craftsky.feed.post/three", did: "did:plc:blocked"},
		{name: "hidden", uri: "at://did:plc:hidden/social.craftsky.feed.post/four", did: "did:plc:hidden"},
		{name: "nonmember", uri: "at://did:plc:nonmember/social.craftsky.feed.post/five", did: "did:plc:nonmember"},
		{name: "missing context", uri: "at://did:plc:context/social.craftsky.feed.post/six", did: "did:plc:context"},
	}
	refs := make([]api.SavedPostRef, 0, len(targets))
	rows := make(map[syntax.ATURI]*api.PostRow)
	summaries := make(map[string]api.EngagementSummary)
	for i, target := range targets {
		refs = append(refs, api.SavedPostRef{PostURI: target.uri, SavedAt: base.Add(time.Duration(i) * time.Minute), FolderID: &folderID})
		row := &api.PostRow{URI: target.uri.String(), DID: target.did.String(), Rkey: target.name, CID: target.name + "-cid", Text: target.name, CreatedAt: base, IndexedAt: base}
		if target.name == "missing context" {
			row.ReplyRootURI = &rootURI
			row.ReplyRootCID = stringPointer("root-cid")
			row.ReplyParentURI = &rootURI
			row.ReplyParentCID = stringPointer("root-cid")
		}
		rows[target.uri] = row
		summaries[target.uri.String()] = api.EngagementSummary{ViewerHasSaved: true, ViewerSavedFolderID: &folderID}
	}
	delete(rows, targets[3].uri)
	delete(rows, targets[4].uri)
	hydrator := &fakeSavedPostHydrator{
		rows:      rows,
		summaries: summaries,
		states: map[syntax.DID]relationships.State{
			targets[1].did: {Muted: true},
			targets[2].did: {Blocking: true},
		},
		contexts: map[syntax.ATURI]bool{targets[5].uri: false},
	}
	refStore := &fakeSavedPostRefStore{refs: refs, cursor: "opaque-next-candidate-page"}
	service := api.NewSavedPostService(refStore, hydrator, fakeSavedPostHandleResolver{})
	page, err := service.ListSavedPosts(context.Background(), "did:plc:alice", api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 50})
	if err != nil {
		t.Fatalf("ListSavedPosts: %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].Post.URI != targets[0].uri.String() || page.Items[1].Post.URI != targets[1].uri.String() {
		t.Fatalf("visible items = %+v", page.Items)
	}
	if page.Cursor != refStore.cursor {
		t.Fatalf("policy-shaped page cursor = %q, want %q", page.Cursor, refStore.cursor)
	}
	if !page.Items[1].Post.Author.Muted {
		t.Fatalf("muted item lost viewer state: %+v", page.Items[1].Post.Author)
	}
	if len(refStore.refs) != len(targets) {
		t.Fatalf("policy shaping mutated private references: %d", len(refStore.refs))
	}

	hydrator.states = map[syntax.DID]relationships.State{}
	for _, target := range targets {
		hydrator.states[target.did] = relationships.State{Blocking: true}
	}
	fullySuppressed, err := service.ListSavedPosts(context.Background(), "did:plc:alice", api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 50})
	if err != nil {
		t.Fatalf("fully suppressed ListSavedPosts: %v", err)
	}
	if len(fullySuppressed.Items) != 0 || fullySuppressed.Cursor != refStore.cursor {
		t.Fatalf("fully suppressed page = %d items/cursor %q, want 0/%q", len(fullySuppressed.Items), fullySuppressed.Cursor, refStore.cursor)
	}

	for _, target := range targets {
		if _, present := rows[target.uri]; !present {
			rows[target.uri] = &api.PostRow{URI: target.uri.String(), DID: target.did.String(), Rkey: target.name, CID: target.name + "-cid", Text: target.name, CreatedAt: base, IndexedAt: base}
		}
	}
	hydrator.states = map[syntax.DID]relationships.State{}
	hydrator.contexts = map[syntax.ATURI]bool{}
	restored, err := service.ListSavedPosts(context.Background(), "did:plc:alice", api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 50})
	if err != nil {
		t.Fatalf("restored ListSavedPosts: %v", err)
	}
	if len(restored.Items) != len(targets) {
		t.Fatalf("restored items = %d, want %d", len(restored.Items), len(targets))
	}
	for i, item := range restored.Items {
		if !item.SavedAt.Equal(refs[i].SavedAt) || item.FolderID == nil || *item.FolderID != folderID {
			t.Fatalf("restored metadata[%d] = %+v, want %+v", i, item, refs[i])
		}
	}
}

func savedPostRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	return req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:alice")))
}

func assertSavedPostError(t *testing.T, resp *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if resp.Code != status {
		t.Fatalf("status = %d, want %d body=%s", resp.Code, status, resp.Body.String())
	}
	var body envelope.Error
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Error != code {
		t.Fatalf("error = %q, want %q", body.Error, code)
	}
}
