package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CleanupErrorObjectDeleteFailed = "object_delete_failed"
	CleanupErrorSettlementPending  = "settlement_pending"
)

var ErrCleanupLeaseLost = errors.New("scheduled cleanup lease lost")

type CleanupClaim struct {
	ID                  uuid.UUID
	ObjectKey           string
	OwnerDID            syntax.DID
	OwnerGeneration     int64
	UploadGeneration    int64
	SourceAttemptID     uuid.UUID
	OutcomeUncertain    bool
	RemoteDeadline      time.Time
	SettlementNotBefore *time.Time
	LeaseToken          uuid.UUID
	AttemptCount        int
}

type cleanupEffectGuard struct {
	conn      *pgxpool.Conn
	objectKey string
	mu        sync.Mutex
	released  bool
}

func (s *Store) SweepExpiredLifecycle(ctx context.Context, now time.Time) error {
	if s == nil || s.pool == nil || now.IsZero() {
		return errors.New("invalid scheduled lifecycle sweep")
	}
	now = now.UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, deleteReferencedPendingCleanupJobsSQL); err != nil {
		return fmt.Errorf("cancel referenced cleanup jobs: %w", err)
	}

	type expiredMedia struct {
		id        uuid.UUID
		objectKey string
	}
	unclaimed := make([]expiredMedia, 0)
	rows, err := tx.Query(ctx, selectExpiredUnclaimedMediaSQL, now)
	if err != nil {
		return fmt.Errorf("select expired unclaimed media: %w", err)
	}
	for rows.Next() {
		var media expiredMedia
		if err := rows.Scan(&media.id, &media.objectKey); err != nil {
			rows.Close()
			return fmt.Errorf("scan expired unclaimed media: %w", err)
		}
		unclaimed = append(unclaimed, media)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	objectKeys := make([]string, 0, len(unclaimed))
	for _, media := range unclaimed {
		result, err := tx.Exec(ctx, deleteExpiredUnclaimedMediaSQL, media.id, now)
		if err != nil {
			return fmt.Errorf("delete expired unclaimed media: %w", err)
		}
		if result.RowsAffected() == 1 {
			objectKeys = append(objectKeys, media.objectKey)
		}
	}

	rows, err = tx.Query(ctx, selectExpiredNeedsAttentionMediaSQL, now)
	if err != nil {
		return fmt.Errorf("select expired scheduled media: %w", err)
	}
	for rows.Next() {
		var objectKey string
		if err := rows.Scan(&objectKey); err != nil {
			rows.Close()
			return fmt.Errorf("scan expired scheduled media: %w", err)
		}
		objectKeys = append(objectKeys, objectKey)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if _, err := tx.Exec(ctx, deleteExpiredNeedsAttentionSQL, now); err != nil {
		return fmt.Errorf("delete expired scheduled posts: %w", err)
	}
	for _, objectKey := range objectKeys {
		if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), objectKey, now); err != nil {
			return fmt.Errorf("enqueue scheduled media cleanup: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, deleteExpiredTombstonesSQL, now); err != nil {
		return fmt.Errorf("delete expired publication tombstones: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *Store) ClaimCleanup(
	ctx context.Context,
	limit int,
	now time.Time,
	leaseDuration time.Duration,
) ([]CleanupClaim, error) {
	if s == nil || s.pool == nil || limit < 1 || limit > 100 || now.IsZero() || leaseDuration <= 0 {
		return nil, errors.New("invalid scheduled cleanup claim")
	}
	now = now.UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	recoveredResult, err := tx.Exec(ctx, recoverExpiredCleanupJobsSQL, now)
	if err != nil {
		return nil, fmt.Errorf("recover cleanup leases: %w", err)
	}
	recoveredCount := recoveredResult.RowsAffected()
	if _, err := tx.Exec(ctx, deleteReferencedPendingCleanupJobsSQL); err != nil {
		return nil, fmt.Errorf("cancel referenced cleanup jobs: %w", err)
	}
	rows, err := tx.Query(ctx, selectDueCleanupJobsSQL, limit, now)
	if err != nil {
		return nil, fmt.Errorf("select cleanup jobs: %w", err)
	}
	type dueCleanup struct {
		id                  uuid.UUID
		objectKey           string
		ownerDID            syntax.DID
		ownerGeneration     int64
		uploadGeneration    int64
		sourceAttemptID     uuid.UUID
		outcomeUncertain    bool
		remoteDeadline      time.Time
		settlementNotBefore *time.Time
		attemptCount        int
	}
	due := make([]dueCleanup, 0, limit)
	for rows.Next() {
		var job dueCleanup
		if err := rows.Scan(
			&job.id,
			&job.objectKey,
			&job.ownerDID,
			&job.ownerGeneration,
			&job.uploadGeneration,
			&job.sourceAttemptID,
			&job.outcomeUncertain,
			&job.remoteDeadline,
			&job.settlementNotBefore,
			&job.attemptCount,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan cleanup job: %w", err)
		}
		due = append(due, job)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	claims := make([]CleanupClaim, 0, len(due))
	for _, job := range due {
		leaseToken := uuid.New()
		if _, err := tx.Exec(
			ctx,
			claimCleanupJobSQL,
			job.id,
			leaseToken,
			now.Add(leaseDuration),
			now,
		); err != nil {
			return nil, fmt.Errorf("claim cleanup job: %w", err)
		}
		claims = append(claims, CleanupClaim{
			ID:                  job.id,
			ObjectKey:           job.objectKey,
			OwnerDID:            job.ownerDID,
			OwnerGeneration:     job.ownerGeneration,
			UploadGeneration:    job.uploadGeneration,
			SourceAttemptID:     job.sourceAttemptID,
			OutcomeUncertain:    job.outcomeUncertain,
			RemoteDeadline:      job.remoteDeadline,
			SettlementNotBefore: job.settlementNotBefore,
			LeaseToken:          leaseToken,
			AttemptCount:        job.attemptCount + 1,
		})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	if s.observer != nil {
		for range recoveredCount {
			s.observer.ObserveScheduledOperation(
				"recover", "success", "lease_expired", 0,
			)
		}
	}
	return claims, nil
}

func (s *Store) RecordCleanupAbsence(
	ctx context.Context,
	claim CleanupClaim,
	absentAt time.Time,
) (ObjectCleanupSettlement, error) {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil ||
		claim.LeaseToken == uuid.Nil || absentAt.IsZero() {
		return ObjectCleanupSettlement{}, errors.New("invalid scheduled cleanup absence")
	}
	var settlement ObjectCleanupSettlement
	if err := s.pool.QueryRow(
		ctx,
		recordCleanupAbsenceSQL,
		claim.ID,
		claim.LeaseToken,
		absentAt.UTC(),
	).Scan(
		&settlement.OutcomeUncertain,
		&settlement.SettlementNotBefore,
		&settlement.LastAbsenceAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ObjectCleanupSettlement{}, ErrCleanupLeaseLost
		}
		return ObjectCleanupSettlement{}, fmt.Errorf("record scheduled cleanup absence: %w", err)
	}
	settlement.RemoteDeadline = claim.RemoteDeadline
	return settlement, nil
}

func (s *Store) CompleteCleanup(ctx context.Context, claim CleanupClaim) error {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil || claim.LeaseToken == uuid.Nil {
		return errors.New("invalid scheduled cleanup completion")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("complete cleanup job: %w", err)
	}
	defer tx.Rollback(ctx)
	var sourceAttemptID uuid.UUID
	if err := tx.QueryRow(ctx, `
		DELETE FROM scheduled_post_cleanup_jobs
		WHERE id=$1 AND state='deleting' AND lease_token=$2
		RETURNING source_attempt_id
	`, claim.ID, claim.LeaseToken).Scan(&sourceAttemptID); errors.Is(err, pgx.ErrNoRows) {
		return ErrCleanupLeaseLost
	} else if err != nil {
		return fmt.Errorf("complete cleanup job: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE account_deletion_safety_tombstones
		SET state='settled',next_attempt_at=NULL,lease_token=NULL,
		    lease_expires_at=NULL,last_result_category='finalAbsenceProven',
		    settled_at=clock_timestamp(),updated_at=clock_timestamp()
		WHERE kind='scheduled_object'
		  AND owner_did=$1 AND owner_generation=$2
		  AND exact_key=$3 AND upload_generation=$4
		  AND source_attempt_id=$5 AND state<>'settled'
	`, claim.OwnerDID, claim.OwnerGeneration, claim.ObjectKey,
		claim.UploadGeneration, sourceAttemptID.String()); err != nil {
		return fmt.Errorf("settle account deletion object safety: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM scheduled_post_object_attempts AS attempts
		WHERE attempts.upload_attempt_id=$1
		  AND NOT EXISTS (
			SELECT 1 FROM scheduled_post_media AS media
			WHERE media.upload_attempt_id=attempts.upload_attempt_id
		  )
	`, sourceAttemptID); err != nil {
		return fmt.Errorf("remove settled object attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit completed cleanup: %w", err)
	}
	return nil
}

func (s *Store) AcquireCleanupEffect(
	ctx context.Context,
	claim CleanupClaim,
) (CleanupEffectGuard, error) {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil || claim.LeaseToken == uuid.Nil ||
		claim.ObjectKey == "" {
		return nil, errors.New("invalid scheduled cleanup effect claim")
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	locked := false
	handedOff := false
	defer func() {
		if handedOff {
			return
		}
		if !locked {
			conn.Release()
			return
		}
		_ = releaseCleanupEffectConnection(conn, claim.ObjectKey)
	}()
	if _, err := conn.Exec(ctx, lockCleanupEffectForSessionSQL, claim.ObjectKey); err != nil {
		return nil, fmt.Errorf("lock scheduled cleanup effect: %w", err)
	}
	locked = true
	var objectKey string
	if err := conn.QueryRow(
		ctx,
		selectCleanupJobEffectFenceSQL,
		claim.ID,
		claim.LeaseToken,
	).Scan(&objectKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCleanupLeaseLost
		}
		return nil, fmt.Errorf("recheck scheduled cleanup effect: %w", err)
	}
	if objectKey != claim.ObjectKey {
		return nil, ErrCleanupLeaseLost
	}
	handedOff = true
	return &cleanupEffectGuard{conn: conn, objectKey: objectKey}, nil
}

func (guard *cleanupEffectGuard) Release(_ context.Context) error {
	if guard == nil {
		return nil
	}
	guard.mu.Lock()
	defer guard.mu.Unlock()
	if guard.released {
		return nil
	}
	err := releaseCleanupEffectConnection(guard.conn, guard.objectKey)
	guard.conn = nil
	guard.released = true
	return err
}

func releaseCleanupEffectConnection(conn *pgxpool.Conn, objectKey string) error {
	cleanupCtx, cancel := context.WithTimeout(
		context.Background(),
		publishingEffectCleanupTimeout,
	)
	defer cancel()
	var unlocked bool
	err := conn.QueryRow(cleanupCtx, unlockCleanupEffectForSessionSQL, objectKey).
		Scan(&unlocked)
	if err == nil && unlocked {
		conn.Release()
		return nil
	}
	hijacked := conn.Hijack()
	closeCtx, closeCancel := context.WithTimeout(
		context.Background(),
		publishingEffectCleanupTimeout,
	)
	defer closeCancel()
	closeErr := hijacked.Close(closeCtx)
	if err != nil {
		if closeErr != nil {
			return fmt.Errorf("unlock scheduled cleanup effect: %w; discard connection: %v", err, closeErr)
		}
		return fmt.Errorf("unlock scheduled cleanup effect: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("scheduled cleanup effect lock was not held; discard connection: %w", closeErr)
	}
	return errors.New("scheduled cleanup effect lock was not held")
}

func (s *Store) PrepareCleanupDelete(
	ctx context.Context,
	claim CleanupClaim,
) (bool, error) {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil || claim.LeaseToken == uuid.Nil ||
		claim.ObjectKey == "" {
		return false, errors.New("invalid scheduled cleanup preparation")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var objectKey string
	if err := tx.QueryRow(
		ctx,
		selectCleanupJobForDeleteSQL,
		claim.ID,
		claim.LeaseToken,
	).Scan(&objectKey); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrCleanupLeaseLost
		}
		return false, fmt.Errorf("lock cleanup job for delete: %w", err)
	}
	if objectKey != claim.ObjectKey {
		return false, ErrCleanupLeaseLost
	}
	var referenced bool
	if err := tx.QueryRow(ctx, scheduledMediaObjectReferencedSQL, objectKey).Scan(&referenced); err != nil {
		return false, fmt.Errorf("check scheduled media reference: %w", err)
	}
	if referenced {
		result, err := tx.Exec(ctx, completeCleanupJobSQL, claim.ID, claim.LeaseToken)
		if err != nil {
			return false, fmt.Errorf("cancel referenced cleanup job: %w", err)
		}
		if result.RowsAffected() != 1 {
			return false, ErrCleanupLeaseLost
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return !referenced, nil
}

func (s *Store) RetryCleanup(
	ctx context.Context,
	claim CleanupClaim,
	nextAttemptAt time.Time,
	safeCode string,
	now time.Time,
) error {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil || claim.LeaseToken == uuid.Nil ||
		now.IsZero() || nextAttemptAt.Before(now) ||
		(safeCode != CleanupErrorObjectDeleteFailed && safeCode != CleanupErrorSettlementPending) {
		return errors.New("invalid scheduled cleanup retry")
	}
	result, err := s.pool.Exec(
		ctx,
		retryCleanupJobSQL,
		claim.ID,
		claim.LeaseToken,
		nextAttemptAt.UTC(),
		safeCode,
		now.UTC(),
	)
	if err != nil {
		return fmt.Errorf("retry cleanup job: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrCleanupLeaseLost
	}
	return nil
}
