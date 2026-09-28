package moderation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

var (
	ErrInvalidCommand   = errors.New("invalid moderation command")
	ErrReplayConflict   = errors.New("moderation replay identity conflict")
	ErrRevisionConflict = errors.New("moderation case revision conflict")
	ErrCaseNotOpen      = errors.New("moderation case is not open")
)

type TrustedCommand struct {
	CaseID           uuid.UUID
	ExpectedRevision int64
	SourceSystem     string
	ReplayID         string
	ActorID          string
	SourceDID        syntax.DID
	Decision         Decision
}

type CommandResult struct {
	CaseID   uuid.UUID
	EventID  uuid.UUID
	Revision int64
	Replayed bool
}

type VisibilityOutputWrite struct {
	SourceDID         syntax.DID
	SubjectType       SubjectType
	SubjectDID        syntax.DID
	SubjectCollection string
	SubjectRkey       string
	SubjectURI        syntax.ATURI
	Value             string
	Action            string
	InternalReason    string
	CreatedAt         time.Time
}

type VisibilityOutputWriter interface {
	InsertOutputTx(context.Context, pgx.Tx, VisibilityOutputWrite) (string, error)
}

type NotificationIntentWriter interface {
	InsertModerationIntentTx(context.Context, pgx.Tx, notifications.ModerationIntent) error
}

type ServiceOption func(*Service)

func WithNotificationWriter(writer NotificationIntentWriter) ServiceOption {
	return func(service *Service) { service.notifications = writer }
}

type OperationObserver interface {
	ObserveModerationCommand(context.Context, string, string, string, string, string, time.Time)
}

func WithOperationObserver(observer OperationObserver) ServiceOption {
	return func(service *Service) { service.observer = observer }
}

type Service struct {
	store         *Store
	visibility    VisibilityOutputWriter
	notifications NotificationIntentWriter
	observer      OperationObserver
	now           func() time.Time
}

func NewService(store *Store, visibility VisibilityOutputWriter, now func() time.Time, options ...ServiceOption) *Service {
	if now == nil {
		now = time.Now
	}
	service := &Service{store: store, visibility: visibility, now: now}
	for _, option := range options {
		option(service)
	}
	return service
}

type lockedCase struct {
	id                uuid.UUID
	ownerDID          syntax.DID
	subjectType       SubjectType
	subjectDID        syntax.DID
	subjectCollection string
	subjectRkey       string
	subjectURI        syntax.ATURI
	state             string
	revision          int64
}

func (s *Service) ResolveCase(ctx context.Context, command TrustedCommand) (result CommandResult, resultErr error) {
	defer func() {
		s.observeCommand(ctx, "decision", command.CaseID, command.SourceSystem, command.ReplayID, result, resultErr)
	}()
	if s == nil || s.store == nil || s.store.pool == nil || s.visibility == nil ||
		command.CaseID == uuid.Nil || command.ExpectedRevision < 0 || command.SourceDID == "" ||
		strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" ||
		strings.TrimSpace(command.ActorID) == "" {
		return CommandResult{}, ErrInvalidCommand
	}
	if err := ValidateDecision(command.Decision); err != nil {
		return CommandResult{}, err
	}
	fingerprint, err := commandFingerprint(command)
	if err != nil {
		return CommandResult{}, err
	}
	now := s.now().UTC().Truncate(time.Microsecond)

	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return CommandResult{}, fmt.Errorf("begin moderation decision: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ownerDID syntax.DID
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_cases WHERE id=$1`, command.CaseID).Scan(&ownerDID); err != nil {
		return CommandResult{}, fmt.Errorf("read moderation case owner: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_account_standings(owner_did,updated_at) VALUES($1,$2) ON CONFLICT(owner_did) DO NOTHING`, ownerDID, now); err != nil {
		return CommandResult{}, fmt.Errorf("ensure moderation standing: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT owner_did FROM moderation_account_standings WHERE owner_did=$1 FOR UPDATE`, ownerDID).Scan(&ownerDID); err != nil {
		return CommandResult{}, fmt.Errorf("lock moderation standing: %w", err)
	}

	caseRow, err := lockCase(ctx, tx, command.CaseID)
	if err != nil {
		return CommandResult{}, err
	}
	if replay, found, err := replayResult(ctx, tx, command.CaseID, command.SourceSystem, command.ReplayID, fingerprint); err != nil {
		return CommandResult{}, err
	} else if found {
		if err := tx.Commit(ctx); err != nil {
			return CommandResult{}, fmt.Errorf("commit moderation replay: %w", err)
		}
		return replay, nil
	}
	if caseRow.revision != command.ExpectedRevision {
		return CommandResult{}, ErrRevisionConflict
	}
	if caseRow.state != "open" {
		return CommandResult{}, ErrCaseNotOpen
	}

	eventID := uuid.New()
	resultRevision := command.ExpectedRevision + 1
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_events(
		id,case_id,event_type,actor_id,source_system,replay_id,request_fingerprint,
		expected_revision,result_revision,created_at
	) VALUES($1,$2,'decision',$3,$4,$5,$6,$7,$8,$9)`, eventID, command.CaseID,
		command.ActorID, command.SourceSystem, command.ReplayID, fingerprint[:],
		command.ExpectedRevision, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("insert moderation decision event: %w", err)
	}
	decisionID := uuid.New()
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_decisions(
		id,case_id,case_event_id,disposition,reason,internal_evidence_notes,
		user_safe_detail,severity_rationale,created_at
	) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, decisionID, command.CaseID, eventID,
		command.Decision.Disposition, nullIfEmpty(string(command.Decision.Reason)),
		nullIfEmpty(command.Decision.Evidence), nullIfEmpty(command.Decision.UserSafeDetail),
		nullIfEmpty(command.Decision.SeverityRationale), now); err != nil {
		return CommandResult{}, fmt.Errorf("insert moderation decision: %w", err)
	}

	for _, effect := range command.Decision.Consequences {
		if err := s.applyDecisionEffect(ctx, tx, caseRow, command, eventID, effect, now); err != nil {
			return CommandResult{}, err
		}
	}
	if err := updateStandingTx(ctx, tx, caseRow.ownerDID, now); err != nil {
		return CommandResult{}, err
	}
	if err := s.insertNotificationTx(ctx, tx, notifications.ModerationEvent{
		Kind: notifications.ModerationDecision, ConsequenceChanged: len(command.Decision.Consequences) > 0,
	}, caseRow.ownerDID, caseRow.id, eventID, now); err != nil {
		return CommandResult{}, err
	}
	if _, err := tx.Exec(ctx, `UPDATE moderation_cases SET state='resolved',revision=$2,resolved_at=$3,updated_at=$3 WHERE id=$1`, command.CaseID, resultRevision, now); err != nil {
		return CommandResult{}, fmt.Errorf("resolve moderation case: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CommandResult{}, fmt.Errorf("commit moderation decision: %w", err)
	}
	return CommandResult{CaseID: command.CaseID, EventID: eventID, Revision: resultRevision}, nil
}

func (s *Service) observeCommand(ctx context.Context, operation string, caseID uuid.UUID, sourceSystem, replayID string, result CommandResult, err error) {
	if s == nil || s.observer == nil {
		return
	}
	resultClass := "success"
	if err != nil {
		resultClass = "failure"
	} else if result.Replayed {
		resultClass = "replayed"
	}
	caseReference := ""
	if reference, referenceErr := FormatCaseReference(caseID); referenceErr == nil {
		caseReference = reference
	}
	requestHash := sha256.Sum256([]byte(sourceSystem + "\x00" + replayID))
	s.observer.ObserveModerationCommand(ctx, operation, resultClass, "moderator", hex.EncodeToString(requestHash[:]), caseReference, time.Now().UTC())
}

func lockCase(ctx context.Context, tx pgx.Tx, caseID uuid.UUID) (lockedCase, error) {
	var row lockedCase
	var subjectCollection, subjectRkey, subjectURI *string
	err := tx.QueryRow(ctx, `SELECT id,owner_did,subject_type,subject_did,subject_collection,
		subject_rkey,subject_uri,state,revision FROM moderation_cases WHERE id=$1 FOR UPDATE`, caseID).Scan(
		&row.id, &row.ownerDID, &row.subjectType, &row.subjectDID, &subjectCollection,
		&subjectRkey, &subjectURI, &row.state, &row.revision)
	if err != nil {
		return lockedCase{}, fmt.Errorf("lock moderation case: %w", err)
	}
	if subjectCollection != nil {
		row.subjectCollection = *subjectCollection
	}
	if subjectRkey != nil {
		row.subjectRkey = *subjectRkey
	}
	if subjectURI != nil {
		row.subjectURI = syntax.ATURI(*subjectURI)
	}
	return row, nil
}

func replayResult(ctx context.Context, tx pgx.Tx, caseID uuid.UUID, sourceSystem, replayID string, fingerprint [sha256.Size]byte) (CommandResult, bool, error) {
	var result CommandResult
	var stored []byte
	err := tx.QueryRow(ctx, `SELECT case_id,id,result_revision,request_fingerprint
		FROM moderation_case_events WHERE source_system=$1 AND replay_id=$2`, sourceSystem, replayID).Scan(
		&result.CaseID, &result.EventID, &result.Revision, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommandResult{}, false, nil
	}
	if err != nil {
		return CommandResult{}, false, fmt.Errorf("read moderation replay: %w", err)
	}
	if result.CaseID != caseID || !bytes.Equal(stored, fingerprint[:]) {
		return CommandResult{}, false, ErrReplayConflict
	}
	result.Replayed = true
	return result, true, nil
}

func (s *Service) insertNotificationTx(ctx context.Context, tx pgx.Tx, event notifications.ModerationEvent, recipient syntax.DID, caseID, eventID uuid.UUID, createdAt time.Time) error {
	return insertNotificationTx(ctx, tx, s.notifications, event, recipient, caseID, eventID, createdAt)
}

func insertNotificationTx(ctx context.Context, tx pgx.Tx, writer NotificationIntentWriter, event notifications.ModerationEvent, recipient syntax.DID, caseID, eventID uuid.UUID, createdAt time.Time) error {
	if writer == nil || !notifications.ShouldNotifyModeration(event) {
		return nil
	}
	reference, err := FormatCaseReference(caseID)
	if err != nil {
		return fmt.Errorf("format moderation notification reference: %w", err)
	}
	intent := notifications.ModerationIntent{RecipientDID: recipient, CaseID: caseID, CaseReference: reference, EventID: eventID, CreatedAt: createdAt}
	if err := writer.InsertModerationIntentTx(ctx, tx, intent); err != nil {
		return fmt.Errorf("insert moderation notification intent: %w", err)
	}
	return nil
}

func (s *Service) applyDecisionEffect(ctx context.Context, tx pgx.Tx, caseRow lockedCase, command TrustedCommand, eventID uuid.UUID, effect EffectType, now time.Time) error {
	logicalID := uuid.New()
	var outputID any
	if isVisibilityEffect(effect) {
		id, err := s.visibility.InsertOutputTx(ctx, tx, VisibilityOutputWrite{
			SourceDID: command.SourceDID, SubjectType: caseRow.subjectType, SubjectDID: caseRow.subjectDID,
			SubjectCollection: caseRow.subjectCollection, SubjectRkey: caseRow.subjectRkey,
			SubjectURI: caseRow.subjectURI, Value: visibilityValue(effect), Action: string(EffectApply),
			InternalReason: command.Decision.Evidence, CreatedAt: now,
		})
		if err != nil {
			return fmt.Errorf("insert moderation visibility output: %w", err)
		}
		outputID = id
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_effect_events(
		id,case_id,case_event_id,logical_effect_id,effect_type,action,moderation_output_id,created_at
	) VALUES($1,$2,$3,$4,$5,'apply',$6,$7)`, uuid.New(), caseRow.id, eventID, logicalID, effect, outputID, now); err != nil {
		return fmt.Errorf("insert moderation effect event: %w", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_active_case_effects(
		case_id,effect_type,logical_effect_id,applied_event_id,moderation_output_id,applied_at
	) VALUES($1,$2,$3,$4,$5,$6)`, caseRow.id, effect, logicalID, eventID, outputID, now); err != nil {
		return fmt.Errorf("project active moderation effect: %w", err)
	}
	if effect == EffectStrike {
		if _, err := tx.Exec(ctx, `INSERT INTO moderation_case_strikes(
			case_id,logical_effect_id,owner_did,issued_at,due_at,updated_at
		) VALUES($1,$2,$3,$4,$5,$4)`, caseRow.id, logicalID, caseRow.ownerDID, now, StrikeDeadline(now)); err != nil {
			return fmt.Errorf("project moderation strike: %w", err)
		}
	}
	return nil
}

func updateStandingTx(ctx context.Context, tx pgx.Tx, ownerDID syntax.DID, now time.Time) error {
	_, err := tx.Exec(ctx, `UPDATE moderation_account_standings AS standing SET
		active_strike_count=(SELECT count(*) FROM moderation_case_strikes WHERE owner_did=$1 AND expired_at IS NULL AND overturned_at IS NULL),
		threshold_suspended=((SELECT count(*) FROM moderation_case_strikes WHERE owner_did=$1 AND expired_at IS NULL AND overturned_at IS NULL) >= 3),
		severe_suspended=EXISTS(SELECT 1 FROM moderation_active_case_effects effect JOIN moderation_cases c ON c.id=effect.case_id WHERE c.owner_did=$1 AND effect.effect_type='severeSuspension'),
		revision=standing.revision+1,updated_at=$2 WHERE owner_did=$1`, ownerDID, now)
	if err != nil {
		return fmt.Errorf("update moderation standing: %w", err)
	}
	return nil
}

func commandFingerprint(command TrustedCommand) ([sha256.Size]byte, error) {
	effects := append([]EffectType(nil), command.Decision.Consequences...)
	sort.Slice(effects, func(i, j int) bool { return effects[i] < effects[j] })
	payload := struct {
		CaseID           string       `json:"caseId"`
		ExpectedRevision int64        `json:"expectedRevision"`
		SourceDID        syntax.DID   `json:"sourceDid"`
		Disposition      Disposition  `json:"disposition"`
		Reason           Reason       `json:"reason"`
		Evidence         string       `json:"evidence"`
		UserSafeDetail   string       `json:"userSafeDetail"`
		Severity         string       `json:"severityRationale"`
		Effects          []EffectType `json:"effects"`
	}{command.CaseID.String(), command.ExpectedRevision, command.SourceDID, command.Decision.Disposition,
		command.Decision.Reason, command.Decision.Evidence, command.Decision.UserSafeDetail,
		command.Decision.SeverityRationale, effects}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("fingerprint moderation command: %w", err)
	}
	return sha256.Sum256(encoded), nil
}
