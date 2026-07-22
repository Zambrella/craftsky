package instagram

import (
	"context"
	"errors"
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

var _ EligibilityRestorationEnqueuer = (*ReconciliationTrigger)(nil)
