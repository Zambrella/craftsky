// appview/internal/scheduledposts/store_draft_resource.go
package scheduledposts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateParams struct {
	ID             uuid.UUID
	OwnerDID       syntax.DID
	OperationID    uuid.UUID
	RequestHash    [32]byte
	ScheduledAt    time.Time
	PayloadBytes   []byte
	PayloadHash    [32]byte
	PayloadVersion int64
	MediaIDs       []uuid.UUID
}

type ScheduledPost struct {
	ID              uuid.UUID
	OwnerDID        syntax.DID
	OwnerGeneration int64
	OperationID     uuid.UUID
	Status          Status
	ScheduledAt     time.Time
	PayloadVersion  int64
	PublicationURI  syntax.ATURI
	PublicationCID  syntax.CID
	PublishedAt     time.Time
	Created         bool
}

type Resource struct {
	ScheduledPost
	PayloadBytes            []byte
	NeedsAttentionExpiresAt *time.Time
}

type UpdateParams struct {
	ID           uuid.UUID
	OwnerDID     syntax.DID
	ScheduledAt  time.Time
	PayloadBytes []byte
	PayloadHash  [32]byte
	MediaIDs     []uuid.UUID
	Now          time.Time
}

type UpdateResult struct {
	PayloadVersion int64
}

func (s *Store) Create(ctx context.Context, params CreateParams) (ScheduledPost, error) {
	if s == nil || s.pool == nil || params.ID == uuid.Nil || params.OwnerDID == "" ||
		params.OperationID == uuid.Nil || params.ScheduledAt.IsZero() ||
		len(params.PayloadBytes) == 0 || params.PayloadVersion < 1 {
		return ScheduledPost{}, errors.New("invalid scheduled post create")
	}
	if len(params.MediaIDs) > 4 || hasDuplicateUUIDs(params.MediaIDs) {
		return ScheduledPost{}, errors.New("invalid scheduled post media")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ScheduledPost{}, err
	}
	defer tx.Rollback(ctx)

	var ownerGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, params.OwnerDID).
		Scan(&ownerGeneration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScheduledPost{}, errors.New("scheduled post owner unavailable")
		}
		return ScheduledPost{}, fmt.Errorf("lock scheduled post owner lifecycle: %w", err)
	}
	var owner syntax.DID
	if err := tx.QueryRow(ctx, lockScheduledPostOwnerSQL, params.OwnerDID).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ScheduledPost{}, errors.New("scheduled post owner unavailable")
		}
		return ScheduledPost{}, fmt.Errorf("lock scheduled post owner: %w", err)
	}

	var existing ScheduledPost
	var existingRequestHash []byte
	err = tx.QueryRow(
		ctx,
		selectScheduledPostByOperationSQL,
		params.OwnerDID,
		ownerGeneration,
		params.OperationID,
	).Scan(
		&existing.ID,
		&existing.OwnerDID,
		&existing.OwnerGeneration,
		&existing.OperationID,
		&existing.Status,
		&existing.ScheduledAt,
		&existing.PayloadVersion,
		&existingRequestHash,
	)
	switch {
	case err == nil:
		if !bytes.Equal(existingRequestHash, params.RequestHash[:]) {
			return ScheduledPost{}, ErrOperationConflict
		}
		existing.ScheduledAt = existing.ScheduledAt.UTC()
		return existing, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return ScheduledPost{}, fmt.Errorf("read scheduled post operation: %w", err)
	}

	var completed ScheduledPost
	var completedRequestHash []byte
	var publicationURI string
	var publicationCID string
	err = tx.QueryRow(
		ctx,
		selectPublicationTombstoneByOperationSQL,
		params.OwnerDID,
		ownerGeneration,
		params.OperationID,
	).Scan(
		&completed.ID,
		&completed.OwnerDID,
		&completed.OwnerGeneration,
		&completed.OperationID,
		&completedRequestHash,
		&publicationURI,
		&publicationCID,
		&completed.PublishedAt,
	)
	switch {
	case err == nil:
		if !bytes.Equal(completedRequestHash, params.RequestHash[:]) {
			return ScheduledPost{}, ErrOperationConflict
		}
		completed.Status = StatusPublished
		completed.PublicationURI = syntax.ATURI(publicationURI)
		completed.PublicationCID = syntax.CID(publicationCID)
		completed.PublishedAt = completed.PublishedAt.UTC()
		return completed, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return ScheduledPost{}, fmt.Errorf("read scheduled publication operation: %w", err)
	}

	var active int
	if err := tx.QueryRow(ctx, countScheduledPostsSQL, params.OwnerDID, ownerGeneration).Scan(&active); err != nil {
		return ScheduledPost{}, fmt.Errorf("count scheduled posts: %w", err)
	}
	if active >= MaximumActivePosts {
		return ScheduledPost{}, ErrCapacityReached
	}

	scheduledAt := params.ScheduledAt.UTC()
	if _, err := tx.Exec(ctx, insertScheduledPostSQL,
		params.ID, params.OwnerDID, ownerGeneration, params.OperationID, params.RequestHash[:],
		StatusScheduled, scheduledAt, params.PayloadBytes, params.PayloadHash[:],
		params.PayloadVersion); err != nil {
		return ScheduledPost{}, fmt.Errorf("insert scheduled post: %w", err)
	}
	for ordinal, mediaID := range params.MediaIDs {
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
				return ScheduledPost{}, ErrScheduledMediaUnavailable
			}
			return ScheduledPost{}, fmt.Errorf("lock scheduled post media: %w", err)
		}
		if state != "ready" || scheduleID != nil {
			return ScheduledPost{}, ErrScheduledMediaUnavailable
		}
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
			return ScheduledPost{}, fmt.Errorf("attach scheduled post media: %w", err)
		}
		if result.RowsAffected() != 1 {
			return ScheduledPost{}, ErrScheduledMediaUnavailable
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return ScheduledPost{}, err
	}

	return ScheduledPost{
		ID:              params.ID,
		OwnerDID:        params.OwnerDID,
		OwnerGeneration: ownerGeneration,
		OperationID:     params.OperationID,
		Status:          StatusScheduled,
		ScheduledAt:     scheduledAt,
		PayloadVersion:  params.PayloadVersion,
		Created:         true,
	}, nil
}

func (s *Store) List(ctx context.Context, ownerDID syntax.DID) ([]Resource, error) {
	if s == nil || s.pool == nil || ownerDID == "" {
		return nil, errors.New("invalid scheduled post list")
	}
	rows, err := s.pool.Query(ctx, listScheduledPostsSQL, ownerDID)
	if err != nil {
		return nil, fmt.Errorf("list scheduled posts: %w", err)
	}
	defer rows.Close()
	resources := make([]Resource, 0, MaximumActivePosts)
	for rows.Next() {
		resource, err := scanScheduledPostResource(rows)
		if err != nil {
			return nil, fmt.Errorf("scan scheduled post list: %w", err)
		}
		resources = append(resources, resource)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read scheduled post list: %w", err)
	}
	return resources, nil
}

func (s *Store) Get(ctx context.Context, ownerDID syntax.DID, id uuid.UUID) (Resource, error) {
	if s == nil || s.pool == nil || ownerDID == "" || id == uuid.Nil {
		return Resource{}, errors.New("invalid scheduled post get")
	}
	resource, err := scanScheduledPostResource(s.pool.QueryRow(ctx, getScheduledPostSQL, ownerDID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Resource{}, ErrScheduleNotFound
	}
	if err != nil {
		return Resource{}, fmt.Errorf("get scheduled post: %w", err)
	}
	return resource, nil
}

type scheduledPostResourceScanner interface {
	Scan(...any) error
}

func scanScheduledPostResource(scanner scheduledPostResourceScanner) (Resource, error) {
	var resource Resource
	var expiresAt *time.Time
	if err := scanner.Scan(
		&resource.ID,
		&resource.OwnerDID,
		&resource.OwnerGeneration,
		&resource.OperationID,
		&resource.Status,
		&resource.ScheduledAt,
		&resource.PayloadBytes,
		&resource.PayloadVersion,
		&expiresAt,
	); err != nil {
		return Resource{}, err
	}
	resource.ScheduledAt = resource.ScheduledAt.UTC()
	if expiresAt != nil {
		utc := expiresAt.UTC()
		resource.NeedsAttentionExpiresAt = &utc
	}
	return resource, nil
}

func (s *Store) Update(ctx context.Context, params UpdateParams) (UpdateResult, error) {
	if s == nil || s.pool == nil || params.ID == uuid.Nil || params.OwnerDID == "" ||
		params.ScheduledAt.IsZero() || len(params.PayloadBytes) == 0 || params.Now.IsZero() {
		return UpdateResult{}, errors.New("invalid scheduled post update")
	}
	if len(params.MediaIDs) > 4 || hasDuplicateUUIDs(params.MediaIDs) {
		return UpdateResult{}, errors.New("invalid scheduled post media")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return UpdateResult{}, err
	}
	defer tx.Rollback(ctx)
	var ownerGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, params.OwnerDID).
		Scan(&ownerGeneration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UpdateResult{}, ErrScheduleNotFound
		}
		return UpdateResult{}, fmt.Errorf("lock scheduled post owner lifecycle: %w", err)
	}
	if _, err := tx.Exec(ctx, lockScheduleEffectForTransactionSQL, params.OwnerDID, params.ID); err != nil {
		return UpdateResult{}, fmt.Errorf("lock scheduled post effect for update: %w", err)
	}

	var status Status
	var currentVersion int64
	if err := tx.QueryRow(ctx, selectScheduledPostForUpdateSQL, params.OwnerDID, params.ID, ownerGeneration).
		Scan(&status, &currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UpdateResult{}, ErrScheduleNotFound
		}
		return UpdateResult{}, fmt.Errorf("lock scheduled post update: %w", err)
	}
	if !status.AllowsMemberMutation() {
		return UpdateResult{}, ErrMutationLocked
	}

	type attachedMedia struct {
		id              uuid.UUID
		objectKey       string
		ownerGeneration int64
	}
	attached := make([]attachedMedia, 0, 4)
	rows, err := tx.Query(
		ctx, selectAttachedScheduledMediaForUpdateSQL, params.OwnerDID, params.ID, ownerGeneration,
	)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("select scheduled post media for update: %w", err)
	}
	for rows.Next() {
		var media attachedMedia
		if err := rows.Scan(&media.id, &media.objectKey, &media.ownerGeneration); err != nil {
			rows.Close()
			return UpdateResult{}, fmt.Errorf("scan scheduled post media for update: %w", err)
		}
		attached = append(attached, media)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return UpdateResult{}, fmt.Errorf("read scheduled post media for update: %w", err)
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
				return UpdateResult{}, ErrScheduledMediaUnavailable
			}
			return UpdateResult{}, fmt.Errorf("lock replacement scheduled post media: %w", err)
		}
		if state != "ready" || (scheduleID != nil && *scheduleID != params.ID) {
			return UpdateResult{}, ErrScheduledMediaUnavailable
		}
	}
	if _, err := tx.Exec(
		ctx, detachScheduledMediaForUpdateSQL, params.OwnerDID, params.ID, ownerGeneration,
	); err != nil {
		return UpdateResult{}, fmt.Errorf("detach scheduled post media for update: %w", err)
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
			return UpdateResult{}, fmt.Errorf("attach replacement scheduled post media: %w", err)
		}
		if result.RowsAffected() != 1 {
			return UpdateResult{}, ErrScheduledMediaUnavailable
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
			ctx, deleteUnclaimedPrivateMediaSQL,
			params.OwnerDID, media.ownerGeneration, media.id,
		)
		if err != nil {
			return UpdateResult{}, fmt.Errorf("delete replaced scheduled post media: %w", err)
		}
		if result.RowsAffected() != 1 {
			return UpdateResult{}, ErrScheduledMediaUnavailable
		}
		if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), media.objectKey, params.Now.UTC()); err != nil {
			return UpdateResult{}, fmt.Errorf("enqueue replaced scheduled post media: %w", err)
		}
	}

	var version int64
	if err := tx.QueryRow(ctx, updateScheduledPostSQL,
		params.OwnerDID, params.ID, ownerGeneration, params.ScheduledAt.UTC(), params.PayloadBytes,
		params.PayloadHash[:], params.Now.UTC()).Scan(&version); err != nil {
		return UpdateResult{}, fmt.Errorf("update scheduled post: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return UpdateResult{}, err
	}
	return UpdateResult{PayloadVersion: version}, nil
}

func (s *Store) Delete(
	ctx context.Context,
	ownerDID syntax.DID,
	id uuid.UUID,
	now time.Time,
) error {
	if s == nil || s.pool == nil || ownerDID == "" || id == uuid.Nil || now.IsZero() {
		return errors.New("invalid scheduled post delete")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var ownerGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, ownerDID).
		Scan(&ownerGeneration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("lock deleted schedule owner lifecycle: %w", err)
	}
	if _, err := tx.Exec(ctx, lockScheduleEffectForTransactionSQL, ownerDID, id); err != nil {
		return fmt.Errorf("lock scheduled post effect for delete: %w", err)
	}
	var status Status
	var version int64
	if err := tx.QueryRow(ctx, selectScheduledPostForUpdateSQL, ownerDID, id, ownerGeneration).
		Scan(&status, &version); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("lock scheduled post delete: %w", err)
	}
	if !status.AllowsMemberMutation() {
		return ErrMutationLocked
	}
	rows, err := tx.Query(
		ctx, selectScheduledMediaForFinalizationSQL, ownerDID, id, ownerGeneration,
	)
	if err != nil {
		return fmt.Errorf("select scheduled post media for delete: %w", err)
	}
	objectKeys := make([]string, 0)
	for rows.Next() {
		var objectKey string
		if err := rows.Scan(&objectKey); err != nil {
			rows.Close()
			return fmt.Errorf("scan scheduled post media for delete: %w", err)
		}
		objectKeys = append(objectKeys, objectKey)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read scheduled post media for delete: %w", err)
	}
	rows.Close()
	if _, err := tx.Exec(ctx, deleteScheduledPostSQL, ownerDID, id, ownerGeneration); err != nil {
		return fmt.Errorf("delete scheduled post: %w", err)
	}
	for _, objectKey := range objectKeys {
		if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), objectKey, now.UTC()); err != nil {
			return fmt.Errorf("enqueue deleted scheduled post media: %w", err)
		}
	}
	return tx.Commit(ctx)
}
