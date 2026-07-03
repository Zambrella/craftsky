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
	"strconv"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/postutil"
)

const craftskyPostNSID = "social.craftsky.feed.post"
const craftskyLikeNSID = "social.craftsky.feed.like"
const craftskyRepostNSID = "social.craftsky.feed.repost"

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
	limits MediaLimits,
	logger *slog.Logger,
) http.Handler {
	limits = normalizeMediaLimits(limits)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("post create: request started",
			pdsLogAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild)...)

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
		if err := ValidatePostCreateWithLimits(req, limits); err != nil {
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
		logger.Debug("post create: validated request",
			pdsLogAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild)...)

		body := lexiconRecordBody(req)
		logger.Debug("post create: prepared PDS record",
			pdsLogAttrs(runID, pdsOperationPostCreate, pdsStageRequestBuild)...)

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), did, craftskyPostNSID, body)
		if err != nil {
			logger.Warn("post: CreateRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write post", runID, err)
			return
		}
		logger.Debug("post create: PDS record created",
			pdsLogSuccessAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest)...)

		row, err := syntheticPostRow(r, store, did, uri, cid, req)
		if err != nil {
			logger.Error("post: hydrate author failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post created but hydrate failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				pdsLogErrorAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest, err)...)
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		logger.Debug("post create: response ready",
			pdsLogSuccessAttrs(runID, pdsOperationPostCreate, pdsStagePDSRequest)...)
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
	if len(req.Images) > 0 {
		images := make([]map[string]any, 0, len(req.Images))
		for _, img := range req.Images {
			one := map[string]any{
				"image": img.Image,
			}
			if strings.TrimSpace(img.Alt) != "" {
				one["alt"] = strings.TrimSpace(img.Alt)
			}
			if img.AspectRatio != nil {
				one["aspectRatio"] = map[string]any{
					"width":  img.AspectRatio.Width,
					"height": img.AspectRatio.Height,
				}
			}
			images = append(images, one)
		}
		body["images"] = images
	}
	if req.Project != nil {
		body["project"] = req.Project
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
		Tags:      postutil.MergeTags(extractRequestTags(req.Text, req.Facets), requestProjectTags(req.Project)),
		CreatedAt: now,
		IndexedAt: now,
		Project:   req.Project,
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
	if len(req.Images) > 0 {
		imagesJSON, err := syntheticPostImagesJSON(req.Images)
		if err != nil {
			return nil, err
		}
		row.Images = imagesJSON
	}

	author, err := store.ReadAuthor(r.Context(), did.String())
	if err != nil {
		return nil, err
	}
	if author != nil {
		row.AuthorDisplayName = author.DisplayName
		row.AuthorAvatarCID = author.AvatarCID
		row.AuthorAvatarMime = author.AvatarMime
	}
	return row, nil
}

func requestProjectTags(project *Project) []string {
	if project == nil {
		return nil
	}
	return postutil.ExtractProjectTags(
		project.Common.Tags,
		requestPatternFacetedTexts(project.Common.Pattern),
		requestMaterialFacetedTexts(project.Common.Materials),
	)
}

func requestPatternFacetedTexts(pattern *ProjectPattern) []postutil.FacetedText {
	if pattern == nil {
		return nil
	}
	return []postutil.FacetedText{
		{Text: stringPtrValue(pattern.Name), Facets: postutil.DecodeFacets(pattern.NameFacets)},
		{Text: stringPtrValue(pattern.Designer), Facets: postutil.DecodeFacets(pattern.DesignerFacets)},
		{Text: stringPtrValue(pattern.Publisher), Facets: postutil.DecodeFacets(pattern.PublisherFacets)},
	}
}

func requestMaterialFacetedTexts(materials []ProjectMaterial) []postutil.FacetedText {
	out := make([]postutil.FacetedText, 0, len(materials))
	for _, material := range materials {
		out = append(out, postutil.FacetedText{
			Text:   material.Text,
			Facets: postutil.DecodeFacets(material.Facets),
		})
	}
	return out
}

func syntheticPostImagesJSON(images []PostImage) (json.RawMessage, error) {
	out := make([]storedPostImage, 0, len(images))
	for _, img := range images {
		ref, _ := img.Image["ref"].(map[string]any)
		cid, _ := ref["$link"].(string)
		mime, _ := img.Image["mimeType"].(string)
		out = append(out, storedPostImage{
			CID:         cid,
			MIME:        mime,
			Size:        int64FromJSONNumber(img.Image["size"]),
			Alt:         strings.TrimSpace(img.Alt),
			AspectRatio: img.AspectRatio,
		})
	}
	return json.Marshal(out)
}

func int64FromJSONNumber(value any) int64 {
	switch n := value.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		v, _ := n.Int64()
		return v
	default:
		return 0
	}
}

// extractRequestTags decodes the raw facets JSON into the indigo typed
// slice (best effort) and returns the same tag set the indexer would
// produce when this record arrives via the firehose. Returns an empty
// (non-nil) slice on decode failure so the response always carries a
// valid tags array.
func extractRequestTags(text string, raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	typed := postutil.DecodeFacets(raw)
	if len(typed) == 0 {
		return []string{}
	}
	return postutil.ExtractTagsForText(text, typed)
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
		viewerDID, _ := middleware.GetDID(r.Context())
		logger.Debug("post get: reading post",
			apiLogAttrs(runID, "post.get")...)
		row, err := store.ReadOne(r.Context(), did.String(), rkey)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"post_not_found", "post not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("post: ReadOne failed",
				apiLogErrorAttrs(runID, "post.get", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post read failed", runID, nil)
			return
		}
		summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), []string{row.URI})
		if err != nil {
			logger.Error("post: EngagementSummaries failed",
				apiLogErrorAttrs(runID, "post.get", "engagement")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			logger.Warn("post: ResolveHandle failed",
				apiLogErrorAttrs(runID, "post.get", "identity")...)
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		resp := BuildPostResponse(row, handle)
		applyEngagementSummary(resp, summaries[row.URI])
		logger.Debug("post get: response ready",
			apiLogSuccessAttrs(runID, "post.get")...)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

// ListCommentRepliesHandler serves GET /v1/posts/{did}/{rkey}/replies.
func ListCommentRepliesHandler(
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
				resp := BuildPostResponse(row, handles[row.DID])
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

		rootPost := BuildPostResponse(root, handles[root.DID])
		applyEngagementSummary(rootPost, summaries[root.URI])
		items := make([]CommentItem, 0, len(comments))
		if focusedCommentRow != nil && !containsPostRow(comments, focusedCommentRow.URI) {
			post := BuildPostResponse(focusedCommentRow, handles[focusedCommentRow.DID])
			applyEngagementSummary(post, summaries[focusedCommentRow.URI])
			replies := ReplyPage{Loaded: false, Items: []ReplyItem{}}
			if focusedReplyRow != nil {
				replies = buildReplyPage(focusedBranchRows, focusedBranchCursor, focusedCommentRow, hydratedRows, handles, summaries)
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
				replies = buildReplyPage(focusedBranchRows, focusedBranchCursor, row, hydratedRows, handles, summaries)
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
		logger.Debug("post delete: request started",
			pdsLogAttrs(runID, pdsOperationPostDelete, pdsStageRequestBuild)...)
		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("post: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationPostDelete, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		if err := pds.DeleteRecord(r.Context(), did, craftskyPostNSID, rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				logger.Debug("post delete: record already absent",
					pdsLogAttrs(runID, pdsOperationPostDelete, pdsStagePDSRequest)...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("post: DeleteRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationPostDelete, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		logger.Debug("post delete: PDS record deleted",
			pdsLogSuccessAttrs(runID, pdsOperationPostDelete, pdsStagePDSRequest)...)
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
		rkey := r.PathValue("rkey")
		logger.Debug("like: resolving target",
			pdsLogAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("like: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveLike(r.Context(), caller.String(), target.URI)
		if err == nil {
			logger.Debug("like: active like already exists",
				pdsLogSuccessAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild)...)
			writeInteractionResponse(w, http.StatusOK, interactionResponseFromRow(active))
			return
		}
		if !errors.Is(err, ErrInteractionNotFound) {
			logger.Error("like: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read like", runID, nil)
			return
		}

		createdAt := time.Now().UTC()
		body := likeRecordBody(target, createdAt)
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("like: creating PDS record",
			pdsLogAttrs(runID, pdsOperationLikeCreate, pdsStageRequestBuild)...)
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("like: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), caller, craftskyLikeNSID, body)
		if err != nil {
			logger.Warn("like: CreateRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeCreate, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write like", runID, err)
			return
		}
		logger.Debug("like: PDS record created",
			pdsLogSuccessAttrs(runID, pdsOperationLikeCreate, pdsStagePDSRequest)...)
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
		rkey := r.PathValue("rkey")
		logger.Debug("unlike: resolving target",
			pdsLogAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("unlike: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveLike(r.Context(), caller.String(), target.URI)
		if errors.Is(err, ErrInteractionNotFound) {
			logger.Debug("unlike: active like absent",
				pdsLogSuccessAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild)...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			logger.Error("unlike: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read like", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("unlike: deleting PDS record",
			pdsLogAttrs(runID, pdsOperationLikeDelete, pdsStageRequestBuild)...)
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("unlike: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		if err := pds.DeleteRecord(r.Context(), caller, craftskyLikeNSID, active.Rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				logger.Debug("unlike: PDS record already absent",
					pdsLogAttrs(runID, pdsOperationLikeDelete, pdsStagePDSRequest)...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("unlike: DeleteRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationLikeDelete, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		logger.Debug("unlike: PDS record deleted",
			pdsLogSuccessAttrs(runID, pdsOperationLikeDelete, pdsStagePDSRequest)...)
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
		rkey := r.PathValue("rkey")
		logger.Debug("repost: resolving target",
			pdsLogAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("repost: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveRepost(r.Context(), caller.String(), target.URI)
		if err == nil {
			logger.Debug("repost: active repost already exists",
				pdsLogSuccessAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild)...)
			writeInteractionResponse(w, http.StatusOK, interactionResponseFromRow(active))
			return
		}
		if !errors.Is(err, ErrInteractionNotFound) {
			logger.Error("repost: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read repost", runID, nil)
			return
		}

		createdAt := time.Now().UTC()
		body := repostRecordBody(target, createdAt)
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("repost: creating PDS record",
			pdsLogAttrs(runID, pdsOperationRepostCreate, pdsStageRequestBuild)...)
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("repost: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		uri, cid, err := pds.CreateRecord(r.Context(), caller, craftskyRepostNSID, body)
		if err != nil {
			logger.Warn("repost: CreateRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostCreate, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not write repost", runID, err)
			return
		}
		logger.Debug("repost: PDS record created",
			pdsLogSuccessAttrs(runID, pdsOperationRepostCreate, pdsStagePDSRequest)...)
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
		rkey := r.PathValue("rkey")
		logger.Debug("unrepost: resolving target",
			pdsLogAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild)...)
		target, err := store.ResolvePostTarget(r.Context(), targetDID.String(), rkey)
		if err != nil {
			if errors.Is(err, ErrPostNotFound) {
				envelope.WriteError(w, http.StatusNotFound,
					"post_not_found", "post not found", runID, nil)
				return
			}
			logger.Error("unrepost: resolve target failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not resolve post", runID, nil)
			return
		}
		active, err := store.FindActiveRepost(r.Context(), caller.String(), target.URI)
		if errors.Is(err, ErrInteractionNotFound) {
			logger.Debug("unrepost: active repost absent",
				pdsLogSuccessAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild)...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err != nil {
			logger.Error("unrepost: find active failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild, err)...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not read repost", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		logger.Debug("unrepost: deleting PDS record",
			pdsLogAttrs(runID, pdsOperationRepostDelete, pdsStageRequestBuild)...)
		pds, err := newPDS(r.Context(), caller, sessionID)
		if err != nil {
			logger.Error("unrepost: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}
		if err := pds.DeleteRecord(r.Context(), caller, craftskyRepostNSID, active.Rkey); err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				logger.Debug("unrepost: PDS record already absent",
					pdsLogAttrs(runID, pdsOperationRepostDelete, pdsStagePDSRequest)...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			logger.Warn("unrepost: DeleteRecord failed",
				pdsLogErrorAttrs(runID, pdsOperationRepostDelete, pdsStagePDSRequest, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "PDS delete failed", runID, err)
			return
		}
		logger.Debug("unrepost: PDS record deleted",
			pdsLogSuccessAttrs(runID, pdsOperationRepostDelete, pdsStagePDSRequest)...)
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
	return listAuthorPostsHandler(store, resolver, logger, "post list", store.ListByAuthor)
}

// ListProjectsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/projects.
func ListProjectsByAuthorHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return listAuthorPostsHandler(store, resolver, logger, "project list", store.ListProjectsByAuthor)
}

// ListCommentsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/comments.
func ListCommentsByAuthorHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return listAuthorPostsHandler(store, resolver, logger, "comment list", store.ListCommentsByAuthor)
}

func listAuthorPostsHandler(
	store PostReader,
	resolver HandleResolver,
	logger *slog.Logger,
	logLabel string,
	list func(context.Context, string, int, string) ([]*PostRow, string, error),
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		operation := postAuthorListOperation(logLabel)
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		logger.Debug(logLabel+": resolving author",
			apiLogAttrs(runID, operation)...)
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn(logLabel+": ResolveDID failed",
					apiLogErrorAttrs(runID, operation, "identity")...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		viewerDID, _ := middleware.GetDID(r.Context())
		logger.Debug(logLabel+": listing author records",
			append(apiLogAttrs(runID, operation),
				slog.Int("limit", limit),
				slog.Bool("has_cursor", cursor != ""))...)

		rows, nextCursor, err := list(r.Context(), did.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error(logLabel+": list failed",
				apiLogErrorAttrs(runID, operation, "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post list failed", runID, nil)
			return
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				postURIs = append(postURIs, row.URI)
			}
			summaries, serr := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if serr != nil {
				logger.Error(logLabel+": EngagementSummaries failed",
					apiLogErrorAttrs(runID, operation, "engagement")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			// Only pay handle-resolution cost when there are rows to render.
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				logger.Warn(logLabel+": ResolveHandle failed",
					apiLogErrorAttrs(runID, operation, "identity")...)
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
		logger.Debug(logLabel+": response ready",
			append(apiLogSuccessAttrs(runID, operation),
				slog.Int("rows", len(rows)),
				slog.Int("items", len(items)),
				slog.Bool("has_next_cursor", nextCursor != ""))...)

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

func resolveCommentAncestor(ctx context.Context, store PostReader, rootURI, parentURI string) (*PostRow, error) {
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

func buildReplyPage(rows []*PostRow, cursor string, commentRow *PostRow, hydratedRows []*PostRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary) ReplyPage {
	items := make([]ReplyItem, 0, len(rows))
	for _, row := range rows {
		var parentRow *PostRow
		if row.ReplyParentURI != nil && *row.ReplyParentURI != commentRow.URI {
			parentRow, _ = findPostRow(hydratedRows, *row.ReplyParentURI)
		}
		items = append(items, buildReplyItem(row, parentRow, commentRow, handles, summaries))
	}
	return ReplyPage{Loaded: true, Items: items, Cursor: cursor}
}

func buildReplyItem(row, parentRow, commentRow *PostRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary) ReplyItem {
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
