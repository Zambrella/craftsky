package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/video"
)

type VideoUploadAuthorizationIssuer interface {
	IssueUpload(context.Context, syntax.DID, string) (video.UploadAuthorization, error)
}

type videoUploadAuthorizationResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

type VideoOperationObserver interface {
	ObserveVideoOperation(context.Context, string, string, string, time.Duration)
}

func VideoUploadAuthorizationHandler(issuer VideoUploadAuthorizationIssuer, _ *slog.Logger, observers ...VideoOperationObserver) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		result, reason := "success", "none"
		defer func() {
			if len(observers) > 0 && observers[0] != nil {
				observers[0].ObserveVideoOperation(r.Context(), "authorization", result, reason, time.Since(started))
			}
		}()
		w.Header().Set("Cache-Control", "no-store")
		runID := middleware.GetRunID(r.Context())
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			result, reason = "rejected", "invalid_request"
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "authenticated account unavailable", runID, nil)
			return
		}
		sessionID, ok := middleware.GetOAuthSessionID(r.Context())
		if !ok || sessionID == "" {
			result, reason = "rejected", "invalid_request"
			envelope.WriteError(w, http.StatusUnauthorized,
				"pds_session_expired", "PDS session expired", runID, nil)
			return
		}
		if issuer == nil {
			result, reason = "unavailable", "upstream"
			envelope.WriteError(w, http.StatusBadGateway,
				"video_service_unavailable", "could not authorize video upload", runID, nil)
			return
		}
		authorization, err := issuer.IssueUpload(r.Context(), owner, sessionID)
		if err != nil {
			result, reason = "unavailable", "upstream"
			envelope.WriteError(w, http.StatusBadGateway,
				"video_service_unavailable", "could not authorize video upload", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, videoUploadAuthorizationResponse{
			Token: authorization.Token, ExpiresAt: authorization.ExpiresAt.UTC().Format(time.RFC3339),
		})
	})
}
