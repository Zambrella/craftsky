package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const deliveryWindow = 6 * time.Hour

type Service struct {
	now                       func() time.Time
	observer                  DecisionObserver
	instagramCoalescingWindow time.Duration
	instagramCountCap         int
}

type ServiceOptions struct {
	InstagramCoalescingWindow time.Duration
	InstagramCountCap         int
}

type DecisionObserver interface{ ObserveNotificationDecision(string, string) }

func NewService(observers ...DecisionObserver) *Service {
	service, _ := NewServiceWithOptions(ServiceOptions{}, observers...)
	return service
}

func NewServiceWithOptions(options ServiceOptions, observers ...DecisionObserver) (*Service, error) {
	var observer DecisionObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	if options.InstagramCoalescingWindow == 0 {
		options.InstagramCoalescingWindow = instagramMatchCoalescingWindow
	}
	if options.InstagramCountCap == 0 {
		options.InstagramCountCap = instagramMatchCountCap
	}
	if options.InstagramCoalescingWindow <= 0 || options.InstagramCoalescingWindow > instagramMatchCoalescingWindow ||
		options.InstagramCountCap <= 0 || options.InstagramCountCap > instagramMatchCountCap {
		return nil, fmt.Errorf("invalid Instagram notification limits")
	}
	return &Service{
		now:                       time.Now,
		observer:                  observer,
		instagramCoalescingWindow: options.InstagramCoalescingWindow,
		instagramCountCap:         options.InstagramCountCap,
	}, nil
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
	decision, err := EvaluateEligibility(EligibilityInput{
		Preference:            preference,
		IsSelf:                activation.RecipientDID == activation.ActorDID,
		RecipientFollowsActor: followsActor,
	})
	if err != nil {
		return err
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
		WITH inserted_event AS (
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
			ON CONFLICT DO NOTHING
			RETURNING id, true AS inserted
		), updated_event AS (
			UPDATE notification_events
			SET source_uri=$6, source_cid=$7, source_rkey=$8,
				subject_uri=NULLIF($14,''), subject_cid=NULLIF($15,''),
				parent_uri=NULLIF($16,''), parent_cid=NULLIF($17,''),
				root_uri=NULLIF($18,''), root_cid=NULLIF($19,''),
				quoted_uri=NULLIF($20,''), quoted_cid=NULLIF($21,''),
				state='active', activity_at=$12, indexed_at=$13,
				retracted_at=NULL, retraction_reason=NULL
			WHERE NOT EXISTS (SELECT 1 FROM inserted_event)
			  AND recipient_did=$2 AND actor_did=$3
			  AND category=$4 AND subject_key=$5
			  AND (
				state='retracted'
				OR source_uri IS DISTINCT FROM $6
				OR source_cid IS DISTINCT FROM $7
			  )
			RETURNING id, false AS inserted
		)
		SELECT id, inserted FROM inserted_event
		UNION ALL
		SELECT id, inserted FROM updated_event
		LIMIT 1
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
