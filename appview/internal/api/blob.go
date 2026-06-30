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
func ImageBlobUploadHandler(newPDS auth.PDSClientFactory, limits MediaLimits, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
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
		logger.Debug("blob upload: request started",
			append(pdsLogAttrs(runID, pdsOperationBlobUpload, pdsStageRequestBuild),
				slog.String("content_type", r.Header.Get("Content-Type")),
				slog.Int64("content_length", r.ContentLength))...)

		uploadReq, payload, err := DecodeImageBlobUploadWithLimits(r.Header.Get("Content-Type"), r.Body, limits)
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
		logger.Debug("blob upload: validated payload",
			append(pdsLogAttrs(runID, pdsOperationBlobUpload, pdsStageRequestBuild),
				slog.String("content_type", uploadReq.ContentType),
				slog.Int64("size", uploadReq.SizeBytes))...)

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("blob upload: newPDS failed",
				pdsLogErrorAttrs(runID, pdsOperationBlobUpload, pdsStageSessionResume, err)...)
			writePDSError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, err)
			return
		}

		uploaded, err := pds.UploadBlob(r.Context(), uploadReq.ContentType, payload)
		if err != nil {
			logger.Warn("blob upload: UploadBlob failed",
				append(pdsLogErrorAttrs(runID, pdsOperationBlobUpload, pdsStagePDSRequest, err),
					slog.String("content_type", uploadReq.ContentType),
					slog.Int64("size", uploadReq.SizeBytes))...)
			writePDSError(w, http.StatusBadGateway,
				"pds_write_failed", "could not upload image", runID, err)
			return
		}

		resp := ImageBlobUploadResponse{
			Blob: uploaded.Raw,
			CID:  uploaded.CID,
			MIME: uploaded.MIME,
			Size: uploaded.Size,
		}
		logger.Debug("blob upload: uploaded to PDS",
			append(pdsLogSuccessAttrs(runID, pdsOperationBlobUpload, pdsStagePDSRequest),
				slog.String("content_type", uploaded.MIME),
				slog.Int64("size", uploaded.Size))...)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}
