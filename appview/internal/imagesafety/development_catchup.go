package imagesafety

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatchUpDevelopment replays unresolved image dependants through normal indexing
// under the dev approval identity. Existing scan verdicts are never relabelled.
// It must only be called against the configured local development database.
func CatchUpDevelopment(ctx context.Context, pool *pgxpool.Pool, config Config) (int64, error) {
	if config.Mode != ScannerModeDevelopmentAutoApprove || !config.Ready() || config.Environment != EnvironmentDevelopment {
		return 0, fmt.Errorf("image catch-up requires development auto-approval")
	}
	var count int64
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
   UPDATE tap_projection_jobs projection
   SET state='pending',dependency_kind=NULL,dependency_key=NULL,
       attempts=0,next_attempt_at=now(),completed_at=NULL,last_reason_code=NULL,updated_at=now()
   WHERE (
      (projection.state='blocked' AND projection.dependency_kind='image_subject_uri')
      OR (projection.state='complete' AND projection.projection_kind='app_bsky_actor_profile')
   ) AND (
      EXISTS (
       SELECT 1 FROM image_subject_requirements requirement
       JOIN image_scan_results result ON result.id=requirement.scan_result_id
       WHERE requirement.subject_uri=projection.source_uri
         AND result.state IN ('pending','unavailable','error')
      ) OR EXISTS (
       SELECT 1 FROM profile_image_candidates candidate
       JOIN image_scan_results result ON result.id=candidate.scan_result_id
       WHERE projection.source_uri='at://' || candidate.profile_did || '/app.bsky.actor.profile/self'
         AND result.state IN ('pending','unavailable','error')
      )
   )`)
		if err != nil {
			return fmt.Errorf("requeue development image dependants: %w", err)
		}
		count = tag.RowsAffected()
		// No worker runs in auto-approve mode. Remove obsolete local queue entries,
		// retaining all results (and matches) under their original scan identity.
		if _, err := tx.Exec(ctx, `DELETE FROM image_scan_jobs job USING image_scan_results result
    WHERE job.scan_result_id=result.id AND result.state IN ('pending','unavailable','error')`); err != nil {
			return fmt.Errorf("remove obsolete development image jobs: %w", err)
		}
		return nil
	})
	return count, err
}
