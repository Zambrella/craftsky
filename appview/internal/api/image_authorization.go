package api

import (
	"context"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/imageupload"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

type ImageUploadAuthorizationIssuer interface {
	IssueUpload(context.Context, syntax.DID, string) (imageupload.Authorization, error)
}

func ImageUploadAuthorizationHandler(issuer ImageUploadAuthorizationIssuer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		runID := middleware.GetRunID(r.Context())
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "authenticated account unavailable", runID, nil)
			return
		}
		sessionID, ok := middleware.GetOAuthSessionID(r.Context())
		if !ok || sessionID == "" {
			envelope.WriteError(w, http.StatusUnauthorized, "pds_session_expired", "PDS session expired", runID, nil)
			return
		}
		if issuer == nil {
			envelope.WriteError(w, http.StatusBadGateway, "image_upload_authorization_unavailable", "could not authorize image upload", runID, nil)
			return
		}
		authorization, err := issuer.IssueUpload(r.Context(), owner, sessionID)
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.ImageUploadAuthorizationHandler", "handler")
			envelope.WriteError(w, http.StatusBadGateway, "image_upload_authorization_unavailable", "could not authorize image upload", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, struct {
			Token          string `json:"token"`
			ExpiresAt      string `json:"expiresAt"`
			UploadEndpoint string `json:"uploadEndpoint"`
		}{authorization.Token, authorization.ExpiresAt.UTC().Format(time.RFC3339), authorization.UploadEndpoint})
	})
}
