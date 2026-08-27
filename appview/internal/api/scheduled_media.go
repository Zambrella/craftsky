package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledMediaService interface {
	Put(context.Context, scheduledposts.PutPrivateMediaParams) (scheduledposts.PrivateMedia, error)
	Open(context.Context, syntax.DID, uuid.UUID) (scheduledposts.OpenedPrivateMedia, error)
	Delete(context.Context, syntax.DID, uuid.UUID, time.Time, ...int64) error
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
	validator ImageValidator,
	logger *slog.Logger,
) http.Handler {
	if validator == nil {
		panic("scheduled image validator is required")
	}
	if logger == nil {
		logger = slog.Default()
	}
	limits = normalizeMediaLimits(limits)
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, mediaID, ok := scheduledMediaIdentity(writer, request)
		if !ok {
			return
		}
		ownerGeneration, ok := scheduledMediaOwnerGeneration(writer, request)
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
		if _, err := validator.Validate(request.Context(), validated.ContentType, payload); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			writeScheduledMediaError(writer, request, err)
			return
		}
		media, err := service.Put(request.Context(), scheduledposts.PutPrivateMediaParams{
			ID: mediaID, OwnerDID: ownerDID, OwnerGeneration: ownerGeneration,
			MIMEType: validated.ContentType,
			Bytes:    payload, Now: time.Now().UTC(),
		})
		if err != nil {
			attributes := []any{slog.String("error_class", scheduledMediaErrorClass(err))}
			if errors.Is(err, scheduledposts.ErrScheduledMediaConflict) {
				attributes = append(attributes, slog.String(
					"conflict_stage", scheduledposts.ScheduledMediaConflictStage(err),
				))
			}
			logger.Warn("scheduled media upload failed", attributes...)
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
		ownerGeneration, ok := scheduledMediaOwnerGeneration(writer, request)
		if !ok {
			return
		}
		if err := service.Delete(request.Context(), ownerDID, mediaID, now().UTC(), ownerGeneration); err != nil {
			logger.Warn("scheduled media delete failed",
				slog.String("error_class", scheduledMediaErrorClass(err)))
			writeScheduledMediaError(writer, request, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
}

func scheduledMediaOwnerGeneration(
	writer http.ResponseWriter,
	request *http.Request,
) (int64, bool) {
	generation, ok := middleware.GetOwnerGeneration(request.Context())
	if !ok {
		envelope.WriteError(
			writer, http.StatusServiceUnavailable, "lifecycle_unavailable",
			"membership unavailable", middleware.GetRunID(request.Context()), nil,
		)
		return 0, false
	}
	return generation, true
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
	case errors.Is(err, ErrScheduledImageInvalid):
		envelope.WriteError(writer, http.StatusUnprocessableEntity, "scheduled_media_invalid", "scheduled media is invalid", runID, nil)
	case errors.Is(err, ErrImageDecodeSaturated):
		writer.Header().Set("Retry-After", "1")
		envelope.WriteError(writer, http.StatusServiceUnavailable, "scheduled_media_busy", "scheduled media validation is temporarily busy", runID, nil)
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
