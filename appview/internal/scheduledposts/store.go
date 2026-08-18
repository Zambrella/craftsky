package scheduledposts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaximumActivePosts             = 3
	publishingEffectCleanupTimeout = 5 * time.Second
	tidClockIDCount                = 1 << 10
)

var (
	ErrCapacityReached           = errors.New("scheduled post capacity reached")
	ErrScheduleNotFound          = errors.New("scheduled post not found")
	ErrMutationLocked            = errors.New("scheduled post mutation locked")
	ErrWorkerLeaseLost           = errors.New("scheduled post worker lease lost")
	ErrOperationConflict         = errors.New("scheduled post operation conflict")
	ErrScheduledMediaUnavailable = errors.New("scheduled post media unavailable")
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

type PublishingEffectGuard struct {
	conn     *pgxpool.Conn
	ownerDID syntax.DID
	id       uuid.UUID
	mu       sync.Mutex
	released bool
}

type publishingEffectConnectionKey struct{}

type publicationDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Store struct {
	pool     *pgxpool.Pool
	observer OperationalObserver
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) SetOperationalObserver(observer OperationalObserver) {
	if s != nil {
		s.observer = observer
	}
}

func (s *Store) publicationDB(ctx context.Context) publicationDatabase {
	if conn, ok := ctx.Value(publishingEffectConnectionKey{}).(*pgxpool.Conn); ok && conn != nil {
		return conn
	}
	return s.pool
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

func hasDuplicateUUIDs(values []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return true
		}
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
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

func (s *Store) AcquirePublishingEffect(
	ctx context.Context,
	claim PublishingClaim,
) (*PublishingEffectGuard, error) {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil || claim.OwnerDID == "" ||
		claim.OwnerGeneration <= 0 || claim.LeaseToken == uuid.Nil || claim.PayloadVersion < 1 {
		return nil, errors.New("invalid publishing effect claim")
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	handedOff := false
	lockHeld := false
	defer func() {
		if !handedOff {
			if lockHeld {
				_, _ = conn.Exec(
					context.Background(),
					unlockScheduleEffectForSessionSQL,
					claim.OwnerDID,
					claim.ID,
				)
			}
			conn.Release()
		}
	}()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var activeGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, claim.OwnerDID).
		Scan(&activeGeneration); err != nil || activeGeneration != claim.OwnerGeneration {
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkerLeaseLost
		}
		return nil, fmt.Errorf("recheck publishing owner lifecycle: %w", err)
	}
	if _, err := tx.Exec(
		ctx,
		lockScheduleEffectForSessionSQL,
		claim.OwnerDID,
		claim.ID,
	); err != nil {
		return nil, fmt.Errorf("lock publishing effect: %w", err)
	}
	lockHeld = true
	var status Status
	var leaseToken *uuid.UUID
	var currentVersion int64
	if err := tx.QueryRow(
		ctx, selectWorkerFenceSQL, claim.OwnerDID, claim.ID, claim.OwnerGeneration,
	).
		Scan(&status, &leaseToken, &currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("recheck publishing effect: %w", err)
	}
	if status != StatusPublishing || leaseToken == nil || *leaseToken != claim.LeaseToken {
		return nil, ErrWorkerLeaseLost
	}
	if err := ValidateWorkerVersion(currentVersion, claim.PayloadVersion); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	handedOff = true
	return &PublishingEffectGuard{
		conn: conn, ownerDID: claim.OwnerDID, id: claim.ID,
	}, nil
}

func (guard *PublishingEffectGuard) Release(_ context.Context) error {
	if guard == nil {
		return nil
	}
	guard.mu.Lock()
	defer guard.mu.Unlock()
	if guard.released {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(
		context.Background(),
		publishingEffectCleanupTimeout,
	)
	defer cancel()
	var unlocked bool
	err := guard.conn.QueryRow(cleanupCtx, unlockScheduleEffectForSessionSQL, guard.ownerDID, guard.id).
		Scan(&unlocked)
	if err == nil && unlocked {
		guard.conn.Release()
		guard.conn = nil
		guard.released = true
		return nil
	}

	hijacked := guard.conn.Hijack()
	guard.conn = nil
	guard.released = true
	closeCtx, closeCancel := context.WithTimeout(
		context.Background(),
		publishingEffectCleanupTimeout,
	)
	defer closeCancel()
	closeErr := hijacked.Close(closeCtx)
	if err != nil {
		if closeErr != nil {
			return fmt.Errorf("unlock publishing effect: %w; discard connection: %v", err, closeErr)
		}
		return fmt.Errorf("unlock publishing effect: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("publishing effect lock was not held; discard connection: %w", closeErr)
	}
	return errors.New("publishing effect lock was not held")
}

// bind keeps publication completion on the same pool connection that holds
// the session advisory lock. The real OAuth effect boundary already occupies
// its own fenced connection, so consulting the general pool here can deadlock
// at small but valid pool capacities.
func (guard *PublishingEffectGuard) bind(ctx context.Context) context.Context {
	if guard == nil {
		return ctx
	}
	guard.mu.Lock()
	conn := guard.conn
	released := guard.released
	guard.mu.Unlock()
	if released || conn == nil {
		return ctx
	}
	return context.WithValue(ctx, publishingEffectConnectionKey{}, conn)
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
