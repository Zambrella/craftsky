package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/linkpreview"
	"social.craftsky/appview/internal/middleware"
)

type LinkPreviewService interface {
	FetchPreview(context.Context, string) (linkpreview.Preview, error)
}

type LinkPreviewObserver interface {
	ObserveLinkPreview(stage, result, errorClass string, status, redirects, bytes int, duration time.Duration)
}

type linkPreviewResponse struct {
	URL         string                        `json:"url"`
	Title       string                        `json:"title"`
	Description string                        `json:"description"`
	Thumbnail   *linkPreviewThumbnailResponse `json:"thumbnail,omitempty"`
}

type linkPreviewThumbnailResponse struct {
	Bytes    string `json:"bytes"`
	MIMEType string `json:"mimeType"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func LinkPreviewHandler(service LinkPreviewService, enabled bool, observers ...LinkPreviewObserver) http.Handler {
	var observer LinkPreviewObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		runID := middleware.GetRunID(r.Context())
		if !enabled || service == nil {
			observeLinkPreview(observer, "admission", "disabled", "none", http.StatusServiceUnavailable, 0, started)
			envelope.WriteError(w, http.StatusServiceUnavailable, "link_preview_unavailable", "link previews are unavailable", runID, nil)
			return
		}
		request, err := DecodeLinkPreviewRequest(r.Body)
		if err != nil {
			observeLinkPreview(observer, "admission", "rejected", "validation", http.StatusBadRequest, 0, started)
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", runID, nil)
			return
		}
		target, err := linkpreview.ValidateURL(request.URL)
		if err != nil {
			observeLinkPreview(observer, "admission", "rejected", "policy", http.StatusUnprocessableEntity, 0, started)
			envelope.WriteError(w, http.StatusUnprocessableEntity, "link_preview_not_allowed", "link preview destination is not allowed", runID, nil)
			return
		}
		preview, err := service.FetchPreview(r.Context(), target.String())
		if err != nil {
			status, errorClass := linkPreviewErrorTelemetry(err)
			observeLinkPreview(observer, "fetch", "failed", errorClass, status, 0, started)
			writeLinkPreviewError(w, runID, err)
			return
		}
		response := linkPreviewResponse{
			URL: preview.URL.String(), Title: preview.Title, Description: preview.Description,
		}
		if preview.Thumbnail != nil {
			response.Thumbnail = &linkPreviewThumbnailResponse{
				Bytes:    base64.StdEncoding.EncodeToString(preview.Thumbnail.Bytes),
				MIMEType: preview.Thumbnail.MIMEType,
				Width:    preview.Thumbnail.Width,
				Height:   preview.Thumbnail.Height,
			}
		}
		bytes := 0
		if preview.Thumbnail != nil {
			bytes = len(preview.Thumbnail.Bytes)
		}
		observeLinkPreview(observer, "complete", "success", "none", http.StatusOK, bytes, started)
		envelope.WriteJSON(w, http.StatusOK, response)
	})
}

func observeLinkPreview(observer LinkPreviewObserver, stage, result, errorClass string, status, bytes int, started time.Time) {
	if observer != nil {
		observer.ObserveLinkPreview(stage, result, errorClass, status, 0, bytes, time.Since(started))
	}
}

func linkPreviewErrorTelemetry(err error) (int, string) {
	switch {
	case errors.Is(err, linkpreview.ErrNotAllowed):
		return http.StatusUnprocessableEntity, "policy"
	case errors.Is(err, linkpreview.ErrUnsupported):
		return http.StatusUnprocessableEntity, "response"
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "timeout"
	default:
		return http.StatusBadGateway, "upstream"
	}
}

func writeLinkPreviewError(w http.ResponseWriter, runID string, err error) {
	switch {
	case errors.Is(err, linkpreview.ErrNotAllowed):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "link_preview_not_allowed", "link preview destination is not allowed", runID, nil)
	case errors.Is(err, linkpreview.ErrUnsupported):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "link_preview_unsupported", "link preview response is unsupported", runID, nil)
	case errors.Is(err, context.DeadlineExceeded):
		envelope.WriteError(w, http.StatusGatewayTimeout, "link_preview_timeout", "link preview request timed out", runID, nil)
	default:
		envelope.WriteError(w, http.StatusBadGateway, "link_preview_upstream_failed", "link preview request failed", runID, nil)
	}
}
