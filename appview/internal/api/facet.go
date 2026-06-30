package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

var ErrMentionNotFound = errors.New("mention not found")

type FacetSuggestionReader interface {
	SearchMentionSuggestions(ctx context.Context, viewerDID syntax.DID, query string, limit int, now time.Time) ([]MentionSuggestionRow, error)
	SearchHashtagSuggestions(ctx context.Context, query string, limit int, now time.Time) ([]HashtagSuggestionRow, error)
}

type FacetMentionResolver interface {
	ResolveMention(ctx context.Context, handle syntax.Handle, now time.Time) (IdentityCacheRow, error)
}

func ListFacetMentionSuggestionsHandler(store FacetSuggestionReader, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := ParseFacetSuggestionRequest(r)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid facet suggestion query", middleware.GetRunID(r.Context()), nil)
			return
		}
		if req.Query == "" {
			writeJSON(w, http.StatusOK, FacetMentionSuggestionsResponse{Items: []FacetMentionSuggestion{}})
			return
		}
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		rows, err := store.SearchMentionSuggestions(r.Context(), viewerDID, req.Query, req.Limit, time.Now().UTC())
		if err != nil {
			logger.Error("facet mention suggestions failed",
				apiLogErrorAttrs(middleware.GetRunID(r.Context()), "facet.mention_suggestions", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "facet_suggestions_unavailable", "facet suggestions unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items := make([]FacetMentionSuggestion, 0, len(rows))
		for _, row := range rows {
			items = append(items, BuildFacetMentionSuggestion(row))
		}
		writeJSON(w, http.StatusOK, FacetMentionSuggestionsResponse{Items: items})
	})
}

func ResolveFacetMentionHandler(resolver FacetMentionResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle, err := ParseFacetMentionHandle(r)
		if err != nil {
			envelope.WriteError(w, http.StatusNotFound, "mention_not_found", "mention not found", middleware.GetRunID(r.Context()), nil)
			return
		}
		row, err := resolver.ResolveMention(r.Context(), handle, time.Now().UTC())
		if errors.Is(err, ErrMentionNotFound) {
			envelope.WriteError(w, http.StatusNotFound, "mention_not_found", "mention not found", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			logger.Error("facet mention resolve failed",
				apiLogErrorAttrs(middleware.GetRunID(r.Context()), "facet.mention.resolve", "store")...)
			envelope.WriteError(w, http.StatusNotFound, "mention_not_found", "mention not found", middleware.GetRunID(r.Context()), nil)
			return
		}
		writeJSON(w, http.StatusOK, BuildFacetMentionResolveResponse(row))
	})
}

func ListFacetHashtagSuggestionsHandler(store FacetSuggestionReader, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := ParseFacetSuggestionRequest(r)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid facet suggestion query", middleware.GetRunID(r.Context()), nil)
			return
		}
		if req.Query == "" {
			writeJSON(w, http.StatusOK, FacetHashtagSuggestionsResponse{Items: []FacetHashtagSuggestion{}})
			return
		}
		rows, err := store.SearchHashtagSuggestions(r.Context(), req.Query, req.Limit, time.Now().UTC())
		if err != nil {
			logger.Error("facet hashtag suggestions failed",
				apiLogErrorAttrs(middleware.GetRunID(r.Context()), "facet.hashtag_suggestions", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "facet_suggestions_unavailable", "facet suggestions unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items := make([]FacetHashtagSuggestion, 0, len(rows))
		for _, row := range rows {
			items = append(items, FacetHashtagSuggestion(row))
		}
		writeJSON(w, http.StatusOK, FacetHashtagSuggestionsResponse{Items: items})
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
