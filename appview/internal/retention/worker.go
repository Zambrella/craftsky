package retention

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/safetyincident"
)

const (
	defaultWorkerID     = "appview-retention"
	defaultLease        = 2 * time.Minute
	defaultMaxAttempts  = 5
	defaultRetryBackoff = time.Minute
)

type Worker struct {
	pool    *pgxpool.Pool
	objects safetyincident.EvidenceStore
	now     func() time.Time
}

type claimedJob struct {
	id             uuid.UUID
	evidenceID     uuid.UUID
	attempt        int
	maxAttempts    int
	startedAt      time.Time
	leaseExpiresAt time.Time
}

func NewWorker(pool *pgxpool.Pool, objects safetyincident.EvidenceStore, now func() time.Time) *Worker {
	if now == nil {
		now = time.Now
	}
	return &Worker{pool: pool, objects: objects, now: now}
}

func (worker *Worker) RunOnce(ctx context.Context, limit int) (int, error) {
	if worker == nil || worker.pool == nil || worker.objects == nil || limit <= 0 {
		return 0, errors.New("retention worker unavailable")
	}
	now := worker.now().UTC().Truncate(time.Microsecond)
	if err := worker.recoverExpiredLeases(ctx, now); err != nil {
		return 0, err
	}
	if err := worker.enqueueDue(ctx, now); err != nil {
		return 0, err
	}
	deletedCount := 0
	for range limit {
		job, found, err := worker.claim(ctx, now)
		if err != nil {
			return deletedCount, err
		}
		if !found {
			break
		}
		deleted, deleteErr := safetyincident.DeleteEvidenceIfUnheld(
			ctx, worker.pool, worker.objects, job.evidenceID,
			defaultWorkerID, "retentionExpired", now,
		)
		if deleteErr != nil {
			if err := worker.fail(ctx, job, now, retentionErrorCategory(deleteErr)); err != nil {
				return deletedCount, err
			}
			continue
		}
		if err := worker.complete(ctx, job, now); err != nil {
			return deletedCount, err
		}
		if deleted {
			deletedCount++
		}
	}
	return deletedCount, nil
}

func (worker *Worker) enqueueDue(ctx context.Context, now time.Time) error {
	_, err := worker.pool.Exec(ctx, `INSERT INTO safety_retention_jobs(
		id,data_class,item_reference,state,attempt_count,max_attempts,created_at,updated_at
	) SELECT gen_random_uuid(),'restrictedEvidence',evidence.id,'queued',0,$2,$1,$1
	FROM safety_evidence evidence
	WHERE evidence.deleted_at IS NULL AND evidence.retention_expires_at <= $1
	  AND NOT EXISTS (
		SELECT 1 FROM safety_legal_hold_evidence scope
		JOIN safety_legal_holds hold ON hold.id=scope.hold_id
		WHERE scope.evidence_id=evidence.id AND hold.released_at IS NULL AND hold.expires_at>$1
	  )
	ON CONFLICT(data_class,item_reference) DO UPDATE SET
		state='retry',next_attempt_at=$1,lease_owner=NULL,lease_expires_at=NULL,
		safe_error_category='dependencyUnavailable',updated_at=$1
	WHERE safety_retention_jobs.state='completed'
	  AND safety_retention_jobs.attempt_count < safety_retention_jobs.max_attempts`, now, defaultMaxAttempts)
	if err != nil {
		return fmt.Errorf("enqueue restricted evidence retention: %w", err)
	}
	return nil
}

func (worker *Worker) recoverExpiredLeases(ctx context.Context, now time.Time) error {
	return pgx.BeginFunc(ctx, worker.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO safety_retention_attempts(
			id,job_id,attempt_number,outcome,safe_error_category,started_at,completed_at
		) SELECT gen_random_uuid(),id,attempt_count,'failed','dependencyUnavailable',updated_at,$1
		FROM safety_retention_jobs
		WHERE state='running' AND lease_expires_at <= $1
		ON CONFLICT(job_id,attempt_number) DO NOTHING`, now); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE safety_retention_jobs SET
			state=CASE WHEN attempt_count >= max_attempts THEN 'deadLetter' ELSE 'retry' END,
			next_attempt_at=CASE WHEN attempt_count >= max_attempts THEN NULL ELSE $1 END,
			lease_owner=NULL,lease_expires_at=NULL,safe_error_category='dependencyUnavailable',updated_at=$1
		WHERE state='running' AND lease_expires_at <= $1`, now)
		return err
	})
}

func (worker *Worker) claim(ctx context.Context, now time.Time) (claimedJob, bool, error) {
	var job claimedJob
	job.startedAt = now
	job.leaseExpiresAt = now.Add(defaultLease)
	err := worker.pool.QueryRow(ctx, `WITH candidate AS (
		SELECT id FROM safety_retention_jobs
		WHERE (state='queued' OR (state='retry' AND next_attempt_at <= $1))
		  AND attempt_count < max_attempts
		ORDER BY COALESCE(next_attempt_at,created_at),id
		FOR UPDATE SKIP LOCKED LIMIT 1
	) UPDATE safety_retention_jobs jobs SET
		state='running',attempt_count=jobs.attempt_count+1,next_attempt_at=NULL,
		lease_owner=$2,lease_expires_at=$3,safe_error_category=NULL,updated_at=$1
	FROM candidate WHERE jobs.id=candidate.id
	RETURNING jobs.id,jobs.item_reference,jobs.attempt_count,jobs.max_attempts`,
		now, defaultWorkerID, job.leaseExpiresAt,
	).Scan(&job.id, &job.evidenceID, &job.attempt, &job.maxAttempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return claimedJob{}, false, nil
	}
	if err != nil {
		return claimedJob{}, false, fmt.Errorf("claim restricted evidence retention: %w", err)
	}
	return job, true, nil
}

func (worker *Worker) complete(ctx context.Context, job claimedJob, now time.Time) error {
	return pgx.BeginFunc(ctx, worker.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO safety_retention_attempts(
			id,job_id,attempt_number,outcome,started_at,completed_at
		) VALUES($1,$2,$3,'succeeded',$4,$5)`, uuid.New(), job.id, job.attempt, job.startedAt, now); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE safety_retention_jobs SET
			state='completed',next_attempt_at=NULL,lease_owner=NULL,lease_expires_at=NULL,
			safe_error_category=NULL,updated_at=$2
		WHERE id=$1 AND state='running' AND lease_owner=$3 AND lease_expires_at=$4`,
			job.id, now, defaultWorkerID, job.leaseExpiresAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("retention lease lost before completion")
		}
		return nil
	})
}

func (worker *Worker) fail(ctx context.Context, job claimedJob, now time.Time, category string) error {
	next := now.Add(defaultRetryBackoff << (job.attempt - 1))
	state := "retry"
	if job.attempt >= job.maxAttempts {
		state = "deadLetter"
		next = time.Time{}
	}
	return pgx.BeginFunc(ctx, worker.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO safety_retention_attempts(
			id,job_id,attempt_number,outcome,safe_error_category,started_at,completed_at
		) VALUES($1,$2,$3,'failed',$4,$5,$6)`, uuid.New(), job.id, job.attempt, category, job.startedAt, now); err != nil {
			return err
		}
		var nextAt any
		if !next.IsZero() {
			nextAt = next
		}
		tag, err := tx.Exec(ctx, `UPDATE safety_retention_jobs SET
			state=$2,next_attempt_at=$3,lease_owner=NULL,lease_expires_at=NULL,
			safe_error_category=$4,updated_at=$5
		WHERE id=$1 AND state='running' AND lease_owner=$6 AND lease_expires_at=$7`,
			job.id, state, nextAt, category, now, defaultWorkerID, job.leaseExpiresAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return errors.New("retention lease lost before failure recording")
		}
		return nil
	})
}

func retentionErrorCategory(err error) string {
	if errors.Is(err, safetyincident.ErrEvidenceStoreBoundary) {
		return "objectDelete"
	}
	return "unknown"
}
