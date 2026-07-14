package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type NotificationNewCountStore interface {
	NotificationNewCount(context.Context, string) (int64, error)
}

type NotificationSeenStore interface {
	MarkNotificationsSeen(context.Context, string) error
}

type NotificationNewCountResponse struct {
	NewCount int64 `json:"newCount"`
}

func (s *PostStore) NotificationNewCount(ctx context.Context, accountDID string) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM notification_events event
		WHERE event.recipient_did = $1
		  AND event.state = 'active'
		  AND event.newness_revision > COALESCE((
			SELECT seen.last_seen_revision
			FROM notification_seen_state seen
			WHERE seen.account_did = $1
		  ), 0)
		  AND NOT EXISTS (
			SELECT 1
			FROM moderation_outputs output
			WHERE output.action = 'apply'
			  AND output.subject_type = 'account'
			  AND output.subject_did = event.actor_did
			  AND output.value IN ('hide', 'takedown')
			  AND (output.expires_at IS NULL OR output.expires_at > now())
			  AND NOT EXISTS (
				SELECT 1
				FROM moderation_outputs negation
				WHERE negation.action = 'negate'
				  AND negation.source_did = output.source_did
				  AND negation.subject_type = output.subject_type
				  AND negation.subject_did = output.subject_did
				  AND negation.value = output.value
				  AND (negation.expires_at IS NULL OR negation.expires_at > now())
				  AND negation.indexed_at > output.indexed_at
			  )
		  )
	`, accountDID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("notification new count: %w", err)
	}
	return count, nil
}

func (s *PostStore) MarkNotificationsSeen(ctx context.Context, accountDID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notification_seen_state (
			account_did,
			last_seen_revision,
			updated_at
		)
		SELECT $1, COALESCE(max(event.newness_revision), 0), now()
		FROM notification_events event
		WHERE event.recipient_did = $1
		ON CONFLICT (account_did) DO UPDATE SET
			last_seen_revision = GREATEST(
				notification_seen_state.last_seen_revision,
				EXCLUDED.last_seen_revision
			),
			updated_at = now()
	`, accountDID)
	if err != nil {
		return fmt.Errorf("mark notifications seen: %w", err)
	}
	return nil
}

func NotificationNewCountHandler(store NotificationNewCountStore, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		count, err := store.NotificationNewCount(r.Context(), did.String())
		if err != nil {
			logger.Error("notification new count failed",
				apiLogErrorAttrs(runID, "notifications.new_count", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "notification new count failed", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NotificationNewCountResponse{NewCount: count})
	})
}

func MarkNotificationsSeenHandler(store NotificationSeenStore, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		if err := store.MarkNotificationsSeen(r.Context(), did.String()); err != nil {
			logger.Error("mark notifications seen failed",
				apiLogErrorAttrs(runID, "notifications.seen", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "mark notifications seen failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
