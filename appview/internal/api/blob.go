package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

type ImageBlobUploadResponse struct {
	Blob map[string]any `json:"blob"`
	CID  string         `json:"cid"`
	MIME string         `json:"mime"`
	Size int64          `json:"size"`
}

// ImageBlobUploadHandler serves POST /v1/blobs/images.
func ImageBlobUploadHandler(newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())

		uploadReq, payload, err := DecodeImageBlobUpload(r.Header.Get("Content-Type"), r.Body)
		if err != nil {
			if fe, ok := err.(*FieldError); ok {
				switch fe.Code {
				case "malformed_body":
					envelope.WriteError(w, http.StatusBadRequest,
						"malformed_body", "could not parse body", runID, fe.Fields)
					return
				case "validation_failed":
					envelope.WriteError(w, http.StatusUnprocessableEntity,
						"validation_failed", "validation failed", runID, fe.Fields)
					return
				}
			}
			envelope.WriteError(w, http.StatusBadRequest,
				"malformed_body", "could not parse body", runID, nil)
			return
		}

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("blob upload: newPDS failed",
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}

		uploaded, err := pds.UploadBlob(r.Context(), uploadReq.ContentType, payload)
		if err != nil {
			logger.Warn("blob upload: UploadBlob failed",
				slog.String("did", did.String()),
				slog.String("mime", uploadReq.ContentType),
				slog.Int64("size", uploadReq.SizeBytes),
				slog.String("err", err.Error()),
				slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "could not upload image", runID, nil)
			return
		}

		resp := ImageBlobUploadResponse{
			Blob: uploaded.Raw,
			CID:  uploaded.CID,
			MIME: uploaded.MIME,
			Size: uploaded.Size,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
