// appview/internal/api/post_author_feed.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type authorFeedHydrationStore interface {
	relationshipStateReader
	engagementSummaryReader
	postQuoteHydrationStore
}

type authorPostsStore interface {
	authorFeedHydrationStore
	ListByAuthor(context.Context, string, int, string) ([]*PostRow, string, error)
}

type authorProjectsStore interface {
	authorFeedHydrationStore
	ListProjectsByAuthor(context.Context, string, int, string) ([]*PostRow, string, error)
}

type authorCommentsStore interface {
	authorFeedHydrationStore
	ListCommentsByAuthor(context.Context, string, int, string) ([]*PostRow, string, error)
}

// ListPostsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/posts.
func ListPostsByAuthorHandler(
	store authorPostsStore,
	resolver HandleResolver,
	logger *slog.Logger,
	pinReader ProfilePinListReader,
	preferenceReaders ...LanguagePreferenceReader,
) http.Handler {
	var filtered authorLanguageList
	if languageStore, ok := store.(interface {
		ListByAuthorWithLanguages(context.Context, string, string, []string, int, string) ([]*PostRow, string, error)
	}); ok {
		filtered = languageStore.ListByAuthorWithLanguages
	}
	return listAuthorPostsHandler(store, resolver, logger, "post list", ProfilePinSlotStandard, pinReader, store.ListByAuthor, filtered, preferenceReaders)
}

// ListProjectsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/projects.
func ListProjectsByAuthorHandler(
	store authorProjectsStore,
	resolver HandleResolver,
	logger *slog.Logger,
	pinReader ProfilePinListReader,
	preferenceReaders ...LanguagePreferenceReader,
) http.Handler {
	var filtered authorLanguageList
	if languageStore, ok := store.(interface {
		ListProjectsByAuthorWithLanguages(context.Context, string, string, []string, int, string) ([]*PostRow, string, error)
	}); ok {
		filtered = languageStore.ListProjectsByAuthorWithLanguages
	}
	return listAuthorPostsHandler(store, resolver, logger, "project list", ProfilePinSlotProject, pinReader, store.ListProjectsByAuthor, filtered, preferenceReaders)
}

// ListCommentsByAuthorHandler serves GET /v1/profiles/{handleOrDid}/comments.
func ListCommentsByAuthorHandler(
	store authorCommentsStore,
	resolver HandleResolver,
	logger *slog.Logger,
	preferenceReaders ...LanguagePreferenceReader,
) http.Handler {
	var filtered authorLanguageList
	if languageStore, ok := store.(interface {
		ListCommentsByAuthorWithLanguages(context.Context, string, string, []string, int, string) ([]*PostRow, string, error)
	}); ok {
		filtered = languageStore.ListCommentsByAuthorWithLanguages
	}
	return listAuthorPostsHandler(store, resolver, logger, "comment list", "", nil, store.ListCommentsByAuthor, filtered, preferenceReaders)
}

type authorLanguageList func(context.Context, string, string, []string, int, string) ([]*PostRow, string, error)

func listAuthorPostsHandler(
	store authorFeedHydrationStore,
	resolver HandleResolver,
	logger *slog.Logger,
	logLabel string,
	pinSlot ProfilePinSlot,
	pinReader ProfilePinListReader,
	list func(context.Context, string, int, string) ([]*PostRow, string, error),
	filteredList authorLanguageList,
	preferenceReaders []LanguagePreferenceReader,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		operation := postAuthorListOperation(logLabel)
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
		logger.Debug(logLabel+": resolving author",
			apiLogAttrs(runID, operation)...)
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn(logLabel+": ResolveDID failed",
					apiLogErrorAttrs(runID, operation, "identity")...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		viewerDID, _ := middleware.GetDID(r.Context())
		relationshipState, err := store.RelationshipState(r.Context(), viewerDID, did)
		if errors.Is(err, relationships.ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"profile_not_found", "profile not found", runID, nil)
			return
		}
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "relationship lookup failed", runID, nil)
			return
		}
		if relationshipState.HasBlock() {
			writeJSON(w, http.StatusOK, struct {
				Items []*PostResponse `json:"items"`
			}{Items: []*PostResponse{}})
			return
		}
		limit := parseLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		logger.Debug(logLabel+": listing author records",
			append(apiLogAttrs(runID, operation),
				slog.Int("limit", limit),
				slog.Bool("has_cursor", cursor != ""))...)

		contentLanguages := []string{}
		if len(preferenceReaders) > 0 {
			var preferenceErr error
			contentLanguages, preferenceErr = authoritativeContentLanguages(r.Context(), viewerDID, preferenceReaders)
			if preferenceErr != nil {
				logger.Error(logLabel+": language preferences failed",
					apiLogErrorAttrs(runID, operation, "language_preferences")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "language preferences lookup failed", runID, nil)
				return
			}
		}

		var profilePin *ProfileListPin
		if pinReader != nil {
			profilePin, err = pinReader.ReadProfileListPin(
				r.Context(),
				did,
				viewerDID,
				pinSlot,
				contentLanguages,
			)
			if err != nil {
				logger.Error(logLabel+": profile pin lookup failed",
					apiLogErrorAttrs(runID, operation, "profile_pin")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "profile pin lookup failed", runID, nil)
				return
			}
		}
		pinStateToken := ""
		if profilePin != nil {
			pinStateToken = profilePin.StateToken
		}
		storeCursor := cursor
		if pinReader != nil && cursor != "" {
			storeCursor, err = unwrapProfileListCursor(cursor, pinSlot, pinStateToken)
			if err != nil {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
		}
		storeLimit := limit
		if cursor != "" && profilePin != nil {
			storeLimit++
		}

		var rows []*PostRow
		var nextCursor string
		if len(preferenceReaders) == 0 {
			rows, nextCursor, err = list(r.Context(), did.String(), storeLimit, storeCursor)
		} else {
			if filteredList == nil {
				logger.Error(logLabel+": language filtering unavailable",
					apiLogErrorAttrs(runID, operation, "language_filter")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post language filtering unavailable", runID, nil)
				return
			}
			rows, nextCursor, err = filteredList(
				r.Context(),
				viewerDID.String(),
				did.String(),
				contentLanguages,
				storeLimit,
				storeCursor,
			)
		}
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error(logLabel+": list failed",
				apiLogErrorAttrs(runID, operation, "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "post list failed", runID, nil)
			return
		}
		var pinnedPostURI *string
		if cursor == "" && profilePin != nil {
			rows, nextCursor, err = promoteProfilePinFirstPage(rows, profilePin.Row, limit, nextCursor)
			if err != nil {
				logger.Error(logLabel+": profile pin cursor failed",
					apiLogErrorAttrs(runID, operation, "profile_pin_cursor")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "profile pin cursor failed", runID, nil)
				return
			}
			pinnedURI := profilePin.Row.URI
			pinnedPostURI = &pinnedURI
		} else if cursor != "" && profilePin != nil {
			rows, nextCursor, err = excludeProfilePinFromLaterPage(rows, profilePin.Row, limit, storeLimit, nextCursor)
			if err != nil {
				logger.Error(logLabel+": profile pin cursor failed",
					apiLogErrorAttrs(runID, operation, "profile_pin_cursor")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "profile pin cursor failed", runID, nil)
				return
			}
		}
		if pinReader != nil && nextCursor != "" {
			nextCursor, err = wrapProfileListCursor(nextCursor, pinSlot, pinStateToken)
			if err != nil {
				logger.Error(logLabel+": profile pin cursor failed",
					apiLogErrorAttrs(runID, operation, "profile_pin_cursor")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "profile pin cursor failed", runID, nil)
				return
			}
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				postURIs = append(postURIs, row.URI)
			}
			summaries, serr := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if serr != nil {
				logger.Error(logLabel+": EngagementSummaries failed",
					apiLogErrorAttrs(runID, operation, "engagement")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			// Only pay handle-resolution cost when there are rows to render.
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				logger.Warn(logLabel+": ResolveHandle failed",
					apiLogErrorAttrs(runID, operation, "identity")...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				resp := BuildPostResponse(row, handle)
				applyEngagementSummary(resp, summaries[row.URI])
				items = append(items, resp)
			}
			if err := attachQuoteViews(r.Context(), store, resolver, items); err != nil {
				logger.Error(logLabel+": QuoteViewRows failed",
					apiLogErrorAttrs(runID, operation, "quote_view")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post quote lookup failed", runID, nil)
				return
			}
		}
		body := struct {
			Items         []*PostResponse `json:"items"`
			Cursor        string          `json:"cursor,omitempty"`
			PinnedPostURI *string         `json:"pinnedPostUri,omitempty"`
		}{Items: items, Cursor: nextCursor, PinnedPostURI: pinnedPostURI}
		logger.Debug(logLabel+": response ready",
			append(apiLogSuccessAttrs(runID, operation),
				slog.Int("rows", len(rows)),
				slog.Int("items", len(items)),
				slog.Bool("has_next_cursor", nextCursor != ""))...)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(body)
	})
}

func promoteProfilePinFirstPage(
	chronological []*PostRow,
	pinned *PostRow,
	limit int,
	nextCursor string,
) ([]*PostRow, string, error) {
	for index, row := range chronological {
		if row.URI != pinned.URI {
			continue
		}
		out := make([]*PostRow, 0, len(chronological))
		out = append(out, pinned)
		out = append(out, chronological[:index]...)
		out = append(out, chronological[index+1:]...)
		return out, nextCursor, nil
	}

	keep := limit - 1
	if keep < 0 {
		keep = 0
	}
	if keep > len(chronological) {
		keep = len(chronological)
	}
	out := make([]*PostRow, 0, 1+keep)
	out = append(out, pinned)
	out = append(out, chronological[:keep]...)
	if len(chronological) < limit {
		return out, "", nil
	}
	if keep == 0 {
		cursor, err := envelope.EncodeCursor(map[string]any{
			"indexedAt": "9999-12-31T23:59:59Z",
			"uri":       "\uffff",
		})
		return out, cursor, err
	}
	last := chronological[keep-1]
	cursor, err := envelope.EncodeCursor(map[string]any{
		"indexedAt": last.ProfileSortAt.UTC().Format(time.RFC3339Nano),
		"uri":       last.URI,
	})
	return out, cursor, err
}

func excludeProfilePinFromLaterPage(
	chronological []*PostRow,
	pinned *PostRow,
	limit int,
	storeLimit int,
	nextCursor string,
) ([]*PostRow, string, error) {
	filtered := make([]*PostRow, 0, len(chronological))
	for _, row := range chronological {
		if row.URI != pinned.URI {
			filtered = append(filtered, row)
		}
	}
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	if len(filtered) == limit && (len(chronological) == storeLimit || nextCursor != "") {
		cursor, err := encodeProfileSeekCursor(filtered[len(filtered)-1])
		return filtered, cursor, err
	}
	return filtered, nextCursor, nil
}

func encodeProfileSeekCursor(row *PostRow) (string, error) {
	return envelope.EncodeCursor(map[string]any{
		"indexedAt": row.ProfileSortAt.UTC().Format(time.RFC3339Nano),
		"uri":       row.URI,
	})
}

func wrapProfileListCursor(cursor string, slot ProfilePinSlot, stateToken string) (string, error) {
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil || len(payload) != 2 {
		return "", envelope.ErrInvalidCursor
	}
	payload["profileListKind"] = string(slot)
	payload["pinStateToken"] = stateToken
	return envelope.EncodeCursor(payload)
}

func unwrapProfileListCursor(cursor string, slot ProfilePinSlot, stateToken string) (string, error) {
	payload, err := envelope.DecodeCursor(cursor)
	if err != nil || len(payload) != 4 {
		return "", envelope.ErrInvalidCursor
	}
	kind, kindOK := payload["profileListKind"].(string)
	token, tokenOK := payload["pinStateToken"].(string)
	if !kindOK || !tokenOK || kind != string(slot) || token != stateToken {
		return "", envelope.ErrInvalidCursor
	}
	delete(payload, "profileListKind")
	delete(payload, "pinStateToken")
	return envelope.EncodeCursor(payload)
}
