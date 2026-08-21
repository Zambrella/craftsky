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

var ErrPublicationConflict = errors.New("scheduled publication result conflicts with tombstone")

type FinalizePublicationParams struct {
	Claim          PublishingClaim
	PublicationURI syntax.ATURI
	PublicationCID syntax.CID
	PublishedAt    time.Time
}

type FinalizePublicationResult struct {
	PublicationURI   syntax.ATURI
	PublicationCID   syntax.CID
	PublishedAt      time.Time
	ExpiresAt        time.Time
	AlreadyFinalized bool
}

func (s *Store) FinalizePublication(
	ctx context.Context,
	params FinalizePublicationParams,
) (FinalizePublicationResult, error) {
	if s == nil || s.pool == nil || params.Claim.ID == uuid.Nil ||
		params.Claim.OwnerDID == "" || params.Claim.OwnerGeneration <= 0 ||
		params.Claim.LeaseToken == uuid.Nil ||
		params.Claim.PayloadVersion < 1 || params.PublicationURI == "" ||
		params.PublicationCID == "" || params.PublishedAt.IsZero() {
		return FinalizePublicationResult{}, errors.New("invalid scheduled publication finalization")
	}
	expectedURI := syntax.ATURI(fmt.Sprintf(
		"at://%s/social.craftsky.feed.post/%s",
		params.Claim.OwnerDID,
		params.Claim.Rkey,
	))
	if params.PublicationURI != expectedURI {
		return FinalizePublicationResult{}, ErrPublicationConflict
	}
	params.PublishedAt = params.PublishedAt.UTC()
	tx, err := s.publicationDB(ctx).Begin(ctx)
	if err != nil {
		return FinalizePublicationResult{}, err
	}
	defer tx.Rollback(ctx)
	var activeGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, params.Claim.OwnerDID).
		Scan(&activeGeneration); err != nil || activeGeneration != params.Claim.OwnerGeneration {
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			return FinalizePublicationResult{}, ErrWorkerLeaseLost
		}
		return FinalizePublicationResult{}, fmt.Errorf("lock finalization owner lifecycle: %w", err)
	}

	var existing FinalizePublicationResult
	var existingURI string
	var existingCID string
	err = tx.QueryRow(
		ctx,
		selectPublicationTombstoneSQL,
		params.Claim.ID,
		params.Claim.OwnerDID,
		params.Claim.OwnerGeneration,
	).Scan(&existingURI, &existingCID, &existing.PublishedAt, &existing.ExpiresAt)
	switch {
	case err == nil:
		existing.PublicationURI = syntax.ATURI(existingURI)
		existing.PublicationCID = syntax.CID(existingCID)
		existing.AlreadyFinalized = true
		if existing.PublicationURI != params.PublicationURI ||
			existing.PublicationCID != params.PublicationCID {
			return FinalizePublicationResult{}, ErrPublicationConflict
		}
		return existing, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return FinalizePublicationResult{}, fmt.Errorf("read publication tombstone: %w", err)
	}

	var operationID uuid.UUID
	var requestHash []byte
	var status Status
	var leaseToken uuid.UUID
	var payloadVersion int64
	err = tx.QueryRow(
		ctx,
		selectScheduledPostForFinalizationSQL,
		params.Claim.OwnerDID,
		params.Claim.ID,
		params.Claim.OwnerGeneration,
	).Scan(&operationID, &requestHash, &status, &leaseToken, &payloadVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return FinalizePublicationResult{}, ErrWorkerLeaseLost
	}
	if err != nil {
		return FinalizePublicationResult{}, fmt.Errorf("read scheduled post for finalization: %w", err)
	}
	if status != StatusPublishing || leaseToken != params.Claim.LeaseToken ||
		payloadVersion != params.Claim.PayloadVersion {
		return FinalizePublicationResult{}, ErrWorkerLeaseLost
	}

	rows, err := tx.Query(
		ctx,
		selectScheduledMediaForFinalizationSQL,
		params.Claim.OwnerDID,
		params.Claim.ID,
		params.Claim.OwnerGeneration,
	)
	if err != nil {
		return FinalizePublicationResult{}, fmt.Errorf("select publication media cleanup: %w", err)
	}
	objectKeys := make([]string, 0)
	for rows.Next() {
		var objectKey string
		if err := rows.Scan(&objectKey); err != nil {
			rows.Close()
			return FinalizePublicationResult{}, fmt.Errorf("scan publication media cleanup: %w", err)
		}
		objectKeys = append(objectKeys, objectKey)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return FinalizePublicationResult{}, err
	}
	rows.Close()

	expiresAt := PublicationTombstoneExpiresAt(params.PublishedAt)
	if _, err := tx.Exec(
		ctx,
		insertPublicationTombstoneSQL,
		params.Claim.ID,
		params.Claim.OwnerDID,
		params.Claim.OwnerGeneration,
		operationID,
		requestHash,
		params.PublicationURI,
		params.PublicationCID,
		params.PublishedAt,
		expiresAt,
	); err != nil {
		return FinalizePublicationResult{}, fmt.Errorf("insert publication tombstone: %w", err)
	}
	result, err := tx.Exec(
		ctx,
		deleteFinalizedScheduledPostSQL,
		params.Claim.OwnerDID,
		params.Claim.ID,
		params.Claim.OwnerGeneration,
		params.Claim.LeaseToken,
		params.Claim.PayloadVersion,
	)
	if err != nil {
		return FinalizePublicationResult{}, fmt.Errorf("delete finalized scheduled post: %w", err)
	}
	if result.RowsAffected() != 1 {
		return FinalizePublicationResult{}, ErrWorkerLeaseLost
	}
	for _, objectKey := range objectKeys {
		if _, err := tx.Exec(
			ctx,
			insertCleanupJobSQL,
			uuid.New(),
			objectKey,
			SuccessfulPublicationCleanupAt(params.PublishedAt),
		); err != nil {
			return FinalizePublicationResult{}, fmt.Errorf("enqueue publication cleanup: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return FinalizePublicationResult{}, err
	}
	return FinalizePublicationResult{
		PublicationURI: params.PublicationURI,
		PublicationCID: params.PublicationCID,
		PublishedAt:    params.PublishedAt,
		ExpiresAt:      expiresAt,
	}, nil
}
