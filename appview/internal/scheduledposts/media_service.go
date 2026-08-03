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
)

var (
	ErrScheduledMediaNotFound = errors.New("scheduled media not found")
	ErrScheduledMediaConflict = errors.New("scheduled media conflict")
)

type PutPrivateMediaParams struct {
	ID       uuid.UUID
	OwnerDID syntax.DID
	MIMEType string
	Bytes    []byte
	Now      time.Time
}

type PrivateMedia struct {
	ID        uuid.UUID
	OwnerDID  syntax.DID
	ObjectKey string
	State     string
	MIMEType  string
	SizeBytes int64
	SHA256    [32]byte
	BlobCID   syntax.CID
	ExpiresAt time.Time
}

type OpenedPrivateMedia struct {
	MIMEType  string
	SizeBytes int64
	Body      io.ReadCloser
}

type PrivateMediaService struct {
	store   *Store
	objects PrivateObjectStore
}

func NewPrivateMediaService(store *Store, objects PrivateObjectStore) *PrivateMediaService {
	return &PrivateMediaService{store: store, objects: objects}
}

func (s *PrivateMediaService) Put(
	ctx context.Context,
	params PutPrivateMediaParams,
) (PrivateMedia, error) {
	if s == nil || s.store == nil || s.objects == nil || params.ID == uuid.Nil ||
		params.OwnerDID == "" || params.MIMEType == "" || len(params.Bytes) == 0 || params.Now.IsZero() {
		return PrivateMedia{}, ErrScheduledMediaConflict
	}
	digest := sha256.Sum256(params.Bytes)
	multihashValue, err := multihash.Sum(params.Bytes, multihash.SHA2_256, -1)
	if err != nil {
		return PrivateMedia{}, ErrScheduledMediaConflict
	}
	predictedCID := syntax.CID(cid.NewCidV1(cid.Raw, multihashValue).String())
	media, err := s.store.reservePrivateMedia(
		ctx,
		params.ID,
		params.OwnerDID,
		params.MIMEType,
		int64(len(params.Bytes)),
		digest,
		params.Now.UTC(),
	)
	if err != nil {
		return PrivateMedia{}, err
	}
	if media.State == "ready" {
		if media.BlobCID != predictedCID {
			return PrivateMedia{}, ErrScheduledMediaConflict
		}
		return media, nil
	}
	if err := s.objects.Put(
		ctx,
		media.ObjectKey,
		bytes.NewReader(params.Bytes),
		int64(len(params.Bytes)),
		params.MIMEType,
	); err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	return s.store.markPrivateMediaReady(ctx, media, predictedCID, params.Now.UTC())
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
) error {
	if s == nil || s.store == nil || ownerDID == "" || mediaID == uuid.Nil || now.IsZero() {
		return ErrScheduledMediaNotFound
	}
	err := s.store.deleteUnclaimedPrivateMedia(ctx, ownerDID, mediaID, now.UTC())
	if errors.Is(err, ErrScheduledMediaNotFound) {
		return nil
	}
	return err
}

func (s *Store) reservePrivateMedia(
	ctx context.Context,
	id uuid.UUID,
	ownerDID syntax.DID,
	mimeType string,
	sizeBytes int64,
	digest [32]byte,
	now time.Time,
) (PrivateMedia, error) {
	objectKey, err := NewObjectKey(id)
	if err != nil {
		return PrivateMedia{}, ErrScheduledMediaConflict
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, lockCleanupObjectForTransactionSQL, objectKey); err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	var cleanupState string
	err = tx.QueryRow(ctx, selectCleanupJobStateForObjectSQL, objectKey).Scan(&cleanupState)
	switch {
	case err == nil && cleanupState == "deleting":
		return PrivateMedia{}, ErrScheduledMediaConflict
	case err == nil && cleanupState == "pending":
		result, deleteErr := tx.Exec(ctx, deletePendingCleanupJobForObjectSQL, objectKey)
		if deleteErr != nil || result.RowsAffected() != 1 {
			return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
		}
	case err == nil:
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
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
		&media.ObjectKey,
		&media.State,
		&scheduleID,
		&media.MIMEType,
		&media.SizeBytes,
		&storedDigest,
		&blobCID,
		&media.ExpiresAt,
	)
	switch {
	case err == nil:
		media.ID = id
		media.ExpiresAt = media.ExpiresAt.UTC()
		copy(media.SHA256[:], storedDigest)
		if blobCID != nil {
			media.BlobCID = syntax.CID(*blobCID)
		}
		if media.OwnerDID != ownerDID {
			return PrivateMedia{}, ErrScheduledMediaNotFound
		}
		if media.MIMEType != mimeType || media.SizeBytes != sizeBytes || media.SHA256 != digest {
			return PrivateMedia{}, ErrScheduledMediaConflict
		}
		return media, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	expiresAt := UnclaimedMediaExpiresAt(now)
	if _, err := tx.Exec(
		ctx,
		insertUploadingPrivateMediaSQL,
		id,
		ownerDID,
		objectKey,
		mimeType,
		sizeBytes,
		digest[:],
		expiresAt,
		now,
	); err != nil {
		return PrivateMedia{}, ErrScheduledMediaConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	return PrivateMedia{
		ID: id, OwnerDID: ownerDID, ObjectKey: objectKey, State: "uploading",
		MIMEType: mimeType, SizeBytes: sizeBytes, SHA256: digest, ExpiresAt: expiresAt,
	}, nil
}

func (s *Store) markPrivateMediaReady(
	ctx context.Context,
	media PrivateMedia,
	blobCID syntax.CID,
	now time.Time,
) (PrivateMedia, error) {
	result, err := s.pool.Exec(
		ctx,
		markPrivateMediaReadySQL,
		media.OwnerDID,
		media.ID,
		blobCID,
		now,
	)
	if err != nil {
		return PrivateMedia{}, ErrPrivateObjectStoreUnavailable
	}
	if result.RowsAffected() != 1 {
		return PrivateMedia{}, ErrScheduledMediaNotFound
	}
	media.State = "ready"
	media.BlobCID = blobCID
	return media, nil
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

func (s *Store) deleteUnclaimedPrivateMedia(
	ctx context.Context,
	ownerDID syntax.DID,
	mediaID uuid.UUID,
	now time.Time,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	defer tx.Rollback(ctx)
	var actualOwner syntax.DID
	var objectKey string
	var state string
	var scheduleID *uuid.UUID
	var mimeType string
	var sizeBytes int64
	var digest []byte
	var blobCID *string
	var expiresAt time.Time
	err = tx.QueryRow(ctx, selectPrivateMediaByIDSQL, mediaID).Scan(
		&actualOwner,
		&objectKey,
		&state,
		&scheduleID,
		&mimeType,
		&sizeBytes,
		&digest,
		&blobCID,
		&expiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) || actualOwner != ownerDID {
		return ErrScheduledMediaNotFound
	}
	if err != nil {
		return ErrPrivateObjectStoreUnavailable
	}
	if scheduleID != nil {
		return ErrScheduledMediaConflict
	}
	result, err := tx.Exec(ctx, deleteUnclaimedPrivateMediaSQL, ownerDID, mediaID)
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
