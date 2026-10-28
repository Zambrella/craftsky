package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

// PostEditSnapshot is server-only. The raw source is never sent to an editor.
type PostEditSnapshot struct {
	Post     *PostRow
	Record   json.RawMessage
	Revision syntax.TID
}

// GetPostEditHandler serves the private owner-only editor seed.
func GetPostEditHandler(store readPostStore, resolver HandleResolver, load PostEditSeedLoader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		runID := middleware.GetRunID(r.Context())
		owner, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, 400, "invalid_identifier", "invalid owner", runID, nil)
			return
		}
		caller, ok := middleware.GetDID(r.Context())
		if !ok || caller != owner {
			envelope.WriteError(w, 403, "forbidden", "cannot edit this post", runID, nil)
			return
		}
		key, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			envelope.WriteError(w, 400, "invalid_identifier", "invalid record key", runID, nil)
			return
		}
		generation, ok := requirePDSEffectGeneration(w, r, runID)
		if !ok {
			return
		}
		session, _ := middleware.GetOAuthSessionID(r.Context())
		if load == nil {
			envelope.WriteError(w, 503, "pds_unavailable", "could not load post", runID, nil)
			return
		}
		snapshot, err := load(r.Context(), owner, key, generation, session)
		if errors.Is(err, ErrPostNotFound) {
			envelope.WriteError(w, 404, "post_not_found", "post not found", runID, nil)
			return
		}
		if errors.Is(err, ErrPostEditRecordInvalid) {
			envelope.WriteError(w, 422, "post_edit_invalid", "post cannot be edited", runID, nil)
			return
		}
		if errors.Is(err, auth.ErrPDSSessionExpired) || errors.Is(err, auth.ErrPDSPermissionsRequired) {
			writePDSError(w, 503, "pds_unavailable", "could not load post", runID, err)
			return
		}
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "query")
			writePDSError(w, 503, "pds_unavailable", "could not load post", runID, err)
			return
		}
		record, err := postEditEditable(snapshot.Record)
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "decode")
			envelope.WriteError(w, 500, "internal_error", "post read failed", runID, nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), owner)
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "identity")
			envelope.WriteError(w, 502, "identity_unavailable", "could not resolve handle", runID, nil)
			return
		}
		post := buildPostResponse(snapshot.Post, handle, store)
		summaries, err := store.EngagementSummaries(r.Context(), owner.String(), nil, []string{snapshot.Post.URI})
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "engagement")
			envelope.WriteError(w, 500, "internal_error", "post engagement lookup failed", runID, nil)
			return
		}
		applyEngagementSummary(post, summaries[snapshot.Post.URI])
		if err := attachQuoteView(r.Context(), store, resolver, post); err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetPostEditHandler", "quote")
			envelope.WriteError(w, 500, "internal_error", "post quote lookup failed", runID, nil)
			return
		}
		writeJSON(w, 200, struct {
			Post           *PostResponse `json:"post"`
			BaseCID        string        `json:"baseCid"`
			RecordRevision syntax.TID    `json:"recordRevision,omitempty"`
			Editable       any           `json:"editable"`
		}{post, snapshot.Post.CID, snapshot.Revision, record})
	})
}
