package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type NotificationDeviceResponse struct {
	AccountSubscriptionID string `json:"accountSubscriptionId"`
}

type NotificationDeviceStore interface {
	RegisterNotificationDevice(context.Context, string, string, string, string) (string, error)
	RemoveNotificationSubscription(context.Context, string, string, string) error
}

var ErrNotificationSubscriptionNotFound = errors.New("notification subscription not found")

func (s *PostStore) RegisterNotificationDevice(ctx context.Context, accountDID, deviceID, platform, token string) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := ownerlifecycle.GuardPrivateMutationTx(ctx, tx, syntax.DID(accountDID), nil); err != nil {
		return "", fmt.Errorf("authorize notification device registration: %w", err)
	}
	var oldInstallation uuid.UUID
	var oldDevice string
	err = tx.QueryRow(ctx, `SELECT id, device_id FROM push_installations WHERE fcm_token = $1 AND active FOR UPDATE`, token).Scan(&oldInstallation, &oldDevice)
	if err != nil && err != pgx.ErrNoRows {
		return "", err
	}
	if err == nil && oldDevice != deviceID {
		if _, err := tx.Exec(ctx, `UPDATE push_installations SET active=false, deactivated_at=now(), updated_at=now() WHERE id=$1`, oldInstallation); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE push_account_subscriptions SET active=false, deactivated_at=now(), updated_at=now() WHERE installation_id=$1 AND active`, oldInstallation); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, `UPDATE push_deliveries SET status='cancelled', updated_at=now() WHERE account_subscription_id IN (SELECT id FROM push_account_subscriptions WHERE installation_id=$1) AND status IN ('pending','retry','leased')`, oldInstallation); err != nil {
			return "", err
		}
	}
	installationID := uuid.New()
	if err := tx.QueryRow(ctx, `
		INSERT INTO push_installations (id, device_id, platform, fcm_token) VALUES ($1,$2,$3,$4)
		ON CONFLICT (device_id) DO UPDATE SET platform=EXCLUDED.platform, fcm_token=EXCLUDED.fcm_token, active=true, deactivated_at=NULL, updated_at=now()
		RETURNING id
	`, installationID, deviceID, platform, token).Scan(&installationID); err != nil {
		return "", err
	}
	subscriptionID, routingID := uuid.New(), uuid.New()
	if err := tx.QueryRow(ctx, `
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id) VALUES ($1,$2,$3,$4)
		ON CONFLICT (installation_id, account_did) DO UPDATE SET active=true, deactivated_at=NULL, updated_at=now()
		RETURNING routing_id
	`, subscriptionID, installationID, accountDID, routingID).Scan(&routingID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return routingID.String(), nil
}

func (s *PostStore) RemoveNotificationSubscription(ctx context.Context, accountDID, deviceID, rawRoutingID string) error {
	routingID, err := uuid.Parse(rawRoutingID)
	if err != nil {
		return ErrNotificationSubscriptionNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var subscriptionID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE push_account_subscriptions subscription
		SET active=false, deactivated_at=now(), updated_at=now()
		FROM push_installations installation
		WHERE subscription.installation_id=installation.id
		  AND subscription.routing_id=$1 AND subscription.account_did=$2
		  AND installation.device_id=$3 AND subscription.active
		RETURNING subscription.id
	`, routingID, accountDID, deviceID).Scan(&subscriptionID)
	if err == pgx.ErrNoRows {
		return ErrNotificationSubscriptionNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE push_deliveries SET status='cancelled', updated_at=now(), lease_owner=NULL, lease_expires_at=NULL WHERE account_subscription_id=$1 AND status IN ('pending','retry','leased')`, subscriptionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PostStore) DeactivateForInstallation(ctx context.Context, accountDID, deviceID string) error {
	return s.deactivateSubscriptions(ctx, `
		SELECT subscription.id FROM push_account_subscriptions subscription
		JOIN push_installations installation ON installation.id=subscription.installation_id
		WHERE subscription.account_did=$1 AND installation.device_id=$2 AND subscription.active
		FOR UPDATE OF subscription
	`, accountDID, deviceID)
}

func (s *PostStore) DeactivateForAccount(ctx context.Context, accountDID string) error {
	return s.deactivateSubscriptions(ctx, `
		SELECT id FROM push_account_subscriptions WHERE account_did=$1 AND active FOR UPDATE
	`, accountDID)
}

// DeactivateForInstallationBefore applies a durable auth-cleanup job without
// crossing a later device registration. Registration and cleanup serialize on
// the subscription row; updated_at is the generation fence between them.
func (s *PostStore) DeactivateForInstallationBefore(
	ctx context.Context,
	accountDID string,
	deviceID string,
	cutoff time.Time,
) error {
	return s.deactivateSubscriptions(ctx, `
		SELECT subscription.id FROM push_account_subscriptions subscription
		JOIN push_installations installation ON installation.id=subscription.installation_id
		WHERE subscription.account_did=$1 AND installation.device_id=$2
		  AND subscription.active AND subscription.updated_at <= $3
		FOR UPDATE OF subscription
	`, accountDID, deviceID, cutoff)
}

// DeactivateForAccountBefore is the DID-wide counterpart. Rows reactivated
// after the cleanup job was committed remain active.
func (s *PostStore) DeactivateForAccountBefore(
	ctx context.Context,
	accountDID string,
	cutoff time.Time,
) error {
	return s.deactivateSubscriptions(ctx, `
		SELECT id FROM push_account_subscriptions
		WHERE account_did=$1 AND active AND updated_at <= $2
		FOR UPDATE
	`, accountDID, cutoff)
}

func (s *PostStore) deactivateSubscriptions(ctx context.Context, query string, args ...any) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	ids, err := scanNotificationSubscriptionIDs(rows)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE push_account_subscriptions SET active=false,deactivated_at=now(),updated_at=now() WHERE id=$1`, id); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE push_deliveries SET status='cancelled',updated_at=now(),lease_owner=NULL,lease_expires_at=NULL WHERE account_subscription_id=$1 AND status IN ('pending','retry','leased')`, id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

type notificationSubscriptionRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close()
}

func scanNotificationSubscriptionIDs(rows notificationSubscriptionRows) ([]uuid.UUID, error) {
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan notification subscription: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification subscriptions: %w", err)
	}
	return ids, nil
}

func RegisterNotificationDeviceHandler(store NotificationDeviceStore, logger *slog.Logger) http.Handler {
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
		deviceID, ok := middleware.GetDeviceID(r.Context())
		if !ok {
			envelope.WriteError(w, 400, "device_id_required", "device id required", runID, nil)
			return
		}
		var request struct {
			Platform string `json:"platform"`
			Token    string `json:"token"`
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if decoder.Decode(&request) != nil || (request.Platform != "ios" && request.Platform != "android") || strings.TrimSpace(request.Token) == "" || len(request.Token) > 4096 {
			envelope.WriteError(w, 400, "invalid_request", "invalid device registration", runID, nil)
			return
		}
		routingID, err := store.RegisterNotificationDevice(r.Context(), did.String(), deviceID, request.Platform, request.Token)
		if err != nil {
			logger.Error("notification device registration failed")
			envelope.WriteError(w, 500, "internal_error", "device registration failed", runID, nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(NotificationDeviceResponse{AccountSubscriptionID: routingID})
	})
}

func RemoveNotificationDeviceHandler(store NotificationDeviceStore, logger *slog.Logger) http.Handler {
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
		deviceID, ok := middleware.GetDeviceID(r.Context())
		if !ok {
			envelope.WriteError(w, 400, "device_id_required", "device id required", runID, nil)
			return
		}
		err := store.RemoveNotificationSubscription(r.Context(), did.String(), deviceID, r.PathValue("accountSubscriptionId"))
		if errors.Is(err, ErrNotificationSubscriptionNotFound) {
			envelope.WriteError(w, 404, "subscription_not_found", "subscription not found", runID, nil)
			return
		}
		if err != nil {
			logger.Error("notification subscription removal failed")
			envelope.WriteError(w, 500, "internal_error", "subscription removal failed", runID, nil)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
