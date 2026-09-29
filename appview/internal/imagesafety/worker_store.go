package imagesafety

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrLeaseLost = errors.New("image scan lease lost")

type BlobSource struct {
	DID          syntax.DID
	URI          syntax.ATURI
	SourceCID    syntax.CID
	BlobCID      syntax.CID
	DeclaredMIME string
	DeclaredSize int64
}

type Claim struct {
	JobID        uuid.UUID
	ResultID     uuid.UUID
	LeaseToken   uuid.UUID
	ScanVersion  int64
	Attempt      int
	LeaseExpires time.Time
	Source       BlobSource
}

type QueueHealth struct {
	Queued       int
	Leased       int
	DeadLettered int
	OldestDueAge time.Duration
	MaxAttempts  int
}

type WorkerStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewWorkerStore(pool *pgxpool.Pool, now func() time.Time) *WorkerStore {
	if now == nil {
		now = time.Now
	}
	return &WorkerStore{pool: pool, now: now}
}

func (store *WorkerStore) Claim(ctx context.Context, workerID string, leaseDuration time.Duration) (Claim, bool, error) {
	if store == nil || store.pool == nil || workerID == "" || leaseDuration <= 0 {
		return Claim{}, false, errors.New("invalid image scan claim")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	leaseToken := uuid.New()
	var claim Claim
	found := false
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
		WITH candidate AS (
			SELECT job.id
			FROM image_scan_jobs job
			JOIN image_scan_results result ON result.id=job.scan_result_id
			WHERE (
				(job.state='queued' AND job.next_attempt_at <= $1)
				OR (job.state='leased' AND job.lease_expires_at <= $1)
			)
			AND EXISTS (SELECT 1 FROM image_blob_sources WHERE blob_cid=result.blob_cid)
			ORDER BY job.next_attempt_at,job.id
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		), leased AS (
			UPDATE image_scan_jobs job
			SET state='leased',attempts=job.attempts+1,lease_owner=$2,
			    lease_token=$3,lease_expires_at=$4,safe_error_category=NULL,updated_at=$1
			FROM candidate
			WHERE job.id=candidate.id
			RETURNING job.id,job.scan_result_id,job.scan_version,job.attempts,job.lease_expires_at
		)
		SELECT leased.id,leased.scan_result_id,leased.scan_version,leased.attempts,leased.lease_expires_at,
		       source.source_did,source.source_uri,source.source_cid,
		       result.blob_cid,source.declared_mime,source.declared_size
		FROM leased
		JOIN image_scan_results result ON result.id=leased.scan_result_id
		JOIN LATERAL (
			SELECT source_did,source_uri,source_cid,declared_mime,declared_size
			FROM image_blob_sources
			WHERE blob_cid=result.blob_cid
			ORDER BY observed_at DESC,source_uri
			LIMIT 1
		) source ON true
		`, now, workerID, leaseToken, now.Add(leaseDuration)).Scan(
			&claim.JobID, &claim.ResultID, &claim.ScanVersion, &claim.Attempt, &claim.LeaseExpires,
			&claim.Source.DID, &claim.Source.URI, &claim.Source.SourceCID,
			&claim.Source.BlobCID, &claim.Source.DeclaredMIME, &claim.Source.DeclaredSize,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim image scan: %w", err)
		}
		claim.LeaseToken = leaseToken
		if _, err := tx.Exec(ctx, `
		INSERT INTO image_scan_events(id,scan_result_id,scan_job_id,event_type,attempt)
		VALUES($1,$2,$3,'leased',$4)
		`, uuid.New(), claim.ResultID, claim.JobID, claim.Attempt); err != nil {
			return fmt.Errorf("record image scan lease: %w", err)
		}
		found = true
		return nil
	})
	if err != nil {
		return Claim{}, false, err
	}
	return claim, found, nil
}

func (store *WorkerStore) Complete(ctx context.Context, claim Claim, state State) ([]string, error) {
	if !state.Terminal() {
		return nil, errors.New("image scan completion requires a terminal state")
	}
	if state == StateMatch {
		return nil, errors.New("image scan match requires restricted incident completion")
	}
	return store.finish(ctx, claim, ScanResult{State: state}, "completed", "", time.Time{}, nil)
}

func (store *WorkerStore) CompleteResult(
	ctx context.Context,
	claim Claim,
	result ScanResult,
	recorder MatchRecorder,
) ([]string, error) {
	if !result.State.Terminal() {
		return nil, errors.New("image scan completion requires a terminal state")
	}
	if result.State == StateMatch && (recorder == nil || result.ProviderReference == "" || result.IntegrityMetadataReference == "") {
		return nil, errors.New("image scan match requires restricted incident metadata")
	}
	return store.finish(ctx, claim, result, "completed", "", time.Time{}, recorder)
}

func (store *WorkerStore) Retry(ctx context.Context, claim Claim, state State, category string, next time.Time) error {
	if state != StateUnavailable && state != StateError || category == "" || next.IsZero() {
		return errors.New("invalid image scan retry")
	}
	_, err := store.finish(ctx, claim, ScanResult{State: state}, "retry_scheduled", category, next, nil)
	return err
}

func (store *WorkerStore) DeadLetter(ctx context.Context, claim Claim, state State, category string) ([]string, error) {
	if state != StateUnavailable && state != StateError || category == "" {
		return nil, errors.New("invalid image scan dead letter")
	}
	return store.finish(ctx, claim, ScanResult{State: state}, "dead_lettered", category, time.Time{}, nil)
}

func (store *WorkerStore) finish(
	ctx context.Context,
	claim Claim,
	result ScanResult,
	eventType string,
	category string,
	next time.Time,
	recorder MatchRecorder,
) ([]string, error) {
	if store == nil || store.pool == nil || claim.JobID == uuid.Nil || claim.ResultID == uuid.Nil ||
		claim.LeaseToken == uuid.Nil || claim.ScanVersion <= 0 || claim.Attempt <= 0 {
		return nil, errors.New("invalid image scan completion")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	wake := make([]string, 0)
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		var fromState State
		err := tx.QueryRow(ctx, `
			SELECT result.state
			FROM image_scan_results result
			JOIN image_scan_jobs job ON job.scan_result_id=result.id
			WHERE result.id=$1 AND result.current_version=$2
			  AND job.id=$3 AND job.state='leased' AND job.scan_version=$2 AND job.lease_token=$4
			FOR UPDATE OF result,job
		`, claim.ResultID, claim.ScanVersion, claim.JobID, claim.LeaseToken).Scan(&fromState)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrLeaseLost
		}
		if err != nil {
			return fmt.Errorf("lock image scan lease: %w", err)
		}
		command, err := tx.Exec(ctx, `
			UPDATE image_scan_results
			SET state=$3,completed_at=CASE WHEN $4::boolean THEN $1::timestamptz ELSE NULL::timestamptz END,
			    provider_reference=$5,integrity_metadata_reference=$6,updated_at=$1
			WHERE id=$2
		`, now, claim.ResultID, result.State, result.State.Terminal(),
			optionalReference(result.ProviderReference), optionalReference(result.IntegrityMetadataReference))
		if err != nil {
			return fmt.Errorf("update image scan result: %w", err)
		}
		if command.RowsAffected() != 1 {
			return ErrLeaseLost
		}
		if result.State == StateMatch {
			if recorder == nil {
				return errors.New("image scan match recorder is unavailable")
			}
			if err := recorder.RecordMatchTx(ctx, tx, MatchDetection{
				ResultID: claim.ResultID, ProviderReference: result.ProviderReference,
				IntegrityMetadataReference: result.IntegrityMetadataReference, DetectedAt: now,
			}); err != nil {
				return fmt.Errorf("record restricted image match: %w", err)
			}
		}

		switch eventType {
		case "completed":
			if _, err := tx.Exec(ctx, `DELETE FROM image_scan_jobs WHERE id=$1 AND lease_token=$2`, claim.JobID, claim.LeaseToken); err != nil {
				return fmt.Errorf("delete completed image scan job: %w", err)
			}
		case "retry_scheduled":
			if _, err := tx.Exec(ctx, `
				UPDATE image_scan_jobs
				SET state='queued',next_attempt_at=$3,lease_owner=NULL,lease_token=NULL,
				    lease_expires_at=NULL,safe_error_category=$4,updated_at=$5
				WHERE id=$1 AND lease_token=$2
			`, claim.JobID, claim.LeaseToken, next.UTC(), category, now); err != nil {
				return fmt.Errorf("schedule image scan retry: %w", err)
			}
		case "dead_lettered":
			if _, err := tx.Exec(ctx, `
				UPDATE image_scan_jobs
				SET state='dead_letter',lease_owner=NULL,lease_token=NULL,lease_expires_at=NULL,
				    safe_error_category=$3,updated_at=$4
				WHERE id=$1 AND lease_token=$2
			`, claim.JobID, claim.LeaseToken, category, now); err != nil {
				return fmt.Errorf("dead-letter image scan: %w", err)
			}
		default:
			return errors.New("unknown image scan event")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO image_scan_events(
				id,scan_result_id,scan_job_id,event_type,from_state,to_state,attempt,created_at
			) VALUES($1,$2,NULL,$3,$4,$5,$6,$7)
		`, uuid.New(), claim.ResultID, eventType, fromState, result.State, claim.Attempt, now); err != nil {
			return fmt.Errorf("record image scan transition: %w", err)
		}
		if eventType == "retry_scheduled" {
			return nil
		}
		rows, err := tx.Query(ctx, `
			SELECT subject_uri FROM image_subject_requirements WHERE scan_result_id=$1
			UNION
			SELECT 'at://' || profile_did || '/app.bsky.actor.profile/self'
			FROM profile_image_candidates WHERE scan_result_id=$1
			ORDER BY 1
		`, claim.ResultID)
		if err != nil {
			return fmt.Errorf("list image scan dependants: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var uri string
			if err := rows.Scan(&uri); err != nil {
				return err
			}
			wake = append(wake, uri)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
		if len(wake) > 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE tap_projection_jobs
				SET state='pending',dependency_kind=NULL,dependency_key=NULL,
				    next_attempt_at=$2,last_reason_code=NULL,updated_at=$2
				WHERE state='blocked' AND dependency_kind='image_subject_uri'
				  AND dependency_key=ANY($1)
			`, wake, now); err != nil {
				return fmt.Errorf("wake image scan dependants: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return wake, nil
}

func optionalReference(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (store *WorkerStore) ManualRetry(ctx context.Context, resultID uuid.UUID, expectedVersion int64) (bool, error) {
	if store == nil || store.pool == nil || resultID == uuid.Nil || expectedVersion <= 0 {
		return false, errors.New("invalid image scan manual retry")
	}
	now := store.now().UTC().Truncate(time.Microsecond)
	retried := false
	err := pgx.BeginFunc(ctx, store.pool, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `
			UPDATE image_scan_jobs job
			SET state='queued',next_attempt_at=$3,lease_owner=NULL,lease_token=NULL,
			    lease_expires_at=NULL,safe_error_category=NULL,updated_at=$3
			WHERE job.scan_result_id=$1 AND job.scan_version=$2 AND job.state='dead_letter'
		`, resultID, expectedVersion, now)
		if err != nil {
			return fmt.Errorf("manual retry image scan: %w", err)
		}
		if command.RowsAffected() == 0 {
			return nil
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO image_scan_events(id,scan_result_id,event_type,attempt,created_at)
			SELECT $1,$2,'manual_retry',attempts,$3 FROM image_scan_jobs WHERE scan_result_id=$2
		`, uuid.New(), resultID, now); err != nil {
			return fmt.Errorf("record image scan manual retry: %w", err)
		}
		retried = true
		return nil
	})
	return retried, err
}

func (store *WorkerStore) Health(ctx context.Context) (QueueHealth, error) {
	if store == nil || store.pool == nil {
		return QueueHealth{}, errors.New("image scan store is unavailable")
	}
	now := store.now().UTC()
	var health QueueHealth
	var oldest *time.Time
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE state='queued'),
		       count(*) FILTER (WHERE state='leased'),
		       count(*) FILTER (WHERE state='dead_letter'),
		       min(next_attempt_at) FILTER (WHERE state='queued'),
		       COALESCE(max(attempts),0)
		FROM image_scan_jobs
	`).Scan(&health.Queued, &health.Leased, &health.DeadLettered, &oldest, &health.MaxAttempts); err != nil {
		return QueueHealth{}, fmt.Errorf("read image scan queue health: %w", err)
	}
	if oldest != nil && oldest.Before(now) {
		health.OldestDueAge = now.Sub(*oldest)
	}
	return health, nil
}
