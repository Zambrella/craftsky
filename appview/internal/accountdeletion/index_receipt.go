package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/tap"
)

type ReceiptObserver struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewReceiptObserver(pool *pgxpool.Pool, now func() time.Time) *ReceiptObserver {
	if now == nil {
		now = time.Now
	}
	return &ReceiptObserver{pool: pool, now: now}
}

func (observer *ReceiptObserver) ObserveHandled(ctx context.Context, event tap.Event) error {
	if event.Action != "delete" || !registeredCraftskyCollection(event.Collection) {
		return nil
	}
	if observer == nil || observer.pool == nil || event.URI == "" || event.DID == "" ||
		event.URI.Authority().String() != event.DID.String() || event.ID == 0 || event.ID > math.MaxInt64 || event.Rev == "" {
		return errors.New("invalid account deletion receipt event")
	}
	if _, err := observer.pool.Exec(ctx, `
		INSERT INTO account_deletion_index_receipts(
			job_id,uri,collection,tap_event_id,repo_revision,handled_at
		)
		SELECT operation.id,$1,$2,$4,$5,$6
		FROM account_deletion_operations operation
		WHERE operation.owner_did=$3
		  AND operation.state IN ('active','retrying','needsAttention')
		ON CONFLICT(job_id,uri) DO UPDATE SET
			tap_event_id=EXCLUDED.tap_event_id,
			repo_revision=EXCLUDED.repo_revision,
			handled_at=EXCLUDED.handled_at
		WHERE EXCLUDED.tap_event_id>account_deletion_index_receipts.tap_event_id
	`, event.URI, event.Collection, event.DID, int64(event.ID), event.Rev, observer.now().UTC()); err != nil {
		return fmt.Errorf("persist account deletion index receipt: %w", err)
	}
	return nil
}
