// appview/internal/api/timeline.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// TimelineReader is the read-side boundary for the home timeline handler.
type TimelineReader interface {
	ListTimeline(ctx context.Context, viewerDID string, limit int, cursor string) ([]*TimelineFeedItemRow, string, error)
	EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
	QuoteViewRows(ctx context.Context, refs []ResponseStrongRef) (map[string]*QuoteViewRow, error)
}

// TimelinePage is the JSON list shape for GET /v1/feed/timeline.
type TimelinePage struct {
	Items  []*TimelineFeedItemResponse `json:"items"`
	Cursor string                      `json:"cursor,omitempty"`
}

// TimelineFeedItemResponse is the home-timeline item shape. Repost
// attribution lives here instead of on post-shaped responses.
type TimelineFeedItemResponse struct {
	ItemKey string                `json:"itemKey"`
	Post    *PostResponse         `json:"post"`
	Reason  *TimelineReasonRepost `json:"reason,omitempty"`
}

// TimelineReasonRepost is the public, lightweight straight-repost reason.
type TimelineReasonRepost struct {
	Type      string     `json:"type"`
	By        PostAuthor `json:"by"`
	URI       string     `json:"uri"`
	CID       string     `json:"cid,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	IndexedAt time.Time  `json:"indexedAt"`
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

		items := make([]*TimelineFeedItemResponse, 0, len(rows))
		if len(rows) > 0 {
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				postURIs = append(postURIs, row.Post.URI)
			}
			summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if err != nil {
				logger.Error("timeline: EngagementSummaries failed",
					apiLogErrorAttrs(runID, "timeline.list", "engagement")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			postRows := timelinePostRows(rows)
			handles, err := resolveHandlesForTimelineItems(r.Context(), rows, postRows, resolver)
			if err != nil {
				logger.Warn("timeline: ResolveHandle failed",
					apiLogErrorAttrs(runID, "timeline.list", "identity")...)
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			postResponses := make([]*PostResponse, 0, len(rows))
			for _, row := range rows {
				resp := BuildPostResponse(row.Post, handles[row.Post.DID])
				applyEngagementSummary(resp, summaries[row.Post.URI])
				postResponses = append(postResponses, resp)
			}
			if err := attachQuoteViews(r.Context(), store, resolver, postResponses); err != nil {
				logger.Error("timeline: QuoteViewRows failed",
					apiLogErrorAttrs(runID, "timeline.list", "quote_view")...)
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "post quote lookup failed", runID, nil)
				return
			}
			for i, row := range rows {
				items = append(items, BuildTimelineFeedItemResponse(row, postResponses[i], handles))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(TimelinePage{Items: items, Cursor: nextCursor})
	})
}

func BuildTimelineFeedItemResponse(row *TimelineFeedItemRow, post *PostResponse, handles map[string]syntax.Handle) *TimelineFeedItemResponse {
	resp := &TimelineFeedItemResponse{
		ItemKey: row.ItemKey,
		Post:    post,
	}
	if row.Repost == nil {
		return resp
	}
	by := PostAuthor{
		DID:         row.Repost.DID,
		Handle:      handles[row.Repost.DID].String(),
		DisplayName: row.Repost.AuthorDisplayName,
		AvatarCID:   row.Repost.AuthorAvatarCID,
	}
	if avatar := synthBlobURL("avatar", row.Repost.DID, row.Repost.AuthorAvatarCID, row.Repost.AuthorAvatarMime); avatar != "" {
		by.Avatar = &avatar
	}
	resp.Reason = &TimelineReasonRepost{
		Type:      "repost",
		By:        by,
		URI:       row.Repost.URI,
		CID:       row.Repost.CID,
		CreatedAt: row.Repost.CreatedAt.UTC(),
		IndexedAt: row.Repost.IndexedAt.UTC(),
	}
	return resp
}

func timelinePostRows(items []*TimelineFeedItemRow) []*PostRow {
	rows := make([]*PostRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, item.Post)
	}
	return rows
}

func resolveHandlesForTimelineItems(ctx context.Context, items []*TimelineFeedItemRow, rows []*PostRow, resolver HandleResolver) (map[string]syntax.Handle, error) {
	handles, err := resolveHandlesForRows(ctx, rows, resolver)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.Repost == nil {
			continue
		}
		if _, ok := handles[item.Repost.DID]; ok {
			continue
		}
		did, err := syntax.ParseDID(item.Repost.DID)
		if err != nil {
			return nil, err
		}
		handle, err := resolver.ResolveHandle(ctx, did)
		if err != nil {
			return nil, err
		}
		handles[item.Repost.DID] = handle
	}
	return handles, nil
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
