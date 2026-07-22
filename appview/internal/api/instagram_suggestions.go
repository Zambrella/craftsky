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
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/followwrite"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

const instagramFollowCollection = followwrite.Collection

var errInstagramPDSUnavailable = errors.New("Instagram follow PDS unavailable")

type InstagramSuggestionService interface {
	ListSuggestions(context.Context, syntax.DID, int, *instagram.SuggestionCursor) ([]instagram.Suggestion, *instagram.SuggestionCursor, error)
	AcceptSuggestion(context.Context, syntax.DID, uuid.UUID, instagram.InstagramFollowWriter) (instagram.Suggestion, error)
	DismissSuggestion(context.Context, syntax.DID, uuid.UUID) error
}

type instagramSuggestionProfileResponse struct {
	DID         syntax.DID    `json:"did"`
	Handle      syntax.Handle `json:"handle"`
	DisplayName *string       `json:"displayName,omitempty"`
	Avatar      *string       `json:"avatar,omitempty"`
}

type instagramSuggestionResponse struct {
	SuggestionID string                              `json:"suggestionId"`
	Profile      instagramSuggestionProfileResponse  `json:"profile"`
	Reason       instagram.InstagramSuggestionReason `json:"reason"`
	State        instagram.InstagramSuggestionState  `json:"state"`
}

type instagramSuggestionPageResponse struct {
	Items  []instagramSuggestionResponse `json:"items"`
	Cursor string                        `json:"cursor,omitempty"`
}

type instagramSuggestionActionResponse struct {
	SuggestionID string                             `json:"suggestionId"`
	State        instagram.InstagramSuggestionState `json:"state"`
}

func ListInstagramSuggestionsHandler(service InstagramSuggestionService, profiles ProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		limit, err := parseInstagramSuggestionLimit(r)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", middleware.GetRunID(r.Context()), nil)
			return
		}
		cursor, err := decodeInstagramSuggestionCursor(r.URL.Query().Get("cursor"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", middleware.GetRunID(r.Context()), nil)
			return
		}
		items, next, err := service.ListSuggestions(r.Context(), owner, limit, cursor)
		if err != nil {
			writeInstagramSuggestionError(w, r, logger, err)
			return
		}
		response := instagramSuggestionPageResponse{Items: make([]instagramSuggestionResponse, 0, len(items))}
		for _, item := range items {
			profile, err := hydrateInstagramSuggestionProfile(r.Context(), owner, item.TargetDID, profiles, resolver)
			if errors.Is(err, ErrProfileNotFound) {
				continue
			}
			if err != nil {
				logger.Error("Instagram suggestion profile hydration failed", slog.String("run_id", middleware.GetRunID(r.Context())), slog.String("error_category", "profile"))
				envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_suggestions_unavailable", "Instagram suggestions unavailable", middleware.GetRunID(r.Context()), nil)
				return
			}
			response.Items = append(response.Items, instagramSuggestionResponse{
				SuggestionID: item.ID.String(), Profile: profile,
				Reason: item.Reason, State: item.State,
			})
		}
		if next != nil {
			response.Cursor, err = encodeInstagramSuggestionCursor(*next)
			if err != nil {
				writeInstagramSuggestionError(w, r, logger, err)
				return
			}
		}
		writeJSONStatus(w, http.StatusOK, response)
	})
}

func AcceptInstagramSuggestionHandler(service InstagramSuggestionService, newPDS auth.PDSClientFactory, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		id, err := uuid.Parse(r.PathValue("suggestionId"))
		if err != nil {
			writeInstagramSuggestionNotFound(w, r)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())
		writer := &instagramPDSFollowWriter{service: followwrite.NewService(newPDS), sessionID: sessionID}
		result, err := service.AcceptSuggestion(r.Context(), owner, id, writer)
		if err != nil {
			writeInstagramSuggestionError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, instagramSuggestionActionResponse{SuggestionID: result.ID.String(), State: result.State})
	})
}

func DeleteInstagramSuggestionHandler(service InstagramSuggestionService, logger *slog.Logger) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		id, err := uuid.Parse(r.PathValue("suggestionId"))
		if err == nil {
			if err := service.DismissSuggestion(r.Context(), owner, id); err != nil {
				logger.Error("Instagram suggestion dismissal failed", slog.String("run_id", middleware.GetRunID(r.Context())), slog.String("error_category", "store"))
				envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_suggestions_unavailable", "Instagram suggestions unavailable", middleware.GetRunID(r.Context()), nil)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

type instagramPDSFollowWriter struct {
	service   *followwrite.Service
	sessionID string
}

func (w *instagramPDSFollowWriter) PutFollow(ctx context.Context, owner, target syntax.DID, rkey syntax.RecordKey, createdAt time.Time) error {
	if w == nil || w.service == nil {
		return errInstagramPDSUnavailable
	}
	return w.service.Write(ctx, owner, target, w.sessionID, &rkey, createdAt)
}

func hydrateInstagramSuggestionProfile(ctx context.Context, owner, target syntax.DID, profiles ProfileReader, resolver HandleResolver) (instagramSuggestionProfileResponse, error) {
	if profiles == nil || resolver == nil {
		return instagramSuggestionProfileResponse{}, errors.New("Instagram suggestion profile dependencies unavailable")
	}
	row, err := profiles.Read(ctx, target.String(), owner.String())
	if err != nil {
		return instagramSuggestionProfileResponse{}, err
	}
	handle, err := resolver.ResolveHandle(ctx, target)
	if err != nil {
		return instagramSuggestionProfileResponse{}, err
	}
	profile := BuildProfileResponse(row, handle, true)
	return instagramSuggestionProfileResponse{
		DID: target, Handle: handle, DisplayName: profile.DisplayName, Avatar: profile.Avatar,
	}, nil
}

func parseInstagramSuggestionLimit(r *http.Request) (int, error) {
	raw, present := r.URL.Query()["limit"]
	if !present {
		return 0, nil
	}
	if len(raw) != 1 {
		return 0, instagram.ErrInvalidInstagramSuggestionPageLimit
	}
	limit, err := strconv.Atoi(raw[0])
	if err != nil || limit < 1 {
		return 0, instagram.ErrInvalidInstagramSuggestionPageLimit
	}
	return limit, nil
}

func encodeInstagramSuggestionCursor(cursor instagram.SuggestionCursor) (string, error) {
	if cursor.ID == uuid.Nil || cursor.CreatedAt.IsZero() {
		return "", envelope.ErrInvalidCursor
	}
	return envelope.EncodeCursor(map[string]any{
		"createdAt": cursor.CreatedAt.UTC().Format(time.RFC3339Nano),
		"id":        cursor.ID.String(),
	})
}

func decodeInstagramSuggestionCursor(value string) (*instagram.SuggestionCursor, error) {
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
	return &instagram.SuggestionCursor{CreatedAt: createdAt.UTC(), ID: id}, nil
}

func writeInstagramSuggestionError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	runID := middleware.GetRunID(r.Context())
	switch {
	case errors.Is(err, instagram.ErrInstagramResourceNotFound):
		writeInstagramSuggestionNotFound(w, r)
	case errors.Is(err, instagram.ErrInstagramSuggestionIneligible):
		envelope.WriteError(w, http.StatusConflict, "instagram_suggestion_ineligible", "Instagram suggestion is no longer eligible", runID, nil)
	case errors.Is(err, instagram.ErrInstagramFollowWriteUnavailable):
		envelope.WriteError(w, http.StatusServiceUnavailable, "follow_write_unavailable", "Follow write unavailable", runID, nil)
	case errors.Is(err, instagram.ErrInvalidInstagramSuggestionCursor), errors.Is(err, envelope.ErrInvalidCursor):
		envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
	case errors.Is(err, instagram.ErrInvalidInstagramSuggestionPageLimit):
		envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request", runID, nil)
	default:
		logger.Error("Instagram suggestion operation failed", slog.String("run_id", runID), slog.String("error_category", "internal"))
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "Instagram suggestion operation failed", runID, nil)
	}
}

func writeInstagramSuggestionNotFound(w http.ResponseWriter, r *http.Request) {
	envelope.WriteError(w, http.StatusNotFound, "instagram_suggestion_not_found", "Instagram suggestion not found", middleware.GetRunID(r.Context()), nil)
}

var _ instagram.InstagramFollowWriter = (*instagramPDSFollowWriter)(nil)
