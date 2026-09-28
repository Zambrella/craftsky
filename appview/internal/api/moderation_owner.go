package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/moderation"
)

func ModerationStandingHandler(store *moderation.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := ctxkeys.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, 500, "internal_error", "authentication context unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		standing, err := store.Standing(r.Context(), did)
		if err != nil {
			envelope.WriteError(w, 500, "internal_error", "moderation standing unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		writeModerationJSON(w, standing)
	})
}

func ModerationHistoryHandler(store *moderation.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := ctxkeys.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, 500, "internal_error", "authentication context unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		limit := 50
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				envelope.WriteError(w, 400, "invalid_request", "invalid limit", ctxkeys.GetRunID(r.Context()), nil)
				return
			}
			limit = parsed
		}
		if limit < 1 || limit > 100 {
			envelope.WriteError(w, 400, "invalid_request", "invalid limit", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		page, err := store.OwnerHistory(r.Context(), did, r.URL.Query().Get("cursor"), limit)
		if errors.Is(err, moderation.ErrInvalidQueueCursor) {
			envelope.WriteError(w, 400, "invalid_cursor", "invalid moderation history cursor", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		if errors.Is(err, moderation.ErrInvalidCommand) {
			envelope.WriteError(w, 400, "invalid_request", "invalid moderation history request", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, 500, "internal_error", "moderation history unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		writeModerationJSON(w, page)
	})
}

func ModerationHistoryEntryHandler(store *moderation.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := ctxkeys.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, 500, "internal_error", "authentication context unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		page, err := store.OwnerEntry(r.Context(), did, r.PathValue("caseReference"))
		if errors.Is(err, moderation.ErrInvalidCaseReference) {
			envelope.WriteError(w, 400, "invalid_case_reference", "invalid moderation case reference", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		if errors.Is(err, moderation.ErrCaseNotFound) {
			envelope.WriteError(w, 404, "moderation_case_not_found", "moderation case not found", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, 500, "internal_error", "moderation history unavailable", ctxkeys.GetRunID(r.Context()), nil)
			return
		}
		writeModerationJSON(w, page)
	})
}

func writeModerationJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
