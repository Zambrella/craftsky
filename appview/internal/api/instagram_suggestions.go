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
	"social.craftsky/appview/internal/ownerlifecycle"
)

type InstagramSuggestionService interface {
	ListPending(context.Context, syntax.DID, int, *instagram.SuggestionCursor) ([]instagram.PrivateSuggestion, *instagram.SuggestionCursor, error)
	Accept(context.Context, syntax.DID, uuid.UUID, string) (instagram.PrivateSuggestion, error)
	Dismiss(context.Context, syntax.DID, uuid.UUID) (bool, error)
}

type instagramSuggestionIdentityResponse struct {
	DID         syntax.DID    `json:"did"`
	Handle      syntax.Handle `json:"handle"`
	DisplayName *string       `json:"displayName,omitempty"`
	Avatar      *string       `json:"avatar,omitempty"`
}

type instagramSuggestionResponse struct {
	SuggestionID string                              `json:"suggestionId"`
	Target       instagramSuggestionIdentityResponse `json:"target"`
	CreatedAt    string                              `json:"createdAt"`
}

type instagramSuggestionPageResponse struct {
	Items  []instagramSuggestionResponse `json:"items"`
	Cursor string                        `json:"cursor,omitempty"`
}

type instagramSuggestionAcceptanceResponse struct {
	SuggestionID string                    `json:"suggestionId"`
	State        instagram.SuggestionState `json:"state"`
}

func ListInstagramSuggestionsHandler(
	service InstagramSuggestionService,
	profiles ProfileReader,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil || profiles == nil || resolver == nil {
			writeInstagramSuggestionError(w, r, logger, errors.New("Instagram suggestion service unavailable"))
			return
		}
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
		items, next, err := service.ListPending(r.Context(), owner, limit, cursor)
		if err != nil {
			writeInstagramSuggestionError(w, r, logger, err)
			return
		}
		response := instagramSuggestionPageResponse{Items: make([]instagramSuggestionResponse, 0, len(items))}
		for _, item := range items {
			profile, err := profiles.Read(r.Context(), item.TargetDID.String(), owner.String())
			if err != nil || profile == nil || !profile.IsCraftskyProfile {
				writeInstagramSuggestionError(w, r, logger, errors.New("suggestion target profile unavailable"))
				return
			}
			handle, err := resolver.ResolveHandle(r.Context(), item.TargetDID)
			if err != nil {
				envelope.WriteError(w, http.StatusBadGateway, "identity_unavailable", "could not resolve suggestion identity", middleware.GetRunID(r.Context()), nil)
				return
			}
			identity := instagramSuggestionIdentityResponse{
				DID: item.TargetDID, Handle: handle, DisplayName: profile.DisplayName,
			}
			if avatar := synthBlobURL("avatar", profile.DID, profile.AvatarCID, profile.AvatarMime); avatar != "" {
				identity.Avatar = &avatar
			}
			response.Items = append(response.Items, instagramSuggestionResponse{
				SuggestionID: item.ID.String(), Target: identity,
				CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano),
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

func AcceptInstagramSuggestionHandler(
	service InstagramSuggestionService,
	logger *slog.Logger,
) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeInstagramSuggestionError(w, r, logger, errors.New("Instagram suggestion service unavailable"))
			return
		}
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		suggestionID, err := uuid.Parse(r.PathValue("suggestionId"))
		if err != nil {
			writeInstagramSuggestionNotFound(w, r)
			return
		}
		sessionID, ok := middleware.GetOAuthSessionID(r.Context())
		if !ok || sessionID == "" {
			envelope.WriteError(w, http.StatusServiceUnavailable, "session_unavailable", "session is temporarily unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		accepted, err := service.Accept(r.Context(), owner, suggestionID, sessionID)
		if err != nil {
			writeInstagramSuggestionError(w, r, logger, err)
			return
		}
		writeJSONStatus(w, http.StatusOK, instagramSuggestionAcceptanceResponse{
			SuggestionID: accepted.ID.String(), State: accepted.State,
		})
	})
}

func DismissInstagramSuggestionHandler(
	service InstagramSuggestionService,
	logger *slog.Logger,
) http.Handler {
	logger = instagramLogger(logger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeInstagramSuggestionError(w, r, logger, errors.New("Instagram suggestion service unavailable"))
			return
		}
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			writeMissingInstagramDID(w, r)
			return
		}
		if suggestionID, err := uuid.Parse(r.PathValue("suggestionId")); err == nil {
			if _, err := service.Dismiss(r.Context(), owner, suggestionID); err != nil {
				writeInstagramSuggestionError(w, r, logger, err)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func parseInstagramSuggestionLimit(r *http.Request) (int, error) {
	raw, present := r.URL.Query()["limit"]
	if !present {
		return 0, nil
	}
	if len(raw) != 1 {
		return 0, errors.New("invalid limit")
	}
	limit, err := strconv.Atoi(raw[0])
	if err != nil || limit < 1 || limit > 50 {
		return 0, errors.New("invalid limit")
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
	rawAt, atOK := payload["createdAt"].(string)
	rawID, idOK := payload["id"].(string)
	createdAt, atErr := time.Parse(time.RFC3339Nano, rawAt)
	id, idErr := uuid.Parse(rawID)
	if !atOK || !idOK || atErr != nil || idErr != nil {
		return nil, envelope.ErrInvalidCursor
	}
	return &instagram.SuggestionCursor{CreatedAt: createdAt.UTC(), ID: id}, nil
}

func writeInstagramSuggestionNotFound(w http.ResponseWriter, r *http.Request) {
	envelope.WriteError(w, http.StatusNotFound, "suggestion_not_found", "suggestion not found", middleware.GetRunID(r.Context()), nil)
}

func writeInstagramSuggestionError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	switch {
	case errors.Is(err, instagram.ErrInstagramResourceNotFound):
		writeInstagramSuggestionNotFound(w, r)
	case errors.Is(err, instagram.ErrSuggestionIneligible),
		errors.Is(err, instagram.ErrSuggestionGenerationChanged),
		errors.Is(err, ownerlifecycle.ErrGenerationChanged),
		errors.Is(err, ownerlifecycle.ErrOwnerNotActive),
		errors.Is(err, ownerlifecycle.ErrTerminalOwner):
		envelope.WriteError(w, http.StatusConflict, "suggestion_unavailable", "suggestion is no longer available", middleware.GetRunID(r.Context()), nil)
	default:
		logger.Error("Instagram suggestion operation failed", slog.String("run_id", middleware.GetRunID(r.Context())), slog.String("error_category", "store_or_effect"))
		envelope.WriteError(w, http.StatusServiceUnavailable, "instagram_unavailable", "Instagram migration unavailable", middleware.GetRunID(r.Context()), nil)
	}
}
