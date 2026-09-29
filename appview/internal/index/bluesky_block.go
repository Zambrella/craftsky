package index

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

const blueskyBlockNSID syntax.NSID = "app.bsky.graph.block"

// BlueskyBlock projects normalized per-URI facts and logical block aggregates.
type BlueskyBlock struct {
	observer RelationshipObserver
}

// RelationshipObserver is the identifier-free operational boundary shared by
// relationship index and backfill paths.
type RelationshipObserver interface {
	ObserveRelationship(operation, result string, duration time.Duration)
}

type relationshipOutcomeObserver interface {
	ObserveRelationshipOutcome(operation, stage, result, errorClass string, duration time.Duration)
}

func NewBlueskyBlock(_ *pgxpool.Pool, observers ...RelationshipObserver) *BlueskyBlock {
	var observer RelationshipObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &BlueskyBlock{observer: observer}
}

func (b *BlueskyBlock) cancelPendingDeliveries(ctx context.Context, db transactionalDatabase, actor, subject syntax.DID) error {
	result, err := db.Exec(ctx, `
		UPDATE push_deliveries delivery
		SET status = 'cancelled', lease_owner = NULL, lease_expires_at = NULL, updated_at = now()
		FROM notification_events event
		WHERE delivery.notification_id = event.id
		  AND delivery.status IN ('pending', 'retry', 'leased')
		  AND (
			(event.recipient_did = $1 AND event.actor_did = $2)
			OR (event.recipient_did = $2 AND event.actor_did = $1)
		  )
	`, actor, subject)
	if err != nil {
		return err
	}
	cancellationResult := "none"
	if result.RowsAffected() > 0 {
		cancellationResult = "some"
	}
	observeRelationshipOutcome(b.observer, "push_cancellation", "delivery", cancellationResult, "none", 0)
	return nil
}

func observeRelationshipOutcome(observer RelationshipObserver, operation, stage, result, errorClass string, duration time.Duration) {
	if observer == nil {
		return
	}
	if detailed, ok := observer.(relationshipOutcomeObserver); ok {
		detailed.ObserveRelationshipOutcome(operation, stage, result, errorClass, duration)
		return
	}
	observer.ObserveRelationship(operation, result, duration)
}
