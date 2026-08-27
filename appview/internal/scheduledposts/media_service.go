package scheduledposts

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/jackc/pgx/v5"
	"github.com/multiformats/go-multihash"

	"social.craftsky/appview/internal/ownerlifecycle"
)

var (
	ErrScheduledMediaNotFound       = errors.New("scheduled media not found")
	ErrScheduledMediaConflict       = errors.New("scheduled media conflict")
	ErrScheduledMediaOutcomeUnknown = errors.New("scheduled media upload outcome is unknown")
)

type PutPrivateMediaParams struct {
	ID              uuid.UUID
	OwnerDID        syntax.DID
	OwnerGeneration int64
	MIMEType        string
	Bytes           []byte
	Now             time.Time
}

type PrivateMedia struct {
	ID                  uuid.UUID
	OwnerDID            syntax.DID
	OwnerGeneration     int64
	UploadGeneration    int64
	UploadAttemptID     uuid.UUID
	ObjectKey           string
	State               string
	MIMEType            string
	SizeBytes           int64
	SHA256              [32]byte
	BlobCID             syntax.CID
	ExpiresAt           time.Time
	RemoteDeadline      time.Time
	SettlementNotBefore *time.Time
}

type OpenedPrivateMedia struct {
	MIMEType  string
	SizeBytes int64
	Body      io.ReadCloser
}

type PrivateMediaService struct {
	store                 *Store
	objects               PrivateObjectStore
	lifecycle             activeOwnerEffects
	putTimeout            time.Duration
	testedSettlementBound time.Duration
	settlementMargin      time.Duration
}

type activeOwnerEffects interface {
	WithActiveEffects(
		context.Context,
		[]ownerlifecycle.ExpectedOwner,
		func(context.Context) error,
	) error
}

type PrivateMediaServiceOptions struct {
	Lifecycle             activeOwnerEffects
	PutTimeout            time.Duration
	TestedSettlementBound time.Duration
	SettlementMargin      time.Duration
}

func NewPrivateMediaService(
	store *Store,
	objects PrivateObjectStore,
	options ...PrivateMediaServiceOptions,
) *PrivateMediaService {
	service := &PrivateMediaService{store: store, objects: objects}
	if len(options) == 1 {
		service.lifecycle = options[0].Lifecycle
		service.putTimeout = options[0].PutTimeout
		service.testedSettlementBound = options[0].TestedSettlementBound
		service.settlementMargin = options[0].SettlementMargin
	}
	return service
}

func (s *PrivateMediaService) Put(
	ctx context.Context,
	params PutPrivateMediaParams,
) (PrivateMedia, error) {
	if s == nil || s.store == nil || s.objects == nil || s.lifecycle == nil ||
		s.putTimeout <= 0 || params.ID == uuid.Nil || params.OwnerDID == "" ||
		params.OwnerGeneration <= 0 || params.MIMEType == "" || len(params.Bytes) == 0 ||
		params.Now.IsZero() || s.testedSettlementBound < 0 || s.settlementMargin < 0 ||
		(s.testedSettlementBound == 0 && s.settlementMargin != 0) {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictInvalidRequest)
	}
	digest := sha256.Sum256(params.Bytes)
	multihashValue, err := multihash.Sum(params.Bytes, multihash.SHA2_256, -1)
	if err != nil {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictDigestConstruction)
	}
	predictedCID := syntax.CID(cid.NewCidV1(cid.Raw, multihashValue).String())
	objectKey, uploadAttemptID, err := NewGenerationObjectKey(
		params.OwnerDID, params.OwnerGeneration, params.ID,
	)
	if err != nil {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictObjectKeyConstruction)
	}
	remoteDeadline := params.Now.UTC().Add(s.putTimeout)
	var settlementNotBefore *time.Time
	if s.testedSettlementBound > 0 {
		boundary := remoteDeadline.Add(s.testedSettlementBound + s.settlementMargin)
		settlementNotBefore = &boundary
	}

	var result PrivateMedia
	err = s.lifecycle.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{
		Owner: params.OwnerDID, Generation: params.OwnerGeneration,
	}}, func(effectCtx context.Context) (effectErr error) {
		lockCtx, cancelLock := context.WithTimeout(effectCtx, s.putTimeout)
		objectFence, err := acquirePrivateObjectFence(lockCtx, s.store.pool, objectKey)
		cancelLock()
		if err != nil {
			return err
		}
		defer func() { effectErr = errors.Join(effectErr, objectFence.Release()) }()

		media, err := s.store.reservePrivateMedia(
			effectCtx,
			params.ID,
			params.OwnerDID,
			params.OwnerGeneration,
			uploadAttemptID,
			objectKey,
			params.MIMEType,
			int64(len(params.Bytes)),
			digest,
			remoteDeadline,
			settlementNotBefore,
			params.Now.UTC(),
		)
		if err != nil {
			return err
		}
		if media.State == "ready" {
			if media.BlobCID != predictedCID {
				return NewScheduledMediaConflict(MediaConflictExistingBlob)
			}
			result = media
			return nil
		}
		if err := s.store.markPrivateMediaDispatched(
			effectCtx, media, params.Now.UTC(),
		); err != nil {
			return err
		}
		remaining := media.RemoteDeadline.Sub(params.Now.UTC())
		if remaining <= 0 {
			return ErrScheduledMediaOutcomeUnknown
		}
		putCtx, cancelPut := context.WithTimeout(effectCtx, remaining)
		putErr := s.objects.Put(
			putCtx,
			media.ObjectKey,
			bytes.NewReader(params.Bytes),
			int64(len(params.Bytes)),
			params.MIMEType,
		)
		cancelPut()
		if putErr != nil {
			if err := s.store.movePrivateMediaToCleanup(
				effectCtx, media, params.Now.UTC(),
			); err != nil {
				return err
			}
			return ErrPrivateObjectStoreUnavailable
		}
		result, err = s.store.markPrivateMediaReady(
			effectCtx, media, predictedCID, params.Now.UTC(),
		)
		return err
	})
	if err != nil {
		return PrivateMedia{}, err
	}
	return result, nil
}

func (s *PrivateMediaService) Open(
	ctx context.Context,
	ownerDID syntax.DID,
	mediaID uuid.UUID,
) (OpenedPrivateMedia, error) {
	if s == nil || s.store == nil || s.objects == nil {
		return OpenedPrivateMedia{}, ErrScheduledMediaNotFound
	}
	media, err := s.store.readReadyPrivateMedia(ctx, ownerDID, mediaID)
	if err != nil {
		return OpenedPrivateMedia{}, err
	}
	body, err := s.objects.Open(ctx, media.ObjectKey)
	if err != nil {
		return OpenedPrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	defer body.Close()
	payload, err := io.ReadAll(io.LimitReader(body, media.SizeBytes+1))
	if err != nil || int64(len(payload)) != media.SizeBytes || sha256.Sum256(payload) != media.SHA256 {
		return OpenedPrivateMedia{}, ErrMediaInvalid
	}
	return OpenedPrivateMedia{
		MIMEType: media.MIMEType, SizeBytes: media.SizeBytes,
		Body: io.NopCloser(bytes.NewReader(payload)),
	}, nil
}

func (s *PrivateMediaService) Delete(
	ctx context.Context,
	ownerDID syntax.DID,
	mediaID uuid.UUID,
	now time.Time,
	ownerGeneration ...int64,
) error {
	if s == nil || s.store == nil || s.lifecycle == nil || ownerDID == "" ||
		mediaID == uuid.Nil || now.IsZero() || len(ownerGeneration) != 1 || ownerGeneration[0] <= 0 {
		return ErrScheduledMediaNotFound
	}
	media, err := s.store.readPrivateMediaIdentity(ctx, ownerDID, mediaID, ownerGeneration[0])
	if errors.Is(err, ErrScheduledMediaNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	err = s.lifecycle.WithActiveEffects(ctx, []ownerlifecycle.ExpectedOwner{{
		Owner: ownerDID, Generation: ownerGeneration[0],
	}}, func(effectCtx context.Context) (effectErr error) {
		lockCtx, cancelLock := context.WithTimeout(effectCtx, s.putTimeout)
		objectFence, err := acquirePrivateObjectFence(lockCtx, s.store.pool, media.ObjectKey)
		cancelLock()
		if err != nil {
			return err
		}
		defer func() { effectErr = errors.Join(effectErr, objectFence.Release()) }()
		return s.store.deleteUnclaimedPrivateMedia(
			effectCtx, ownerDID, mediaID, ownerGeneration[0], now.UTC(),
		)
	})
	if errors.Is(err, ErrScheduledMediaNotFound) {
		return nil
	}
	return err
}

func (s *Store) reservePrivateMedia(
	ctx context.Context,
	id uuid.UUID,
	ownerDID syntax.DID,
	ownerGeneration int64,
	uploadAttemptID uuid.UUID,
	objectKey string,
	mimeType string,
	sizeBytes int64,
	digest [32]byte,
	remoteDeadline time.Time,
	settlementNotBefore *time.Time,
	now time.Time,
) (PrivateMedia, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	defer tx.Rollback(ctx)
	var cleanupState string
	err = tx.QueryRow(ctx, selectCleanupJobStateForObjectSQL, objectKey).Scan(&cleanupState)
	switch {
	case err == nil:
		return PrivateMedia{}, ErrScheduledMediaOutcomeUnknown
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	var media PrivateMedia
	var scheduleID *uuid.UUID
	var storedDigest []byte
	var blobCID *string
	err = tx.QueryRow(ctx, selectPrivateMediaByIDSQL, id).Scan(
		&media.OwnerDID,
		&media.OwnerGeneration,
		&media.UploadGeneration,
		&media.UploadAttemptID,
		&media.ObjectKey,
		&media.State,
		&scheduleID,
		&media.MIMEType,
		&media.SizeBytes,
		&storedDigest,
		&blobCID,
		&media.ExpiresAt,
		&media.RemoteDeadline,
		&media.SettlementNotBefore,
	)
	switch {
	case err == nil:
		media.ID = id
		media.ExpiresAt = media.ExpiresAt.UTC()
		media.RemoteDeadline = media.RemoteDeadline.UTC()
		if media.SettlementNotBefore != nil {
			boundary := media.SettlementNotBefore.UTC()
			media.SettlementNotBefore = &boundary
		}
		copy(media.SHA256[:], storedDigest)
		if blobCID != nil {
			media.BlobCID = syntax.CID(*blobCID)
		}
		if media.OwnerDID != ownerDID {
			return PrivateMedia{}, ErrScheduledMediaNotFound
		}
		if media.OwnerGeneration != ownerGeneration {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingOwnerGeneration)
		}
		if media.UploadGeneration != ownerGeneration {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingUploadGeneration)
		}
		if media.UploadAttemptID != uploadAttemptID {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingUploadAttempt)
		}
		if media.ObjectKey != objectKey {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingObjectKey)
		}
		if media.MIMEType != mimeType {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingMIMEType)
		}
		if media.SizeBytes != sizeBytes {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingSize)
		}
		if media.SHA256 != digest {
			return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictExistingDigest)
		}
		if media.State == "uploading" {
			var outcome string
			if err := tx.QueryRow(
				ctx, selectObjectAttemptOutcomeSQL, uploadAttemptID,
			).Scan(&outcome); err != nil {
				return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
			}
			if outcome != "prepared" {
				return PrivateMedia{}, ErrScheduledMediaOutcomeUnknown
			}
		}
		return media, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	if _, err := tx.Exec(
		ctx,
		insertPrivateObjectAttemptSQL,
		uploadAttemptID,
		id,
		ownerDID,
		ownerGeneration,
		objectKey,
		digest[:],
		now,
		remoteDeadline,
		settlementNotBefore,
	); err != nil {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictObjectAttemptReservation)
	}
	var (
		storedOwner      syntax.DID
		storedGeneration int64
		storedMediaID    uuid.UUID
		storedObjectKey  string
		attemptDigest    []byte
		storedOutcome    string
		storedDeadline   time.Time
	)
	if err := tx.QueryRow(
		ctx, selectPrivateObjectAttemptIdentitySQL, uploadAttemptID,
	).Scan(
		&storedOwner, &storedGeneration, &storedMediaID, &storedObjectKey,
		&attemptDigest, &storedOutcome, &storedDeadline,
	); err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	if storedOwner != ownerDID {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictAttemptOwner)
	}
	if storedGeneration != ownerGeneration {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictAttemptGeneration)
	}
	if storedMediaID != id {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictAttemptMedia)
	}
	if storedObjectKey != objectKey {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictAttemptObjectKey)
	}
	if !bytes.Equal(attemptDigest, digest[:]) {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictAttemptDigest)
	}
	if storedOutcome != "prepared" {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictAttemptOutcome)
	}
	// PostgreSQL is authoritative for the attempt deadline. Comparing it with a
	// newly computed time is invalid because timestamptz has microsecond precision.
	remoteDeadline = storedDeadline.UTC()
	expiresAt := UnclaimedMediaExpiresAt(now)
	if _, err := tx.Exec(
		ctx,
		insertUploadingPrivateMediaSQL,
		id,
		ownerDID,
		ownerGeneration,
		uploadAttemptID,
		objectKey,
		mimeType,
		sizeBytes,
		digest[:],
		expiresAt,
		now,
	); err != nil {
		return PrivateMedia{}, NewScheduledMediaConflict(MediaConflictMediaReservation)
	}
	if err := tx.Commit(ctx); err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	return PrivateMedia{
		ID: id, OwnerDID: ownerDID, OwnerGeneration: ownerGeneration,
		UploadGeneration: ownerGeneration, UploadAttemptID: uploadAttemptID,
		ObjectKey: objectKey, State: "uploading",
		MIMEType: mimeType, SizeBytes: sizeBytes, SHA256: digest, ExpiresAt: expiresAt,
		RemoteDeadline: remoteDeadline, SettlementNotBefore: settlementNotBefore,
	}, nil
}

func (s *Store) markPrivateMediaDispatched(
	ctx context.Context,
	media PrivateMedia,
	now time.Time,
) error {
	result, err := s.pool.Exec(
		ctx,
		markPrivateObjectAttemptDispatchedSQL,
		media.UploadAttemptID,
		now,
	)
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	if result.RowsAffected() != 1 {
		return ErrScheduledMediaOutcomeUnknown
	}
	return nil
}

func (s *Store) markPrivateMediaReady(
	ctx context.Context,
	media PrivateMedia,
	blobCID syntax.CID,
	now time.Time,
) (PrivateMedia, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(
		ctx, markPrivateObjectAttemptAcceptedSQL, media.UploadAttemptID, now,
	)
	if err != nil || result.RowsAffected() != 1 {
		return PrivateMedia{}, ErrScheduledMediaOutcomeUnknown
	}
	result, err = tx.Exec(
		ctx,
		markPrivateMediaReadySQL,
		media.OwnerDID,
		media.OwnerGeneration,
		media.ID,
		media.UploadAttemptID,
		blobCID,
		now,
	)
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	if result.RowsAffected() != 1 {
		return PrivateMedia{}, ErrScheduledMediaNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	media.State = "ready"
	media.BlobCID = blobCID
	return media, nil
}

func (s *Store) movePrivateMediaToCleanup(
	ctx context.Context,
	media PrivateMedia,
	now time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(
		ctx,
		deleteUploadingPrivateMediaSQL,
		media.OwnerDID,
		media.OwnerGeneration,
		media.ID,
		media.UploadAttemptID,
	)
	if err != nil || result.RowsAffected() != 1 {
		return ErrPrivateObjectStoreUnavailable
	}
	if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), media.ObjectKey, now); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	return nil
}

func (s *Store) readReadyPrivateMedia(
	ctx context.Context,
	ownerDID syntax.DID,
	mediaID uuid.UUID,
) (PrivateMedia, error) {
	var media PrivateMedia
	var digest []byte
	var blobCID string
	err := s.pool.QueryRow(ctx, selectReadyPrivateMediaSQL, ownerDID, mediaID).Scan(
		&media.ObjectKey,
		&media.OwnerGeneration,
		&media.UploadGeneration,
		&media.UploadAttemptID,
		&media.MIMEType,
		&media.SizeBytes,
		&digest,
		&blobCID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PrivateMedia{}, ErrScheduledMediaNotFound
	}
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	media.ID = mediaID
	media.OwnerDID = ownerDID
	media.State = "ready"
	copy(media.SHA256[:], digest)
	media.BlobCID = syntax.CID(blobCID)
	return media, nil
}

func (s *Store) readPrivateMediaIdentity(
	ctx context.Context,
	ownerDID syntax.DID,
	mediaID uuid.UUID,
	ownerGeneration int64,
) (PrivateMedia, error) {
	var media PrivateMedia
	err := s.pool.QueryRow(
		ctx, selectPrivateMediaIdentitySQL, ownerDID, mediaID, ownerGeneration,
	).Scan(
		&media.ObjectKey,
		&media.UploadGeneration,
		&media.UploadAttemptID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return PrivateMedia{}, ErrScheduledMediaNotFound
	}
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	media.ID = mediaID
	media.OwnerDID = ownerDID
	media.OwnerGeneration = ownerGeneration
	return media, nil
}

func (s *Store) deleteUnclaimedPrivateMedia(
	ctx context.Context,
	ownerDID syntax.DID,
	mediaID uuid.UUID,
	ownerGeneration int64,
	now time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	defer tx.Rollback(ctx)
	var actualOwner syntax.DID
	var actualOwnerGeneration int64
	var uploadGeneration int64
	var uploadAttemptID uuid.UUID
	var objectKey string
	var state string
	var scheduleID *uuid.UUID
	var mimeType string
	var sizeBytes int64
	var digest []byte
	var blobCID *string
	var expiresAt time.Time
	var remoteDeadline time.Time
	var settlementNotBefore *time.Time
	err = tx.QueryRow(ctx, selectPrivateMediaByIDSQL, mediaID).Scan(
		&actualOwner,
		&actualOwnerGeneration,
		&uploadGeneration,
		&uploadAttemptID,
		&objectKey,
		&state,
		&scheduleID,
		&mimeType,
		&sizeBytes,
		&digest,
		&blobCID,
		&expiresAt,
		&remoteDeadline,
		&settlementNotBefore,
	)
	if errors.Is(err, pgx.ErrNoRows) || actualOwner != ownerDID ||
		actualOwnerGeneration != ownerGeneration {
		return ErrScheduledMediaNotFound
	}
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	if scheduleID != nil {
		return ErrScheduledMediaConflict
	}
	result, err := tx.Exec(
		ctx, deleteUnclaimedPrivateMediaSQL, ownerDID, ownerGeneration, mediaID,
	)
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	if result.RowsAffected() != 1 {
		return ErrScheduledMediaNotFound
	}
	if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), objectKey, now); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	return nil
}
