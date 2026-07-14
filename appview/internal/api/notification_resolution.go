package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

var ErrNotificationNotFound = errors.New("notification not found")

type NotificationResolution struct {
	ID     string             `json:"id"`
	Type   NotificationType   `json:"type"`
	State  string             `json:"state"`
	Target NotificationTarget `json:"target"`
}

type NotificationResolver interface {
	ResolveNotification(context.Context, string, string) (*NotificationResolution, error)
}

func ResolveNotificationHandler(store NotificationResolver, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, 500, "internal_error", "no did in context", runID, nil)
			return
		}
		resolution, err := store.ResolveNotification(r.Context(), did.String(), r.PathValue("notificationId"))
		if errors.Is(err, ErrNotificationNotFound) {
			envelope.WriteError(w, 404, "notification_not_found", "notification not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("notification resolution failed")
			envelope.WriteError(w, 500, "internal_error", "notification resolution failed", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resolution)
	})
}

type NotificationTarget struct {
	Kind string `json:"kind"`
	URI  string `json:"uri,omitempty"`
	DID  string `json:"did,omitempty"`
}

func (s *PostStore) ResolveNotification(ctx context.Context, viewerDID, rawID string) (*NotificationResolution, error) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, ErrNotificationNotFound
	}
	var category NotificationType
	var state, actorDID, sourceURI string
	var subjectURI, parentURI, rootURI, quotedURI sql.NullString
	err = s.pool.QueryRow(ctx, `
		SELECT category, state, actor_did, source_uri, subject_uri, parent_uri, root_uri, quoted_uri
		FROM notification_events
		WHERE id = $1 AND recipient_did = $2
	`, id, viewerDID).Scan(&category, &state, &actorDID, &sourceURI, &subjectURI, &parentURI, &rootURI, &quotedURI)
	if err == pgx.ErrNoRows {
		return nil, ErrNotificationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("resolve notification: %w", err)
	}
	resolution := &NotificationResolution{ID: id.String(), Type: category, State: state}
	postTarget := func(candidates ...string) (NotificationTarget, error) {
		for _, uri := range candidates {
			if uri == "" {
				continue
			}
			visible, err := s.notificationPostVisible(ctx, uri)
			if err != nil {
				return NotificationTarget{}, err
			}
			if visible {
				return NotificationTarget{Kind: "post", URI: uri}, nil
			}
		}
		return NotificationTarget{Kind: "notifications"}, nil
	}
	actorTarget := func() (NotificationTarget, error) {
		visible, err := s.notificationActorVisible(ctx, actorDID)
		if err != nil {
			return NotificationTarget{}, err
		}
		if visible {
			return NotificationTarget{Kind: "actorProfile", DID: actorDID}, nil
		}
		return NotificationTarget{Kind: "notifications"}, nil
	}
	switch category {
	case NotificationTypeFollow:
		resolution.Target, err = actorTarget()
	case NotificationTypeMention:
		resolution.Target, err = postTarget(sourceURI)
		if err == nil && resolution.Target.Kind == "notifications" {
			resolution.Target, err = actorTarget()
		}
	case NotificationTypeReply:
		resolution.Target, err = postTarget(sourceURI, parentURI.String, rootURI.String)
	case NotificationTypeQuote:
		resolution.Target, err = postTarget(sourceURI, quotedURI.String)
	case NotificationTypeLike, NotificationTypeRepost:
		resolution.Target, err = postTarget(subjectURI.String)
	default:
		resolution.Target = NotificationTarget{Kind: "notifications"}
	}
	if err != nil {
		return nil, fmt.Errorf("resolve target: %w", err)
	}
	return resolution, nil
}

func (s *PostStore) notificationPostVisible(ctx context.Context, uri string) (bool, error) {
	var visible bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM craftsky_posts p
			WHERE p.uri=$1
			  AND NOT EXISTS (
				SELECT 1 FROM moderation_outputs mo
				WHERE mo.action='apply'
				  AND mo.value IN ('hide','takedown')
				  AND (mo.expires_at IS NULL OR mo.expires_at>now())
				  AND ((mo.subject_type='post' AND mo.subject_uri=p.uri)
				       OR (mo.subject_type='account' AND mo.subject_did=p.did))
				  AND NOT EXISTS (
					SELECT 1 FROM moderation_outputs neg
					WHERE neg.action='negate'
					  AND neg.source_did=mo.source_did
					  AND neg.subject_type=mo.subject_type
					  AND neg.subject_did=mo.subject_did
					  AND neg.value=mo.value
					  AND (neg.expires_at IS NULL OR neg.expires_at>now())
					  AND neg.indexed_at>mo.indexed_at
					  AND (mo.subject_type='account' OR neg.subject_uri=mo.subject_uri)
				  )
			  )
		)`, uri).Scan(&visible)
	return visible, err
}

func (s *PostStore) notificationActorVisible(ctx context.Context, did string) (bool, error) {
	var visible bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM bluesky_profiles p
			WHERE p.did=$1
			  AND NOT EXISTS (
				SELECT 1 FROM moderation_outputs mo
				WHERE mo.action='apply'
				  AND mo.subject_type='account'
				  AND mo.subject_did=p.did
				  AND mo.value IN ('hide','takedown')
				  AND (mo.expires_at IS NULL OR mo.expires_at>now())
				  AND NOT EXISTS (
					SELECT 1 FROM moderation_outputs neg
					WHERE neg.action='negate'
					  AND neg.source_did=mo.source_did
					  AND neg.subject_type=mo.subject_type
					  AND neg.subject_did=mo.subject_did
					  AND neg.value=mo.value
					  AND (neg.expires_at IS NULL OR neg.expires_at>now())
					  AND neg.indexed_at>mo.indexed_at
				  )
			  )
		)`, did).Scan(&visible)
	return visible, err
}
