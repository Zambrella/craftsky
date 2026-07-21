package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

var ErrSavedPostIdentityUnavailable = errors.New("saved post: identity unavailable")

type SavedPostState struct {
	SavedAt  time.Time `json:"savedAt"`
	FolderID *string   `json:"folderId"`
}

type SavedPostFolder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SavedPostItem struct {
	Post     *PostResponse `json:"post"`
	SavedAt  time.Time     `json:"savedAt"`
	FolderID *string       `json:"folderId"`
}

type SavedPostRef struct {
	PostURI  syntax.ATURI
	SavedAt  time.Time
	FolderID *string
}

type SavedPostListFilter struct {
	Scope    SavedPostScope
	FolderID string
	Sort     SavedPostSort
	Limit    int
	Cursor   string
}

type SavedPostPage struct {
	Items  []SavedPostItem `json:"items"`
	Cursor string          `json:"cursor,omitempty"`
}

type savedPostFolderPage struct {
	Items  []SavedPostFolder `json:"items"`
	Cursor string            `json:"cursor,omitempty"`
}

type SavedPostTargetResolver interface {
	ResolveSavedPostTarget(context.Context, syntax.DID, syntax.DID, syntax.RecordKey) (syntax.ATURI, error)
}

type SavedPostMutationStore interface {
	Save(context.Context, syntax.DID, syntax.ATURI, FolderAssignment) (SaveMutationResult, error)
	Unsave(context.Context, syntax.DID, syntax.ATURI) error
}

type SavedPostFolderStore interface {
	CreateFolder(context.Context, syntax.DID, string) (SavedPostFolder, error)
	RenameFolder(context.Context, syntax.DID, string, string) (SavedPostFolder, error)
	DeleteFolder(context.Context, syntax.DID, string) error
	ListFolders(context.Context, syntax.DID, int, string) ([]SavedPostFolder, string, error)
}

type SavedPostListService interface {
	ListSavedPosts(context.Context, syntax.DID, SavedPostListFilter) (SavedPostPage, error)
}

type SavedPostRefReader interface {
	ListSavedRefs(context.Context, syntax.DID, SavedPostListFilter) ([]SavedPostRef, string, error)
}

type SavedPostHydrator interface {
	ReadEligiblePostsByURI(context.Context, syntax.DID, []syntax.ATURI) (map[syntax.ATURI]*PostRow, error)
	EngagementSummaries(context.Context, string, []string) (map[string]EngagementSummary, error)
	QuoteViewRows(context.Context, []ResponseStrongRef) (map[string]*QuoteViewRow, error)
}

type SavedPostService struct {
	refs     SavedPostRefReader
	hydrator SavedPostHydrator
	resolver HandleResolver
}

func NewSavedPostService(refs SavedPostRefReader, hydrator SavedPostHydrator, resolver HandleResolver) *SavedPostService {
	return &SavedPostService{refs: refs, hydrator: hydrator, resolver: resolver}
}

func (s *SavedPostService) ListSavedPosts(ctx context.Context, owner syntax.DID, filter SavedPostListFilter) (SavedPostPage, error) {
	refs, next, err := s.refs.ListSavedRefs(ctx, owner, filter)
	if err != nil {
		return SavedPostPage{}, err
	}
	uris := make([]syntax.ATURI, 0, len(refs))
	for _, ref := range refs {
		uris = append(uris, ref.PostURI)
	}
	rowsByURI, err := s.hydrator.ReadEligiblePostsByURI(ctx, owner, uris)
	if err != nil {
		return SavedPostPage{}, err
	}
	contextStates := make(map[syntax.ATURI]bool, len(uris))
	for _, uri := range uris {
		contextStates[uri] = true
	}
	if contextReader, ok := s.hydrator.(interface {
		RequiredContextStates(context.Context, syntax.DID, []syntax.ATURI) (map[syntax.ATURI]bool, error)
	}); ok {
		contextStates, err = contextReader.RequiredContextStates(ctx, owner, uris)
		if err != nil {
			return SavedPostPage{}, err
		}
	}
	subjects := make([]syntax.DID, 0, len(rowsByURI))
	seenSubjects := make(map[syntax.DID]struct{}, len(rowsByURI))
	for _, row := range rowsByURI {
		did, err := syntax.ParseDID(row.DID)
		if err != nil {
			return SavedPostPage{}, ErrSavedPostIdentityUnavailable
		}
		if _, seen := seenSubjects[did]; !seen {
			seenSubjects[did] = struct{}{}
			subjects = append(subjects, did)
		}
	}
	states := make(map[syntax.DID]relationships.State, len(subjects))
	if relationshipReader, ok := s.hydrator.(interface {
		RelationshipStates(context.Context, syntax.DID, []syntax.DID) (map[syntax.DID]relationships.State, error)
	}); ok {
		states, err = relationshipReader.RelationshipStates(ctx, owner, subjects)
		if err != nil {
			return SavedPostPage{}, err
		}
	}
	rows := make([]*PostRow, 0, len(refs))
	postURIs := make([]string, 0, len(refs))
	for _, ref := range refs {
		row := rowsByURI[ref.PostURI]
		if row == nil || ((row.ReplyRootURI != nil || row.ReplyParentURI != nil) && !contextStates[ref.PostURI]) {
			continue
		}
		did := syntax.DID(row.DID)
		if !savedPostPolicyAllows(states[did]) {
			continue
		}
		rows = append(rows, row)
		postURIs = append(postURIs, row.URI)
	}
	handles, err := resolveHandlesForRows(ctx, rows, s.resolver)
	if err != nil {
		return SavedPostPage{}, ErrSavedPostIdentityUnavailable
	}
	summaries, err := s.hydrator.EngagementSummaries(ctx, owner.String(), postURIs)
	if err != nil {
		return SavedPostPage{}, err
	}
	items := make([]SavedPostItem, 0, len(rows))
	responses := make([]*PostResponse, 0, len(rows))
	for _, ref := range refs {
		row := rowsByURI[ref.PostURI]
		if row == nil || ((row.ReplyRootURI != nil || row.ReplyParentURI != nil) && !contextStates[ref.PostURI]) {
			continue
		}
		did := syntax.DID(row.DID)
		if !savedPostPolicyAllows(states[did]) {
			continue
		}
		response := BuildPostResponse(row, handles[row.DID])
		applyEngagementSummary(response, summaries[row.URI])
		ApplyPostAuthorViewerState(response, states[did])
		items = append(items, SavedPostItem{Post: response, SavedAt: ref.SavedAt, FolderID: ref.FolderID})
		responses = append(responses, response)
	}
	if err := attachQuoteViews(ctx, s.hydrator, s.resolver, responses); err != nil {
		if errors.Is(err, ErrHandleUnavailable) {
			return SavedPostPage{}, ErrSavedPostIdentityUnavailable
		}
		return SavedPostPage{}, err
	}
	return SavedPostPage{Items: items, Cursor: next}, nil
}

func savedPostPolicyAllows(state relationships.State) bool {
	return !state.HasBlock()
}

type SaveMutationResult struct {
	State   SavedPostState
	Created bool
	Changed bool
}

func SaveMutationHTTPStatus(result SaveMutationResult) int {
	if result.Created {
		return http.StatusCreated
	}
	return http.StatusOK
}

func savedAtForMutation(existing *time.Time, now time.Time) time.Time {
	if existing != nil {
		return *existing
	}
	return now
}

func folderUpdatedAtForMutation(current, now time.Time, renamed bool) time.Time {
	if renamed {
		return now
	}
	return current
}

type savedPostFolderOperation string

const (
	savedPostFolderAssignment savedPostFolderOperation = "assignment"
	savedPostFolderRename     savedPostFolderOperation = "rename"
	savedPostFolderScopedList savedPostFolderOperation = "scoped_list"
	savedPostFolderDelete     savedPostFolderOperation = "delete"
)

func mapSavedPostFolderError(operation savedPostFolderOperation, err error) (status int, code string, handled bool) {
	if !errors.Is(err, ErrSavedPostFolderNotFound) {
		return 0, "", false
	}
	if operation == savedPostFolderDelete {
		return http.StatusNoContent, "", true
	}
	return http.StatusNotFound, "saved_post_folder_not_found", true
}

func SavePostHandler(targets SavedPostTargetResolver, store SavedPostMutationStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		did, rkey, ok := savedPostPath(w, r)
		if !ok {
			return
		}
		assignment, err := DecodeSavePostRequest(r.Body)
		if err != nil {
			writeSavedPostFieldError(w, r, err)
			return
		}
		uri, err := targets.ResolveSavedPostTarget(r.Context(), owner, did, rkey)
		if errors.Is(err, ErrPostNotFound) {
			writeSavedPostError(w, r, http.StatusNotFound, "post_not_found", "post not found", nil)
			return
		}
		if err != nil {
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post target lookup failed", nil)
			return
		}
		result, err := store.Save(r.Context(), owner, uri, assignment)
		if errors.Is(err, ErrSavedPostFolderNotFound) {
			writeSavedPostError(w, r, http.StatusNotFound, "saved_post_folder_not_found", "saved post folder not found", nil)
			return
		}
		if err != nil {
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post mutation failed", nil)
			return
		}
		writeSavedPostJSON(w, SaveMutationHTTPStatus(result), result.State)
	})
}

func UnsavePostHandler(store SavedPostMutationStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		did, rkey, ok := savedPostPath(w, r)
		if !ok {
			return
		}
		uri := syntax.ATURI("at://" + did.String() + "/" + craftskyPostNSID + "/" + rkey.String())
		if err := store.Unsave(r.Context(), owner, uri); err != nil {
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post delete failed", nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func CreateSavedPostFolderHandler(store SavedPostFolderStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		name, err := DecodeSavedPostFolderRequest(r.Body)
		if err != nil {
			writeSavedPostFieldError(w, r, err)
			return
		}
		folder, err := store.CreateFolder(r.Context(), owner, name)
		if err != nil {
			if _, ok := err.(*FieldError); ok {
				writeSavedPostFieldError(w, r, err)
				return
			}
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post folder create failed", nil)
			return
		}
		writeSavedPostJSON(w, http.StatusCreated, folder)
	})
}

func RenameSavedPostFolderHandler(store SavedPostFolderStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		name, err := DecodeSavedPostFolderRequest(r.Body)
		if err != nil {
			writeSavedPostFieldError(w, r, err)
			return
		}
		folder, err := store.RenameFolder(r.Context(), owner, r.PathValue("folderId"), name)
		if status, code, handled := mapSavedPostFolderError(savedPostFolderRename, err); handled {
			writeSavedPostError(w, r, status, code, "saved post folder not found", nil)
			return
		}
		if err != nil {
			if _, ok := err.(*FieldError); ok {
				writeSavedPostFieldError(w, r, err)
				return
			}
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post folder rename failed", nil)
			return
		}
		writeSavedPostJSON(w, http.StatusOK, folder)
	})
}

func DeleteSavedPostFolderHandler(store SavedPostFolderStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		err := store.DeleteFolder(r.Context(), owner, r.PathValue("folderId"))
		if status, _, handled := mapSavedPostFolderError(savedPostFolderDelete, err); handled {
			w.WriteHeader(status)
			return
		}
		if err != nil {
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post folder delete failed", nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func ListSavedPostFoldersHandler(store SavedPostFolderStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		limit, cursor, err := ParseSavedPostFolderListQuery(r.URL.Query())
		if err != nil {
			writeSavedPostQueryError(w, r, err)
			return
		}
		items, next, err := store.ListFolders(r.Context(), owner, limit, cursor)
		if errors.Is(err, envelope.ErrInvalidCursor) {
			writeSavedPostError(w, r, http.StatusBadRequest, "invalid_cursor", "invalid cursor", nil)
			return
		}
		if err != nil {
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post folder list failed", nil)
			return
		}
		if items == nil {
			items = []SavedPostFolder{}
		}
		writeSavedPostJSON(w, http.StatusOK, savedPostFolderPage{Items: items, Cursor: next})
	})
}

func ListSavedPostsHandler(service SavedPostListService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := savedPostOwner(w, r)
		if !ok {
			return
		}
		filter, err := ParseSavedPostListQuery(r.URL.Query())
		if err != nil {
			writeSavedPostQueryError(w, r, err)
			return
		}
		page, err := service.ListSavedPosts(r.Context(), owner, filter)
		switch {
		case errors.Is(err, ErrSavedPostFolderNotFound):
			writeSavedPostError(w, r, http.StatusNotFound, "saved_post_folder_not_found", "saved post folder not found", nil)
			return
		case errors.Is(err, envelope.ErrInvalidCursor):
			writeSavedPostError(w, r, http.StatusBadRequest, "invalid_cursor", "invalid cursor", nil)
			return
		case errors.Is(err, ErrSavedPostIdentityUnavailable):
			writeSavedPostError(w, r, http.StatusBadGateway, "identity_unavailable", "could not resolve handle", nil)
			return
		case err != nil:
			writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "saved post list failed", nil)
			return
		}
		if page.Items == nil {
			page.Items = []SavedPostItem{}
		}
		writeSavedPostJSON(w, http.StatusOK, page)
	})
}

func savedPostOwner(w http.ResponseWriter, r *http.Request) (syntax.DID, bool) {
	owner, ok := middleware.GetDID(r.Context())
	if !ok {
		writeSavedPostError(w, r, http.StatusInternalServerError, "internal_error", "no did in context", nil)
	}
	return owner, ok
}

func savedPostPath(w http.ResponseWriter, r *http.Request) (syntax.DID, syntax.RecordKey, bool) {
	did, err := syntax.ParseDID(r.PathValue("did"))
	if err != nil {
		writeSavedPostError(w, r, http.StatusBadRequest, "invalid_identifier", "not a valid DID", nil)
		return "", "", false
	}
	rkey, err := syntax.ParseRecordKey(r.PathValue("rkey"))
	if err != nil {
		writeSavedPostError(w, r, http.StatusBadRequest, "invalid_identifier", "not a valid record key", nil)
		return "", "", false
	}
	return did, rkey, true
}

func writeSavedPostFieldError(w http.ResponseWriter, r *http.Request, err error) {
	fieldErr, ok := err.(*FieldError)
	if !ok {
		writeSavedPostError(w, r, http.StatusBadRequest, "malformed_body", "could not parse request", nil)
		return
	}
	status := http.StatusBadRequest
	message := "request rejected"
	if fieldErr.Code == "validation_failed" {
		status = http.StatusUnprocessableEntity
		message = "validation failed"
	}
	writeSavedPostError(w, r, status, fieldErr.Code, message, fieldErr.Fields)
}

func writeSavedPostQueryError(w http.ResponseWriter, r *http.Request, err error) {
	fieldErr, ok := err.(*FieldError)
	if !ok {
		writeSavedPostError(w, r, http.StatusBadRequest, "validation_failed", "invalid query", nil)
		return
	}
	writeSavedPostError(w, r, http.StatusBadRequest, fieldErr.Code, "invalid query", fieldErr.Fields)
}

func writeSavedPostError(w http.ResponseWriter, r *http.Request, status int, code, message string, fields map[string]string) {
	envelope.WriteError(w, status, code, message, middleware.GetRunID(r.Context()), fields)
}

func writeSavedPostJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
