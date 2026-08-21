package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type InstagramMatchActivation struct {
	RecipientDID syntax.DID
	ActorDID     syntax.DID
	SuggestionID uuid.UUID
	ActivityAt   time.Time
}

// ActivateInstagramMatch records the private notification in the caller's
// transaction. The suggestion ID is the idempotency key, so reconciliation
// replays cannot create another event or delivery.
func (s *Service) ActivateInstagramMatch(
	ctx context.Context,
	tx pgx.Tx,
	activation InstagramMatchActivation,
) error {
	if s == nil || tx == nil || activation.RecipientDID == "" ||
		activation.ActorDID == "" || activation.RecipientDID == activation.ActorDID ||
		activation.SuggestionID == uuid.Nil {
		return errors.New("invalid Instagram match notification activation")
	}
	activityAt := activation.ActivityAt.UTC()
	if activityAt.IsZero() {
		activityAt = s.now().UTC()
	}
	indexedAt := s.now().UTC()

	preference := defaultPreference
	err := tx.QueryRow(ctx, `
		SELECT scope, push_enabled
		FROM notification_preferences
		WHERE account_did=$1 AND category='instagramMatch'
	`, activation.RecipientDID).Scan(&preference.Scope, &preference.PushEnabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read Instagram match preference: %w", err)
	}
	if preference.Scope != Everyone {
		return errors.New("invalid Instagram match notification scope")
	}

	notificationID := uuid.New()
	var insertedID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			eligibility_scope, recipient_followed_actor,
			push_enabled_snapshot, state, first_activity_at, activity_at,
			indexed_at, initial_push_evaluated_at
		) VALUES (
			$1,$2,$3,'instagramMatch',$4,
			'everyone',false,$5,'active',$6,$6,$7,$7
		)
		ON CONFLICT DO NOTHING
		RETURNING id
	`, notificationID, activation.RecipientDID, activation.ActorDID,
		activation.SuggestionID.String(), preference.PushEnabled, activityAt, indexedAt).Scan(&insertedID)
	if errors.Is(err, pgx.ErrNoRows) {
		if s.observer != nil {
			s.observer.ObserveNotificationDecision(string(InstagramMatch), "duplicate")
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("create Instagram match notification: %w", err)
	}

	if preference.PushEnabled {
		if _, err := tx.Exec(ctx, `
			INSERT INTO push_deliveries (
				id, notification_id, account_subscription_id,
				status, next_attempt_at, deadline_at
			)
			SELECT gen_random_uuid(), $1, subscription.id,
			       'pending', $2, $3
			FROM push_account_subscriptions subscription
			JOIN push_installations installation
			  ON installation.id=subscription.installation_id
			WHERE subscription.account_did=$4
			  AND subscription.active
			  AND installation.active
			ON CONFLICT (notification_id, account_subscription_id) DO NOTHING
		`, insertedID, indexedAt, activityAt.Add(deliveryWindow), activation.RecipientDID); err != nil {
			return fmt.Errorf("schedule Instagram match notification: %w", err)
		}
	}
	if s.observer != nil {
		s.observer.ObserveNotificationDecision(string(InstagramMatch), "created")
	}
	return nil
}
