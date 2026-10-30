package notifications

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

// RestrictStoredPublication withdraws existing reasons and cancels undispatched
// work in the caller's source/acceptance transaction. It never creates events,
// advances activity/newness, re-enables a delivery, or recalls completed sends.
func RestrictStoredPublication(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, allowed []string, now time.Time) error {
	_, err := tx.Exec(ctx, `WITH retracted AS (
 UPDATE notification_events SET state='retracted',retracted_at=$3,retraction_reason='publicationReasonRemoved'
 WHERE source_uri=$1 AND state='active' AND category IN ('reply','mention','quote')
 AND NOT (recipient_did=ANY(COALESCE($2::text[],'{}'::text[]))) RETURNING id
 ) UPDATE push_deliveries delivery SET status='cancelled',updated_at=$3,lease_owner=NULL,lease_expires_at=NULL
 FROM retracted WHERE delivery.notification_id=retracted.id AND delivery.status IN ('pending','retry','leased')`, uri, allowed, now)
	return err
}
