package api

import (
	"context"
	"encoding/base64"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/linkpreview"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/profilelinks"
)

type ProfileLinkPreviewService interface {
	FetchProfileLinkPreview(context.Context, string) (linkpreview.ProfileLinkPreview, error)
}

func ProfileLinkPreviewHandler(service LinkPreviewService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		request, err := DecodeLinkPreviewRequest(r.Body)
		if err != nil {
			envelope.WriteError(w, 400, "invalid_request", "invalid profile link", runID, nil)
			return
		}
		target, err := profilelinks.Normalize(request.URL)
		if err != nil {
			envelope.WriteError(w, 422, "validation_failed", "invalid profile link", runID, nil)
			return
		}
		previewer, ok := service.(ProfileLinkPreviewService)
		if !ok {
			envelope.WriteJSON(w, 200, map[string]any{"reachability": "unknown"})
			return
		}
		preview, err := previewer.FetchProfileLinkPreview(r.Context(), target)
		response := map[string]any{"reachability": "unreachable"}
		if err == nil && preview.Reachable {
			response["reachability"] = "reachable"
		}
		if preview.Icon != nil {
			response["icon"] = linkPreviewThumbnailResponse{Bytes: base64.StdEncoding.EncodeToString(preview.Icon.Bytes), MIMEType: preview.Icon.MIMEType, Width: preview.Icon.Width, Height: preview.Icon.Height}
		}
		w.Header().Set("Cache-Control", "no-store")
		envelope.WriteJSON(w, 200, response)
	})
}
