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

var ErrInvalidAppealTransition = errors.New("invalid moderation appeal transition")

type AppealStatus string

const (
	AppealStatusNone    AppealStatus = ""
	AppealStatusPending AppealStatus = "pending"
	AppealStatusUpheld  AppealStatus = "upheld"
	AppealStatusChanged AppealStatus = "changed"
)

func ValidateAppealConfirmation(ownerVisible bool, current AppealStatus) error {
	if !ownerVisible || current != AppealStatusNone {
		return ErrInvalidAppealTransition
	}
	return nil
}

func ValidateAppealResolution(current, outcome AppealStatus) error {
	if current != AppealStatusPending || (outcome != AppealStatusUpheld && outcome != AppealStatusChanged) {
		return ErrInvalidAppealTransition
	}
	return nil
}

type AppealCorrespondence struct {
	CaseReference   string
	SourceSystem    string
	ReplayID        string
	SenderReference string
	ReceivedAt      time.Time
}

type CorrespondenceResult struct {
	ID       uuid.UUID
	CaseID   uuid.UUID
	Replayed bool
}

func (s *Store) RecordAppealCorrespondence(ctx context.Context, input AppealCorrespondence) (CorrespondenceResult, error) {
	if s == nil || s.pool == nil || strings.TrimSpace(input.SourceSystem) == "" ||
		strings.TrimSpace(input.ReplayID) == "" || input.ReceivedAt.IsZero() {
		return CorrespondenceResult{}, ErrInvalidCommand
	}
	reference, err := ParseCaseReference(input.CaseReference)
	if err != nil {
		return CorrespondenceResult{}, err
	}
	result := CorrespondenceResult{ID: uuid.New(), CaseID: reference.UUID()}
	var senderHash any
	if input.SenderReference != "" {
		hash := sha256.Sum256([]byte(input.SenderReference))
		senderHash = hash[:]
	}
	command, err := s.pool.Exec(ctx, `INSERT INTO moderation_appeal_correspondence(
		id,case_id,source_system,replay_id,sender_reference_hash,received_at,created_at
	) VALUES($1,$2,$3,$4,$5,$6,$6) ON CONFLICT(source_system,replay_id) DO NOTHING`,
		result.ID, result.CaseID, input.SourceSystem, input.ReplayID, senderHash, input.ReceivedAt.UTC())
	if err != nil {
		return CorrespondenceResult{}, fmt.Errorf("record appeal correspondence: %w", err)
	}
	if command.RowsAffected() == 1 {
		return result, nil
	}
	if err := s.pool.QueryRow(ctx, `SELECT id,case_id FROM moderation_appeal_correspondence WHERE source_system=$1 AND replay_id=$2`, input.SourceSystem, input.ReplayID).Scan(&result.ID, &result.CaseID); err != nil {
		return CorrespondenceResult{}, fmt.Errorf("read appeal correspondence replay: %w", err)
	}
	if result.CaseID != reference.UUID() {
		return CorrespondenceResult{}, ErrReplayConflict
	}
	result.Replayed = true
	return result, nil
}

type AppealCommand struct {
	CaseID           uuid.UUID
	ExpectedRevision int64
	SourceSystem     string
	ReplayID         string
	ActorID          string
	CorrespondenceID uuid.UUID
}

type AppealResolutionCommand struct {
	CaseID           uuid.UUID
	ExpectedRevision int64
	SourceSystem     string
	ReplayID         string
	ActorID          string
	SourceDID        syntax.DID
	Outcome          AppealStatus
	Negate           []EffectType
	Rationale        string
}

func (s *Service) ConfirmAppeal(ctx context.Context, command AppealCommand) (result CommandResult, resultErr error) {
	defer func() {
		s.observeCommand(ctx, "appeal_confirmation", command.CaseID, command.SourceSystem, command.ReplayID, result, resultErr)
	}()
	fingerprint, err := appealFingerprint("confirm", command.CaseID, command.ExpectedRevision, command.CorrespondenceID.String())
	if err != nil || command.CaseID == uuid.Nil || command.ExpectedRevision < 0 || strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" || strings.TrimSpace(command.ActorID) == "" {
		return CommandResult{}, ErrInvalidCommand
	}
	return s.applyAppealCommand(ctx, command.CaseID, command.ExpectedRevision, command.SourceSystem, command.ReplayID, command.ActorID, fingerprint, func(ctx context.Context, tx pgx.Tx, caseRow lockedCase, eventID uuid.UUID, now time.Time) error {
		var visible bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM moderation_decisions d JOIN moderation_effect_events e ON e.case_event_id=d.case_event_id WHERE d.case_id=$1 AND d.disposition='violation')`, caseRow.id).Scan(&visible); err != nil {
			return fmt.Errorf("read owner-visible decision: %w", err)
		}
		var status AppealStatus
		err := tx.QueryRow(ctx, `SELECT status FROM moderation_appeals WHERE case_id=$1`, caseRow.id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			status = AppealStatusNone
		} else if err != nil {
			return fmt.Errorf("read moderation appeal: %w", err)
		}
		if err := ValidateAppealConfirmation(visible, status); err != nil {
			return err
		}
		if command.CorrespondenceID != uuid.Nil {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM moderation_appeal_correspondence WHERE id=$1 AND case_id=$2)`, command.CorrespondenceID, caseRow.id).Scan(&exists); err != nil || !exists {
				return ErrInvalidAppealTransition
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO moderation_appeals(case_id,confirmed_event_id,status,confirmed_at) VALUES($1,$2,'pending',$3)`, caseRow.id, eventID, now)
		if err != nil {
			return fmt.Errorf("confirm moderation appeal: %w", err)
		}
		return nil
	})
}

func (s *Service) ResolveAppeal(ctx context.Context, command AppealResolutionCommand) (result CommandResult, resultErr error) {
	defer func() {
		s.observeCommand(ctx, "appeal_resolution", command.CaseID, command.SourceSystem, command.ReplayID, result, resultErr)
	}()
	if strings.TrimSpace(command.Rationale) == "" {
		return CommandResult{}, ErrInvalidAppealTransition
	}
	if command.Outcome == AppealStatusUpheld && len(command.Negate) != 0 ||
		command.Outcome == AppealStatusChanged && (command.SourceDID == "" || len(command.Negate) == 0 || !validEffectChangeTypes(command.Negate)) {
		return CommandResult{}, ErrInvalidAppealTransition
	}
	effects := append([]EffectType(nil), command.Negate...)
	sort.Slice(effects, func(i, j int) bool { return effects[i] < effects[j] })
	fingerprint, err := appealFingerprint("resolve", command.CaseID, command.ExpectedRevision, fmt.Sprintf("%s:%s:%v", command.Outcome, command.Rationale, effects))
	if err != nil || command.CaseID == uuid.Nil || command.ExpectedRevision < 0 || strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" || strings.TrimSpace(command.ActorID) == "" {
		return CommandResult{}, ErrInvalidCommand
	}
	return s.applyAppealCommand(ctx, command.CaseID, command.ExpectedRevision, command.SourceSystem, command.ReplayID, command.ActorID, fingerprint, func(ctx context.Context, tx pgx.Tx, caseRow lockedCase, eventID uuid.UUID, now time.Time) error {
		var current AppealStatus
		if err := tx.QueryRow(ctx, `SELECT status FROM moderation_appeals WHERE case_id=$1 FOR UPDATE`, caseRow.id).Scan(&current); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInvalidAppealTransition
			}
			return fmt.Errorf("lock moderation appeal: %w", err)
		}
		if err := ValidateAppealResolution(current, command.Outcome); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE moderation_appeals SET status=$2,resolved_event_id=$3,resolved_at=$4 WHERE case_id=$1`, caseRow.id, command.Outcome, eventID, now)
		if err != nil {
			return fmt.Errorf("resolve moderation appeal: %w", err)
		}
		if command.Outcome == AppealStatusChanged {
			effectCommand := EffectChangeCommand{SourceDID: command.SourceDID, Rationale: command.Rationale}
			for _, effect := range command.Negate {
				if err := s.negateEffectTx(ctx, tx, caseRow, effectCommand, eventID, effect, now); err != nil {
					return err
				}
			}
			if err := updateStandingTx(ctx, tx, caseRow.ownerDID, now); err != nil {
				return err
			}
		}
		return s.insertNotificationTx(ctx, tx, notifications.ModerationEvent{
			Kind: notifications.ModerationAppealResolved, ConsequenceChanged: command.Outcome == AppealStatusChanged,
		}, caseRow.ownerDID, caseRow.id, eventID, now)
	})
}

func (s *Service) applyAppealCommand(ctx context.Context, caseID uuid.UUID, expectedRevision int64, sourceSystem, replayID, actorID string, fingerprint [sha256.Size]byte, apply func(context.Context, pgx.Tx, lockedCase, uuid.UUID, time.Time) error) (CommandResult, error) {
	if s == nil || s.store == nil || s.store.pool == nil {
		return CommandResult{}, ErrInvalidCommand
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return CommandResult{}, fmt.Errorf("begin moderation appeal command: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var owner string
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_cases WHERE id=$1`, caseID).Scan(&owner); err != nil {
		return CommandResult{}, fmt.Errorf("read appeal case owner: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_account_standings(owner_did,updated_at) VALUES($1,$2) ON CONFLICT(owner_did) DO NOTHING`, owner, now); err != nil {
		return CommandResult{}, fmt.Errorf("ensure appeal standing: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_account_standings WHERE owner_did=$1 FOR UPDATE`, owner).Scan(&owner); err != nil {
		return CommandResult{}, fmt.Errorf("lock appeal standing: %w", err)
	}
	caseRow, err := lockCase(ctx, tx, caseID)
	if err != nil {
		return CommandResult{}, err
	}
	if replay, found, err := replayResult(ctx, tx, caseID, sourceSystem, replayID, fingerprint); err != nil {
		return CommandResult{}, err
	} else if found {
		if err := tx.Commit(ctx); err != nil {
			return CommandResult{}, fmt.Errorf("commit appeal replay: %w", err)
		}
		return replay, nil
	}
	if caseRow.revision != expectedRevision {
		return CommandResult{}, ErrRevisionConflict
	}
	if caseRow.state != "resolved" {
		return CommandResult{}, ErrInvalidAppealTransition
	}
	eventID := uuid.New()
	eventType := "appealConfirmed"
	// A pending appeal already exists only for resolution commands.
	var pending bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM moderation_appeals WHERE case_id=$1 AND status='pending')`, caseID).Scan(&pending); err != nil {
		return CommandResult{}, fmt.Errorf("read pending appeal: %w", err)
	}
	if pending {
		eventType = "appealResolved"
	}
	resultRevision := expectedRevision + 1
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_events(id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,expected_revision,result_revision,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, eventID, caseID, eventType, actorID, sourceSystem, replayID, fingerprint[:], expectedRevision, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("insert moderation appeal event: %w", err)
	}
	if err := apply(ctx, tx, caseRow, eventID, now); err != nil {
		return CommandResult{}, err
	}
	if eventType == "appealConfirmed" {
		if err := s.insertNotificationTx(ctx, tx, notifications.ModerationEvent{Kind: notifications.ModerationAppealConfirmed}, caseRow.ownerDID, caseID, eventID, now); err != nil {
			return CommandResult{}, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE moderation_cases SET revision=$2,updated_at=$3 WHERE id=$1`, caseID, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("revise moderation appeal case: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CommandResult{}, fmt.Errorf("commit moderation appeal command: %w", err)
	}
	return CommandResult{CaseID: caseID, EventID: eventID, Revision: resultRevision}, nil
}

func appealFingerprint(kind string, caseID uuid.UUID, expectedRevision int64, detail string) ([sha256.Size]byte, error) {
	encoded, err := json.Marshal(struct {
		Kind             string `json:"kind"`
		CaseID           string `json:"caseId"`
		ExpectedRevision int64  `json:"expectedRevision"`
		Detail           string `json:"detail"`
	}{kind, caseID.String(), expectedRevision, detail})
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint moderation appeal: %w", err)
	}
	return sha256.Sum256(encoded), nil
}
