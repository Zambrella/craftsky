// appview/internal/api/post_read.go
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type readPostStore interface {
	postByKeyReader
	relationshipStateReader
	engagementSummaryReader
	postQuoteHydrationStore
}

// GetPostHandler serves GET /v1/posts/{did}/{rkey}.
func GetPostHandler(store readPostStore, resolver HandleResolver, logger *slog.Logger) http.Handler {
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
		relationshipState, err := store.RelationshipState(r.Context(), viewerDID, did)
		if errors.Is(err, relationships.ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"profile_not_found", "profile not found", runID, nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post relationship lookup failed", runID, nil)
			return
		}
		if relationshipState.HasBlock() {
			resp := buildPostResponse(row, "", store)
			ApplyPostRelationshipPolicy(resp, relationshipState, relationships.SurfaceDirectPost)
			writeJSON(w, http.StatusOK, resp)
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
		resp := buildPostResponse(row, handle, store)
		ApplyPostAuthorViewerState(resp, relationshipState)
		applyEngagementSummary(resp, summaries[row.URI])
		if err := attachQuoteView(r.Context(), store, resolver, resp); err != nil {
			logger.Error("post: QuoteViewRows failed",
				apiLogErrorAttrs(runID, "post.get", "quote_view")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post quote lookup failed", runID, nil)
			return
		}
		logger.Debug("post get: response ready",
			apiLogSuccessAttrs(runID, "post.get")...)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
