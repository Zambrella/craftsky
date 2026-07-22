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

const (
	instagramMatchCoalescingWindow = 5 * time.Minute
	instagramMatchCountCap         = 99
)

type InstagramMatchActivation struct {
	RecipientDID syntax.DID
	SuggestionID uuid.UUID
	ActivityAt   time.Time
}

type InstagramMatchRetraction struct {
	SuggestionID uuid.UUID
	Reason       string
}

// ActivateInstagramMatch attaches one newly eligible future suggestion to the
// recipient's current fixed five-minute actorless digest. Reconciliation
// replays for a suggestion already attached to any digest are a no-op.
func (s *Service) ActivateInstagramMatch(ctx context.Context, tx pgx.Tx, activation InstagramMatchActivation) error {
	if s == nil || tx == nil || activation.RecipientDID == "" || activation.SuggestionID == uuid.Nil {
		return errors.New("invalid Instagram match notification activation")
	}
	at := activation.ActivityAt.UTC()
	if at.IsZero() {
		at = s.now().UTC()
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 7))`, activation.RecipientDID); err != nil {
		return fmt.Errorf("lock Instagram match notification: %w", err)
	}

	var alreadyAttached bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM instagram_notification_suggestions
			WHERE suggestion_id=$1
		)
	`, activation.SuggestionID).Scan(&alreadyAttached); err != nil {
		return fmt.Errorf("read Instagram match support: %w", err)
	}
	if alreadyAttached {
		return nil
	}

	preference := defaultPreference
	err := tx.QueryRow(ctx, `
		SELECT scope, push_enabled
		FROM notification_preferences
		WHERE account_did=$1 AND category=$2
	`, activation.RecipientDID, InstagramMatch).Scan(&preference.Scope, &preference.PushEnabled)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read Instagram match preference: %w", err)
	}
	if preference.Scope != Everyone {
		return errors.New("invalid Instagram match notification scope")
	}

	var eventID uuid.UUID
	var coalesceUntil time.Time
	created := false
	err = tx.QueryRow(ctx, `
		SELECT id, coalesce_until
		FROM notification_events
		WHERE recipient_did=$1
		  AND kind='system'
		  AND category='instagramMatch'
		  AND state='active'
		  AND coalesce_until>$2
		ORDER BY coalesce_until DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`, activation.RecipientDID, at).Scan(&eventID, &coalesceUntil)
	if errors.Is(err, pgx.ErrNoRows) {
		created = true
		eventID = uuid.New()
		coalesceUntil = at.Add(s.instagramCoalescingWindow)
		groupKey := at.Format(time.RFC3339Nano)
		if _, err := tx.Exec(ctx, `
			INSERT INTO notification_events (
				id, recipient_did, kind, category, subject_key,
				eligibility_scope, recipient_followed_actor,
				push_enabled_snapshot, state, first_activity_at,
				activity_at, indexed_at, initial_push_evaluated_at,
				system_count, system_count_capped, system_destination,
				system_group_key, coalesce_until
			) VALUES (
				$1, $2, 'system', 'instagramMatch', $3,
				'everyone', false, $4, 'active', $5,
				$5, $5, $5, 1, false, 'instagramMigration', $3, $6
			)
		`, eventID, activation.RecipientDID, groupKey, preference.PushEnabled, at, coalesceUntil); err != nil {
			return fmt.Errorf("create Instagram match notification: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("read Instagram match notification: %w", err)
	}

	inserted, err := tx.Exec(ctx, `
		INSERT INTO instagram_notification_suggestions (notification_id, suggestion_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
	`, eventID, activation.SuggestionID, at)
	if err != nil {
		return fmt.Errorf("attach Instagram match suggestion: %w", err)
	}
	if inserted.RowsAffected() == 0 {
		return nil
	}

	if !created {
		var supportCount int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM instagram_notification_suggestions
			WHERE notification_id=$1
		`, eventID).Scan(&supportCount); err != nil {
			return fmt.Errorf("count Instagram match suggestions: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE notification_events
			SET system_count=LEAST($2, $5::integer),
			    system_count_capped=$2>$5::integer,
			    push_enabled_snapshot=$3,
			    activity_at=GREATEST(activity_at,$4),
			    indexed_at=GREATEST(indexed_at,$4)
			WHERE id=$1 AND kind='system' AND state='active'
		`, eventID, supportCount, preference.PushEnabled, at, s.instagramCountCap); err != nil {
			return fmt.Errorf("coalesce Instagram match notification: %w", err)
		}
	}

	if created && preference.PushEnabled {
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
		`, eventID, coalesceUntil, coalesceUntil.Add(deliveryWindow), activation.RecipientDID); err != nil {
			return fmt.Errorf("schedule Instagram match notification: %w", err)
		}
	}

	if s.observer != nil {
		result := "created"
		if !created {
			result = "coalesced"
		}
		s.observer.ObserveNotificationDecision(string(InstagramMatch), result)
	}
	return nil
}

// RetractInstagramMatch removes one suggestion's support from its digest.
// Remaining support is recounted; zero support retracts the actorless item and
// cancels every unsent delivery, including a currently leased one.
func (s *Service) RetractInstagramMatch(ctx context.Context, tx pgx.Tx, retraction InstagramMatchRetraction) error {
	if s == nil || tx == nil || retraction.SuggestionID == uuid.Nil {
		return errors.New("invalid Instagram match notification retraction")
	}
	now := s.now().UTC()
	reason := safeInstagramRetractionReason(retraction.Reason)

	var eventID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT support.notification_id
		FROM instagram_notification_suggestions support
		JOIN notification_events event ON event.id=support.notification_id
		WHERE support.suggestion_id=$1
		FOR UPDATE OF support, event
	`, retraction.SuggestionID).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read Instagram match retraction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM instagram_notification_suggestions
		WHERE notification_id=$1 AND suggestion_id=$2
	`, eventID, retraction.SuggestionID); err != nil {
		return fmt.Errorf("remove Instagram match support: %w", err)
	}

	var supportCount int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM instagram_notification_suggestions
		WHERE notification_id=$1
	`, eventID).Scan(&supportCount); err != nil {
		return fmt.Errorf("recount Instagram match support: %w", err)
	}
	if supportCount > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE notification_events
			SET system_count=LEAST($2,$3::integer), system_count_capped=$2>$3::integer
			WHERE id=$1 AND kind='system' AND state='active'
		`, eventID, supportCount, s.instagramCountCap); err != nil {
			return fmt.Errorf("update Instagram match count: %w", err)
		}
		return nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notification_events
		SET state='retracted', retracted_at=$2, retraction_reason=$3
		WHERE id=$1 AND kind='system' AND state='active'
	`, eventID, now, reason); err != nil {
		return fmt.Errorf("retract Instagram match notification: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE push_deliveries
		SET status='cancelled', lease_owner=NULL, lease_expires_at=NULL, updated_at=$2
		WHERE notification_id=$1 AND status IN ('pending','retry','leased')
	`, eventID, now); err != nil {
		return fmt.Errorf("cancel Instagram match delivery: %w", err)
	}
	if s.observer != nil {
		s.observer.ObserveNotificationDecision(string(InstagramMatch), "retracted")
	}
	return nil
}

func safeInstagramRetractionReason(reason string) string {
	switch reason {
	case "suggestion_invalidated", "import_deleted", "link_revoked", "membership_inactive", "eligibility_changed":
		return reason
	default:
		return "suggestion_invalidated"
	}
}
