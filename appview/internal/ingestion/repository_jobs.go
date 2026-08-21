package ingestion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RepositoryJobKind string

const (
	RepositoryJobTapAddRepo   RepositoryJobKind = "tap_add_repo"
	RepositoryJobPDSReconcile RepositoryJobKind = "pds_reconcile"
)

type RepositoryJob struct {
	ID                    uuid.UUID
	DID                   syntax.DID
	Kind                  RepositoryJobKind
	State                 string
	Attempts              int
	NextAttemptAt         time.Time
	LeaseOwner            string
	LeaseToken            uuid.UUID
	LeaseExpiresAt        time.Time
	LastReasonCode        string
	AuthoritativeRevision string
	LastSuccessfulAt      *time.Time
}

type RepositoryClaimRequest struct {
	Worker        string
	LeaseToken    uuid.UUID
	LeaseDuration time.Duration
	Limit         int
}

type RepositoryClaim struct {
	RepositoryJob
}

type RepositoryJobHandler func(context.Context, RepositoryClaim) (authoritativeRevision string, err error)

func (store *Store) EnqueueRepositoryJob(ctx context.Context, did syntax.DID, kind RepositoryJobKind) error {
	if did == "" || !validRepositoryJobKind(kind) {
		return errors.New("invalid Tap repository job")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	return pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		return enqueueRepositoryJob(ctx, tx, did, string(kind), now)
	})
}

func validRepositoryJobKind(kind RepositoryJobKind) bool {
	return kind == RepositoryJobTapAddRepo || kind == RepositoryJobPDSReconcile
}

func (store *Store) ClaimRepositoryJobs(ctx context.Context, request RepositoryClaimRequest) ([]RepositoryClaim, error) {
	if strings.TrimSpace(request.Worker) == "" || len(request.Worker) > 256 ||
		request.LeaseToken == uuid.Nil || request.LeaseDuration <= 0 || request.Limit <= 0 {
		return nil, errors.New("invalid Tap repository claim")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	expires := now.Add(request.LeaseDuration).Truncate(time.Microsecond)
	rows, err := store.pool.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM tap_repository_jobs
			WHERE next_attempt_at <= $1
			  AND (state='pending' OR (state='processing' AND lease_expires_at <= $1))
			ORDER BY next_attempt_at,id
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE tap_repository_jobs AS job
		SET state='processing',attempts=job.attempts+1,
		    lease_owner=$3,lease_token=$4,lease_expires_at=$5,updated_at=$1
		FROM candidates
		WHERE job.id=candidates.id
		RETURNING job.id,job.did,job.job_kind,job.state,job.attempts,
		          job.next_attempt_at,job.lease_owner,job.lease_token,
		          job.lease_expires_at,job.last_reason_code,
		          job.authoritative_revision,job.last_successful_at
	`, now, request.Limit, strings.TrimSpace(request.Worker), request.LeaseToken, expires)
	if err != nil {
		return nil, fmt.Errorf("claim Tap repository jobs: %w", err)
	}
	defer rows.Close()
	claims := make([]RepositoryClaim, 0, request.Limit)
	for rows.Next() {
		claim, err := scanRepositoryClaim(rows)
		if err != nil {
			return nil, err
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Tap repository claims: %w", err)
	}
	return claims, nil
}

func (store *Store) RunRepositoryJob(ctx context.Context, claim RepositoryClaim, handler RepositoryJobHandler) error {
	return store.runRepositoryJob(ctx, claim, handler, time.Second)
}

func (store *Store) runRepositoryJob(ctx context.Context, claim RepositoryClaim, handler RepositoryJobHandler, retryDelay time.Duration) error {
	if claim.ID == uuid.Nil || claim.DID == "" || !validRepositoryJobKind(claim.Kind) ||
		claim.LeaseToken == uuid.Nil || handler == nil {
		return ErrProjectionLeaseLost
	}
	if retryDelay <= 0 {
		return errors.New("tap repository retry delay must be positive")
	}
	revision, handlerErr := handler(ctx, claim)
	now := store.now().UTC().Truncate(time.Microsecond)
	if handlerErr != nil {
		rescheduleErr := store.rescheduleRepositoryJob(ctx, claim, now.Add(retryDelay), "remote_unavailable")
		return errors.Join(handlerErr, rescheduleErr)
	}
	result, err := store.pool.Exec(ctx, `
		UPDATE tap_repository_jobs
		SET state='complete',lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_reason_code=NULL,authoritative_revision=$3,
		    last_successful_at=$4,updated_at=$4
		WHERE id=$1 AND state='processing' AND lease_token=$2 AND lease_expires_at>$4
	`, claim.ID, claim.LeaseToken, nullableString(strings.TrimSpace(revision)), now)
	if err != nil {
		return fmt.Errorf("complete Tap repository job: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrProjectionLeaseLost
	}
	return nil
}

func (store *Store) rescheduleRepositoryJob(ctx context.Context, claim RepositoryClaim, next time.Time, reason string) error {
	now := store.now().UTC().Truncate(time.Microsecond)
	result, err := store.pool.Exec(ctx, `
		UPDATE tap_repository_jobs
		SET state='pending',next_attempt_at=$3,
		    lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
		    last_reason_code=$4,updated_at=$5
		WHERE id=$1 AND state='processing' AND lease_token=$2 AND lease_expires_at>$5
	`, claim.ID, claim.LeaseToken, next.UTC().Truncate(time.Microsecond), reason, now)
	if err != nil {
		return fmt.Errorf("reschedule Tap repository job: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrProjectionLeaseLost
	}
	return nil
}

func (store *Store) RepositoryJob(ctx context.Context, did syntax.DID, kind RepositoryJobKind) (RepositoryJob, error) {
	if did == "" || !validRepositoryJobKind(kind) {
		return RepositoryJob{}, errors.New("invalid Tap repository job key")
	}
	return scanRepositoryJob(store.pool.QueryRow(ctx, `
		SELECT id,did,job_kind,state,attempts,next_attempt_at,
		       lease_owner,lease_token,lease_expires_at,last_reason_code,
		       authoritative_revision,last_successful_at
		FROM tap_repository_jobs WHERE did=$1 AND job_kind=$2
	`, did, kind))
}

func scanRepositoryClaim(row rowScanner) (RepositoryClaim, error) {
	job, err := scanRepositoryJob(row)
	return RepositoryClaim{RepositoryJob: job}, err
}

func scanRepositoryJob(row rowScanner) (RepositoryJob, error) {
	var job RepositoryJob
	var leaseOwner, lastReason, revision *string
	var leaseToken *uuid.UUID
	var leaseExpires *time.Time
	if err := row.Scan(&job.ID, &job.DID, &job.Kind, &job.State, &job.Attempts,
		&job.NextAttemptAt, &leaseOwner, &leaseToken, &leaseExpires,
		&lastReason, &revision, &job.LastSuccessfulAt); err != nil {
		return RepositoryJob{}, fmt.Errorf("scan Tap repository job: %w", err)
	}
	if leaseOwner != nil {
		job.LeaseOwner = *leaseOwner
	}
	if leaseToken != nil {
		job.LeaseToken = *leaseToken
	}
	if leaseExpires != nil {
		job.LeaseExpiresAt = *leaseExpires
	}
	if lastReason != nil {
		job.LastReasonCode = *lastReason
	}
	if revision != nil {
		job.AuthoritativeRevision = *revision
	}
	return job, nil
}
