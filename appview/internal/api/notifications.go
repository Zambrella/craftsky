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
	NotificationHandles(ctx context.Context, dids []string) (map[string]syntax.Handle, error)
	EngagementSummaries(ctx context.Context, viewerDID string, postURIs []string) (map[string]EngagementSummary, error)
}

type NotificationPage struct {
	Items  []*NotificationItem `json:"items"`
	Cursor string              `json:"cursor,omitempty"`
}

type NotificationItem struct {
	ID               string                 `json:"id"`
	URI              string                 `json:"uri,omitempty"`
	CID              string                 `json:"cid,omitempty"`
	Rkey             string                 `json:"rkey,omitempty"`
	Type             NotificationType       `json:"type"`
	Actor            NotificationActor      `json:"actor"`
	References       NotificationReferences `json:"references"`
	CreatedAt        string                 `json:"createdAt"`
	IndexedAt        string                 `json:"indexedAt"`
	SubjectPost      *PostResponse          `json:"subjectPost,omitempty"`
	Reply            *NotificationReplyRef  `json:"reply,omitempty"`
	ContentAvailable *bool                  `json:"contentAvailable,omitempty"`
}

type NotificationActor struct {
	Available         bool    `json:"available"`
	DID               string  `json:"did"`
	Handle            string  `json:"handle"`
	DisplayName       *string `json:"displayName,omitempty"`
	Avatar            *string `json:"avatar,omitempty"`
	AvatarCID         *string `json:"avatarCid,omitempty"`
	ViewerIsFollowing bool    `json:"viewerIsFollowing"`
	Muted             bool    `json:"muted"`
	Blocking          bool    `json:"blocking"`
	BlockedBy         bool    `json:"blockedBy"`
}

func ListNotificationsHandler(store NotificationReader, _ HandleResolver, logger *slog.Logger) http.Handler {
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
			logger.Error("notifications: list failed",
				apiLogErrorAttrs(runID, "notifications.list", "store")...)
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
			handles, err := store.NotificationHandles(r.Context(), dids)
			if err != nil {
				logger.Error("notifications: indexed handle batch failed",
					apiLogErrorAttrs(runID, "notifications.list", "store")...)
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "notification identity lookup failed", runID, nil)
				return
			}
			summaries, err := store.EngagementSummaries(r.Context(), viewerDID.String(), postURIs)
			if err != nil {
				logger.Error("notifications: EngagementSummaries failed",
					apiLogErrorAttrs(runID, "notifications.list", "engagement")...)
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
	actorHandle, actorAvailable := handles[row.ActorDID]
	item := &NotificationItem{
		ID:   row.ID,
		Type: row.Type,
		Actor: NotificationActor{
			Available:         actorAvailable,
			DID:               row.ActorDID,
			Handle:            actorHandle.String(),
			DisplayName:       row.ActorDisplayName,
			AvatarCID:         row.ActorAvatarCID,
			ViewerIsFollowing: row.ActorViewerIsFollowing,
		},
		CreatedAt:  row.CreatedAt.UTC().Format(time.RFC3339),
		IndexedAt:  row.IndexedAt.UTC().Format(time.RFC3339),
		Reply:      row.Reply,
		References: row.References,
	}
	if avatar := synthBlobURL("avatar", row.ActorDID, row.ActorAvatarCID, row.ActorAvatarMime); avatar != "" {
		item.Actor.Avatar = &avatar
	}
	if row.References.Source.Available {
		item.URI = row.References.Source.URI
		item.CID = row.References.Source.CID
		item.Rkey = row.References.Source.Rkey
	}
	if !actorAvailable {
		item.Actor.DisplayName = nil
		item.Actor.Avatar = nil
		item.Actor.AvatarCID = nil
	}
	if row.SubjectPost != nil {
		post := BuildPostResponse(row.SubjectPost, handles[row.SubjectPost.DID])
		applyEngagementSummary(post, summaries[row.SubjectPost.URI])
		item.SubjectPost = post
	}
	switch row.Type {
	case NotificationTypeLike, NotificationTypeRepost:
		available := row.References.Subject != nil && row.References.Subject.Available
		item.ContentAvailable = &available
	case NotificationTypeReply, NotificationTypeMention, NotificationTypeQuote:
		available := row.References.Source.Available
		item.ContentAvailable = &available
	}
	return item
}
