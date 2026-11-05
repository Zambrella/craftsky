package api

import (
	"context"
	"errors"
	"net/http"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/video"
	"time"
)

func BunnyVideoStatusHandler(service *video.Bunny) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		runID := middleware.GetRunID(r.Context())
		owner, _ := middleware.GetDID(r.Context())
		session, _ := middleware.GetOAuthSessionID(r.Context())
		if session == "" {
			envelope.WriteError(w, 401, "pds_session_expired", "PDS session expired", runID, nil)
			return
		}
		if service == nil {
			envelope.WriteError(w, 503, "video_service_unavailable", "video service unavailable", runID, nil)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 100*time.Second)
		defer cancel()
		status, err := service.Status(ctx, owner, session, r.PathValue("jobId"))
		if errors.Is(err, video.ErrVideoJobNotFound) {
			envelope.WriteError(w, 404, "video_job_not_found", "video job not found", runID, nil)
			return
		}
		if errors.Is(err, auth.ErrPDSSessionExpired) || errors.Is(err, auth.ErrPDSPermissionsRequired) {
			writePDSError(w, 401, "pds_session_expired", pdsSessionExpiredMessage, runID, err)
			return
		}
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.BunnyVideoStatusHandler", "handler")
			envelope.WriteError(w, 502, "video_service_unavailable", "video service unavailable", runID, nil)
			return
		}
		envelope.WriteJSON(w, 200, map[string]any{"jobStatus": status})
	})
}
