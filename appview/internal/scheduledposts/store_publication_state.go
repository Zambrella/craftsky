// appview/internal/scheduledposts/store_publication_state.go
package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type FrozenRecordParams struct {
	ID              uuid.UUID
	OwnerDID        syntax.DID
	OwnerGeneration int64
	LeaseToken      uuid.UUID
	PayloadVersion  int64
	RecordBytes     []byte
	RecordHash      [32]byte
	Now             time.Time
}

type PublishingClaim struct {
	ID              uuid.UUID
	OwnerDID        syntax.DID
	OwnerGeneration int64
	LeaseToken      uuid.UUID
	PayloadVersion  int64
	Rkey            syntax.RecordKey
	CreatedAt       time.Time
}

func (s *Store) PrepareManualPublication(
	ctx context.Context,
	params UpdateParams,
) (WorkItem, error) {
	if s == nil || s.pool == nil || params.ID == uuid.Nil || params.OwnerDID == "" ||
		params.ScheduledAt.IsZero() || len(params.PayloadBytes) == 0 || params.Now.IsZero() {
		return WorkItem{}, errors.New("invalid manual scheduled publication")
	}
	if len(params.MediaIDs) > 4 || hasDuplicateUUIDs(params.MediaIDs) {
		return WorkItem{}, errors.New("invalid scheduled post media")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WorkItem{}, err
	}
	defer tx.Rollback(ctx)
	var ownerGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, params.OwnerDID).
		Scan(&ownerGeneration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkItem{}, ErrScheduleNotFound
		}
		return WorkItem{}, fmt.Errorf("lock manual publication owner lifecycle: %w", err)
	}
	if _, err := tx.Exec(ctx, lockScheduledPostOwnerForTransactionSQL, params.OwnerDID); err != nil {
		return WorkItem{}, fmt.Errorf("lock manual publication owner: %w", err)
	}
	if _, err := tx.Exec(ctx, lockScheduleEffectForTransactionSQL, params.OwnerDID, params.ID); err != nil {
		return WorkItem{}, fmt.Errorf("lock manual scheduled publication: %w", err)
	}

	var status Status
	var currentVersion int64
	if err := tx.QueryRow(ctx, selectScheduledPostForUpdateSQL, params.OwnerDID, params.ID, ownerGeneration).
		Scan(&status, &currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WorkItem{}, ErrScheduleNotFound
		}
		return WorkItem{}, fmt.Errorf("lock manual scheduled post: %w", err)
	}
	if !status.AllowsMemberMutation() {
		return WorkItem{}, ErrMutationLocked
	}

	type attachedMedia struct {
		id              uuid.UUID
		objectKey       string
		ownerGeneration int64
	}
	attached := make([]attachedMedia, 0, 4)
	rows, err := tx.Query(
		ctx,
		selectAttachedScheduledMediaForUpdateSQL,
		params.OwnerDID,
		params.ID,
		ownerGeneration,
	)
	if err != nil {
		return WorkItem{}, fmt.Errorf("select manual publication media: %w", err)
	}
	for rows.Next() {
		var media attachedMedia
		if err := rows.Scan(&media.id, &media.objectKey, &media.ownerGeneration); err != nil {
			rows.Close()
			return WorkItem{}, fmt.Errorf("scan manual publication media: %w", err)
		}
		attached = append(attached, media)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return WorkItem{}, fmt.Errorf("read manual publication media: %w", err)
	}
	rows.Close()

	for _, mediaID := range params.MediaIDs {
		var state string
		var scheduleID *uuid.UUID
		if err := tx.QueryRow(
			ctx,
			selectScheduledMediaForClaimSQL,
			params.OwnerDID,
			mediaID,
			ownerGeneration,
		).Scan(&state, &scheduleID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return WorkItem{}, ErrScheduledMediaUnavailable
			}
			return WorkItem{}, fmt.Errorf("lock manual publication media: %w", err)
		}
		if state != "ready" || (scheduleID != nil && *scheduleID != params.ID) {
			return WorkItem{}, ErrScheduledMediaUnavailable
		}
	}
	if _, err := tx.Exec(
		ctx,
		detachScheduledMediaForUpdateSQL,
		params.OwnerDID,
		params.ID,
		ownerGeneration,
	); err != nil {
		return WorkItem{}, fmt.Errorf("detach manual publication media: %w", err)
	}
	for ordinal, mediaID := range params.MediaIDs {
		result, err := tx.Exec(
			ctx,
			attachScheduledMediaSQL,
			params.OwnerDID,
			mediaID,
			ownerGeneration,
			params.ID,
			ordinal,
		)
		if err != nil {
			return WorkItem{}, fmt.Errorf("attach manual publication media: %w", err)
		}
		if result.RowsAffected() != 1 {
			return WorkItem{}, ErrScheduledMediaUnavailable
		}
	}
	retained := make(map[uuid.UUID]struct{}, len(params.MediaIDs))
	for _, mediaID := range params.MediaIDs {
		retained[mediaID] = struct{}{}
	}
	for _, media := range attached {
		if _, ok := retained[media.id]; ok {
			continue
		}
		result, err := tx.Exec(
			ctx,
			deleteUnclaimedPrivateMediaSQL,
			params.OwnerDID,
			media.ownerGeneration,
			media.id,
		)
		if err != nil {
			return WorkItem{}, fmt.Errorf("delete replaced manual publication media: %w", err)
		}
		if result.RowsAffected() != 1 {
			return WorkItem{}, ErrScheduledMediaUnavailable
		}
		if _, err := tx.Exec(
			ctx,
			insertCleanupJobSQL,
			uuid.New(),
			media.objectKey,
			params.Now.UTC(),
		); err != nil {
			return WorkItem{}, fmt.Errorf("enqueue replaced manual publication media: %w", err)
		}
	}

	rkey, err := allocatePublicationRkey(
		ctx,
		tx,
		params.OwnerDID,
		params.ID,
		params.Now.UTC(),
	)
	if err != nil {
		return WorkItem{}, err
	}
	leaseToken := uuid.New()
	var version int64
	if err := tx.QueryRow(
		ctx,
		updateAndClaimManualPublicationSQL,
		params.OwnerDID,
		params.ID,
		ownerGeneration,
		params.Now.UTC(),
		params.PayloadBytes,
		params.PayloadHash[:],
		leaseToken,
		params.Now.UTC().Add(DefaultPublicationLeaseDuration),
		rkey,
	).Scan(&version); err != nil {
		return WorkItem{}, fmt.Errorf("prepare manual scheduled publication: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return WorkItem{}, err
	}
	return WorkItem{
		ID: params.ID, OwnerDID: params.OwnerDID, OwnerGeneration: ownerGeneration,
		LeaseToken:     leaseToken,
		PayloadVersion: version, Rkey: rkey, CreatedAt: params.Now.UTC(), Manual: true,
	}, nil
}

func (s *Store) SaveFrozenRecord(ctx context.Context, params FrozenRecordParams) error {
	if s == nil || s.pool == nil || params.ID == uuid.Nil || params.OwnerDID == "" ||
		params.OwnerGeneration <= 0 || params.LeaseToken == uuid.Nil || params.PayloadVersion < 1 ||
		len(params.RecordBytes) == 0 || params.Now.IsZero() {
		return errors.New("invalid frozen publication record")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var activeGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, params.OwnerDID).
		Scan(&activeGeneration); err != nil || activeGeneration != params.OwnerGeneration {
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			return ErrWorkerLeaseLost
		}
		return fmt.Errorf("lock frozen publication owner lifecycle: %w", err)
	}

	var status Status
	var leaseToken *uuid.UUID
	var currentVersion int64
	if err := tx.QueryRow(
		ctx, selectWorkerFenceSQL, params.OwnerDID, params.ID, params.OwnerGeneration,
	).
		Scan(&status, &leaseToken, &currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrScheduleNotFound
		}
		return fmt.Errorf("lock scheduled post worker fence: %w", err)
	}
	if status != StatusPublishing || leaseToken == nil || *leaseToken != params.LeaseToken {
		return ErrWorkerLeaseLost
	}
	if err := ValidateWorkerVersion(currentVersion, params.PayloadVersion); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, saveFrozenRecordSQL,
		params.OwnerDID, params.ID, params.OwnerGeneration, params.LeaseToken, params.PayloadVersion,
		params.RecordBytes, params.RecordHash[:], params.Now.UTC()); err != nil {
		return fmt.Errorf("save frozen publication record: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *Store) ClaimDue(
	ctx context.Context,
	limit int,
	now time.Time,
	leaseDuration time.Duration,
) ([]PublishingClaim, error) {
	if s == nil || s.pool == nil || limit < 1 || limit > 100 || now.IsZero() || leaseDuration <= 0 {
		return nil, errors.New("invalid scheduled post claim")
	}
	now = now.UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	recoveredResult, err := tx.Exec(ctx, recoverExpiredPublishingLeasesSQL, now)
	if err != nil {
		return nil, fmt.Errorf("recover scheduled post leases: %w", err)
	}
	recoveredCount := recoveredResult.RowsAffected()
	type dueRow struct {
		id              uuid.UUID
		ownerDID        syntax.DID
		ownerGeneration int64
		payloadVersion  int64
		rkey            *string
		createdAt       *time.Time
		finalRecovery   bool
	}
	due := make([]dueRow, 0, limit)
	appendRows := func(rows pgx.Rows, finalRecovery bool) error {
		defer rows.Close()
		for rows.Next() {
			var row dueRow
			if err := rows.Scan(
				&row.id, &row.ownerDID, &row.ownerGeneration, &row.payloadVersion,
				&row.rkey, &row.createdAt,
			); err != nil {
				return fmt.Errorf("scan due scheduled post: %w", err)
			}
			row.finalRecovery = finalRecovery
			due = append(due, row)
		}
		return rows.Err()
	}
	rows, err := tx.Query(ctx, selectExpiredFinalPublishingSQL, limit, now)
	if err != nil {
		return nil, fmt.Errorf("select expired final scheduled posts: %w", err)
	}
	if err := appendRows(rows, true); err != nil {
		return nil, err
	}
	if remaining := limit - len(due); remaining > 0 {
		rows, err = tx.Query(ctx, selectDueScheduledPostsSQL, remaining, now)
		if err != nil {
			return nil, fmt.Errorf("select due scheduled posts: %w", err)
		}
		if err := appendRows(rows, false); err != nil {
			return nil, err
		}
	}

	claims := make([]PublishingClaim, 0, len(due))
	for _, row := range due {
		var rkey syntax.RecordKey
		createdAt := now
		if row.rkey != nil && row.createdAt != nil {
			parsed, err := syntax.ParseRecordKey(*row.rkey)
			if err != nil {
				return nil, fmt.Errorf("parse frozen publication key: %w", err)
			}
			rkey = parsed
			createdAt = row.createdAt.UTC()
		} else if row.finalRecovery {
			return nil, errors.New("final publication recovery is missing frozen identity")
		} else {
			var ownerLocked bool
			if err := tx.QueryRow(
				ctx, tryLockScheduledPostOwnerForTransactionSQL, row.ownerDID,
			).Scan(&ownerLocked); err != nil {
				return nil, fmt.Errorf("lock publication-key owner: %w", err)
			}
			if !ownerLocked {
				continue
			}
			var err error
			rkey, err = allocatePublicationRkey(ctx, tx, row.ownerDID, row.id, now)
			if err != nil {
				return nil, err
			}
		}
		leaseToken := uuid.New()
		if row.finalRecovery {
			if _, err := tx.Exec(ctx, reclaimExpiredFinalPublishingSQL,
				row.ownerDID, row.id, row.ownerGeneration, leaseToken,
				now.Add(leaseDuration), now,
			); err != nil {
				return nil, fmt.Errorf("reclaim final scheduled post: %w", err)
			}
		} else if _, err := tx.Exec(ctx, claimScheduledPostSQL,
			row.ownerDID, row.id, row.ownerGeneration, leaseToken,
			now.Add(leaseDuration), rkey, createdAt,
		); err != nil {
			return nil, fmt.Errorf("claim scheduled post: %w", err)
		}
		claims = append(claims, PublishingClaim{
			ID: row.id, OwnerDID: row.ownerDID, OwnerGeneration: row.ownerGeneration,
			LeaseToken:     leaseToken,
			PayloadVersion: row.payloadVersion, Rkey: rkey, CreatedAt: createdAt,
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

func allocatePublicationRkey(
	ctx context.Context,
	tx pgx.Tx,
	ownerDID syntax.DID,
	scheduleID uuid.UUID,
	now time.Time,
) (syntax.RecordKey, error) {
	firstClockID := (uint(scheduleID[14])<<8 | uint(scheduleID[15])) & (tidClockIDCount - 1)
	for offset := uint(0); offset < tidClockIDCount; offset++ {
		clockID := (firstClockID + offset) & (tidClockIDCount - 1)
		candidate, err := syntax.ParseRecordKey(syntax.NewTIDFromTime(now, clockID).String())
		if err != nil {
			return "", fmt.Errorf("allocate publication key: %w", err)
		}
		var available bool
		if err := tx.QueryRow(ctx, publicationRkeyAvailableSQL, ownerDID, candidate).Scan(&available); err != nil {
			return "", fmt.Errorf("check publication key availability: %w", err)
		}
		if available {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("allocate publication key: no record key available")
}
