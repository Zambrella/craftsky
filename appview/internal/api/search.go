package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// SearchStore is the Postgres-backed search implementation. Search handlers
// depend on this concrete store for production route wiring; focused handler
// tests use smaller interfaces in follow-up TDD loops.
type SearchStore struct {
	pool      *pgxpool.Pool
	postStore *PostStore
}

func NewSearchStore(pool *pgxpool.Pool) *SearchStore {
	return &SearchStore{pool: pool, postStore: NewPostStore(pool)}
}

func SearchHashtagPostsHandler(store *SearchStore, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tag, err := NormalizeHashtagPathValue(r.PathValue("tag"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid hashtag", middleware.GetRunID(r.Context()), nil)
			return
		}
		sort, err := parseSearchSort(r.URL.Query().Get("sort"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid search query", middleware.GetRunID(r.Context()), nil)
			return
		}
		limit, err := parseBoundedSearchLimit(r.URL.Query().Get("limit"), SearchDefaultLimit, SearchMaxLimit)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid search query", middleware.GetRunID(r.Context()), nil)
			return
		}
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		rows, nextCursor, err := store.SearchHashtagPosts(r.Context(), tag, sort, limit, r.URL.Query().Get("cursor"), time.Now().UTC())
		if errors.Is(err, envelope.ErrInvalidCursor) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "invalid cursor", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			logger.Error("hashtag search failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items, err := buildSearchPostResponses(r.Context(), rows, viewerDID.String(), store, resolver)
		if err != nil {
			logger.Error("hashtag search response failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		writeJSON(w, http.StatusOK, SearchPostPageResponse{Hashtag: tag, Items: items, Cursor: nextCursor})
	})
}

func SearchProfilesHandler(store *SearchStore, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := ParseProfileSearchRequest(r)
		if err != nil {
			code := "validation_error"
			status := http.StatusBadRequest
			message := "invalid profile search query"
			if errors.Is(err, envelope.ErrInvalidCursor) {
				code = "invalid_cursor"
				message = "invalid cursor"
			}
			envelope.WriteError(w, status, code, message, middleware.GetRunID(r.Context()), nil)
			return
		}
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		rows, nextCursor, err := store.SearchProfiles(r.Context(), viewerDID.String(), req)
		if err != nil {
			logger.Error("profile search failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items := make([]ProfileSearchSummary, 0, len(rows))
		for _, row := range rows {
			items = append(items, BuildProfileSearchSummary(row))
		}
		writeJSON(w, http.StatusOK, SearchProfilePageResponse{Items: items, Cursor: nextCursor})
	})
}

func SearchPostsHandler(store *SearchStore, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := ParsePostSearchRequest(r)
		if err != nil {
			code := "validation_error"
			message := "invalid post search query"
			if errors.Is(err, envelope.ErrInvalidCursor) {
				code = "invalid_cursor"
				message = "invalid cursor"
			}
			envelope.WriteError(w, http.StatusBadRequest, code, message, middleware.GetRunID(r.Context()), nil)
			return
		}
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		rows, nextCursor, err := store.SearchPosts(r.Context(), req, time.Now().UTC())
		if errors.Is(err, envelope.ErrInvalidCursor) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "invalid cursor", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			logger.Error("post search failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items, err := buildSearchPostResponses(r.Context(), rows, viewerDID.String(), store, resolver)
		if err != nil {
			logger.Error("post search response failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		writeJSON(w, http.StatusOK, SearchPostPageResponse{Items: items, Cursor: nextCursor})
	})
}

func SearchProjectsHandler(store *SearchStore, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := ParseProjectSearchRequest(r)
		if err != nil {
			code := "validation_error"
			message := "invalid project search query"
			if errors.Is(err, envelope.ErrInvalidCursor) {
				code = "invalid_cursor"
				message = "invalid cursor"
			}
			envelope.WriteError(w, http.StatusBadRequest, code, message, middleware.GetRunID(r.Context()), nil)
			return
		}
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		rows, nextCursor, err := store.SearchProjects(r.Context(), req, time.Now().UTC())
		if errors.Is(err, envelope.ErrInvalidCursor) {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "invalid cursor", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err != nil {
			logger.Error("project search failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items, err := buildSearchPostResponses(r.Context(), rows, viewerDID.String(), store, resolver)
		if err != nil {
			logger.Error("project search response failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		writeJSON(w, http.StatusOK, SearchPostPageResponse{Items: items, Cursor: nextCursor})
	})
}

func TopHashtagsHandler(store *SearchStore, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := ParseTopHashtagsRequest(r)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid top hashtag query", middleware.GetRunID(r.Context()), nil)
			return
		}
		groups, err := store.TopHashtags(r.Context(), req, time.Now().UTC())
		if err != nil {
			logger.Error("top hashtags failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "search_unavailable", "search unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		writeJSON(w, http.StatusOK, TopHashtagsResponse{Groups: groups})
	})
}

func ListRecentSearchesHandler(store *SearchStore, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		rows, err := store.ListRecentSearches(r.Context(), viewerDID.String())
		if err != nil {
			logger.Error("list recent searches failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "recent_searches_unavailable", "recent searches unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		items := make([]RecentSearchResponse, 0, len(rows))
		for _, row := range rows {
			item, err := BuildRecentSearchResponse(row)
			if err != nil {
				envelope.WriteError(w, http.StatusInternalServerError, "recent_searches_unavailable", "recent searches unavailable", middleware.GetRunID(r.Context()), nil)
				return
			}
			items = append(items, item)
		}
		writeJSON(w, http.StatusOK, RecentSearchPageResponse{Items: items})
	})
}

func SaveRecentSearchHandler(store *SearchStore, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		req, err := DecodeSaveRecentSearchRequest(r)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "validation_error", "invalid recent search", middleware.GetRunID(r.Context()), nil)
			return
		}
		row, err := store.SaveRecentSearch(r.Context(), viewerDID.String(), req, time.Now().UTC())
		if err != nil {
			logger.Error("save recent search failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())), slog.String("type", req.Type))
			envelope.WriteError(w, http.StatusInternalServerError, "recent_searches_unavailable", "recent searches unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		item, err := BuildRecentSearchResponse(row)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError, "recent_searches_unavailable", "recent searches unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		writeJSON(w, http.StatusOK, item)
	})
}

func DeleteRecentSearchHandler(store *SearchStore, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", middleware.GetRunID(r.Context()), nil)
			return
		}
		if err := store.DeleteRecentSearch(r.Context(), viewerDID.String(), r.PathValue("id")); err != nil {
			logger.Error("delete recent search failed", slog.String("err", err.Error()), slog.String("run_id", middleware.GetRunID(r.Context())))
			envelope.WriteError(w, http.StatusInternalServerError, "recent_searches_unavailable", "recent searches unavailable", middleware.GetRunID(r.Context()), nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func searchNotImplementedHandler(logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logger != nil {
			logger.Debug("search endpoint reached before store loop", slog.String("run_id", middleware.GetRunID(r.Context())))
		}
		envelope.WriteError(w, http.StatusNotImplemented, "not_implemented", "search endpoint not implemented", middleware.GetRunID(r.Context()), nil)
	})
}

func buildSearchPostResponses(ctx context.Context, rows []SearchPostRow, viewerDID string, store *SearchStore, resolver HandleResolver) ([]*PostResponse, error) {
	items := make([]*PostResponse, 0, len(rows))
	uris := make([]string, 0, len(rows))
	for _, row := range rows {
		uris = append(uris, row.Post.URI)
	}
	summaries, err := store.EngagementSummaries(ctx, viewerDID, uris)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		handle := syntax.Handle(row.Post.DID)
		if resolver != nil {
			if resolved, err := resolver.ResolveHandle(ctx, syntax.DID(row.Post.DID)); err == nil && resolved != "" {
				handle = resolved
			}
		}
		resp := BuildPostResponse(row.Post, handle)
		applyEngagementSummary(resp, summaries[row.Post.URI])
		items = append(items, resp)
	}
	return items, nil
}
