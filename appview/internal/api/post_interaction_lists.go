package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type PostInteractionTargetReader interface {
	ResolveInteractionTarget(context.Context, syntax.DID, syntax.DID, syntax.RecordKey, PostInteractionKind) (*PostInteractionTarget, error)
}

type PostInteractionAccountListReader interface {
	PostInteractionTargetReader
	ListPostInteractionAccounts(context.Context, syntax.DID, *PostInteractionTarget, PostInteractionKind, int, string) (ProfileAccountPage, error)
}

type PostQuoteListReader interface {
	PostInteractionTargetReader
	ListQuotePostResponses(context.Context, syntax.DID, *PostInteractionTarget, []string, int, string, HandleResolver) ([]*PostResponse, string, error)
}

func ListPostLikesHandler(store PostInteractionAccountListReader, logger *slog.Logger) http.Handler {
	return listPostInteractionAccountsHandler(store, PostInteractionLikes, logger)
}

func ListPostRepostsHandler(store PostInteractionAccountListReader, logger *slog.Logger) http.Handler {
	return listPostInteractionAccountsHandler(store, PostInteractionReposts, logger)
}

func ListPostQuotesHandler(
	store PostQuoteListReader,
	resolver HandleResolver,
	logger *slog.Logger,
	preferenceReaders ...LanguagePreferenceReader,
) http.Handler {
	return postInteractionListHandler(store, PostInteractionQuotes, logger, func(
		w http.ResponseWriter,
		r *http.Request,
		viewer syntax.DID,
		target *PostInteractionTarget,
		limit int,
		cursor string,
	) (postInteractionListResult, error) {
		contentLanguages, err := authoritativeContentLanguages(r.Context(), viewer, preferenceReaders)
		if err != nil {
			return postInteractionListResult{}, err
		}
		items, next, err := store.ListQuotePostResponses(
			r.Context(), viewer, target, contentLanguages, limit, cursor, resolver,
		)
		if err != nil {
			return postInteractionListResult{}, err
		}
		envelope.WriteJSON(w, http.StatusOK, struct {
			Items  []*PostResponse `json:"items"`
			Cursor string          `json:"cursor,omitempty"`
		}{Items: items, Cursor: next})
		return postInteractionListResult{rowCount: len(items), hasCursor: next != ""}, nil
	})
}

func listPostInteractionAccountsHandler(store PostInteractionAccountListReader, kind PostInteractionKind, logger *slog.Logger) http.Handler {
	return postInteractionListHandler(store, kind, logger, func(
		w http.ResponseWriter,
		r *http.Request,
		viewer syntax.DID,
		target *PostInteractionTarget,
		limit int,
		cursor string,
	) (postInteractionListResult, error) {
		page, err := store.ListPostInteractionAccounts(r.Context(), viewer, target, kind, limit, cursor)
		if err != nil {
			return postInteractionListResult{}, err
		}
		envelope.WriteJSON(w, http.StatusOK, page)
		return postInteractionListResult{
			rowCount:  len(page.Items),
			hasCursor: page.Cursor != nil,
		}, nil
	})
}

type postInteractionListResult struct {
	rowCount  int
	hasCursor bool
}

type postInteractionListOperation func(http.ResponseWriter, *http.Request, syntax.DID, *PostInteractionTarget, int, string) (postInteractionListResult, error)

func postInteractionListHandler(store PostInteractionTargetReader, kind PostInteractionKind, logger *slog.Logger, list postInteractionListOperation) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		viewer, ok := middleware.GetDID(r.Context())
		if !ok {
			logPostInteractionListResult(logger, runID, kind, "error", "internal", 0, postInteractionListResult{}, false)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		owner, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			logPostInteractionListResult(logger, runID, kind, "error", "validation", 0, postInteractionListResult{}, false)
			envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		rkey, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			logPostInteractionListResult(logger, runID, kind, "error", "validation", 0, postInteractionListResult{}, false)
			envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "rkey path segment is not a valid record key", runID, nil)
			return
		}

		limit := parseLimit(r.URL.Query().Get("limit"))
		target, err := store.ResolveInteractionTarget(r.Context(), viewer, owner, rkey, kind)
		if err != nil {
			logPostInteractionListResult(logger, runID, kind, "error", postInteractionListErrorClass(err), limit, postInteractionListResult{}, true)
			writePostInteractionListError(w, runID, err)
			return
		}
		result, err := list(w, r, viewer, target, limit, r.URL.Query().Get("cursor"))
		if err != nil {
			logPostInteractionListResult(logger, runID, kind, "error", postInteractionListErrorClass(err), limit, postInteractionListResult{}, true)
			writePostInteractionListError(w, runID, err)
			return
		}
		logPostInteractionListResult(logger, runID, kind, "success", "", limit, result, true)
	})
}

func logPostInteractionListResult(
	logger *slog.Logger,
	runID string,
	kind PostInteractionKind,
	result string,
	errorClass string,
	limit int,
	page postInteractionListResult,
	includePageFields bool,
) {
	if logger == nil {
		return
	}
	attrs := append(apiLogAttrs(runID, string(kind)), slog.String("result", result))
	if errorClass != "" {
		attrs = append(attrs, slog.String("error_class", errorClass))
	}
	if includePageFields {
		attrs = append(attrs,
			slog.Int("limit", limit),
			slog.Int("row_count", page.rowCount),
			slog.Bool("has_cursor", page.hasCursor),
		)
	}
	level := slog.LevelInfo
	if result != "success" {
		level = slog.LevelWarn
	}
	logger.Log(context.Background(), level, "post interaction list completed", attrs...)
}

func postInteractionListErrorClass(err error) string {
	switch {
	case errors.Is(err, envelope.ErrInvalidCursor):
		return "validation"
	case errors.Is(err, ErrPostNotFound):
		return "unavailable"
	case errors.Is(err, ErrPostInteractionIdentityUnavailable), errors.Is(err, ErrHandleUnavailable):
		return "identity"
	default:
		return "internal"
	}
}

func writePostInteractionListError(w http.ResponseWriter, runID string, err error) {
	switch {
	case errors.Is(err, envelope.ErrInvalidCursor):
		envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
	case errors.Is(err, ErrPostNotFound):
		envelope.WriteError(w, http.StatusNotFound, "post_not_found", "post not found", runID, nil)
	case errors.Is(err, ErrPostInteractionIdentityUnavailable), errors.Is(err, ErrHandleUnavailable):
		envelope.WriteError(w, http.StatusBadGateway, "identity_unavailable", "could not resolve identity", runID, nil)
	default:
		envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "post interaction list failed", runID, nil)
	}
}
