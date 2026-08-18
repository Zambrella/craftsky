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

	"social.craftsky/appview/internal/ownerlifecycle"
)

// AccountDeletion removes unpublished scheduled content inside the caller's
// actor-deletion transaction and leaves only content-free object cleanup work.
type AccountDeletion struct {
	pool       *pgxpool.Pool
	now        func() time.Time
	ownerFence ownerExclusiveFence
}

type ownerExclusiveFence interface {
	WithExclusive(context.Context, []syntax.DID, func(context.Context) error) error
}

const maximumDepartureAttachedMedia = MaximumActivePosts * 4

func NewAccountDeletion(
	pool *pgxpool.Pool,
	now func() time.Time,
	ownerFence ...ownerExclusiveFence,
) *AccountDeletion {
	deletion := &AccountDeletion{pool: pool, now: now}
	if len(ownerFence) == 1 {
		deletion.ownerFence = ownerFence[0]
	}
	return deletion
}

func (*AccountDeletion) Name() string { return "scheduledPosts" }

// DepartureParticipant cancels all generation-bound scheduled work inside the
// lifecycle transition transaction. Compose it on every active-to-non-active
// transition; the caller already holds the exclusive owner fence and the
// lifecycle row lock, so no claimed generation can outlive the commit.
func (deletion *AccountDeletion) DepartureParticipant() ownerlifecycle.TransitionParticipant {
	return func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if before.State != ownerlifecycle.StateActive || after.State == ownerlifecycle.StateActive {
			return nil
		}
		if before.Owner != after.Owner || before.Owner == "" {
			return errors.New("scheduled lifecycle participant owner mismatch")
		}
		return deletion.cancelScheduledGeneration(ctx, tx, before.Owner, before.Generation)
	}
}

// cancelScheduledGeneration performs only fixed-cardinality transition work.
// The product limit permits at most three schedules with four attached media
// objects each. Unclaimed generation-bound media stays inaccessible and is
// settled by the existing expiry or accepted-deletion worker.
func (deletion *AccountDeletion) cancelScheduledGeneration(
	ctx context.Context,
	tx pgx.Tx,
	ownerDID syntax.DID,
	ownerGeneration int64,
) error {
	if deletion == nil || deletion.now == nil || tx == nil || ownerDID == "" || ownerGeneration <= 0 {
		return errors.New("scheduled departure cancellation is unavailable")
	}
	now := deletion.now().UTC()
	if now.IsZero() {
		return errors.New("scheduled departure cancellation time is unavailable")
	}
	rows, err := tx.Query(
		ctx,
		selectScheduledPostIDsForDepartureSQL,
		ownerDID,
		ownerGeneration,
		MaximumActivePosts+1,
	)
	if err != nil {
		return fmt.Errorf("select departing scheduled work: %w", err)
	}
	scheduleIDs := make([]uuid.UUID, 0, MaximumActivePosts)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan departing scheduled work: %w", err)
		}
		scheduleIDs = append(scheduleIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read departing scheduled work: %w", err)
	}
	rows.Close()
	if len(scheduleIDs) > MaximumActivePosts {
		return errors.New("scheduled departure capacity invariant violated")
	}
	for _, id := range scheduleIDs {
		if _, err := tx.Exec(ctx, lockScheduleEffectForTransactionSQL, ownerDID, id); err != nil {
			return fmt.Errorf("lock departing scheduled publication: %w", err)
		}
	}

	type attachedObject struct {
		id        uuid.UUID
		objectKey string
	}
	attached := make([]attachedObject, 0, maximumDepartureAttachedMedia)
	if len(scheduleIDs) > 0 {
		rows, err = tx.Query(
			ctx,
			selectScheduledMediaForDepartureSQL,
			ownerDID,
			ownerGeneration,
			scheduleIDs,
			maximumDepartureAttachedMedia+1,
		)
		if err != nil {
			return fmt.Errorf("select departing scheduled media: %w", err)
		}
		for rows.Next() {
			var object attachedObject
			if err := rows.Scan(&object.id, &object.objectKey); err != nil {
				rows.Close()
				return fmt.Errorf("scan departing scheduled media: %w", err)
			}
			attached = append(attached, object)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("read departing scheduled media: %w", err)
		}
		rows.Close()
	}
	if len(attached) > maximumDepartureAttachedMedia {
		return errors.New("scheduled departure media invariant violated")
	}
	for _, object := range attached {
		if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), object.objectKey, now); err != nil {
			return fmt.Errorf("enqueue departing scheduled media: %w", err)
		}
	}
	mediaResult, err := tx.Exec(
		ctx, deleteScheduledMediaForDepartureSQL, ownerDID, ownerGeneration,
	)
	if err != nil {
		return fmt.Errorf("delete departing scheduled media: %w", err)
	}
	if mediaResult.RowsAffected() != int64(len(attached)) {
		return errors.New("scheduled departure media set changed")
	}
	postResult, err := tx.Exec(
		ctx, deleteScheduledPostsForDepartureSQL, ownerDID, ownerGeneration,
	)
	if err != nil {
		return fmt.Errorf("delete departing scheduled posts: %w", err)
	}
	if postResult.RowsAffected() != int64(len(scheduleIDs)) {
		return errors.New("scheduled departure work set changed")
	}
	return nil
}

// Purge removes scheduled state for an accepted deletion job and records only
// the content-free object-cleanup job IDs needed by the terminal gate.
func (deletion *AccountDeletion) Purge(ctx context.Context, ownerDID syntax.DID) error {
	if deletion == nil || deletion.pool == nil || deletion.ownerFence == nil || ownerDID == "" {
		return errors.New("scheduled account deletion is unavailable")
	}
	return deletion.ownerFence.WithExclusive(ctx, []syntax.DID{ownerDID}, func(fenceCtx context.Context) error {
		return pgx.BeginFunc(fenceCtx, deletion.pool, func(tx pgx.Tx) error {
			return deletion.hardDeleteByActor(fenceCtx, tx, ownerDID)
		})
	})
}

// PurgeAccepted inventories every outcome-uncertain object Put for the exact
// accepted deletion operation and lifecycle generation before removing the
// private media rows. The resulting safety rows are server-internal,
// non-secret, exact-key authority; they are not user-visible progress state.
func (deletion *AccountDeletion) PurgeAccepted(
	ctx context.Context,
	operationID uuid.UUID,
	ownerDID syntax.DID,
	ownerGeneration int64,
) error {
	if deletion == nil || deletion.pool == nil || deletion.ownerFence == nil ||
		operationID == uuid.Nil || ownerDID == "" || ownerGeneration <= 0 {
		return errors.New("scheduled accepted account deletion is unavailable")
	}
	return deletion.ownerFence.WithExclusive(ctx, []syntax.DID{ownerDID}, func(fenceCtx context.Context) error {
		return pgx.BeginFunc(fenceCtx, deletion.pool, func(tx pgx.Tx) error {
			if err := deletion.adoptUncertainObjectSafety(
				fenceCtx, tx, operationID, ownerDID, ownerGeneration,
			); err != nil {
				return err
			}
			return deletion.hardDeleteByActor(fenceCtx, tx, ownerDID)
		})
	})
}

func (deletion *AccountDeletion) adoptUncertainObjectSafety(
	ctx context.Context,
	tx pgx.Tx,
	operationID uuid.UUID,
	ownerDID syntax.DID,
	ownerGeneration int64,
) error {
	rows, err := tx.Query(ctx, `
		SELECT upload_attempt_id,object_key,upload_generation,
		       remote_deadline,settlement_not_before
		FROM scheduled_post_object_attempts
		WHERE owner_did=$1 AND owner_generation=$2
		  AND remote_outcome='dispatched'
		ORDER BY upload_attempt_id
		FOR UPDATE
	`, ownerDID, ownerGeneration)
	if err != nil {
		return fmt.Errorf("select uncertain scheduled objects: %w", err)
	}
	type uncertainObject struct {
		attemptID           uuid.UUID
		objectKey           string
		uploadGeneration    int64
		remoteDeadline      time.Time
		settlementNotBefore *time.Time
	}
	objects := make([]uncertainObject, 0)
	for rows.Next() {
		var object uncertainObject
		if err := rows.Scan(
			&object.attemptID,
			&object.objectKey,
			&object.uploadGeneration,
			&object.remoteDeadline,
			&object.settlementNotBefore,
		); err != nil {
			rows.Close()
			return fmt.Errorf("scan uncertain scheduled object: %w", err)
		}
		objects = append(objects, object)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read uncertain scheduled objects: %w", err)
	}
	rows.Close()

	now := deletion.now().UTC()
	for _, object := range objects {
		if _, err := tx.Exec(ctx, `
			INSERT INTO account_deletion_safety_tombstones(
				id,operation_id,owner_did,owner_generation,kind,exact_key,
				upload_generation,source_attempt_id,state,remote_deadline,
				settlement_not_before,next_attempt_at,created_at,updated_at
			) VALUES (
				$1,$2,$3,$4,'scheduled_object',$5,$6,$7,'pending',$8,$9,$10,$10,$10
			)
			ON CONFLICT (operation_id,kind,exact_key,upload_generation)
			DO NOTHING
		`, uuid.New(), operationID, ownerDID, ownerGeneration, object.objectKey,
			object.uploadGeneration, object.attemptID.String(), object.remoteDeadline,
			object.settlementNotBefore, now); err != nil {
			return fmt.Errorf("adopt uncertain scheduled object: %w", err)
		}
	}
	return nil
}

func (deletion *AccountDeletion) HandleIdentityDeleted(
	ctx context.Context,
	ownerDID syntax.DID,
) error {
	if deletion == nil || deletion.pool == nil || deletion.ownerFence == nil {
		return errors.New("scheduled account deletion is unavailable")
	}
	return deletion.ownerFence.WithExclusive(ctx, []syntax.DID{ownerDID}, func(fenceCtx context.Context) error {
		return pgx.BeginFunc(fenceCtx, deletion.pool, func(tx pgx.Tx) error {
			return deletion.hardDeleteByActor(fenceCtx, tx, ownerDID)
		})
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
		if _, err := tx.Exec(ctx, insertCleanupJobSQL, uuid.New(), objectKey, now); err != nil {
			return fmt.Errorf("enqueue scheduled account media: %w", err)
		}
	}
	return nil
}
