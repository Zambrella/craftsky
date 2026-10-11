package ingestion

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

const PostNotificationsProjection = "post_notifications"

// Source updates/deletes invalidate old notification leases and dependencies.
// Copy the serving job's authority decision rather than bypassing lifecycle or
// uncertain-source ordering gates for auxiliary work.
func resetPostNotificationJob(ctx context.Context, tx pgx.Tx, uri syntax.ATURI, collection syntax.NSID) error {
	if collection != "social.craftsky.feed.post" {
		return nil
	}
	_, err := tx.Exec(ctx, `
		UPDATE tap_projection_jobs auxiliary
		SET source_event_id=serving.source_event_id,state=serving.state,
		    dependency_kind=serving.dependency_kind,dependency_key=serving.dependency_key,
		    attempts=0,next_attempt_at=serving.next_attempt_at,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_reason_code=serving.last_reason_code,completed_at=serving.completed_at,
		    updated_at=serving.updated_at
		FROM tap_projection_jobs serving
		WHERE auxiliary.source_uri=$1 AND auxiliary.projection_kind=$2
		  AND serving.source_uri=auxiliary.source_uri AND serving.projection_kind=$3
	`, uri, PostNotificationsProjection, projectionKind(collection))
	if err != nil {
		return fmt.Errorf("reset post notification job: %w", err)
	}
	return nil
}

// EnqueuePostNotifications commits notification work with the serving post.
// Reprojecting an unchanged source must not reset a completed/blocked job.
func EnqueuePostNotifications(ctx context.Context, tx pgx.Tx, source SourceRecord) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO tap_projection_jobs(source_uri,projection_kind,source_event_id,state)
		VALUES($1,$2,$3,'pending')
		ON CONFLICT(source_uri,projection_kind) DO NOTHING
	`, source.URI, PostNotificationsProjection, source.SourceEventID)
	if err != nil {
		return fmt.Errorf("enqueue post notifications: %w", err)
	}
	return nil
}
