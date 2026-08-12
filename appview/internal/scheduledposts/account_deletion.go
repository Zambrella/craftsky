package scheduledposts

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

// AccountDeletion removes unpublished scheduled content inside the caller's
// actor-deletion transaction and leaves only content-free object cleanup work.
type AccountDeletion struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewAccountDeletion(pool *pgxpool.Pool, now func() time.Time) *AccountDeletion {
	return &AccountDeletion{pool: pool, now: now}
}

func (*AccountDeletion) Name() string { return "scheduledPosts" }

// Purge removes scheduled state for an accepted deletion job and records only
// the content-free object-cleanup job IDs needed by the terminal gate.
func (deletion *AccountDeletion) Purge(ctx context.Context, ownerDID syntax.DID) error {
	if deletion == nil || deletion.pool == nil || ownerDID == "" {
		return errors.New("scheduled account deletion is unavailable")
	}
	return pgx.BeginFunc(ctx, deletion.pool, func(tx pgx.Tx) error {
		return deletion.hardDeleteByActor(ctx, tx, ownerDID)
	})
}

func (deletion *AccountDeletion) HandleIdentityDeleted(
	ctx context.Context,
	ownerDID syntax.DID,
) error {
	if deletion == nil || deletion.pool == nil {
		return errors.New("scheduled account deletion is unavailable")
	}
	return pgx.BeginFunc(ctx, deletion.pool, func(tx pgx.Tx) error {
		return deletion.hardDeleteByActor(ctx, tx, ownerDID)
	})
}

func (deletion *AccountDeletion) HardDeleteByActor(
	ctx context.Context,
	tx pgx.Tx,
	ownerDID syntax.DID,
) error {
	return deletion.hardDeleteByActor(ctx, tx, ownerDID)
}

func (deletion *AccountDeletion) hardDeleteByActor(
	ctx context.Context,
	tx pgx.Tx,
	ownerDID syntax.DID,
) error {
	if deletion == nil || deletion.pool == nil || deletion.now == nil || tx == nil || ownerDID == "" {
		return errors.New("scheduled account deletion is unavailable")
	}
	now := deletion.now().UTC()
	if now.IsZero() {
		return errors.New("scheduled account deletion time is unavailable")
	}

	// Create serializes capacity against this same profile row. Holding it until
	// the outer actor-deletion transaction commits prevents a new schedule from
	// appearing after the private-state snapshot.
	var lockedOwner syntax.DID
	err := tx.QueryRow(ctx, lockScheduledPostOwnerSQL, ownerDID).Scan(&lockedOwner)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lock scheduled account owner: %w", err)
	}

	rows, err := tx.Query(ctx, selectScheduledPostIDsForAccountDeletionSQL, ownerDID)
	if err != nil {
		return fmt.Errorf("select scheduled account work: %w", err)
	}
	scheduleIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan scheduled account work: %w", err)
		}
		scheduleIDs = append(scheduleIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read scheduled account work: %w", err)
	}
	rows.Close()
	for _, id := range scheduleIDs {
		if _, err := tx.Exec(ctx, lockScheduleEffectForTransactionSQL, ownerDID, id); err != nil {
			return fmt.Errorf("lock scheduled account publication: %w", err)
		}
	}

	rows, err = tx.Query(ctx, selectScheduledMediaForAccountDeletionSQL, ownerDID)
	if err != nil {
		return fmt.Errorf("select scheduled account media: %w", err)
	}
	objectKeys := make([]string, 0)
	for rows.Next() {
		var objectKey string
		if err := rows.Scan(&objectKey); err != nil {
			rows.Close()
			return fmt.Errorf("scan scheduled account media: %w", err)
		}
		objectKeys = append(objectKeys, objectKey)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read scheduled account media: %w", err)
	}
	rows.Close()

	if _, err := tx.Exec(ctx, deleteScheduledMediaForAccountSQL, ownerDID); err != nil {
		return fmt.Errorf("delete scheduled account media: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteScheduledPostsForAccountSQL, ownerDID); err != nil {
		return fmt.Errorf("delete scheduled account posts: %w", err)
	}
	if _, err := tx.Exec(ctx, deleteScheduledTombstonesForAccountSQL, ownerDID); err != nil {
		return fmt.Errorf("delete scheduled account tombstones: %w", err)
	}
	for _, objectKey := range objectKeys {
		if _, err := tx.Exec(ctx, `
			INSERT INTO scheduled_post_cleanup_jobs (
				id, object_key, next_attempt_at, created_at, updated_at
			) VALUES ($1, $2, $3, $3, $3)
			ON CONFLICT (object_key) DO UPDATE SET object_key=EXCLUDED.object_key
		`, uuid.New(), objectKey, now); err != nil {
			return fmt.Errorf("enqueue scheduled account media: %w", err)
		}
	}
	return nil
}
