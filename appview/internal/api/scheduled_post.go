package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

type scheduledPostCreator interface {
	Create(context.Context, scheduledposts.CreateParams) (scheduledposts.ScheduledPost, error)
}

type scheduledPostLister interface {
	List(context.Context, syntax.DID) ([]scheduledposts.Resource, error)
}

type scheduledPostReader interface {
	Get(context.Context, syntax.DID, uuid.UUID) (scheduledposts.Resource, error)
}

type scheduledPostUpdater interface {
	scheduledPostReader
	Update(context.Context, scheduledposts.UpdateParams) (scheduledposts.UpdateResult, error)
}

type scheduledPostDeleter interface {
	Delete(context.Context, syntax.DID, uuid.UUID, time.Time) error
}

func CreateScheduledPostHandler(
	store scheduledPostCreator,
	limits MediaLimits,
	now func() time.Time,
	logger *slog.Logger,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, ok := middleware.GetDID(request.Context())
		if !ok {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "authenticated account unavailable", middleware.GetRunID(request.Context()), nil)
			return
		}
		decoded, err := decodeScheduledPostCreate(request.Body)
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "malformed_body", "could not parse body", middleware.GetRunID(request.Context()), nil)
			return
		}
		acceptedAt := now().UTC()
		operationID, mediaIDs, payloadBytes, err := validateScheduledPostRequest(acceptedAt, decoded, limits)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		payloadHash := sha256.Sum256(payloadBytes)
		requestHashInput, _ := json.Marshal(struct {
			ScheduledAt time.Time       `json:"scheduledAt"`
			Payload     json.RawMessage `json:"payload"`
		}{ScheduledAt: decoded.ScheduledAt.UTC(), Payload: payloadBytes})
		requestHash := sha256.Sum256(requestHashInput)
		created, err := store.Create(request.Context(), scheduledposts.CreateParams{
			ID: uuid.New(), OwnerDID: ownerDID, OperationID: operationID,
			RequestHash: requestHash, ScheduledAt: decoded.ScheduledAt.UTC(),
			PayloadBytes: payloadBytes, PayloadHash: payloadHash,
			PayloadVersion: 1, MediaIDs: mediaIDs,
		})
		if err != nil {
			logger.Warn("scheduled post create failed", slog.String("error_class", scheduledPostErrorClass(err)))
			writeScheduledPostError(writer, request, err)
			return
		}
		status := http.StatusOK
		if created.Created {
			status = http.StatusCreated
		}
		envelope.WriteJSON(writer, status, scheduledPostResponse{
			ID: created.ID.String(), OperationID: created.OperationID.String(),
			Status: created.Status, ScheduledAt: created.ScheduledAt.UTC(),
			Payload: decoded.Payload.canonical(),
		})
	})
}

func ListScheduledPostsHandler(store scheduledPostLister, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, ok := scheduledPostOwner(writer, request)
		if !ok {
			return
		}
		resources, err := store.List(request.Context(), ownerDID)
		if err != nil {
			logger.Warn("scheduled post list failed", slog.String("error_class", scheduledPostErrorClass(err)))
			writeScheduledPostError(writer, request, err)
			return
		}
		response := scheduledPostListResponse{Items: make([]scheduledPostSummaryResponse, 0, len(resources)), Count: len(resources)}
		for _, resource := range resources {
			payload, err := scheduledposts.DecodePayload(resource.PayloadBytes)
			if err != nil {
				writeScheduledPostError(writer, request, err)
				return
			}
			item := scheduledPostSummaryResponse{
				ID: resource.ID.String(), Status: resource.Status,
				ScheduledAt: resource.ScheduledAt, Kind: payload.Kind,
				TextPreview:             boundedScheduledPreview(payload.Text, 160),
				NeedsAttentionExpiresAt: resource.NeedsAttentionExpiresAt,
			}
			if resource.Status == scheduledposts.StatusNeedsAttention {
				response.NeedsAttentionCount++
			}
			if len(payload.Media) > 0 {
				item.FirstMediaID = payload.Media[0].ID
			}
			if len(payload.Project) > 0 {
				var project Project
				if json.Unmarshal(payload.Project, &project) == nil && project.Common.Title != nil {
					item.ProjectTitle = *project.Common.Title
				}
			}
			response.Items = append(response.Items, item)
		}
		envelope.WriteJSON(writer, http.StatusOK, response)
	})
}

func GetScheduledPostHandler(store scheduledPostReader, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, ok := scheduledPostOwner(writer, request)
		if !ok {
			return
		}
		id, err := uuid.Parse(request.PathValue("id"))
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "invalid_scheduled_post_id", "scheduled post ID is invalid", middleware.GetRunID(request.Context()), nil)
			return
		}
		resource, err := store.Get(request.Context(), ownerDID, id)
		if err != nil {
			logger.Warn("scheduled post get failed", slog.String("error_class", scheduledPostErrorClass(err)))
			writeScheduledPostError(writer, request, err)
			return
		}
		payload, err := scheduledposts.DecodePayload(resource.PayloadBytes)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		envelope.WriteJSON(writer, http.StatusOK, scheduledPostResponse{
			ID: resource.ID.String(), OperationID: resource.OperationID.String(),
			Status: resource.Status, ScheduledAt: resource.ScheduledAt,
			Payload: payload,
		})
	})
}

func UpdateScheduledPostHandler(
	store scheduledPostUpdater,
	limits MediaLimits,
	now func() time.Time,
	logger *slog.Logger,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, ok := scheduledPostOwner(writer, request)
		if !ok {
			return
		}
		id, err := uuid.Parse(request.PathValue("id"))
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "invalid_scheduled_post_id", "scheduled post ID is invalid", middleware.GetRunID(request.Context()), nil)
			return
		}
		decoded, err := decodeScheduledPostUpdate(request.Body)
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "malformed_body", "could not parse body", middleware.GetRunID(request.Context()), nil)
			return
		}
		acceptedAt := now().UTC()
		validationRequest := scheduledPostCreateRequest{
			OperationID: uuid.NewString(), ScheduledAt: decoded.ScheduledAt, Payload: decoded.Payload,
		}
		_, mediaIDs, payloadBytes, err := validateScheduledPostRequest(acceptedAt, validationRequest, limits)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		payloadHash := sha256.Sum256(payloadBytes)
		if _, err := store.Update(request.Context(), scheduledposts.UpdateParams{
			ID: id, OwnerDID: ownerDID, ScheduledAt: decoded.ScheduledAt.UTC(),
			PayloadBytes: payloadBytes, PayloadHash: payloadHash,
			MediaIDs: mediaIDs, Now: acceptedAt,
		}); err != nil {
			logger.Warn("scheduled post update failed", slog.String("error_class", scheduledPostErrorClass(err)))
			writeScheduledPostError(writer, request, err)
			return
		}
		resource, err := store.Get(request.Context(), ownerDID, id)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		payload, err := scheduledposts.DecodePayload(resource.PayloadBytes)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		envelope.WriteJSON(writer, http.StatusOK, scheduledPostResponse{
			ID: resource.ID.String(), OperationID: resource.OperationID.String(),
			Status: resource.Status, ScheduledAt: resource.ScheduledAt,
			Payload: payload,
		})
	})
}

func DeleteScheduledPostHandler(
	store scheduledPostDeleter,
	now func() time.Time,
	logger *slog.Logger,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, ok := scheduledPostOwner(writer, request)
		if !ok {
			return
		}
		id, err := uuid.Parse(request.PathValue("id"))
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "invalid_scheduled_post_id", "scheduled post ID is invalid", middleware.GetRunID(request.Context()), nil)
			return
		}
		if err := store.Delete(request.Context(), ownerDID, id, now().UTC()); err != nil {
			logger.Warn("scheduled post delete failed", slog.String("error_class", scheduledPostErrorClass(err)))
			writeScheduledPostError(writer, request, err)
			return
		}
		writer.WriteHeader(http.StatusNoContent)
	})
}

// PublishScheduledPostHandler applies the editor's complete payload and makes
// the retained item immediately due. The ordinary background publisher owns
// the PDS side effect and its crash-safe reconciliation protocol.
func PublishScheduledPostHandler(
	publisher scheduledposts.ManualPublisher,
	limits MediaLimits,
	now func() time.Time,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ownerDID, ok := middleware.GetDID(request.Context())
		if !ok {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "request context is unavailable", middleware.GetRunID(request.Context()), nil)
			return
		}
		id, err := uuid.Parse(request.PathValue("id"))
		if err != nil {
			writeScheduledPostError(writer, request, scheduledposts.ErrScheduleNotFound)
			return
		}
		decoded, err := decodeScheduledPostPublication(request.Body)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		acceptedAt := now().UTC()
		validation := scheduledPostCreateRequest{
			OperationID: uuid.NewString(), ScheduledAt: acceptedAt.Add(5 * time.Minute), Payload: decoded.Payload,
		}
		_, mediaIDs, payloadBytes, err := validateScheduledPostRequest(acceptedAt, validation, limits)
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		outcome, err := publisher.PublishManual(request.Context(), scheduledposts.UpdateParams{
			ID: id, OwnerDID: ownerDID, ScheduledAt: acceptedAt,
			PayloadBytes: payloadBytes, PayloadHash: sha256.Sum256(payloadBytes),
			MediaIDs: mediaIDs, Now: acceptedAt,
		})
		if err != nil {
			writeScheduledPostError(writer, request, err)
			return
		}
		if outcome == scheduledposts.ManualPublicationReconciling {
			logger.Debug("manual scheduled publication requires reconciliation")
			envelope.WriteJSON(writer, http.StatusAccepted, map[string]string{"status": "publishing"})
			return
		}
		logger.Debug("manual scheduled publication completed")
		envelope.WriteJSON(writer, http.StatusOK, map[string]string{"status": "published"})
	})
}

func scheduledPostOwner(writer http.ResponseWriter, request *http.Request) (syntax.DID, bool) {
	ownerDID, ok := middleware.GetDID(request.Context())
	if !ok {
		envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "authenticated account unavailable", middleware.GetRunID(request.Context()), nil)
		return "", false
	}
	return ownerDID, true
}

func boundedScheduledPreview(value string, limit int) string {
	if limit < 1 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func writeScheduledPostError(writer http.ResponseWriter, request *http.Request, err error) {
	runID := middleware.GetRunID(request.Context())
	var fieldError *FieldError
	switch {
	case errors.Is(err, scheduledposts.ErrInvalidScheduledAt):
		envelope.WriteError(writer, http.StatusUnprocessableEntity, "invalid_scheduled_at", "scheduled time is invalid", runID, map[string]string{"scheduledAt": "must be a whole minute from five minutes through 28 days"})
	case errors.Is(err, scheduledposts.ErrIneligibleScheduledPost):
		envelope.WriteError(writer, http.StatusUnprocessableEntity, "scheduled_post_ineligible", "post is not eligible for scheduling", runID, nil)
	case errors.Is(err, scheduledposts.ErrCapacityReached):
		envelope.WriteError(writer, http.StatusConflict, "scheduled_post_capacity", "three scheduled posts are already retained", runID, nil)
	case errors.Is(err, scheduledposts.ErrOperationConflict):
		envelope.WriteError(writer, http.StatusConflict, "scheduled_operation_conflict", "operation conflicts with an existing request", runID, nil)
	case errors.Is(err, scheduledposts.ErrScheduledMediaUnavailable):
		envelope.WriteError(writer, http.StatusUnprocessableEntity, "scheduled_media_not_found", "scheduled media is unavailable", runID, nil)
	case errors.Is(err, scheduledposts.ErrScheduleNotFound):
		envelope.WriteError(writer, http.StatusNotFound, "scheduled_post_not_found", "scheduled post not found", runID, nil)
	case errors.Is(err, scheduledposts.ErrMutationLocked):
		envelope.WriteError(writer, http.StatusConflict, "scheduled_post_publishing", "scheduled post publication has started", runID, nil)
	case errors.Is(err, scheduledposts.ErrManualPublicationFailed):
		envelope.WriteError(writer, http.StatusBadGateway, "scheduled_publication_failed", "scheduled post could not be published", runID, nil)
	case errors.As(err, &fieldError):
		envelope.WriteError(writer, http.StatusUnprocessableEntity, "scheduled_post_invalid", "scheduled post is invalid", runID, fieldError.Fields)
	default:
		envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "scheduled post request failed", runID, nil)
	}
}

func scheduledPostErrorClass(err error) string {
	switch {
	case errors.Is(err, scheduledposts.ErrInvalidScheduledAt):
		return "invalid_time"
	case errors.Is(err, scheduledposts.ErrIneligibleScheduledPost):
		return "ineligible"
	case errors.Is(err, scheduledposts.ErrCapacityReached):
		return "capacity"
	case errors.Is(err, scheduledposts.ErrOperationConflict):
		return "operation_conflict"
	case errors.Is(err, scheduledposts.ErrScheduledMediaUnavailable):
		return "media_unavailable"
	default:
		return "unknown"
	}
}
