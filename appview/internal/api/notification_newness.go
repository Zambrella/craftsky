package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
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
		FROM notification_events e
		WHERE e.recipient_did = $1
		  AND e.state = 'active'
		  AND e.is_new
		  `+notificationVisibilityPredicate+`
	`, accountDID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("notification new count: %w", err)
	}
	return count, nil
}

func (s *PostStore) MarkNotificationsSeen(ctx context.Context, accountDID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mark notifications seen begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := ownerlifecycle.GuardPrivateMutationTx(ctx, tx, syntax.DID(accountDID), nil); err != nil {
		return fmt.Errorf("mark notifications seen authorization: %w", err)
	}
	// Capture each activation in the statement snapshot. Sequence allocation
	// order can differ from commit order, and a concurrent reactivation must
	// remain new if its row changes while this update is waiting for a lock.
	_, err = tx.Exec(ctx, `
		WITH unseen AS MATERIALIZED (
			SELECT id, newness_revision
			FROM notification_events
			WHERE recipient_did = $1 AND state = 'active' AND is_new
		)
		UPDATE notification_events AS event
		SET is_new = false
		FROM unseen
		WHERE event.id = unseen.id
		  AND event.newness_revision = unseen.newness_revision
		  AND event.state = 'active'
		  AND event.is_new
	`, accountDID)
	if err != nil {
		return fmt.Errorf("mark notifications seen: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("mark notifications seen commit: %w", err)
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
				apiLogErrorAttrs(r.Context(), runID, "notifications.new_count", "store", err)...)
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
				apiLogErrorAttrs(r.Context(), runID, "notifications.seen", "store", err)...)
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "mark notifications seen failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
