package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/relationships"
)

const deliveryWindow = 6 * time.Hour

type Service struct {
	now                  func() time.Time
	observer             DecisionObserver
	relationshipObserver relationshipOutcomeObserver
}

type DecisionObserver interface{ ObserveNotificationDecision(string, string) }
type relationshipOutcomeObserver interface {
	ObserveRelationshipOutcome(operation, stage, result, errorClass string, duration time.Duration)
}

func NewService(observers ...DecisionObserver) *Service {
	var observer DecisionObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	service := &Service{now: time.Now, observer: observer}
	if detailed, ok := observer.(relationshipOutcomeObserver); ok {
		service.relationshipObserver = detailed
	}
	return service
}

func (s *Service) Activate(ctx context.Context, tx pgx.Tx, activation Activation) error {
	preference := defaultPreference
	err := tx.QueryRow(ctx, `
		SELECT scope, push_enabled
		FROM notification_preferences
		WHERE account_did = $1 AND category = $2
	`, activation.RecipientDID, activation.Category).Scan(&preference.Scope, &preference.PushEnabled)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("read effective preference: %w", err)
	}

	followsActor := false
	if preference.Scope == PeopleIFollow {
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM atproto_follows
				WHERE did = $1 AND subject_did = $2
			)
		`, activation.RecipientDID, activation.ActorDID).Scan(&followsActor); err != nil {
			return fmt.Errorf("read event-time follow state: %w", err)
		}
	}
	var relationship relationships.State
	if err := tx.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM actor_mutes WHERE owner_did = $1 AND subject_did = $2),
			EXISTS (SELECT 1 FROM atproto_blocks WHERE blocker_did = $1 AND subject_did = $2),
			EXISTS (SELECT 1 FROM atproto_blocks WHERE blocker_did = $2 AND subject_did = $1)
	`, activation.RecipientDID, activation.ActorDID).Scan(
		&relationship.Muted,
		&relationship.Blocking,
		&relationship.BlockedBy,
	); err != nil {
		return fmt.Errorf("read event-time relationship state: %w", err)
	}
	decision, err := EvaluateEligibility(EligibilityInput{
		Preference:            preference,
		IsSelf:                activation.RecipientDID == activation.ActorDID,
		RecipientFollowsActor: followsActor,
		Relationship:          relationship,
	})
	if err != nil {
		return err
	}
	if relationship.Muted || relationship.HasBlock() {
		if s.observer != nil {
			s.observer.ObserveNotificationDecision(string(activation.Category), "suppressed")
		}
		if s.relationshipObserver != nil {
			s.relationshipObserver.ObserveRelationshipOutcome(
				"notification_suppression", "policy", "suppressed", "none", 0,
			)
		}
	}
	if !decision.Accepted {
		if s.observer != nil {
			s.observer.ObserveNotificationDecision(string(activation.Category), "suppressed")
		}
		return nil
	}

	now := s.now().UTC()
	notificationID := uuid.New()
	var insertedID uuid.UUID
	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO notification_events (
			id, recipient_did, actor_did, category, subject_key,
			source_uri, source_cid, source_rkey, subject_uri, subject_cid,
			parent_uri,parent_cid,root_uri,root_cid,quoted_uri,quoted_cid,
			eligibility_scope, recipient_followed_actor, push_enabled_snapshot,
			state, first_activity_at, activity_at, indexed_at, initial_push_evaluated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, NULLIF($14, ''), NULLIF($15, ''),
			NULLIF($16,''),NULLIF($17,''),NULLIF($18,''),NULLIF($19,''),NULLIF($20,''),NULLIF($21,''),
			$9, $10, $11,
			'active', $12, $12, $13, $13
		)
		ON CONFLICT (recipient_did, actor_did, category, subject_key) DO UPDATE SET
			source_uri = EXCLUDED.source_uri,
			source_cid = EXCLUDED.source_cid,
			source_rkey = EXCLUDED.source_rkey,
			subject_uri=EXCLUDED.subject_uri,subject_cid=EXCLUDED.subject_cid,
			parent_uri=EXCLUDED.parent_uri,parent_cid=EXCLUDED.parent_cid,
			root_uri=EXCLUDED.root_uri,root_cid=EXCLUDED.root_cid,
			quoted_uri=EXCLUDED.quoted_uri,quoted_cid=EXCLUDED.quoted_cid,
			state = 'active',
			activity_at = EXCLUDED.activity_at,
			indexed_at = EXCLUDED.indexed_at,
			retracted_at = NULL,
			retraction_reason = NULL
		WHERE notification_events.state = 'retracted'
		   OR notification_events.source_uri IS DISTINCT FROM EXCLUDED.source_uri
		   OR notification_events.source_cid IS DISTINCT FROM EXCLUDED.source_cid
		RETURNING id, (xmax = 0) AS inserted
	`, notificationID, activation.RecipientDID, activation.ActorDID, activation.Category, activation.SubjectKey,
		activation.SourceURI, activation.SourceCID, activation.SourceRkey,
		preference.Scope, followsActor, decision.PushEnabled, activation.ActivityAt, now,
		activation.SubjectURI, activation.SubjectCID, activation.ParentURI, activation.ParentCID, activation.RootURI, activation.RootCID, activation.QuotedURI, activation.QuotedCID).Scan(&insertedID, &inserted)
	if err == pgx.ErrNoRows {
		if s.observer != nil {
			s.observer.ObserveNotificationDecision(string(activation.Category), "duplicate")
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	if s.observer != nil {
		s.observer.ObserveNotificationDecision(string(activation.Category), "created")
	}
	if !inserted || !decision.PushEnabled {
		return nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO push_deliveries (
			id, notification_id, account_subscription_id,
			status, next_attempt_at, deadline_at
		)
		SELECT gen_random_uuid(), $1, subscription.id,
		       'pending', $2, $3
		FROM push_account_subscriptions subscription
		JOIN push_installations installation ON installation.id = subscription.installation_id
		WHERE subscription.account_did = $4
		  AND subscription.active
		  AND installation.active
		ON CONFLICT (notification_id, account_subscription_id) DO NOTHING
	`, insertedID, now, activation.ActivityAt.Add(deliveryWindow), activation.RecipientDID); err != nil {
		return fmt.Errorf("fan out notification: %w", err)
	}
	return nil
}

func (s *Service) Retract(ctx context.Context, tx pgx.Tx, retraction Retraction) error {
	now := s.now().UTC()
	if _, err := tx.Exec(ctx, `
		WITH retracted AS (
			UPDATE notification_events
			SET state = 'retracted', retracted_at = $2, retraction_reason = $3
			WHERE state = 'active' AND (
				source_uri = $1 OR subject_uri = $1 OR parent_uri = $1 OR root_uri = $1 OR quoted_uri = $1
			)
			RETURNING id
		)
		UPDATE push_deliveries delivery
		SET status = 'cancelled', updated_at = $2,
		    lease_owner = NULL, lease_expires_at = NULL
		FROM retracted
		WHERE delivery.notification_id = retracted.id
		  AND delivery.status IN ('pending', 'retry', 'leased')
	`, retraction.SourceURI, now, retraction.Reason); err != nil {
		return fmt.Errorf("retract notification: %w", err)
	}
	return nil
}
