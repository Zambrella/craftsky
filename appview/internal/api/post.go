// appview/internal/api/post.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/postutil"
)

const craftskyPostNSID = "social.craftsky.feed.post"
const craftskyLikeNSID = "social.craftsky.feed.like"
const craftskyRepostNSID = "social.craftsky.feed.repost"

const threadMaxDepth = 6
const threadMaxPosts = 500

// LikeStore is the read-side interaction subset needed by like/unlike handlers.
type LikeStore interface {
	ResolvePostTarget(ctx context.Context, did, rkey string) (*PostTargetRef, error)
	FindActiveLike(ctx context.Context, did, subjectURI string) (*InteractionRow, error)
}

// RepostStore is the read-side interaction subset needed by repost/unrepost handlers.
type RepostStore interface {
	ResolvePostTarget(ctx context.Context, did, rkey string) (*PostTargetRef, error)
	FindActiveRepost(ctx context.Context, did, subjectURI string) (*InteractionRow, error)
}

// CreatePostHandler serves POST /v1/posts.
func CreatePostHandler(
	store PostReader,
	newPDS auth.PDSClientFactory,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())

		req, err := DecodePostCreate(r.Body)
		if err != nil {
			fe, isFE := err.(*FieldError)
			switch {
			case isFE && fe.Code == "malformed_body":
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, fe.Fields)
			case isFE:
				envelope.WriteError(w, http.StatusBadRequest,
					fe.Code, "request body rejected", runID, fe.Fields)
			default:
				envelope.WriteError(w, http.StatusBadRequest,
					"malformed_body", "could not parse body", runID, nil)
			}
			return
		}
		if err := ValidatePostCreate(req); err != nil {
			fe, isFE := err.(*FieldError)
			if isFE {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}

		body := lexiconRecordBody(req)

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), did, craftskyPostNSID, body)
		if err != nil {
			logger.Warn("post: CreateRecord failed",
				slog.String("did", did.String()), slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write post", runID, nil)
			return
		}

		row, err := syntheticPostRow(r, store, did, uri, cid, req)
		if err != nil {
			logger.Error("post: hydrate author failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post created but hydrate failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// lexiconRecordBody translates the wire request into the lexicon-shaped
// record body that goes to the PDS. Facets are pass-through raw JSON so
// the PDS sees exactly what the client sent (including any "$type"
// discriminators on union variants).
func lexiconRecordBody(req PostCreateRequest) map[string]any {
	body := map[string]any{
		"$type":     craftskyPostNSID,
		"text":      req.Text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}
	if len(req.Facets) > 0 {
		body["facets"] = req.Facets
	}
	if req.Reply != nil {
		body["reply"] = map[string]any{
			"root":   map[string]any{"uri": req.Reply.Root.URI, "cid": req.Reply.Root.CID},
			"parent": map[string]any{"uri": req.Reply.Parent.URI, "cid": req.Reply.Parent.CID},
		}
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		body["embed"] = map[string]any{
			"$type": craftskyPostNSID + "#quoteEmbed",
			"record": map[string]any{
				"uri": req.Embed.Quote.URI,
				"cid": req.Embed.Quote.CID,
			},
		}
	}
	return body
}

// syntheticPostRow assembles the PostRow that BuildPostResponse needs
// from the request body, the PDS-assigned (uri, cid), and a single
// author lookup against the store. We don't wait for the firehose to
// land the row.
//
// Tags are extracted from req.Facets via a non-strict decode into the
// indigo richtext-facet typed slice. Errors on that decode produce
// empty tags (the PDS will still validate the lexicon shape) — we
// don't fail the whole request just because tag extraction couldn't
// parse facets the PDS may yet accept.
func syntheticPostRow(
	r *http.Request,
	store PostReader,
	did syntax.DID,
	uri syntax.ATURI,
	cid syntax.CID,
	req PostCreateRequest,
) (*PostRow, error) {
	now := time.Now().UTC()
	row := &PostRow{
		URI:       string(uri),
		DID:       did.String(),
		Rkey:      path.Base(string(uri)),
		CID:       string(cid),
		Text:      req.Text,
		Tags:      extractRequestTags(req.Facets),
		CreatedAt: now,
		IndexedAt: now,
	}
	if len(req.Facets) > 0 {
		row.Facets = req.Facets
	}
	if req.Reply != nil {
		row.ReplyRootURI = strPtr(req.Reply.Root.URI)
		row.ReplyRootCID = strPtr(req.Reply.Root.CID)
		row.ReplyParentURI = strPtr(req.Reply.Parent.URI)
		row.ReplyParentCID = strPtr(req.Reply.Parent.CID)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		row.QuoteURI = strPtr(req.Embed.Quote.URI)
		row.QuoteCID = strPtr(req.Embed.Quote.CID)
	}

	author, err := store.ReadAuthor(r.Context(), did.String())
	if err != nil {
		return nil, err
	}
	if author != nil {
		row.AuthorDisplayName = author.DisplayName
		row.AuthorAvatarCID = author.AvatarCID
	}
	return row, nil
}

// extractRequestTags decodes the raw facets JSON into the indigo typed
// slice (best effort) and returns the same tag set the indexer would
// produce when this record arrives via the firehose. Returns an empty
// (non-nil) slice on decode failure so the response always carries a
// valid tags array.
func extractRequestTags(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var typed []*appbsky.RichtextFacet
	if err := json.Unmarshal(raw, &typed); err != nil {
		return []string{}
	}
	return postutil.ExtractTags(typed)
}

func strPtr(s string) *string { return &s }

// GetPostHandler serves GET /v1/posts/{did}/{rkey}.
func GetPostHandler(store PostReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		row, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post: ReadOne failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}
		viewerDID, _ := middleware.GetDID(r.Context())
		summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), []string{row.URI})
		if err != nil {
			logger.Error("post: EngagementSummaries failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				slog.String("did", did.String()),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		applyEngagementSummary(resp, summaries[row.URI])
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// ListDirectRepliesHandler serves GET /v1/posts/{did}/{rkey}/replies.
func ListDirectRepliesHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		target, err := store.ResolvePostTarget(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post replies: ResolvePostTarget failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}

		limit := parseCommentLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		rows, nextCursor, err := store.ListDirectReplies(r.Context(), target.URI, limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("post replies: ListDirectReplies failed",
				slog.String("target_uri", target.URI),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "reply list failed", runID, nil)
			return
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			viewerDID, _ := middleware.GetDID(r.Context())
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				postURIs = append(postURIs, row.URI)
			}
			summaries, serr := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if serr != nil {
				logger.Error("post replies: EngagementSummaries failed",
					slog.String("target_uri", target.URI),
					slog.String("err", serr.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			handles, herr := resolveHandlesForRows(r.Context(), rows, resolver)
			if herr != nil {
				logger.Warn("post replies: ResolveHandle failed",
					slog.String("err", herr.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				resp := BuildPostResponse(row, handles[row.DID])
				applyEngagementSummary(resp, summaries[row.URI])
				items = append(items, resp)
			}
		}
		body := struct {
			Items  []*PostResponse `json:"items"`
			Cursor string          `json:"cursor,omitempty"`
		}{Items: items, Cursor: nextCursor}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// GetPostCommentsHandler serves GET /v1/posts/{did}/{rkey}/comments.
func GetPostCommentsHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		root, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post comments: ReadOne failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}

		viewerDID, _ := middleware.GetDID(r.Context())
		sortValue := parseCommentSort(r.URL.Query().Get("sort"))
		limit := parseCommentLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		focus := (*FocusContext)(nil)
		focusedURI := ""
		var focusedCommentRow *PostRow
		var focusedReplyRow *PostRow
		var focusedReplyParentRow *PostRow
		if focusRaw := r.URL.Query().Get("focus"); focusRaw != "" {
			if _, ferr := syntax.ParseATURI(focusRaw); ferr != nil {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_focus", "focus query parameter is not a valid AT-URI", runID, nil)
				return
			}
			focus = &FocusContext{URI: focusRaw, Status: "notFound"}
			focusedRow, ferr := store.ReadPostByURI(r.Context(), focusRaw)
			if ferr != nil && !errors.Is(ferr, ErrPostNotFound) {
				logger.Error("post comments: ReadPostByURI failed",
					slog.String("focus", focusRaw),
					slog.String("err", ferr.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "focus read failed", runID, nil)
				return
			}
			if focusedRow != nil {
				switch {
				case focusedRow.URI == root.URI:
					focus.Status = "included"
					focus.Kind = "root"
				case focusedRow.ReplyParentURI != nil && *focusedRow.ReplyParentURI == root.URI:
					focus.Status = "included"
					focus.Kind = "comment"
					focusedURI = focusedRow.URI
					focusedCommentRow = focusedRow
				case focusedRow.ReplyRootURI != nil && *focusedRow.ReplyRootURI == root.URI:
					if focusedRow.ReplyParentURI != nil {
						parentRow, perr := store.ReadPostByURI(r.Context(), *focusedRow.ReplyParentURI)
						if perr != nil && !errors.Is(perr, ErrPostNotFound) {
							logger.Error("post comments: ReadPostByURI parent failed",
								slog.String("parent", *focusedRow.ReplyParentURI),
								slog.String("err", perr.Error()),
								slog.String("run_id", runID))
							envelope.WriteError(w, http.StatusInternalServerError,
								"internal_error", "focus parent read failed", runID, nil)
							return
						}
						if parentRow != nil && parentRow.ReplyParentURI != nil && *parentRow.ReplyParentURI == root.URI {
							focus.Status = "included"
							focus.Kind = "reply"
							focus.CommentURI = parentRow.URI
							focusedURI = parentRow.URI
							focusedCommentRow = parentRow
							focusedReplyRow = focusedRow
							focusedReplyParentRow = parentRow
						} else if parentRow != nil && parentRow.ReplyRootURI != nil && *parentRow.ReplyRootURI == root.URI && parentRow.ReplyParentURI != nil {
							commentRow, cerr := store.ReadPostByURI(r.Context(), *parentRow.ReplyParentURI)
							if cerr != nil && !errors.Is(cerr, ErrPostNotFound) {
								logger.Error("post comments: ReadPostByURI comment ancestor failed",
									slog.String("ancestor", *parentRow.ReplyParentURI),
									slog.String("err", cerr.Error()),
									slog.String("run_id", runID))
								envelope.WriteError(w, http.StatusInternalServerError,
									"internal_error", "focus ancestor read failed", runID, nil)
								return
							}
							if commentRow != nil && commentRow.ReplyParentURI != nil && *commentRow.ReplyParentURI == root.URI {
								focus.Status = "included"
								focus.Kind = "reply"
								focus.CommentURI = commentRow.URI
								focusedURI = commentRow.URI
								focusedCommentRow = commentRow
								focusedReplyRow = focusedRow
								focusedReplyParentRow = parentRow
							}
						}
					}
				default:
					focus.Status = "mismatchedRoot"
				}
			}
		}
		comments, nextCursor, err := store.ListRootComments(r.Context(), root.URI, viewerDID.String(), sortValue, limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("post comments: ListRootComments failed",
				slog.String("root_uri", root.URI),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "comment list failed", runID, nil)
			return
		}

		hydratedRows := append([]*PostRow{root}, comments...)
		if focusedCommentRow != nil && !containsPostRow(comments, focusedCommentRow.URI) {
			hydratedRows = append(hydratedRows, focusedCommentRow)
		}
		if focusedReplyRow != nil {
			hydratedRows = append(hydratedRows, focusedReplyRow)
		}
		if focusedReplyParentRow != nil && focusedReplyParentRow.URI != focusedCommentRow.URI {
			hydratedRows = append(hydratedRows, focusedReplyParentRow)
		}
		postURIs := make([]string, 0, len(hydratedRows))
		for _, row := range hydratedRows {
			postURIs = append(postURIs, row.URI)
		}
		summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
		if err != nil {
			logger.Error("post comments: EngagementSummaries failed",
				slog.String("root_uri", root.URI),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		handles, err := resolveHandlesForRows(r.Context(), hydratedRows, resolver)
		if err != nil {
			logger.Warn("post comments: ResolveHandle failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}

		rootPost := BuildPostResponse(root, handles[root.DID])
		applyEngagementSummary(rootPost, summaries[root.URI])
		items := make([]CommentItem, 0, len(comments))
		if focusedCommentRow != nil && !containsPostRow(comments, focusedCommentRow.URI) {
			post := BuildPostResponse(focusedCommentRow, handles[focusedCommentRow.DID])
			applyEngagementSummary(post, summaries[focusedCommentRow.URI])
			replies := ReplyPage{Loaded: false, Items: []ReplyItem{}}
			if focusedReplyRow != nil {
				replies = ReplyPage{Loaded: true, Items: []ReplyItem{buildFocusedReplyItem(focusedReplyRow, focusedReplyParentRow, focusedCommentRow, handles, summaries)}}
			}
			items = append(items, CommentItem{
				Post:      post,
				Placement: "focused",
				Replies:   replies,
			})
		}
		for _, row := range comments {
			post := BuildPostResponse(row, handles[row.DID])
			applyEngagementSummary(post, summaries[row.URI])
			placement := "normal"
			if focusedURI != "" && row.URI == focusedURI {
				placement = "focused"
			}
			replies := ReplyPage{Loaded: false, Items: []ReplyItem{}}
			if focusedReplyRow != nil && row.URI == focusedURI {
				replies = ReplyPage{Loaded: true, Items: []ReplyItem{buildFocusedReplyItem(focusedReplyRow, focusedReplyParentRow, row, handles, summaries)}}
			}
			items = append(items, CommentItem{
				Post:      post,
				Placement: placement,
				Replies:   replies,
			})
		}
		body := &CommentSectionResponse{
			Post:     rootPost,
			Comments: CommentPage{Items: items, Cursor: nextCursor},
			Sort:     sortValue,
			Focus:    focus,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// GetPostThreadHandler serves GET /v1/posts/{did}/{rkey}/thread.
func GetPostThreadHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		target, err := store.ResolvePostTarget(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post thread: ResolvePostTarget failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}

		targetRow, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post thread: ReadOne failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}

		rootURI := target.URI
		if targetRow.ReplyRootURI != nil {
			rootURI = *targetRow.ReplyRootURI
		}
		rows, err := store.LoadThreadCandidates(r.Context(), rootURI, target.URI, threadMaxPosts+1)
		if err != nil {
			logger.Error("post thread: LoadThreadCandidates failed",
				slog.String("target_uri", target.URI),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "thread read failed", runID, nil)
			return
		}
		if !containsPostRow(rows, target.URI) {
			rows = append(rows, targetRow)
		}
		ancestors := []*PostRow{}
		if targetRow.ReplyParentURI != nil {
			ancestors, err = store.LoadThreadAncestors(r.Context(), rootURI, *targetRow.ReplyParentURI, target.URI, threadMaxDepth+1)
			if err != nil {
				logger.Error("post thread: LoadThreadAncestors failed",
					slog.String("target_uri", target.URI),
					slog.String("err", err.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "thread ancestor read failed", runID, nil)
				return
			}
		}

		rowTree, renderedRows, truncated := buildThreadRowTree(rows, target.URI)
		viewerDID, _ := middleware.GetDID(r.Context())
		hydratedRows := append([]*PostRow{}, ancestors...)
		hydratedRows = append(hydratedRows, renderedRows...)
		postURIs := make([]string, 0, len(hydratedRows))
		for _, row := range hydratedRows {
			postURIs = append(postURIs, row.URI)
		}
		summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
		if err != nil {
			logger.Error("post thread: EngagementSummaries failed",
				slog.String("target_uri", target.URI),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		handles, err := resolveHandlesForRows(r.Context(), hydratedRows, resolver)
		if err != nil {
			logger.Warn("post thread: ResolveHandle failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}

		body := buildThreadResponse(rowTree, ancestors, handles, summaries, truncated)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// DeletePostHandler serves DELETE /v1/posts/{did}/{rkey}. Idempotent —
// returns 204 even if the underlying record was already gone.
func DeletePostHandler(newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		if did != caller {
			envelope.WriteError(w, http.StatusForbidden,
				"forbidden", "cannot delete another user's post", runID, nil)
			return
		}
		rkey := r.PathValue("rkey")
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		if err := pds.DeleteRecord(r.Context(), did, craftskyPostNSID, rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("post: DeleteRecord failed",
				slog.String("did", did.String()),
				slog.String("rkey", rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// LikePostHandler serves POST /v1/posts/{did}/{rkey}/likes.
func LikePostHandler(store LikeStore, newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		if err := rejectNonEmptyBody(r); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"unexpected_field", "request body rejected", runID, nil)
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), r.PathValue("rkey"))
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("like: resolve target failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveLike(r.Context(), caller.String(), target.URI)
		if err == nil {
			writeInteractionResponse(w, http.StatusOK, interactionResponseFromRow(active))
			return
		}
		if !errors.Is(err, ErrInteractionNotFound) {
			logger.Error("like: find active failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read like", runID, nil)
			return
		}

		createdAt := time.Now().UTC()
		body := likeRecordBody(target, createdAt)
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("like: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), caller, craftskyLikeNSID, body)
		if err != nil {
			logger.Warn("like: CreateRecord failed",
				slog.String("did", caller.String()), slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write like", runID, nil)
			return
		}
		writeInteractionResponse(w, http.StatusCreated, &InteractionWriteResponse{
			URI:       string(uri),
			CID:       string(cid),
			Rkey:      path.Base(string(uri)),
			Subject:   ResponseStrongRef{URI: target.URI, CID: target.CID},
			CreatedAt: createdAt,
		})
	})
}

// UnlikePostHandler serves DELETE /v1/posts/{did}/{rkey}/likes.
func UnlikePostHandler(store LikeStore, newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), r.PathValue("rkey"))
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("unlike: resolve target failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveLike(r.Context(), caller.String(), target.URI)
		if errors.Is(err, ErrInteractionNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			logger.Error("unlike: find active failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read like", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("unlike: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		if err := pds.DeleteRecord(r.Context(), caller, craftskyLikeNSID, active.Rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("unlike: DeleteRecord failed",
				slog.String("did", caller.String()),
				slog.String("rkey", active.Rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

// RepostPostHandler serves POST /v1/posts/{did}/{rkey}/reposts.
func RepostPostHandler(store RepostStore, newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		if err := rejectNonEmptyBody(r); err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"unexpected_field", "request body rejected", runID, nil)
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), r.PathValue("rkey"))
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("repost: resolve target failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveRepost(r.Context(), caller.String(), target.URI)
		if err == nil {
			writeInteractionResponse(w, http.StatusOK, interactionResponseFromRow(active))
			return
		}
		if !errors.Is(err, ErrInteractionNotFound) {
			logger.Error("repost: find active failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read repost", runID, nil)
			return
		}

		createdAt := time.Now().UTC()
		body := repostRecordBody(target, createdAt)
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("repost: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), caller, craftskyRepostNSID, body)
		if err != nil {
			logger.Warn("repost: CreateRecord failed",
				slog.String("did", caller.String()), slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write repost", runID, nil)
			return
		}
		writeInteractionResponse(w, http.StatusCreated, &InteractionWriteResponse{
			URI:       string(uri),
			CID:       string(cid),
			Rkey:      path.Base(string(uri)),
			Subject:   ResponseStrongRef{URI: target.URI, CID: target.CID},
			CreatedAt: createdAt,
		})
	})
}

// UnrepostPostHandler serves DELETE /v1/posts/{did}/{rkey}/reposts.
func UnrepostPostHandler(store RepostStore, newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		targetDID, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), r.PathValue("rkey"))
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("unrepost: resolve target failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveRepost(r.Context(), caller.String(), target.URI)
		if errors.Is(err, ErrInteractionNotFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			logger.Error("unrepost: find active failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read repost", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("unrepost: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}
		if err := pds.DeleteRecord(r.Context(), caller, craftskyRepostNSID, active.Rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("unrepost: DeleteRecord failed",
				slog.String("did", caller.String()),
				slog.String("rkey", active.Rkey),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func rejectNonEmptyBody(r *http.Request) error {
	if r.Body == nil {
		return nil
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(raw)) != "" {
		return errors.New("body must be empty")
	}
	return nil
}

func likeRecordBody(target *PostTargetRef, createdAt time.Time) map[string]any {
	return interactionRecordBody(craftskyLikeNSID, target, createdAt)
}

func repostRecordBody(target *PostTargetRef, createdAt time.Time) map[string]any {
	return interactionRecordBody(craftskyRepostNSID, target, createdAt)
}

func interactionRecordBody(nsid string, target *PostTargetRef, createdAt time.Time) map[string]any {
	return map[string]any{
		"$type": nsid,
		"subject": map[string]any{
			"uri": target.URI,
			"cid": target.CID,
		},
		"createdAt": createdAt.Format(time.RFC3339),
	}
}

func interactionResponseFromRow(row *InteractionRow) *InteractionWriteResponse {
	return &InteractionWriteResponse{
		URI:       row.URI,
		CID:       row.CID,
		Rkey:      row.Rkey,
		Subject:   ResponseStrongRef{URI: row.SubjectURI, CID: row.SubjectCID},
		CreatedAt: row.CreatedAt.UTC(),
	}
}

func writeInteractionResponse(w http.ResponseWriter, status int, resp *InteractionWriteResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// ListPostsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/posts.
func ListPostsByAuthorHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("post list: ResolveDID failed",
					slog.String("input", raw),
					slog.String("err", err.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")

		rows, nextCursor, err := store.ListByAuthor(r.Context(), did.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("post list: ListByAuthor failed",
				slog.String("did", did.String()),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post list failed", runID, nil)
			return
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			viewerDID, _ := middleware.GetDID(r.Context())
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				postURIs = append(postURIs, row.URI)
			}
			summaries, serr := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if serr != nil {
				logger.Error("post list: EngagementSummaries failed",
					slog.String("did", did.String()),
					slog.String("err", serr.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			// Only pay handle-resolution cost when there are rows to render.
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				logger.Warn("post list: ResolveHandle failed",
					slog.String("did", did.String()),
					slog.String("err", herr.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				resp := BuildPostResponse(row, handle)
				applyEngagementSummary(resp, summaries[row.URI])
				items = append(items, resp)
			}
		}
		body := struct {
			Items  []*PostResponse `json:"items"`
			Cursor string          `json:"cursor,omitempty"`
		}{Items: items, Cursor: nextCursor}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

func resolveHandlesForRows(ctx context.Context, rows []*PostRow, resolver HandleResolver) (map[string]syntax.Handle, error) {
	handles := make(map[string]syntax.Handle)
	for _, row := range rows {
		if _, ok := handles[row.DID]; ok {
			continue
		}
		did, err := syntax.ParseDID(row.DID)
		if err != nil {
			return nil, err
		}
		handle, err := resolver.ResolveHandle(ctx, did)
		if err != nil {
			return nil, err
		}
		handles[row.DID] = handle
	}
	return handles, nil
}

func buildFocusedReplyItem(row, parentRow, commentRow *PostRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary) ReplyItem {
	post := BuildPostResponse(row, handles[row.DID])
	applyEngagementSummary(post, summaries[row.URI])
	item := ReplyItem{Post: post, Flattened: false}
	if parentRow != nil && commentRow != nil && parentRow.URI != commentRow.URI {
		item.Flattened = true
		item.ReplyingTo = &ReplyingToAuthor{
			URI:         parentRow.URI,
			DID:         parentRow.DID,
			Handle:      handles[parentRow.DID].String(),
			DisplayName: parentRow.AuthorDisplayName,
		}
	}
	return item
}

type threadRowNode struct {
	row     *PostRow
	replies []*threadRowNode
}

func containsPostRow(rows []*PostRow, uri string) bool {
	for _, row := range rows {
		if row.URI == uri {
			return true
		}
	}
	return false
}

func buildThreadRowTree(rows []*PostRow, targetURI string) (*threadRowNode, []*PostRow, bool) {
	rowsByURI := make(map[string]*PostRow, len(rows))
	childrenByParent := make(map[string][]*PostRow)
	truncated := len(rows) > threadMaxPosts
	for _, row := range rows {
		if _, ok := rowsByURI[row.URI]; ok {
			continue
		}
		rowsByURI[row.URI] = row
		if row.URI != targetURI && row.ReplyParentURI != nil {
			childrenByParent[*row.ReplyParentURI] = append(childrenByParent[*row.ReplyParentURI], row)
		}
	}
	for parentURI := range childrenByParent {
		sort.SliceStable(childrenByParent[parentURI], func(i, j int) bool {
			left := childrenByParent[parentURI][i]
			right := childrenByParent[parentURI][j]
			if left.CreatedAt.Equal(right.CreatedAt) {
				return left.URI < right.URI
			}
			return left.CreatedAt.Before(right.CreatedAt)
		})
	}

	target := rowsByURI[targetURI]
	root := &threadRowNode{row: target, replies: []*threadRowNode{}}
	rendered := make([]*PostRow, 0, min(len(rows), threadMaxPosts))
	count := 0
	var addChildren func(node *threadRowNode, depth int)
	addChildren = func(node *threadRowNode, depth int) {
		if count >= threadMaxPosts {
			truncated = true
			return
		}
		rendered = append(rendered, node.row)
		count++
		children := childrenByParent[node.row.URI]
		if len(children) == 0 {
			return
		}
		if depth >= threadMaxDepth {
			truncated = true
			return
		}
		for _, childRow := range children {
			if _, ok := rowsByURI[childRow.URI]; !ok {
				continue
			}
			if count >= threadMaxPosts {
				truncated = true
				return
			}
			child := &threadRowNode{row: childRow, replies: []*threadRowNode{}}
			node.replies = append(node.replies, child)
			addChildren(child, depth+1)
		}
	}
	addChildren(root, 0)
	return root, rendered, truncated
}

func buildThreadResponse(root *threadRowNode, ancestors []*PostRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary, truncated bool) *ThreadResponse {
	var convert func(*threadRowNode) *ThreadNode
	convert = func(node *threadRowNode) *ThreadNode {
		post := BuildPostResponse(node.row, handles[node.row.DID])
		applyEngagementSummary(post, summaries[node.row.URI])
		out := &ThreadNode{Post: post, Replies: make([]*ThreadNode, 0, len(node.replies))}
		for _, child := range node.replies {
			out.Replies = append(out.Replies, convert(child))
		}
		return out
	}
	rootPost := BuildPostResponse(root.row, handles[root.row.DID])
	applyEngagementSummary(rootPost, summaries[root.row.URI])
	out := &ThreadResponse{
		Post:      rootPost,
		Ancestors: make([]*PostResponse, 0, len(ancestors)),
		Replies:   make([]*ThreadNode, 0, len(root.replies)),
		Truncated: truncated,
	}
	for _, row := range ancestors {
		post := BuildPostResponse(row, handles[row.DID])
		applyEngagementSummary(post, summaries[row.URI])
		out.Ancestors = append(out.Ancestors, post)
	}
	for _, child := range root.replies {
		out.Replies = append(out.Replies, convert(child))
	}
	return out
}

// parseLimit returns the validated limit, defaulting to 50 and capping
// at 100. Per pagination spec §5: caps are silent (we don't 400 on
// overshoot, we cap).
func parseLimit(raw string) int {
	const defaultLimit, maxLimit = 50, 100
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

func parseCommentLimit(raw string) int {
	limit := parseLimit(raw)
	if limit > 10 {
		return 10
	}
	return limit
}

func parseCommentSort(raw string) string {
	switch raw {
	case "newest", "follows":
		return raw
	default:
		return "oldest"
	}
}
