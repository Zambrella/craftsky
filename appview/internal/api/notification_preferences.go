package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
)

type NotificationPreferencesResponse struct {
	Preferences map[notifications.Category]notifications.Preference `json:"preferences"`
}

type NotificationPreferenceStore interface {
	NotificationPreferences(context.Context, string) (map[notifications.Category]notifications.Preference, error)
	PatchNotificationPreferences(context.Context, string, map[notifications.Category]notifications.PreferencePatch) (map[notifications.Category]notifications.Preference, error)
}

func (s *PostStore) NotificationPreferences(ctx context.Context, did string) (map[notifications.Category]notifications.Preference, error) {
	rows, err := s.pool.Query(ctx, `SELECT category, scope, push_enabled FROM notification_preferences WHERE account_did = $1`, did)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	persisted := map[notifications.Category]notifications.Preference{}
	for rows.Next() {
		var category notifications.Category
		var preference notifications.Preference
		if err := rows.Scan(&category, &preference.Scope, &preference.PushEnabled); err != nil {
			return nil, err
		}
		persisted[category] = preference
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notifications.ResolvePreferences(persisted, nil)
}

func (s *PostStore) PatchNotificationPreferences(ctx context.Context, did string, patch map[notifications.Category]notifications.PreferencePatch) (map[notifications.Category]notifications.Preference, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT category, scope, push_enabled FROM notification_preferences WHERE account_did = $1 FOR UPDATE`, did)
	if err != nil {
		return nil, err
	}
	persisted := map[notifications.Category]notifications.Preference{}
	for rows.Next() {
		var category notifications.Category
		var preference notifications.Preference
		if err := rows.Scan(&category, &preference.Scope, &preference.PushEnabled); err != nil {
			rows.Close()
			return nil, err
		}
		persisted[category] = preference
	}
	rows.Close()
	resolved, err := notifications.ResolvePreferences(persisted, patch)
	if err != nil {
		return nil, err
	}
	for category := range patch {
		preference := resolved[category]
		if _, err := tx.Exec(ctx, `
			INSERT INTO notification_preferences (account_did, category, scope, push_enabled)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (account_did, category) DO UPDATE SET scope = EXCLUDED.scope, push_enabled = EXCLUDED.push_enabled, updated_at = now()
		`, did, category, preference.Scope, preference.PushEnabled); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return resolved, nil
}

func GetNotificationPreferencesHandler(store NotificationPreferenceStore, logger *slog.Logger) http.Handler {
	return notificationPreferencesHandler(store, logger, false)
}

func PatchNotificationPreferencesHandler(store NotificationPreferenceStore, logger *slog.Logger) http.Handler {
	return notificationPreferencesHandler(store, logger, true)
}

func notificationPreferencesHandler(store NotificationPreferenceStore, logger *slog.Logger, patching bool) http.Handler {
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
		var preferences map[notifications.Category]notifications.Preference
		var err error
		if patching {
			var request struct {
				Preferences map[notifications.Category]notifications.PreferencePatch `json:"preferences"`
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if decoder.Decode(&request) != nil || request.Preferences == nil {
				envelope.WriteError(w, 400, "invalid_request", "invalid preferences", runID, nil)
				return
			}
			preferences, err = store.PatchNotificationPreferences(r.Context(), did.String(), request.Preferences)
		} else {
			preferences, err = store.NotificationPreferences(r.Context(), did.String())
		}
		if err != nil {
			if patching {
				envelope.WriteError(w, 400, "invalid_preferences", "invalid notification preferences", runID, nil)
				return
			}
			logger.Error("notification preferences failed")
			envelope.WriteError(w, 500, "internal_error", "notification preferences failed", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NotificationPreferencesResponse{Preferences: preferences})
	})
}
