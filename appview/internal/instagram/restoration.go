package instagram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EligibilityRestorationReason string

const (
	RestorationModerationCleared EligibilityRestorationReason = "moderationCleared"
	RestorationRelationshipSafe  EligibilityRestorationReason = "relationshipSafetyRestored"
)

func (r EligibilityRestorationReason) Valid() bool {
	return r == RestorationModerationCleared || r == RestorationRelationshipSafe
}

type EligibilityRestorationEnqueuer interface {
	EnqueueEligibilityRestoration(context.Context, syntax.DID, syntax.DID, EligibilityRestorationReason) error
}

// ReconciliationTrigger is the narrow hook future moderation, block, and mute
// owners can inject without depending on worker internals.
type ReconciliationTrigger struct {
	pool  *pgxpool.Pool
	now   func() time.Time
	newID func() uuid.UUID
}

func NewReconciliationTrigger(pool *pgxpool.Pool, now func() time.Time) *ReconciliationTrigger {
	if now == nil {
		now = time.Now
	}
	return &ReconciliationTrigger{pool: pool, now: now, newID: uuid.New}
}

func (t *ReconciliationTrigger) EnqueueEligibilityRestoration(ctx context.Context, importer, target syntax.DID, reason EligibilityRestorationReason) error {
	if t == nil || t.pool == nil || importer == "" || target == "" || importer == target || !reason.Valid() {
		return errors.New("invalid Instagram eligibility restoration")
	}
	now := t.now().UTC()
	_, err := t.pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs (
			id,owner_did,target_did,reason,status,next_attempt_at,created_at,updated_at
		) VALUES ($1,$2,$3,$4,'queued',$5,$5,$5)
	`, t.newID(), importer, target, reason, now)
	return err
}

func (t *ReconciliationTrigger) EnqueueRelationshipSafetyRestoration(
	ctx context.Context,
	left, right syntax.DID,
) error {
	if t == nil || t.pool == nil || left == "" || right == "" || left == right {
		return errors.New("invalid Instagram relationship restoration")
	}
	now := t.now().UTC()
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, pair := range [][2]syntax.DID{{left, right}, {right, left}} {
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_reconciliation_jobs (
				id,owner_did,target_did,reason,status,next_attempt_at,
				created_at,updated_at
			) VALUES ($1,$2,$3,$4,'queued',$5,$5,$5)
		`, t.newID(), pair[0], pair[1], RestorationRelationshipSafe, now); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (t *ReconciliationTrigger) EnqueueModerationRestoration(
	ctx context.Context,
	target syntax.DID,
) error {
	if t == nil || t.pool == nil || target == "" {
		return errors.New("invalid Instagram moderation restoration")
	}
	now := t.now().UTC()
	_, err := t.pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs (
			id,owner_did,link_id,reason,status,next_attempt_at,
			created_at,updated_at
		)
		SELECT $1,link.owner_did,link.id,$2,'queued',$3,$3,$3
		FROM instagram_account_links link
		WHERE link.owner_did=$4
		  AND link.state='active'
		  AND link.discoverable
		  AND NOT link.conflict_pending
		ORDER BY link.updated_at DESC,link.id DESC
		LIMIT 1
	`, t.newID(), RestorationModerationCleared, now, target)
	return err
}

func (t *ReconciliationTrigger) EnqueueExpiredModerationRestorations(
	ctx context.Context,
	limit int,
) (int, error) {
	if t == nil || t.pool == nil || limit < 1 || limit > MaxRetentionBatch {
		return 0, errors.New("invalid expired Instagram moderation restoration")
	}
	now := t.now().UTC()
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT output.id,output.subject_did,link.id
		FROM moderation_outputs output
		JOIN LATERAL (
			SELECT id
			FROM instagram_account_links
			WHERE owner_did=output.subject_did
			  AND state='active'
			  AND discoverable
			  AND NOT conflict_pending
			ORDER BY updated_at DESC,id DESC
			LIMIT 1
		) link ON true
		WHERE output.subject_type='account'
		  AND output.action='apply'
		  AND output.value IN ('hide','takedown')
		  AND output.expires_at IS NOT NULL
		  AND output.expires_at <= $1
		  AND NOT EXISTS (
			SELECT 1
			FROM moderation_outputs negated
			WHERE negated.source_did=output.source_did
			  AND negated.subject_type=output.subject_type
			  AND negated.subject_did=output.subject_did
			  AND negated.value=output.value
			  AND negated.action='negate'
			  AND negated.indexed_at > output.indexed_at
		  )
		  AND NOT EXISTS (
			SELECT 1
			FROM instagram_reconciliation_jobs job
			WHERE job.reason='moderationExpired:' || output.id::text
		  )
		ORDER BY output.expires_at,output.id
		LIMIT $2
		FOR UPDATE OF output SKIP LOCKED
	`, now, limit)
	if err != nil {
		return 0, err
	}
	type expiredRestoration struct {
		outputID uuid.UUID
		target   syntax.DID
		linkID   uuid.UUID
	}
	var restorations []expiredRestoration
	for rows.Next() {
		var restoration expiredRestoration
		if err := rows.Scan(
			&restoration.outputID,
			&restoration.target,
			&restoration.linkID,
		); err != nil {
			rows.Close()
			return 0, err
		}
		restorations = append(restorations, restoration)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	for _, restoration := range restorations {
		reason := fmt.Sprintf("moderationExpired:%s", restoration.outputID)
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_reconciliation_jobs (
				id,owner_did,link_id,reason,status,next_attempt_at,
				created_at,updated_at
			) VALUES ($1,$2,$3,$4,'queued',$5,$5,$5)
		`, t.newID(), restoration.target, restoration.linkID, reason, now); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(restorations), nil
}

var _ EligibilityRestorationEnqueuer = (*ReconciliationTrigger)(nil)
