package moderation

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/notifications"
)

var ErrInvalidEffectChange = errors.New("invalid moderation effect change")

type EffectChangeCommand struct {
	CaseID           uuid.UUID
	ExpectedRevision int64
	SourceSystem     string
	ReplayID         string
	ActorID          string
	SourceDID        syntax.DID
	Negate           []EffectType
	Apply            []EffectType
	Rationale        string
}

func (s *Service) ChangeEffects(ctx context.Context, command EffectChangeCommand) (result CommandResult, resultErr error) {
	defer func() {
		s.observeCommand(ctx, "effect_change", command.CaseID, command.SourceSystem, command.ReplayID, result, resultErr)
	}()
	if s == nil || s.store == nil || s.store.pool == nil || s.visibility == nil ||
		command.CaseID == uuid.Nil || command.ExpectedRevision < 0 || command.SourceDID == "" ||
		strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" ||
		strings.TrimSpace(command.ActorID) == "" || strings.TrimSpace(command.Rationale) == "" ||
		(len(command.Negate) == 0 && len(command.Apply) == 0) || !validEffectChangeTypes(command.Negate, command.Apply) {
		return CommandResult{}, ErrInvalidEffectChange
	}
	fingerprint, err := effectChangeFingerprint(command)
	if err != nil {
		return CommandResult{}, err
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return CommandResult{}, fmt.Errorf("begin moderation effect change: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var owner syntax.DID
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_cases WHERE id=$1`, command.CaseID).Scan(&owner); err != nil {
		return CommandResult{}, fmt.Errorf("read effect-change owner: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_account_standings(owner_did,updated_at) VALUES($1,$2) ON CONFLICT(owner_did) DO NOTHING`, owner, now); err != nil {
		return CommandResult{}, fmt.Errorf("ensure effect-change standing: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_account_standings WHERE owner_did=$1 FOR UPDATE`, owner).Scan(&owner); err != nil {
		return CommandResult{}, fmt.Errorf("lock effect-change standing: %w", err)
	}
	caseRow, err := lockCase(ctx, tx, command.CaseID)
	if err != nil {
		return CommandResult{}, err
	}
	if replay, found, err := replayResult(ctx, tx, command.CaseID, command.SourceSystem, command.ReplayID, fingerprint); err != nil {
		return CommandResult{}, err
	} else if found {
		if err := tx.Commit(ctx); err != nil {
			return CommandResult{}, fmt.Errorf("commit effect-change replay: %w", err)
		}
		return replay, nil
	}
	if caseRow.revision != command.ExpectedRevision {
		return CommandResult{}, ErrRevisionConflict
	}
	if caseRow.state != "resolved" {
		return CommandResult{}, ErrInvalidEffectChange
	}

	eventID := uuid.New()
	resultRevision := command.ExpectedRevision + 1
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_events(
		id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,
		expected_revision,result_revision,created_at
	) VALUES($1,$2,'effectsChanged',$3,$4,$5,$6,$7,$8,$9)`, eventID, command.CaseID,
		command.ActorID, command.SourceSystem, command.ReplayID, fingerprint[:],
		command.ExpectedRevision, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("insert effect-change event: %w", err)
	}
	for _, effect := range command.Negate {
		if err := s.negateEffectTx(ctx, tx, caseRow, command, eventID, effect, now); err != nil {
			return CommandResult{}, err
		}
	}
	for _, effect := range command.Apply {
		if err := s.reapplyEffectTx(ctx, tx, caseRow, command, eventID, effect, now); err != nil {
			return CommandResult{}, err
		}
	}
	if err := updateStandingTx(ctx, tx, owner, now); err != nil {
		return CommandResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE moderation_cases SET revision=$2,updated_at=$3 WHERE id=$1`, command.CaseID, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("revise moderation case effects: %w", err)
	}
	if err := s.insertNotificationTx(ctx, tx, notifications.ModerationEvent{
		Kind: notifications.ModerationEffectsChanged, ConsequenceChanged: true,
	}, owner, command.CaseID, eventID, now); err != nil {
		return CommandResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return CommandResult{}, fmt.Errorf("commit moderation effect change: %w", err)
	}
	return CommandResult{CaseID: command.CaseID, EventID: eventID, Revision: resultRevision}, nil
}

func (s *Service) negateEffectTx(ctx context.Context, tx pgx.Tx, caseRow lockedCase, command EffectChangeCommand, eventID uuid.UUID, effect EffectType, now time.Time) error {
	var logicalID uuid.UUID
	err := tx.QueryRow(ctx, `SELECT logical_effect_id FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type=$2 FOR UPDATE`, caseRow.id, effect).Scan(&logicalID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidEffectChange
	}
	if err != nil {
		return fmt.Errorf("lock active moderation effect: %w", err)
	}
	var outputID any
	if isVisibilityEffect(effect) {
		id, err := s.visibility.InsertOutputTx(ctx, tx, VisibilityOutputWrite{
			SourceDID: command.SourceDID, SubjectType: caseRow.subjectType, SubjectDID: caseRow.subjectDID,
			SubjectCollection: caseRow.subjectCollection, SubjectRkey: caseRow.subjectRkey, SubjectURI: caseRow.subjectURI,
			Value: visibilityValue(effect), Action: string(EffectNegate), InternalReason: command.Rationale, CreatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("negate moderation visibility output: %w", err)
		}
		outputID = id
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_effect_events(id,case_id,case_event_id,logical_effect_id,effect_type,action,moderation_output_id,rationale,created_at)
		VALUES($1,$2,$3,$4,$5,'negate',$6,$7,$8)`, uuid.New(), caseRow.id, eventID, logicalID, effect, outputID, command.Rationale, now); err != nil {
		return fmt.Errorf("insert moderation effect negation: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type=$2`, caseRow.id, effect); err != nil {
		return fmt.Errorf("remove active moderation effect: %w", err)
	}
	if effect == EffectStrike {
		if _, err := tx.Exec(ctx, `UPDATE moderation_case_strikes SET overturned_at=$2,expired_at=NULL,updated_at=$2 WHERE case_id=$1`, caseRow.id, now); err != nil {
			return fmt.Errorf("overturn moderation strike: %w", err)
		}
	}
	return nil
}

func (s *Service) reapplyEffectTx(ctx context.Context, tx pgx.Tx, caseRow lockedCase, command EffectChangeCommand, eventID uuid.UUID, effect EffectType, now time.Time) error {
	var active bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type=$2)`, caseRow.id, effect).Scan(&active); err != nil {
		return fmt.Errorf("check active moderation effect: %w", err)
	}
	if active {
		return ErrInvalidEffectChange
	}
	var logicalID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT logical_effect_id FROM moderation_effect_events WHERE case_id=$1 AND effect_type=$2 ORDER BY created_at DESC,id DESC LIMIT 1`, caseRow.id, effect).Scan(&logicalID); errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidEffectChange
	} else if err != nil {
		return fmt.Errorf("read logical moderation effect: %w", err)
	}
	var outputID any
	if isVisibilityEffect(effect) {
		id, err := s.visibility.InsertOutputTx(ctx, tx, VisibilityOutputWrite{
			SourceDID: command.SourceDID, SubjectType: caseRow.subjectType, SubjectDID: caseRow.subjectDID,
			SubjectCollection: caseRow.subjectCollection, SubjectRkey: caseRow.subjectRkey, SubjectURI: caseRow.subjectURI,
			Value: visibilityValue(effect), Action: string(EffectApply), InternalReason: command.Rationale, CreatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("reapply moderation visibility output: %w", err)
		}
		outputID = id
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_effect_events(id,case_id,case_event_id,logical_effect_id,effect_type,action,moderation_output_id,rationale,created_at)
		VALUES($1,$2,$3,$4,$5,'apply',$6,$7,$8)`, uuid.New(), caseRow.id, eventID, logicalID, effect, outputID, command.Rationale, now); err != nil {
		return fmt.Errorf("insert moderation effect reapplication: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_active_case_effects(case_id,effect_type,logical_effect_id,applied_event_id,moderation_output_id,applied_at)
		VALUES($1,$2,$3,$4,$5,$6)`, caseRow.id, effect, logicalID, eventID, outputID, now); err != nil {
		return fmt.Errorf("project reapplied moderation effect: %w", err)
	}
	if effect == EffectStrike {
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_strikes(case_id,logical_effect_id,owner_did,issued_at,due_at,updated_at)
			VALUES($1,$2,$3,$4,$5,$4) ON CONFLICT(case_id) DO UPDATE SET issued_at=EXCLUDED.issued_at,due_at=EXCLUDED.due_at,expired_at=NULL,overturned_at=NULL,updated_at=EXCLUDED.updated_at`,
			caseRow.id, logicalID, caseRow.ownerDID, now, StrikeDeadline(now)); err != nil {
			return fmt.Errorf("reapply moderation strike: %w", err)
		}
	}
	return nil
}

func validEffectChangeTypes(groups ...[]EffectType) bool {
	seen := map[EffectType]bool{}
	for _, group := range groups {
		for _, effect := range group {
			if !validEffect(effect) || seen[effect] {
				return false
			}
			seen[effect] = true
		}
	}
	return true
}

func effectChangeFingerprint(command EffectChangeCommand) ([sha256.Size]byte, error) {
	negate := append([]EffectType(nil), command.Negate...)
	apply := append([]EffectType(nil), command.Apply...)
	sort.Slice(negate, func(i, j int) bool { return negate[i] < negate[j] })
	sort.Slice(apply, func(i, j int) bool { return apply[i] < apply[j] })
	encoded, err := json.Marshal(struct {
		CaseID           string       `json:"caseId"`
		ExpectedRevision int64        `json:"expectedRevision"`
		SourceDID        syntax.DID   `json:"sourceDid"`
		Negate           []EffectType `json:"negate"`
		Apply            []EffectType `json:"apply"`
		Rationale        string       `json:"rationale"`
	}{command.CaseID.String(), command.ExpectedRevision, command.SourceDID, negate, apply, command.Rationale})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint moderation effect change: %w", err)
	}
	return sha256.Sum256(encoded), nil
}
