package moderation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/notifications"
)

type ExpiryProcessor struct {
	store         *Store
	notifications NotificationIntentWriter
	now           func() time.Time
	observer      ExpiryObserver
}

type ExpiryObserver interface {
	ObserveModerationExpiry(time.Time, time.Time) bool
}

type ExpiryWorker struct {
	processor *ExpiryProcessor
	batchSize int
}

func NewExpiryWorker(processor *ExpiryProcessor, batchSize int) *ExpiryWorker {
	return &ExpiryWorker{processor: processor, batchSize: batchSize}
}

func (w *ExpiryWorker) ProcessBatch(ctx context.Context) (int, error) {
	if w == nil || w.processor == nil || w.batchSize < 1 {
		return 0, ErrInvalidCommand
	}
	return w.processor.ProcessDue(ctx, w.batchSize)
}

func NewExpiryProcessor(store *Store, notifications NotificationIntentWriter, now func() time.Time, observers ...ExpiryObserver) *ExpiryProcessor {
	if now == nil {
		now = time.Now
	}
	processor := &ExpiryProcessor{store: store, notifications: notifications, now: now}
	if len(observers) > 0 {
		processor.observer = observers[0]
	}
	return processor
}

func (p *ExpiryProcessor) ProcessDue(ctx context.Context, limit int) (int, error) {
	if p == nil || p.store == nil || p.store.pool == nil || limit < 1 {
		return 0, ErrInvalidCommand
	}
	now := p.now().UTC().Truncate(time.Microsecond)
	rows, err := p.store.pool.Query(ctx, `SELECT DISTINCT owner_did FROM moderation_case_strikes
		WHERE due_at <= $1 AND expired_at IS NULL AND overturned_at IS NULL ORDER BY owner_did LIMIT $2`, now, limit)
	if err != nil {
		return 0, fmt.Errorf("list due moderation strike owners: %w", err)
	}
	var owners []syntax.DID
	for rows.Next() {
		var owner syntax.DID
		if err := rows.Scan(&owner); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan due moderation strike owner: %w", err)
		}
		owners = append(owners, owner)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate due moderation strike owners: %w", err)
	}
	rows.Close()

	processed := 0
	for _, owner := range owners {
		count, err := p.processOwner(ctx, owner, now, limit-processed)
		if err != nil {
			return processed, err
		}
		processed += count
		if processed >= limit {
			break
		}
	}
	return processed, nil
}

func (p *ExpiryProcessor) processOwner(ctx context.Context, owner syntax.DID, now time.Time, limit int) (int, error) {
	tx, err := p.store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin strike expiry: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var before bool
	if err := tx.QueryRow(ctx, `SELECT effective_suspended FROM moderation_account_standings WHERE owner_did=$1 FOR UPDATE`, owner).Scan(&before); err != nil {
		return 0, fmt.Errorf("lock expiry standing: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT strike.case_id,strike.logical_effect_id,strike.due_at,c.revision
		FROM moderation_case_strikes strike JOIN moderation_cases c ON c.id=strike.case_id
		WHERE strike.owner_did=$1 AND strike.due_at <= $2 AND strike.expired_at IS NULL AND strike.overturned_at IS NULL
		ORDER BY strike.due_at,strike.case_id LIMIT $3 FOR UPDATE OF strike,c`, owner, now, limit)
	if err != nil {
		return 0, fmt.Errorf("lock due moderation strikes: %w", err)
	}
	type dueStrike struct {
		caseID, logicalID uuid.UUID
		dueAt             time.Time
		revision          int64
	}
	var due []dueStrike
	for rows.Next() {
		var strike dueStrike
		if err := rows.Scan(&strike.caseID, &strike.logicalID, &strike.dueAt, &strike.revision); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan due moderation strike: %w", err)
		}
		due = append(due, strike)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate due moderation strikes: %w", err)
	}
	rows.Close()
	if len(due) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return 0, err
		}
		return 0, nil
	}
	for _, strike := range due {
		if p.observer != nil {
			p.observer.ObserveModerationExpiry(strike.dueAt, now)
		}
		eventID := uuid.New()
		replayID := "strike-expiry:" + strike.caseID.String() + ":" + strike.dueAt.UTC().Format(time.RFC3339Nano)
		fingerprint := sha256.Sum256([]byte(replayID))
		_, err := tx.Exec(ctx, `INSERT INTO moderation_case_events(id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,expected_revision,result_revision,created_at)
			VALUES($1,$2,'strikeExpired','system','expiry',$3,$4,$5,$6,$7)`, eventID, strike.caseID, replayID, fingerprint[:], strike.revision, strike.revision+1, now)
		if err != nil {
			return 0, fmt.Errorf("insert strike expiry event: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_effect_events(id,case_id,case_event_id,logical_effect_id,effect_type,action,created_at)
			VALUES($1,$2,$3,$4,'strike','expire',$5)`, uuid.New(), strike.caseID, eventID, strike.logicalID, now); err != nil {
			return 0, fmt.Errorf("insert strike expiry effect: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type='strike'`, strike.caseID); err != nil {
			return 0, fmt.Errorf("remove expired active strike: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE moderation_case_strikes SET expired_at=$2,updated_at=$2 WHERE case_id=$1`, strike.caseID, now); err != nil {
			return 0, fmt.Errorf("project expired strike: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE moderation_cases SET revision=$2,updated_at=$3 WHERE id=$1`, strike.caseID, strike.revision+1, now); err != nil {
			return 0, fmt.Errorf("revise expired strike case: %w", err)
		}
		if err := updateStandingTx(ctx, tx, owner, now); err != nil {
			return 0, err
		}
		var after bool
		if err := tx.QueryRow(ctx, `SELECT effective_suspended FROM moderation_account_standings WHERE owner_did=$1`, owner).Scan(&after); err != nil {
			return 0, fmt.Errorf("read standing after expiry: %w", err)
		}
		if err := insertNotificationTx(ctx, tx, p.notifications, notifications.ModerationEvent{
			Kind: notifications.ModerationStrikeExpired, EnforcementChanged: before != after,
		}, owner, strike.caseID, eventID, now); err != nil {
			return 0, err
		}
		before = after
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit strike expiry: %w", err)
	}
	if observer, ok := p.observer.(OperationObserver); ok {
		for _, strike := range due {
			reference, _ := FormatCaseReference(strike.caseID)
			replayID := "strike-expiry:" + strike.caseID.String() + ":" + strike.dueAt.UTC().Format(time.RFC3339Nano)
			requestHash := sha256.Sum256([]byte("expiry\x00" + replayID))
			observer.ObserveModerationCommand(ctx, "strike_expiry", "success", "system", hex.EncodeToString(requestHash[:]), reference, now)
		}
	}
	return len(due), nil
}
