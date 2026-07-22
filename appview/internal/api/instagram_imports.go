package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

type InstagramImportService interface {
	CreateImport(context.Context, syntax.DID, instagram.ImportSourceType, bool, []instagram.ImportEntry) (instagram.CreateImportResult, error)
	ListImports(context.Context, syntax.DID, int, *instagram.ImportCursor) ([]instagram.GraphImport, *instagram.ImportCursor, error)
	GetImport(context.Context, syntax.DID, uuid.UUID) (instagram.GraphImport, error)
	UpdateImport(context.Context, syntax.DID, uuid.UUID, *bool, *bool) (instagram.GraphImport, error)
	DeleteImport(context.Context, syntax.DID, uuid.UUID) error
}

type instagramImportResponse struct {
	ImportID           string                         `json:"importId"`
	State              instagram.InstagramImportState `json:"state"`
	SourceType         instagram.ImportSourceType     `json:"sourceType"`
	RetainUnmatched    bool                           `json:"retainUnmatched"`
	RetentionExpiresAt string                         `json:"retentionExpiresAt,omitempty"`
	FollowingCount     int                            `json:"followingCount"`
	CreatedAt          string                         `json:"createdAt"`
}

type instagramImportCountsResponse struct {
	FollowingCount int `json:"followingCount"`
}

type instagramImportCreateResponse struct {
	Import                 instagramImportResponse       `json:"import"`
	Counts                 instagramImportCountsResponse `json:"counts"`
	InitialSuggestionCount int                           `json:"initialSuggestionCount"`
}

type instagramImportPageResponse struct {
	Items  []instagramImportResponse `json:"items"`
	Cursor string                    `json:"cursor,omitempty"`
}

func CreateInstagramImportHandler(service InstagramImportService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		var request struct {
			SourceType      instagram.ImportSourceType `json:"sourceType"`
			RetainUnmatched *bool                      `json:"retainUnmatched"`
			Entries         []instagram.ImportEntry    `json:"entries"`
		}
		if err := decodeStrictJSONObject(r, &request); err != nil || !request.SourceType.Valid() || request.RetainUnmatched == nil || request.Entries == nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
			return
		}
		created, err := service.CreateImport(r.Context(), owner, request.SourceType, *request.RetainUnmatched, request.Entries)
		if err != nil {
			writeInstagramImportError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusCreated, instagramImportCreateResponse{
			Import: importResponse(created.Import),
			Counts: instagramImportCountsResponse{
				FollowingCount: created.Counts.Following,
			},
			InitialSuggestionCount: created.InitialSuggestionCount,
		})
	})
}

func ListInstagramImportsHandler(service InstagramImportService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		limit := 0
		if rawLimit, present := r.URL.Query()["limit"]; present {
			if len(rawLimit) != 1 {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
				return
			}
			parsed, err := strconv.Atoi(rawLimit[0])
			if err != nil || parsed < 1 {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
				return
			}
			limit = parsed
		}
		cursor, err := decodeInstagramImportCursor(r.URL.Query().Get("cursor"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", middleware.GetRunID(r.Context()), nil)
			return
		}
		items, next, err := service.ListImports(r.Context(), owner, limit, cursor)
		if err != nil {
			writeInstagramImportError(w, r, logger, err)
			return
		}
		response := instagramImportPageResponse{Items: make([]instagramImportResponse, 0, len(items))}
		for _, item := range items {
			response.Items = append(response.Items, importResponse(item))
		}
		if next != nil {
			response.Cursor, err = encodeInstagramImportCursor(*next)
			if err != nil {
				writeInstagramImportError(w, r, logger, err)
				return
			}
		}
		writeJSONStatus(w, http.StatusOK, response)
	})
}

func GetInstagramImportHandler(service InstagramImportService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		id, err := uuid.Parse(r.PathValue("importId"))
		if err != nil {
			writeInstagramImportNotFound(w, r)
			return
		}
		item, err := service.GetImport(r.Context(), owner, id)
		if err != nil {
			writeInstagramImportError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, importResponse(item))
	})
}

func PatchInstagramImportHandler(service InstagramImportService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		id, err := uuid.Parse(r.PathValue("importId"))
		if err != nil {
			writeInstagramImportNotFound(w, r)
			return
		}
		var request struct {
			RetainUnmatched *bool `json:"retainUnmatched"`
			Reactivate      *bool `json:"reactivate"`
		}
		if err := decodeStrictJSONObject(r, &request); err != nil || (request.RetainUnmatched == nil && request.Reactivate == nil) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
			return
		}
		item, err := service.UpdateImport(r.Context(), owner, id, request.RetainUnmatched, request.Reactivate)
		if err != nil {
			writeInstagramImportError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, importResponse(item))
	})
}

func DeleteInstagramImportHandler(service InstagramImportService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		id, err := uuid.Parse(r.PathValue("importId"))
		if err == nil {
			if err := service.DeleteImport(r.Context(), owner, id); err != nil {
				logger.Error("Instagram import deletion failed", slog.String("run_id", middleware.GetRunID(r.Context())), slog.String("error_category", "store"))
				envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_unavailable", "Instagram migration unavailable", middleware.GetRunID(r.Context()), nil)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func importResponse(item instagram.GraphImport) instagramImportResponse {
	response := instagramImportResponse{
		ImportID: item.ID.String(), State: item.State, SourceType: item.SourceType,
		RetainUnmatched: item.RetainUnmatched, FollowingCount: item.FollowingCount,
		CreatedAt: instagramTime(item.CreatedAt),
	}
	if item.RetentionExpiresAt != nil {
		response.RetentionExpiresAt = instagramTime(*item.RetentionExpiresAt)
	}
	return response
}

func encodeInstagramImportCursor(cursor instagram.ImportCursor) (string, error) {
	if cursor.ID == uuid.Nil || cursor.CreatedAt.IsZero() {
		return "", envelope.ErrInvalidCursor
	}
	return envelope.EncodeCursor(map[string]any{
		"createdAt": cursor.CreatedAt.UTC().Format(time.RFC3339Nano),
		"id":        cursor.ID.String(),
	})
}

func decodeInstagramImportCursor(value string) (*instagram.ImportCursor, error) {
	if value == "" {
		return nil, nil
	}
	payload, err := envelope.DecodeCursor(value)
	if err != nil || len(payload) != 2 {
		return nil, envelope.ErrInvalidCursor
	}
	createdRaw, createdOK := payload["createdAt"].(string)
	idRaw, idOK := payload["id"].(string)
	createdAt, createdErr := time.Parse(time.RFC3339Nano, createdRaw)
	id, idErr := uuid.Parse(idRaw)
	if !createdOK || !idOK || createdErr != nil || idErr != nil {
		return nil, envelope.ErrInvalidCursor
	}
	return &instagram.ImportCursor{CreatedAt: createdAt.UTC(), ID: id}, nil
}

func writeInstagramImportError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	runID := middleware.GetRunID(r.Context())
	switch {
	case errors.Is(err, instagram.ErrInstagramResourceNotFound):
		writeInstagramImportNotFound(w, r)
	case errors.Is(err, instagram.ErrInstagramImportInactive):
		envelope.WriteError(w, http.StatusConflict, "instagram_import_inactive", "Instagram import inactive", runID, nil)
	case errors.Is(err, instagram.ErrInstagramImportExpired):
		envelope.WriteError(w, http.StatusConflict, "instagram_import_expired", "Instagram import expired", runID, nil)
	case errors.Is(err, instagram.ErrUnmatchedDataUnavailable):
		envelope.WriteError(w, http.StatusConflict, "unmatched_data_unavailable", "Unmatched Instagram data unavailable", runID, nil)
	case errors.Is(err, instagram.ErrInvalidInstagramImportCursor), errors.Is(err, envelope.ErrInvalidCursor):
		envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
	case errors.Is(err, instagram.ErrInvalidInstagramImport),
		errors.Is(err, instagram.ErrInvalidInstagramUsername),
		errors.Is(err, instagram.ErrTooManyImportEntries):
		envelope.WriteError(w, http.StatusUnprocessableEntity, "invalid_instagram_import", "Instagram import is invalid", runID, nil)
	case errors.Is(err, instagram.ErrInvalidInstagramPageLimit):
		envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", runID, nil)
	default:
		logger.Error("Instagram import operation failed", slog.String("run_id", runID), slog.String("error_category", "internal"))
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Instagram import failed", runID, nil)
	}
}

func writeInstagramImportNotFound(w http.ResponseWriter, r *http.Request) {
	envelope.WriteError(w, http.StatusNotFound, "instagram_import_not_found", "Instagram import not found", middleware.GetRunID(r.Context()), nil)
}

func writeMissingInstagramDID(w http.ResponseWriter, r *http.Request) {
	envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
}
