package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

const DefaultPublicationLeaseDuration = time.Minute

type publicationMedia struct {
	ObjectKey string
	MIMEType  string
	SizeBytes int64
	SHA256    [32]byte
	BlobCID   syntax.CID
}

type publicationSnapshot struct {
	Payload      []byte
	FrozenRecord []byte
	AttemptCount int
	ScheduledAt  time.Time
	Media        []publicationMedia
}

func (s *Store) ClaimBatch(ctx context.Context, limit int, now time.Time) ([]WorkItem, error) {
	claims, err := s.ClaimDue(ctx, limit, now, DefaultPublicationLeaseDuration)
	if err != nil {
		return nil, err
	}
	items := make([]WorkItem, 0, len(claims))
	for _, claim := range claims {
		items = append(items, WorkItem{
			ID: claim.ID, OwnerDID: claim.OwnerDID, LeaseToken: claim.LeaseToken,
			PayloadVersion: claim.PayloadVersion, Rkey: claim.Rkey, CreatedAt: claim.CreatedAt,
		})
	}
	return items, nil
}

func workItemClaim(item WorkItem) PublishingClaim {
	return PublishingClaim{ID: item.ID, OwnerDID: item.OwnerDID, LeaseToken: item.LeaseToken,
		PayloadVersion: item.PayloadVersion, Rkey: item.Rkey, CreatedAt: item.CreatedAt}
}

func (s *Store) publicationSnapshot(ctx context.Context, claim PublishingClaim) (publicationSnapshot, error) {
	var snapshot publicationSnapshot
	err := s.pool.QueryRow(ctx, selectPublicationSnapshotSQL, claim.OwnerDID, claim.ID,
		claim.LeaseToken, claim.PayloadVersion).Scan(
		&snapshot.Payload, &snapshot.FrozenRecord, &snapshot.AttemptCount, &snapshot.ScheduledAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return publicationSnapshot{}, ErrWorkerLeaseLost
	}
	if err != nil {
		return publicationSnapshot{}, fmt.Errorf("read publication snapshot: %w", err)
	}
	rows, err := s.pool.Query(ctx, selectPublicationMediaSQL, claim.OwnerDID, claim.ID)
	if err != nil {
		return publicationSnapshot{}, fmt.Errorf("read publication media: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var media publicationMedia
		var digest []byte
		if err := rows.Scan(
			&media.ObjectKey,
			&media.MIMEType,
			&media.SizeBytes,
			&digest,
			&media.BlobCID,
		); err != nil {
			return publicationSnapshot{}, err
		}
		copy(media.SHA256[:], digest)
		snapshot.Media = append(snapshot.Media, media)
	}
	return snapshot, rows.Err()
}

func (s *Store) failPublication(ctx context.Context, claim PublishingClaim, decision FailureDecision, now time.Time) (Status, error) {
	snapshot, err := s.publicationSnapshot(ctx, claim)
	if err != nil {
		return "", err
	}
	status := StatusRetrying
	nextAttempt, eligible := RetryAttemptAt(snapshot.ScheduledAt, snapshot.AttemptCount, 0)
	var needsAt any
	var needsExpires any
	if decision.Disposition == FailureNeedsAttention || !eligible || snapshot.AttemptCount >= 6 {
		status = StatusNeedsAttention
		nextAttempt = now.UTC()
		needsAt = now.UTC()
		needsExpires = NeedsAttentionExpiresAt(now.UTC())
	}
	result, err := s.pool.Exec(ctx, failScheduledPublicationSQL,
		claim.OwnerDID, claim.ID, claim.LeaseToken, claim.PayloadVersion,
		status, nextAttempt.UTC(), decision.SafeCode, now.UTC(), needsAt, needsExpires,
	)
	if err != nil {
		return "", fmt.Errorf("record publication failure: %w", err)
	}
	if result.RowsAffected() != 1 {
		return "", ErrWorkerLeaseLost
	}
	return status, nil
}
