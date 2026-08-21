package instagram

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

const MaxModerationRestorationBatch = 100

// ModerationRestorationRelay atomically promotes durable moderation
// restoration intents into the existing Instagram reconciliation queue. It
// performs database work only; downstream reconciliation owns all eligibility
// evaluation and the approved private-suggestion outcome.
type ModerationRestorationRelay struct {
	pool               *pgxpool.Pool
	lifecycles         *ownerlifecycle.Store
	now                func() time.Time
	newID              func() uuid.UUID
	beforeStatusUpdate func(string) error
}

func NewModerationRestorationRelay(
	pool *pgxpool.Pool,
	lifecycles *ownerlifecycle.Store,
	now func() time.Time,
) (*ModerationRestorationRelay, error) {
	if pool == nil || lifecycles == nil {
		return nil, errors.New("moderation restoration relay requires database and lifecycle stores")
	}
	if now == nil {
		now = time.Now
	}
	return &ModerationRestorationRelay{
		pool: pool, lifecycles: lifecycles, now: now, newID: uuid.New,
	}, nil
}

func (relay *ModerationRestorationRelay) PromotePending(
	ctx context.Context,
	limit int,
) (int, error) {
	if relay == nil || relay.pool == nil || relay.lifecycles == nil || relay.now == nil || relay.newID == nil {
		return 0, errors.New("moderation restoration relay is not initialized")
	}
	if limit < 1 || limit > MaxModerationRestorationBatch {
		return 0, errors.New("moderation restoration relay batch limit is invalid")
	}

	rows, err := relay.pool.Query(ctx, `
		SELECT
			outbox.moderation_output_id,
			output.source_did,
			output.subject_did
		FROM moderation_restoration_outbox AS outbox
		JOIN moderation_outputs AS output ON output.id=outbox.moderation_output_id
		WHERE outbox.status='pending'
		ORDER BY outbox.created_at,outbox.moderation_output_id
		LIMIT $1
	`, limit)
	if err != nil {
		return 0, fmt.Errorf("select moderation restoration intents: %w", err)
	}
	type intent struct {
		outputID string
		source   syntax.DID
		subject  syntax.DID
	}
	intents := make([]intent, 0, limit)
	for rows.Next() {
		var selected intent
		if err := rows.Scan(&selected.outputID, &selected.source, &selected.subject); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan moderation restoration intent: %w", err)
		}
		intents = append(intents, selected)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate moderation restoration intents: %w", err)
	}
	rows.Close()

	processed := 0
	for _, selected := range intents {
		changed, err := relay.promoteOne(ctx, selected.outputID, selected.source, selected.subject)
		if err != nil {
			return processed, err
		}
		if changed {
			processed++
		}
	}
	return processed, nil
}

func (relay *ModerationRestorationRelay) promoteOne(
	ctx context.Context,
	outputID string,
	source syntax.DID,
	subject syntax.DID,
) (bool, error) {
	changed := false
	err := relay.lifecycles.WithOwnerStates(
		ctx,
		[]syntax.DID{source, subject},
		func(fenceCtx context.Context, tx pgx.Tx, states map[syntax.DID]ownerlifecycle.Lifecycle) error {
			var (
				target     syntax.DID
				storedSrc  syntax.DID
				storedSubj syntax.DID
				hasLink    bool
				owner      syntax.DID
				linkID     uuid.UUID
			)
			err := tx.QueryRow(fenceCtx, `
				SELECT outbox.target_did,output.source_did,output.subject_did,
				       link.id IS NOT NULL,COALESCE(link.owner_did,''),
				       COALESCE(link.id,'00000000-0000-0000-0000-000000000000'::uuid)
				FROM moderation_restoration_outbox AS outbox
				JOIN moderation_outputs AS output ON output.id=outbox.moderation_output_id
				LEFT JOIN LATERAL (
					SELECT id,owner_did
					FROM instagram_account_links
					WHERE owner_did=outbox.target_did
					  AND state='active' AND discoverable AND NOT conflict_pending
					ORDER BY updated_at DESC,id DESC LIMIT 1
				) AS link ON true
				WHERE outbox.moderation_output_id=$1 AND outbox.status='pending'
				FOR UPDATE OF outbox SKIP LOCKED
			`, outputID).Scan(&target, &storedSrc, &storedSubj, &hasLink, &owner, &linkID)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("lock moderation restoration intent: %w", err)
			}
			if target != subject || storedSubj != subject || storedSrc != source {
				return errors.New("moderation restoration participant identity changed")
			}

			processedAt := relay.now().UTC()
			status := ""
			targetState, targetKnown := states[subject]
			sourceState, sourceKnown := states[source]
			switch {
			case targetKnown && targetState.State == ownerlifecycle.StateTerminal:
				status = "cancelled_target_terminal"
			case !targetKnown || targetState.State != ownerlifecycle.StateActive:
				status = "no_work"
			case sourceKnown && sourceState.State != ownerlifecycle.StateActive:
				status = "no_work"
			case !hasLink:
				status = "no_work"
			}
			if status != "" {
				if err := relay.callBeforeStatusUpdate(outputID); err != nil {
					return err
				}
				command, err := tx.Exec(fenceCtx, `
					UPDATE moderation_restoration_outbox
					SET status=$1,reconciliation_job_id=NULL,processed_at=$2
					WHERE moderation_output_id=$3 AND status='pending'
				`, status, processedAt, outputID)
				if err != nil {
					return fmt.Errorf("settle moderation restoration without work: %w", err)
				}
				changed = command.RowsAffected() == 1
				return nil
			}

			jobID := relay.newID()
			if jobID == uuid.Nil || owner == "" || linkID == uuid.Nil {
				return errors.New("moderation restoration promotion identity is invalid")
			}
			reason := "moderationCleared:" + outputID
			if _, err := tx.Exec(fenceCtx, `
				INSERT INTO instagram_reconciliation_jobs (
					id,owner_did,link_id,reason,status,next_attempt_at,created_at,updated_at
				) VALUES ($1,$2,$3,$4,'queued',$5,$5,$5)
			`, jobID, owner, linkID, reason, processedAt); err != nil {
				return fmt.Errorf("insert moderation reconciliation job: %w", err)
			}
			if err := relay.callBeforeStatusUpdate(outputID); err != nil {
				return err
			}
			command, err := tx.Exec(fenceCtx, `
				UPDATE moderation_restoration_outbox
				SET status='queued',reconciliation_job_id=$1,processed_at=$2
				WHERE moderation_output_id=$3 AND status='pending'
			`, jobID, processedAt, outputID)
			if err != nil {
				return fmt.Errorf("mark moderation restoration queued: %w", err)
			}
			if command.RowsAffected() != 1 {
				return errors.New("moderation restoration queued transition lost ownership")
			}
			changed = true
			return nil
		},
	)
	return changed, err
}

func (relay *ModerationRestorationRelay) callBeforeStatusUpdate(outputID string) error {
	if relay.beforeStatusUpdate == nil {
		return nil
	}
	if err := relay.beforeStatusUpdate(outputID); err != nil {
		return fmt.Errorf("moderation restoration status hook: %w", err)
	}
	return nil
}
