package moderation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/notifications"
)

type SevereRestorationCommand struct {
	CaseID           uuid.UUID
	EffectID         uuid.UUID
	ExpectedRevision int64
	SourceSystem     string
	ReplayID         string
	ActorID          string
	Rationale        string
}

func (s *Service) RestoreSevereSuspension(ctx context.Context, command SevereRestorationCommand) (CommandResult, error) {
	if s == nil || s.store == nil || s.store.pool == nil || command.CaseID == uuid.Nil ||
		command.EffectID == uuid.Nil || command.ExpectedRevision < 0 ||
		strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" ||
		strings.TrimSpace(command.ActorID) == "" || strings.TrimSpace(command.Rationale) == "" {
		return CommandResult{}, ErrInvalidEffectChange
	}
	fingerprint, err := severeRestorationFingerprint(command)
	if err != nil {
		return CommandResult{}, err
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return CommandResult{}, fmt.Errorf("begin severe suspension restoration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var owner syntax.DID
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_cases WHERE id=$1`, command.CaseID).Scan(&owner); err != nil {
		return CommandResult{}, fmt.Errorf("read restoration owner: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_account_standings(owner_did,updated_at) VALUES($1,$2) ON CONFLICT(owner_did) DO NOTHING`, owner, now); err != nil {
		return CommandResult{}, fmt.Errorf("ensure restoration standing: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_account_standings WHERE owner_did=$1 FOR UPDATE`, owner).Scan(&owner); err != nil {
		return CommandResult{}, fmt.Errorf("lock restoration standing: %w", err)
	}
	caseRow, err := lockCase(ctx, tx, command.CaseID)
	if err != nil {
		return CommandResult{}, err
	}
	if replay, found, err := replayResult(ctx, tx, command.CaseID, command.SourceSystem, command.ReplayID, fingerprint); err != nil {
		return CommandResult{}, err
	} else if found {
		if err := tx.Commit(ctx); err != nil {
			return CommandResult{}, fmt.Errorf("commit restoration replay: %w", err)
		}
		return replay, nil
	}
	if caseRow.revision != command.ExpectedRevision {
		return CommandResult{}, ErrRevisionConflict
	}
	if caseRow.state != "resolved" {
		return CommandResult{}, ErrInvalidEffectChange
	}
	var activeEffectID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT logical_effect_id FROM moderation_active_case_effects
		WHERE case_id=$1 AND effect_type='severeSuspension' FOR UPDATE`, command.CaseID).Scan(&activeEffectID)
	if errors.Is(err, pgx.ErrNoRows) || err == nil && activeEffectID != command.EffectID {
		return CommandResult{}, ErrInvalidEffectChange
	}
	if err != nil {
		return CommandResult{}, fmt.Errorf("lock severe suspension effect: %w", err)
	}

	eventID := uuid.New()
	resultRevision := command.ExpectedRevision + 1
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_events(
		id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,
		expected_revision,result_revision,created_at
	) VALUES($1,$2,'severeRestored',$3,$4,$5,$6,$7,$8,$9)`, eventID, command.CaseID,
		command.ActorID, command.SourceSystem, command.ReplayID, fingerprint[:],
		command.ExpectedRevision, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("insert severe restoration event: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_effect_events(
		id,case_id,case_event_id,logical_effect_id,effect_type,action,rationale,created_at
	) VALUES($1,$2,$3,$4,'severeSuspension','restore',$5,$6)`, uuid.New(), command.CaseID,
		eventID, command.EffectID, command.Rationale, now); err != nil {
		return CommandResult{}, fmt.Errorf("insert severe restoration effect: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM moderation_active_case_effects
		WHERE case_id=$1 AND effect_type='severeSuspension' AND logical_effect_id=$2`, command.CaseID, command.EffectID); err != nil {
		return CommandResult{}, fmt.Errorf("remove severe suspension effect: %w", err)
	}
	if err := updateStandingTx(ctx, tx, caseRow.ownerDID, now); err != nil {
		return CommandResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE moderation_cases SET revision=$2,updated_at=$3 WHERE id=$1`, command.CaseID, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("revise restored moderation case: %w", err)
	}
	if err := s.insertNotificationTx(ctx, tx, notifications.ModerationEvent{
		Kind: notifications.ModerationSevereRestored, ConsequenceChanged: true,
	}, caseRow.ownerDID, command.CaseID, eventID, now); err != nil {
		return CommandResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommandResult{}, fmt.Errorf("commit severe suspension restoration: %w", err)
	}
	return CommandResult{CaseID: command.CaseID, EventID: eventID, Revision: resultRevision}, nil
}

func severeRestorationFingerprint(command SevereRestorationCommand) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(struct {
		CaseID           string `json:"caseId"`
		EffectID         string `json:"effectId"`
		ExpectedRevision int64  `json:"expectedRevision"`
		Rationale        string `json:"rationale"`
	}{command.CaseID.String(), command.EffectID.String(), command.ExpectedRevision, command.Rationale})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint severe suspension restoration: %w", err)
	}
	return sha256.Sum256(encoded), nil
}
