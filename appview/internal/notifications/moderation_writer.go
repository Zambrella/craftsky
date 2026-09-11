package notifications

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ModerationWriter persists one actorless event per authoritative case event.
// The current preference controls push fan-out only, never moderation history.
type ModerationWriter struct{}

func (ModerationWriter) InsertModerationIntentTx(ctx context.Context, tx pgx.Tx, intent ModerationIntent) error {
	if intent.RecipientDID == "" || intent.CaseID == uuid.Nil || intent.CaseReference == "" || intent.EventID == uuid.Nil || intent.CreatedAt.IsZero() {
		return fmt.Errorf("invalid moderation notification intent")
	}
	notificationID := uuid.New()
	var pushEnabledSnapshot bool
	err := tx.QueryRow(ctx, `INSERT INTO notification_events(
		id,recipient_did,category,subject_key,moderation_case_reference,moderation_event_id,
		eligibility_scope,recipient_followed_actor,push_enabled_snapshot,state,
		first_activity_at,activity_at,indexed_at,initial_push_evaluated_at
	) VALUES($1,$2,'moderation',$3,$4,$5,'everyone',false,
		COALESCE((SELECT push_enabled FROM notification_preferences WHERE account_did=$2 AND category='moderation'),true),
		'active',$6,$6,$6,$6)
	ON CONFLICT (moderation_event_id) WHERE category='moderation' DO NOTHING
	RETURNING push_enabled_snapshot`, notificationID, intent.RecipientDID, intent.EventID.String(), intent.CaseReference, intent.EventID, intent.CreatedAt.UTC()).Scan(&pushEnabledSnapshot)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert moderation notification event: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO push_deliveries(id,notification_id,account_subscription_id,status,next_attempt_at,deadline_at)
		SELECT gen_random_uuid(),$1,subscription.id,'pending',$2::timestamptz,$2::timestamptz+interval '6 hours'
		FROM push_account_subscriptions subscription
		JOIN push_installations installation ON installation.id=subscription.installation_id
		WHERE subscription.account_did=$3 AND subscription.active AND installation.active
		ON CONFLICT(notification_id,account_subscription_id) DO NOTHING`, notificationID, intent.CreatedAt.UTC(), intent.RecipientDID)
	if err != nil {
		return fmt.Errorf("fan out moderation notification: %w", err)
	}
	return nil
}
