// appview/internal/api/notifications.go
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

type NotificationReader interface {
	ListNotifications(ctx context.Context, viewerDID string, limit int, cursor string) ([]*NotificationRow, string, error)
	EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

type NotificationPage struct {
	Items  []*NotificationItem `json:"items"`
	Cursor string              `json:"cursor,omitempty"`
}

type NotificationItem struct {
	URI         string                `json:"uri"`
	CID         string                `json:"cid"`
	Rkey        string                `json:"rkey"`
	Type        NotificationType      `json:"type"`
	Actor       NotificationActor     `json:"actor"`
	CreatedAt   string                `json:"createdAt"`
	IndexedAt   string                `json:"indexedAt"`
	SubjectPost *PostResponse         `json:"subjectPost,omitempty"`
	Reply       *NotificationReplyRef `json:"reply,omitempty"`
}

type NotificationActor struct {
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName,omitempty"`
	AvatarCID   *string `json:"avatarCid,omitempty"`
}

func ListNotificationsHandler(store NotificationReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		viewerDID, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}

		limit := parseNotificationLimit(r.URL.Query().Get("limit"))
		cursor := r.URL.Query().Get("cursor")
		rows, nextCursor, err := store.ListNotifications(r.Context(), viewerDID.String(), limit, cursor)
		if err != nil {
			if errors.Is(err, envelope.ErrInvalidCursor) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			logger.Error("notifications: list failed", slog.String("did", viewerDID.String()), slog.String("err", err.Error()), slog.String("run_id", runID))
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "notification list failed", runID, nil)
			return
		}

		items := make([]*NotificationItem, 0, len(rows))
		if len(rows) > 0 {
			dids := make([]string, 0, len(rows)*2)
			postURIs := make([]string, 0, len(rows))
			for _, row := range rows {
				dids = append(dids, row.ActorDID)
				if row.SubjectPost != nil {
					dids = append(dids, row.SubjectPost.DID)
					postURIs = append(postURIs, row.SubjectPost.URI)
				}
			}
			handles, err := resolveHandlesForDIDs(r.Context(), dids, resolver)
			if err != nil {
				logger.Warn("notifications: ResolveHandle failed", slog.String("err", err.Error()), slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway, "identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if err != nil {
				logger.Error("notifications: EngagementSummaries failed", slog.String("did", viewerDID.String()), slog.String("err", err.Error()), slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "post engagement lookup failed", runID, nil)
				return
			}
			for _, row := range rows {
				items = append(items, buildNotificationItem(row, handles, summaries))
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(NotificationPage{Items: items, Cursor: nextCursor})
	})
}

func parseNotificationLimit(raw string) int {
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

func buildNotificationItem(row *NotificationRow, handles map[string]syntax.Handle, summaries map[string]EngagementSummary) *NotificationItem {
	item := &NotificationItem{
		URI: row.URI, CID: row.CID, Rkey: row.Rkey, Type: row.Type,
		Actor: NotificationActor{
			DID:         row.ActorDID,
			Handle:      handles[row.ActorDID].String(),
			DisplayName: row.ActorDisplayName,
			AvatarCID:   row.ActorAvatarCID,
		},
		CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339),
		IndexedAt: row.IndexedAt.UTC().Format(time.RFC3339),
		Reply:     row.Reply,
	}
	if row.SubjectPost != nil {
		post := BuildPostResponse(row.SubjectPost, handles[row.SubjectPost.DID])
		applyEngagementSummary(post, summaries[row.SubjectPost.URI])
		item.SubjectPost = post
	}
	return item
}

func resolveHandlesForDIDs(ctx context.Context, dids []string, resolver HandleResolver) (map[string]syntax.Handle, error) {
	out := make(map[string]syntax.Handle, len(dids))
	for _, did := range dids {
		if did == "" {
			continue
		}
		if _, ok := out[did]; ok {
			continue
		}
		handle, err := resolver.ResolveHandle(ctx, syntax.DID(did))
		if err != nil {
			return nil, err
		}
		out[did] = handle
	}
	return out, nil
}
