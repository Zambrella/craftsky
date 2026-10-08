package imagesafety

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RescanStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewRescanStore(pool *pgxpool.Pool, now func() time.Time) *RescanStore {
	if now == nil {
		now = time.Now
	}
	return &RescanStore{pool: pool, now: now}
}

// Target starts a new scan generation only when expectedVersion is still the
// current clear generation. Callers own authorization and audit policy.
func (store *RescanStore) Target(ctx context.Context, resultID uuid.UUID, expectedVersion int64) (bool, error) {
	if store == nil || store.pool == nil {
		return false, fmt.Errorf("target rescan: store is unavailable")
	}
	if resultID == uuid.Nil || expectedVersion <= 0 {
		return false, fmt.Errorf("target rescan: invalid result or version")
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("target rescan: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var nextVersion int64
	err = tx.QueryRow(ctx, `
		UPDATE image_scan_results
		SET state='pending',completed_at=NULL,current_version=current_version+1,updated_at=$3
		WHERE id=$1 AND current_version=$2 AND state='clear'
		RETURNING current_version
	`, resultID, expectedVersion, store.now()).Scan(&nextVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("target rescan: reset result: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO image_scan_jobs(id,scan_result_id,state,scan_version,next_attempt_at)
		VALUES($1,$2,'queued',$3,$4)
		ON CONFLICT(scan_result_id) DO UPDATE SET
			state='queued',scan_version=EXCLUDED.scan_version,attempts=0,
			next_attempt_at=EXCLUDED.next_attempt_at,
			lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			safe_error_category=NULL,updated_at=EXCLUDED.next_attempt_at
	`, uuid.New(), resultID, nextVersion, store.now()); err != nil {
		return false, fmt.Errorf("target rescan: queue generation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE image_subject_states subject
		SET visibility_state='blocked',updated_at=$2
		FROM image_subject_requirements requirement
		WHERE requirement.scan_result_id=$1
		  AND subject.subject_uri=requirement.subject_uri
		  AND subject.source_cid=requirement.source_cid
	`, resultID, store.now()); err != nil {
		return false, fmt.Errorf("target rescan: block subjects: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bluesky_profiles profile
		SET avatar_cid=NULL,avatar_mime=NULL
		FROM profile_image_candidates candidate
		WHERE candidate.scan_result_id=$1
		  AND candidate.slot='avatar'
		  AND candidate.profile_did=profile.did
		  AND candidate.source_cid=profile.record_cid
	`, resultID); err != nil {
		return false, fmt.Errorf("target rescan: hide profile avatars: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bluesky_profiles profile
		SET banner_cid=NULL,banner_mime=NULL
		FROM profile_image_candidates candidate
		WHERE candidate.scan_result_id=$1
		  AND candidate.slot='banner'
		  AND candidate.profile_did=profile.did
		  AND candidate.source_cid=profile.record_cid
	`, resultID); err != nil {
		return false, fmt.Errorf("target rescan: hide profile banners: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tap_projection_jobs projection
		SET state='blocked',dependency_kind='image_subject_uri',dependency_key=projection.source_uri,
			attempts=0,next_attempt_at=$2,
			lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
			last_reason_code='image_scan_pending',completed_at=NULL,updated_at=$2
		WHERE projection.state <> 'permanent_denied'
		  AND (
		    EXISTS (
			SELECT 1 FROM image_subject_requirements requirement
			WHERE requirement.scan_result_id=$1
			  AND requirement.subject_uri=projection.source_uri
		    )
		    OR EXISTS (
			SELECT 1 FROM profile_image_candidates candidate
			WHERE candidate.scan_result_id=$1
			  AND projection.source_uri='at://' || candidate.profile_did || '/app.bsky.actor.profile/self'
		    )
		  )
	`, resultID, store.now()); err != nil {
		return false, fmt.Errorf("target rescan: block projections: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("target rescan: commit: %w", err)
	}
	return true, nil
}
