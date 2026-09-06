// appview/internal/api/post_conversation.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type postByURIReader interface {
	ReadPostByURI(context.Context, string) (*PostRow, error)
}

type commentRepliesStore interface {
	postByKeyReader
	postByURIReader
	engagementSummaryReader
	postQuoteHydrationStore
	ListCommentBranchReplies(context.Context, string, string, int, string) ([]*PostRow, string, error)
}

type postCommentsStore interface {
	postByKeyReader
	postByURIReader
	relationshipStateReader
	engagementSummaryReader
	postQuoteHydrationStore
	ListRootComments(context.Context, string, string, string, int, string) ([]*PostRow, string, error)
	ListCommentBranchReplies(context.Context, string, string, int, string) ([]*PostRow, string, error)
	ListCommentBranchRepliesAround(context.Context, string, string, string, int) ([]*PostRow, string, error)
}

// ListCommentRepliesHandler serves GET /v1/posts/{did}/{rkey}/replies.
func ListCommentRepliesHandler(
	store commentRepliesStore,
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
		logger.Debug("post replies: resolving target",
			apiLogAttrs(runID, "post.replies.list")...)
		target, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post replies: ReadOne failed",
				apiLogErrorAttrs(runID, "post.replies.list", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		if !target.IsComment() {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_post_role", "post must be a top-level comment", runID, nil)
			return
		}

		limit := parseCommentLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		logger.Debug("post replies: listing branch replies",
			append(apiLogAttrs(runID, "post.replies.list"),
				slog.Int("limit", limit))...)
		rows, nextCursor, err := store.ListCommentBranchReplies(r.Context(), target.URI, *target.ReplyRootURI, limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("post replies: ListCommentBranchReplies failed",
				apiLogErrorAttrs(runID, "post.replies.list", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "reply list failed", runID, nil)
			return
		}

		items := make([]ReplyItem, 0, len(rows))
		if len(rows) > 0 {
			viewerDID, _ := middleware.GetDID(r.Context())
			postURIs := make([]string, 0, len(rows))
			hydratedRows := make([]*PostRow, 0, len(rows)*2)
			for _, row := range rows {
				postURIs = append(postURIs, row.URI)
				hydratedRows = append(hydratedRows, row)
				if row.ReplyParentURI != nil && *row.ReplyParentURI != target.URI {
					parentRow, perr := store.ReadPostByURI(r.Context(), *row.ReplyParentURI)
					if perr != nil && !errors.Is(perr, ErrPostNotFound) {
						logger.Error("post replies: ReadPostByURI parent failed",
							apiLogErrorAttrs(runID, "post.replies.list", "store")...)
						envelope.WriteError(w, http.StatusInternalServerError,
							"internal_error", "reply parent lookup failed", runID, nil)
						return
					}
					if parentRow != nil {
						hydratedRows = append(hydratedRows, parentRow)
					}
				}
			}
			states, stateErr := relationshipStatesForRows(r.Context(), store, viewerDID, hydratedRows)
			if stateErr != nil {
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "reply relationship lookup failed", runID, nil)
				return
			}
			summaries, serr := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if serr != nil {
				logger.Error("post replies: EngagementSummaries failed",
					apiLogErrorAttrs(runID, "post.replies.list", "engagement")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			handles, herr := resolveHandlesForRows(r.Context(), hydratedRows, resolver)
			if herr != nil {
				logger.Warn("post replies: ResolveHandle failed",
					apiLogErrorAttrs(runID, "post.replies.list", "identity")...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				resp := buildPostResponse(row, handles[row.DID], store)
				applyEngagementSummary(resp, summaries[row.URI])
				item := ReplyItem{Post: resp, Flattened: false}
				if row.ReplyParentURI != nil && *row.ReplyParentURI != target.URI {
					item.Flattened = true
					parentRow, _ := findPostRow(hydratedRows, *row.ReplyParentURI)
					if parentRow != nil {
						item.ReplyingTo = &ReplyingToAuthor{
							URI:         parentRow.URI,
							DID:         parentRow.DID,
							Handle:      handles[parentRow.DID].String(),
							DisplayName: parentRow.AuthorDisplayName,
						}
					}
				}
				items = append(items, item)
			}
			items = shapeReplyItems(items, rows, states)
			responses := make([]*PostResponse, 0, len(items))
			for i := range items {
				responses = append(responses, items[i].Post)
			}
			if err := attachQuoteViews(r.Context(), store, resolver, responses); err != nil {
				logger.Error("post replies: QuoteViewRows failed",
					apiLogErrorAttrs(runID, "post.replies.list", "quote_view")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post quote lookup failed", runID, nil)
				return
			}
		}
		body := ReplyPage{Loaded: true, Items: items, Cursor: nextCursor}
		logger.Debug("post replies: response ready",
			append(apiLogSuccessAttrs(runID, "post.replies.list"),
				slog.Int("rows", len(rows)),
				slog.Int("items", len(items)))...)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// GetPostCommentsHandler serves GET /v1/posts/{did}/{rkey}/comments.
func GetPostCommentsHandler(
	store postCommentsStore,
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
		logger.Debug("post comments: resolving root",
			apiLogAttrs(runID, "post.comments.list")...)
		root, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post comments: ReadOne failed",
				apiLogErrorAttrs(runID, "post.comments.list", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}
		if !root.IsRoot() {
			envelope.WriteError(w, http.StatusBadRequest,
				"invalid_post_role", "post must be a root post", runID, nil)
			return
		}

		viewerDID, _ := middleware.GetDID(r.Context())
		rootRelationship, err := store.RelationshipState(r.Context(), viewerDID, did)
		if errors.Is(err, relationships.ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound, "profile_not_found", "profile not found", runID, nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "post relationship lookup failed", runID, nil)
			return
		}
		if rootRelationship.HasBlock() {
			rootPost := buildPostResponse(root, "", store)
			ApplyPostRelationshipPolicy(rootPost, rootRelationship, relationships.SurfaceDirectPost)
			writeJSON(w, http.StatusOK, &CommentSectionResponse{
				Post: rootPost, Comments: CommentPage{Items: []CommentItem{}}, Sort: parseCommentSort(r.URL.Query().Get("sort")),
			})
			return
		}
		sortValue := parseCommentSort(r.URL.Query().Get("sort"))
		limit := parseCommentLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		logger.Debug("post comments: listing root comments",
			append(apiLogAttrs(runID, "post.comments.list"),
				slog.String("sort", sortValue),
				slog.Int("limit", limit),
				slog.Bool("has_cursor", cursor != ""),
				slog.Bool("has_focus", r.URL.Query().Get("focus") != ""))...)
		focus := (*FocusContext)(nil)
		focusedURI := ""
		var focusedCommentRow *PostRow
		var focusedReplyRow *PostRow
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
					apiLogErrorAttrs(runID, "post.comments.list", "store")...)
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
						commentRow, cerr := resolveCommentAncestor(r.Context(), store, root.URI, *focusedRow.ReplyParentURI)
						if cerr != nil {
							logger.Error("post comments: resolve focus ancestor failed",
								apiLogErrorAttrs(runID, "post.comments.list", "store")...)
							envelope.WriteError(w, http.StatusInternalServerError,
								"internal_error", "focus ancestor read failed", runID, nil)
							return
						}
						if commentRow != nil {
							focus.Status = "included"
							focus.Kind = "reply"
							focus.CommentURI = commentRow.URI
							focusedURI = commentRow.URI
							focusedCommentRow = commentRow
							focusedReplyRow = focusedRow
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
				apiLogErrorAttrs(runID, "post.comments.list", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "comment list failed", runID, nil)
			return
		}

		var focusedBranchRows []*PostRow
		focusedBranchCursor := ""
		focusedBranchParentRows := []*PostRow{}
		if focusedReplyRow != nil && focusedCommentRow != nil {
			focusedBranchRows, focusedBranchCursor, err = store.ListCommentBranchReplies(r.Context(), focusedCommentRow.URI, root.URI, parseCommentLimit(""), "")
			if err != nil {
				logger.Error("post comments: ListCommentBranchReplies failed",
					apiLogErrorAttrs(runID, "post.comments.list", "store")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "reply list failed", runID, nil)
				return
			}
			if !containsPostRow(focusedBranchRows, focusedReplyRow.URI) {
				focusedBranchRows, focusedBranchCursor, err = store.ListCommentBranchRepliesAround(r.Context(), focusedCommentRow.URI, root.URI, focusedReplyRow.URI, parseCommentLimit(""))
				if err != nil {
					logger.Error("post comments: ListCommentBranchRepliesAround failed",
						apiLogErrorAttrs(runID, "post.comments.list", "store")...)
					envelope.WriteError(w, http.StatusInternalServerError,
						"internal_error", "reply list failed", runID, nil)
					return
				}
			}
			for _, row := range focusedBranchRows {
				if row.ReplyParentURI == nil || *row.ReplyParentURI == focusedCommentRow.URI || containsPostRow(focusedBranchRows, *row.ReplyParentURI) || containsPostRow(focusedBranchParentRows, *row.ReplyParentURI) {
					continue
				}
				parentRow, perr := store.ReadPostByURI(r.Context(), *row.ReplyParentURI)
				if perr != nil && !errors.Is(perr, ErrPostNotFound) {
					logger.Error("post comments: ReadPostByURI branch parent failed",
						apiLogErrorAttrs(runID, "post.comments.list", "store")...)
					envelope.WriteError(w, http.StatusInternalServerError,
						"internal_error", "reply parent lookup failed", runID, nil)
					return
				}
				if parentRow != nil {
					focusedBranchParentRows = append(focusedBranchParentRows, parentRow)
				}
			}
		}

		hydratedRows := append([]*PostRow{root}, comments...)
		if focusedCommentRow != nil && !containsPostRow(comments, focusedCommentRow.URI) {
			hydratedRows = append(hydratedRows, focusedCommentRow)
		}
		hydratedRows = append(hydratedRows, focusedBranchRows...)
		hydratedRows = append(hydratedRows, focusedBranchParentRows...)
		states, err := relationshipStatesForRows(r.Context(), store, viewerDID, hydratedRows)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "thread relationship lookup failed", runID, nil)
			return
		}
		postURIs := make([]string, 0, len(hydratedRows))
		for _, row := range hydratedRows {
			postURIs = append(postURIs, row.URI)
		}
		summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
		if err != nil {
			logger.Error("post comments: EngagementSummaries failed",
				apiLogErrorAttrs(runID, "post.comments.list", "engagement")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		handles, err := resolveHandlesForRows(r.Context(), hydratedRows, resolver)
		if err != nil {
			logger.Warn("post comments: ResolveHandle failed",
				apiLogErrorAttrs(runID, "post.comments.list", "identity")...)
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}

		rootPost := buildPostResponse(root, handles[root.DID], store)
		applyEngagementSummary(rootPost, summaries[root.URI])
		items := make([]CommentItem, 0, len(comments))
		if focusedCommentRow != nil && !containsPostRow(comments, focusedCommentRow.URI) {
			post := buildPostResponse(focusedCommentRow, handles[focusedCommentRow.DID], store)
			applyEngagementSummary(post, summaries[focusedCommentRow.URI])
			replies := ReplyPage{Loaded: false, Items: []ReplyItem{}}
			if focusedReplyRow != nil {
				replies = buildReplyPage(focusedBranchRows, focusedBranchCursor, focusedCommentRow, hydratedRows, handles, summaries, store)
				replies.Items = shapeReplyItems(replies.Items, focusedBranchRows, states)
			}
			items = append(items, CommentItem{
				Post:      post,
				Placement: "focused",
				Replies:   replies,
			})
		}
		for _, row := range comments {
			post := buildPostResponse(row, handles[row.DID], store)
			applyEngagementSummary(post, summaries[row.URI])
			placement := "normal"
			if focusedURI != "" && row.URI == focusedURI {
				placement = "focused"
			}
			replies := ReplyPage{Loaded: false, Items: []ReplyItem{}}
			if focusedReplyRow != nil && row.URI == focusedURI {
				replies = buildReplyPage(focusedBranchRows, focusedBranchCursor, row, hydratedRows, handles, summaries, store)
				replies.Items = shapeReplyItems(replies.Items, focusedBranchRows, states)
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
		body.Comments.Items = shapeCommentItems(body.Comments.Items, hydratedRows, states)
		if err := attachQuoteViews(r.Context(), store, resolver, collectCommentSectionPostResponses(body)); err != nil {
			logger.Error("post comments: QuoteViewRows failed",
				apiLogErrorAttrs(runID, "post.comments.list", "quote_view")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post quote lookup failed", runID, nil)
			return
		}
		logger.Debug("post comments: response ready",
			append(apiLogSuccessAttrs(runID, "post.comments.list"),
				slog.Int("comments", len(comments)),
				slog.Int("items", len(items)),
				slog.Bool("has_next_cursor", nextCursor != ""),
				slog.Bool("has_focus", focus != nil))...)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

func relationshipStatesForRows(
	ctx context.Context,
	store relationshipStatesReader,
	viewer syntax.DID,
	rows []*PostRow,
) (map[syntax.DID]relationships.State, error) {
	seen := make(map[syntax.DID]struct{})
	subjects := make([]syntax.DID, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		did, err := syntax.ParseDID(row.DID)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[did]; ok {
			continue
		}
		seen[did] = struct{}{}
		subjects = append(subjects, did)
	}
	return store.RelationshipStates(ctx, viewer, subjects)
}

func shapeReplyItems(items []ReplyItem, rows []*PostRow, states map[syntax.DID]relationships.State) []ReplyItem {
	out := make([]ReplyItem, 0, len(items))
	protected := make(map[string]bool)
	for i, item := range items {
		if i >= len(rows) || item.Post == nil {
			continue
		}
		row := rows[i]
		if row.ReplyParentURI != nil && protected[*row.ReplyParentURI] {
			protected[row.URI] = true
			continue
		}
		did, _ := syntax.ParseDID(row.DID)
		state := states[did]
		if state.HasBlock() {
			protected[row.URI] = true
			continue
		}
		if state.Muted {
			ApplyPostRelationshipPolicy(item.Post, state, relationships.SurfaceThread)
			item.ReplyingTo = nil
			protected[row.URI] = true
		}
		if item.ReplyingTo != nil {
			parentDID, _ := syntax.ParseDID(item.ReplyingTo.DID)
			if states[parentDID].HasBlock() {
				item.ReplyingTo = nil
			}
		}
		out = append(out, item)
	}
	return out
}

func shapeCommentItems(items []CommentItem, rows []*PostRow, states map[syntax.DID]relationships.State) []CommentItem {
	out := make([]CommentItem, 0, len(items))
	rowsByURI := make(map[string]*PostRow, len(rows))
	for _, row := range rows {
		rowsByURI[row.URI] = row
	}
	for _, item := range items {
		if item.Post == nil {
			continue
		}
		row := rowsByURI[item.Post.URI]
		if row == nil {
			continue
		}
		did, _ := syntax.ParseDID(row.DID)
		state := states[did]
		if state.HasBlock() {
			continue
		}
		if state.Muted {
			ApplyPostRelationshipPolicy(item.Post, state, relationships.SurfaceThread)
			item.Replies = ReplyPage{Loaded: false, Items: []ReplyItem{}}
		}
		out = append(out, item)
	}
	return out
}

func resolveCommentAncestor(ctx context.Context, store postByURIReader, rootURI, parentURI string) (*PostRow, error) {
	seen := make(map[string]struct{})
	for uri := parentURI; uri != "" && len(seen) < 64; {
		if _, ok := seen[uri]; ok {
			return nil, nil
		}
		seen[uri] = struct{}{}

		row, err := store.ReadPostByURI(ctx, uri)
		if errors.Is(err, ErrPostNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if row.ReplyParentURI != nil && *row.ReplyParentURI == rootURI {
			return row, nil
		}
		if row.ReplyRootURI == nil || *row.ReplyRootURI != rootURI || row.ReplyParentURI == nil {
			return nil, nil
		}
		uri = *row.ReplyParentURI
	}
	return nil, nil
}

func buildReplyPage(rows []*PostRow, cursor string, commentRow *PostRow, hydratedRows []*PostRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary, playbackSource any) ReplyPage {
	items := make([]ReplyItem, 0, len(rows))
	for _, row := range rows {
		var parentRow *PostRow
		if row.ReplyParentURI != nil && *row.ReplyParentURI != commentRow.URI {
			parentRow, _ = findPostRow(hydratedRows, *row.ReplyParentURI)
		}
		items = append(items, buildReplyItem(row, parentRow, commentRow, handles, summaries, playbackSource))
	}
	return ReplyPage{Loaded: true, Items: items, Cursor: cursor}
}

func collectCommentSectionPostResponses(body *CommentSectionResponse) []*PostResponse {
	if body == nil {
		return nil
	}
	responses := make([]*PostResponse, 0, 1+len(body.Comments.Items))
	if body.Post != nil {
		responses = append(responses, body.Post)
	}
	for i := range body.Comments.Items {
		item := &body.Comments.Items[i]
		if item.Post != nil {
			responses = append(responses, item.Post)
		}
		responses = append(responses, collectReplyPagePostResponses(item.Replies)...)
	}
	return responses
}

func collectReplyPagePostResponses(page ReplyPage) []*PostResponse {
	responses := make([]*PostResponse, 0, len(page.Items))
	for i := range page.Items {
		if page.Items[i].Post != nil {
			responses = append(responses, page.Items[i].Post)
		}
	}
	return responses
}

func buildReplyItem(row, parentRow, commentRow *PostRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary, playbackSource any) ReplyItem {
	post := buildPostResponse(row, handles[row.DID], playbackSource)
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

func containsPostRow(rows []*PostRow, uri string) bool {
	for _, row := range rows {
		if row.URI == uri {
			return true
		}
	}
	return false
}

func findPostRow(rows []*PostRow, uri string) (*PostRow, bool) {
	for _, row := range rows {
		if row.URI == uri {
			return row, true
		}
	}
	return nil, false
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
