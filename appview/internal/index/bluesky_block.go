package index

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/tap"
)

const blueskyBlockNSID syntax.NSID = "app.bsky.graph.block"

// BlueskyBlock is the sole writer of the public block projection.
type BlueskyBlock struct {
	pool     *pgxpool.Pool
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

var _ Indexer = (*BlueskyBlock)(nil)

func NewBlueskyBlock(pool *pgxpool.Pool, observers ...RelationshipObserver) *BlueskyBlock {
	var observer RelationshipObserver
	if len(observers) > 0 {
		observer = observers[0]
	}
	return &BlueskyBlock{pool: pool, observer: observer}
}

func (b *BlueskyBlock) Handle(ctx context.Context, ev tap.Event) (err error) {
	if ev.Collection != blueskyBlockNSID {
		return nil
	}
	started := time.Now()
	operation := "index_" + ev.Action
	stage := "request"
	errorClass := "none"
	defer func() {
		if b.observer == nil {
			return
		}
		result := "success"
		if err != nil {
			result = "error"
		}
		observeRelationshipOutcome(b.observer, operation, stage, result, errorClass, time.Since(started))
	}()

	switch ev.Action {
	case "create", "update":
		stage = "decode"
		var record bsky.GraphBlock
		if err := json.Unmarshal(ev.Record, &record); err != nil {
			errorClass = "validation"
			return fmt.Errorf("unmarshal block %s: %w", ev.URI, err)
		}
		stage = "validate"
		subject, err := syntax.ParseDID(record.Subject)
		if err != nil {
			errorClass = "validation"
			return fmt.Errorf("parse block subject on %s: %w", ev.URI, err)
		}
		createdAt, err := time.Parse(time.RFC3339, record.CreatedAt)
		if err != nil {
			errorClass = "validation"
			return fmt.Errorf("parse block createdAt on %s: %w", ev.URI, err)
		}
		observeRelationshipOutcome(b.observer, operation, "lag", "success", "none", time.Since(createdAt))
		stage = "store"
		tx, err := b.pool.Begin(ctx)
		if err != nil {
			errorClass = "store"
			return fmt.Errorf("begin block %s: %w", ev.URI, err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `
			INSERT INTO atproto_blocks (
				uri, blocker_did, rkey, cid, subject_did, record, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (uri) DO UPDATE SET
				blocker_did = EXCLUDED.blocker_did,
				rkey = EXCLUDED.rkey,
				cid = EXCLUDED.cid,
				subject_did = EXCLUDED.subject_did,
				record = EXCLUDED.record,
				created_at = EXCLUDED.created_at,
				indexed_at = now()
		`, ev.URI, ev.DID, ev.Rkey, ev.CID, subject, ev.Record, createdAt); err != nil {
			errorClass = "store"
			return fmt.Errorf("upsert block %s: %w", ev.URI, err)
		}
		cancelTag, err := tx.Exec(ctx, `
			UPDATE push_deliveries delivery
			SET status = 'cancelled', lease_owner = NULL, lease_expires_at = NULL, updated_at = now()
			FROM notification_events event
			WHERE delivery.notification_id = event.id
			  AND delivery.status IN ('pending', 'retry', 'leased')
			  AND (
				(event.recipient_did = $1 AND event.actor_did = $2)
				OR (event.recipient_did = $2 AND event.actor_did = $1)
			  )
		`, ev.DID, subject)
		if err != nil {
			errorClass = "store"
			return fmt.Errorf("cancel block deliveries %s: %w", ev.URI, err)
		}
		if err := tx.Commit(ctx); err != nil {
			errorClass = "store"
			return fmt.Errorf("commit block %s: %w", ev.URI, err)
		}
		cancellationResult := "none"
		if cancelTag.RowsAffected() > 0 {
			cancellationResult = "some"
		}
		observeRelationshipOutcome(b.observer, "push_cancellation", "delivery", cancellationResult, "none", 0)
		return nil
	case "delete":
		stage = "store"
		if _, err := b.pool.Exec(ctx, `DELETE FROM atproto_blocks WHERE uri = $1`, ev.URI); err != nil {
			errorClass = "store"
			return fmt.Errorf("delete block %s: %w", ev.URI, err)
		}
		return nil
	default:
		stage = "validate"
		errorClass = "validation"
		return fmt.Errorf("unknown block action %q on %s", ev.Action, ev.URI)
	}
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
