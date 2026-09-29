package api

import (
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

func VideoLaunchDisabledHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteError(w, http.StatusServiceUnavailable, "video_disabled", "video is unavailable for this launch", middleware.GetRunID(r.Context()), nil)
	})
}
