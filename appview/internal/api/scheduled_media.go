package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledMediaService interface {
	Put(context.Context, scheduledposts.PutPrivateMediaParams) (scheduledposts.PrivateMedia, error)
	Open(context.Context, syntax.DID, uuid.UUID) (scheduledposts.OpenedPrivateMedia, error)
	Delete(context.Context, syntax.DID, uuid.UUID, time.Time) error
}

type scheduledMediaResponse struct {
	ID        string `json:"id"`
	State     string `json:"state"`
	MIMEType  string `json:"mimeType"`
	SizeBytes int64  `json:"sizeBytes"`
	BlobCID   string `json:"blobCid"`
}

func PutScheduledMediaHandler(
	service scheduledMediaService,
	limits MediaLimits,
	logger *slog.Logger,
) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	limits = normalizeMediaLimits(limits)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, mediaID, ok := scheduledMediaIdentity(writer, request)
		if !ok {
			return
		}
		validated, payload, err := DecodeImageBlobUploadWithLimits(
			request.Header.Get("Content-Type"),
			request.Body,
			limits,
		)
		if err != nil {
			writeScheduledMediaError(writer, request, err)
			return
		}
		if err := validateScheduledImageContent(validated.ContentType, payload); err != nil {
			writeScheduledMediaError(writer, request, err)
			return
		}
		media, err := service.Put(request.Context(), scheduledposts.PutPrivateMediaParams{
			ID: mediaID, OwnerDID: ownerDID, MIMEType: validated.ContentType,
			Bytes: payload, Now: time.Now().UTC(),
		})
		if err != nil {
			logger.Warn("scheduled media upload failed",
				slog.String("error_class", scheduledMediaErrorClass(err)))
			writeScheduledMediaError(writer, request, err)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(writer).Encode(scheduledMediaResponse{
			ID: media.ID.String(), State: media.State, MIMEType: media.MIMEType,
			SizeBytes: media.SizeBytes, BlobCID: media.BlobCID.String(),
		})
	})
}

func validateScheduledImageContent(contentType string, payload []byte) error {
	fields := map[string]string{}
	if len(payload) == 0 {
		fields["body"] = "must contain an image"
	} else {
		_, format, err := image.Decode(bytes.NewReader(payload))
		decodedContentType := map[string]string{
			"jpeg": "image/jpeg",
			"png":  "image/png",
			"webp": "image/webp",
		}[format]
		if err != nil || decodedContentType == "" {
			fields["body"] = "must contain a valid image"
		} else if decodedContentType != contentType {
			fields["body"] = "must match the declared image type"
		}
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func GetScheduledMediaHandler(service scheduledMediaService, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, mediaID, ok := scheduledMediaIdentity(writer, request)
		if !ok {
			return
		}
		media, err := service.Open(request.Context(), ownerDID, mediaID)
		if err != nil {
			logger.Warn("scheduled media read failed",
				slog.String("error_class", scheduledMediaErrorClass(err)))
			writeScheduledMediaError(writer, request, err)
			return
		}
		defer media.Body.Close()
		writer.Header().Set("Content-Type", media.MIMEType)
		writer.Header().Set("Content-Length", strconv.FormatInt(media.SizeBytes, 10))
		writer.Header().Set("Cache-Control", "private, no-store")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.WriteHeader(http.StatusOK)
		_, _ = io.Copy(writer, media.Body)
	})
}

func DeleteScheduledMediaHandler(
	service scheduledMediaService,
	now func() time.Time,
	logger *slog.Logger,
) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, mediaID, ok := scheduledMediaIdentity(writer, request)
		if !ok {
			return
		}
		if err := service.Delete(request.Context(), ownerDID, mediaID, now().UTC()); err != nil {
			logger.Warn("scheduled media delete failed",
				slog.String("error_class", scheduledMediaErrorClass(err)))
			writeScheduledMediaError(writer, request, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
}

func scheduledMediaIdentity(
	writer http.ResponseWriter,
	request *http.Request,
) (syntax.DID, uuid.UUID, bool) {
	ownerDID, ok := middleware.GetDID(request.Context())
	if !ok {
		envelope.WriteError(
			writer, http.StatusInternalServerError, "internal_error",
			"authenticated account unavailable", middleware.GetRunID(request.Context()), nil,
		)
		return "", uuid.Nil, false
	}
	mediaID, err := uuid.Parse(request.PathValue("mediaId"))
	if err != nil {
		envelope.WriteError(
			writer, http.StatusBadRequest, "invalid_media_id",
			"media ID is invalid", middleware.GetRunID(request.Context()), nil,
		)
		return "", uuid.Nil, false
	}
	return ownerDID, mediaID, true
}

func writeScheduledMediaError(
	writer http.ResponseWriter,
	request *http.Request,
	err error,
) {
	runID := middleware.GetRunID(request.Context())
	var fieldError *FieldError
	switch {
	case errors.As(err, &fieldError):
		status := http.StatusUnprocessableEntity
		if fieldError.Code == "malformed_body" {
			status = http.StatusBadRequest
		}
		envelope.WriteError(writer, status, "scheduled_media_invalid", "scheduled media is invalid", runID, fieldError.Fields)
	case errors.Is(err, scheduledposts.ErrScheduledMediaNotFound):
		envelope.WriteError(writer, http.StatusNotFound, "scheduled_media_not_found", "scheduled media not found", runID, nil)
	case errors.Is(err, scheduledposts.ErrScheduledMediaConflict):
		envelope.WriteError(writer, http.StatusConflict, "scheduled_media_conflict", "scheduled media conflicts with an existing upload", runID, nil)
	case errors.Is(err, scheduledposts.ErrMediaInvalid):
		envelope.WriteError(writer, http.StatusUnprocessableEntity, "scheduled_media_invalid", "scheduled media is invalid", runID, nil)
	case errors.Is(err, scheduledposts.ErrPrivateObjectStoreUnavailable):
		envelope.WriteError(writer, http.StatusServiceUnavailable, "scheduled_media_unavailable", "scheduled media is temporarily unavailable", runID, nil)
	default:
		envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "scheduled media request failed", runID, nil)
	}
}

func scheduledMediaErrorClass(err error) string {
	switch {
	case errors.Is(err, scheduledposts.ErrScheduledMediaNotFound):
		return "not_found"
	case errors.Is(err, scheduledposts.ErrScheduledMediaConflict):
		return "conflict"
	case errors.Is(err, scheduledposts.ErrMediaInvalid):
		return "invalid"
	case errors.Is(err, scheduledposts.ErrPrivateObjectStoreUnavailable):
		return "dependency_unavailable"
	default:
		return "unknown"
	}
}
