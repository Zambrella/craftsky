// appview/internal/api/timeline.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// TimelineReader is the read-side boundary for the home timeline handler.
type TimelineReader interface {
	ListTimeline(ctx context.Context, viewerDID string, limit int, cursor string) ([]*PostRow, string, error)
	EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

// TimelinePage is the JSON list shape for GET /v1/feed/timeline.
type TimelinePage struct {
	Items  []*PostResponse `json:"items"`
	Cursor string          `json:"cursor,omitempty"`
}

// ListTimelineHandler serves GET /v1/feed/timeline.
func ListTimelineHandler(store TimelineReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}

		limit := parseTimelineLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		rows, nextCursor, err := store.ListTimeline(r.Context(), viewerDID.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("timeline: list failed",
				apiLogErrorAttrs(runID, "timeline.list", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "timeline list failed", runID, nil)
			return
		}

		items := make([]*PostResponse, 0, len(rows))
		if len(rows) > 0 {
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				postURIs = append(postURIs, row.URI)
			}
			summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if err != nil {
				logger.Error("timeline: EngagementSummaries failed",
					apiLogErrorAttrs(runID, "timeline.list", "engagement")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			handles, err := resolveHandlesForRows(r.Context(), rows, resolver)
			if err != nil {
				logger.Warn("timeline: ResolveHandle failed",
					apiLogErrorAttrs(runID, "timeline.list", "identity")...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			for _, row := range rows {
				resp := BuildPostResponse(row, handles[row.DID])
				applyEngagementSummary(resp, summaries[row.URI])
				items = append(items, resp)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TimelinePage{Items: items, Cursor: nextCursor})
	})
}

func parseTimelineLimit(raw string) int {
	const defaultLimit, maxLimit = 20, 50
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}
