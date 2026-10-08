package safetyincident

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidWorkflowTransition = errors.New("invalid safety workflow transition")
	ErrAuthorityNotVerified      = errors.New("authority request is not independently verified")
	ErrLawfulApprovalRequired    = errors.New("lawful-process approval is required")
)

type WorkflowService struct {
	pool      *pgxpool.Pool
	deadlines DeadlinePolicy
	now       func() time.Time
}

func NewWorkflowService(pool *pgxpool.Pool, deadlines DeadlinePolicy, now func() time.Time) *WorkflowService {
	if now == nil {
		now = time.Now
	}
	return &WorkflowService{pool: pool, deadlines: deadlines, now: now}
}

type IntimateImageCommand struct {
	IntakeID         uuid.UUID
	ReceivedAt       time.Time
	Actor            Actor
	StandingCode     string
	DeclarationsCode string
}

func (service *WorkflowService) OpenIntimateImage(ctx context.Context, command IntimateImageCommand) (uuid.UUID, Deadline, error) {
	if service == nil || service.pool == nil || command.ReceivedAt.IsZero() ||
		strings.TrimSpace(command.StandingCode) == "" || strings.TrimSpace(command.DeclarationsCode) == "" {
		return uuid.Nil, Deadline{}, ErrInvalidWorkflowTransition
	}
	if !command.Actor.HasPermission(PermissionWorkflowManage) {
		return uuid.Nil, Deadline{}, ErrUnauthorized
	}
	deadline, err := service.deadlines.Calculate(WorkflowIntimateImage, command.ReceivedAt)
	if err != nil {
		return uuid.Nil, Deadline{}, err
	}
	id := uuid.New()
	now := service.now().UTC()
	err = pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO safety_workflows(
			id,intake_id,workflow_class,state,urgency,responsible_actor,received_at,target_seconds,deadline_at,created_at,updated_at
		) VALUES($1,$2,'intimateImage','inProgress','priority',$3,$4,$5,$6,$7,$7)`,
			id, nullableUUID(command.IntakeID), command.Actor.ID, command.ReceivedAt.UTC(),
			int64(deadline.DueAt.Sub(command.ReceivedAt.UTC()).Seconds()), deadline.DueAt, now); err != nil {
			return fmt.Errorf("open intimate-image workflow: %w", err)
		}
		for index, event := range []struct{ kind, detail string }{
			{"received", "qualifyingComplaint"},
			{"standingRecorded", command.StandingCode},
			{"declarationsRecorded", command.DeclarationsCode},
		} {
			if err := insertWorkflowEvent(ctx, tx, id, int64(index+1), event.kind, command.Actor.ID, event.detail, now); err != nil {
				return err
			}
		}
		return nil
	})
	return id, deadline, err
}

type IntimateImageOutcome struct {
	WorkflowID                  uuid.UUID
	Actor                       Actor
	JudgmentCode                string
	SameImageSearchCode         string
	SubstantiallySameSearchCode string
	ActionCode                  string
	ExceptionCode               string
	OutcomeCode                 string
}

func (service *WorkflowService) ResolveIntimateImage(ctx context.Context, outcome IntimateImageOutcome) error {
	values := []string{outcome.Actor.ID, outcome.JudgmentCode, outcome.SameImageSearchCode,
		outcome.SubstantiallySameSearchCode, outcome.ActionCode, outcome.OutcomeCode}
	if service == nil || service.pool == nil || outcome.WorkflowID == uuid.Nil || slices.ContainsFunc(values, func(value string) bool { return strings.TrimSpace(value) == "" }) {
		return ErrInvalidWorkflowTransition
	}
	if !outcome.Actor.HasPermission(PermissionWorkflowManage) {
		return ErrUnauthorized
	}
	now := service.now().UTC()
	return pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		var class, state string
		var sequence int64
		if err := tx.QueryRow(ctx, `SELECT workflow_class,state,
			COALESCE((SELECT max(sequence) FROM safety_workflow_events WHERE workflow_id=$1),0)
			FROM safety_workflows WHERE id=$1 FOR UPDATE`, outcome.WorkflowID).Scan(&class, &state, &sequence); err != nil {
			return err
		}
		if class != string(WorkflowIntimateImage) || state != "inProgress" {
			return ErrInvalidWorkflowTransition
		}
		events := []struct{ kind, detail string }{
			{"judgmentRecorded", outcome.JudgmentCode},
			{"sameImageSearched", outcome.SameImageSearchCode},
			{"substantiallySameSearched", outcome.SubstantiallySameSearchCode},
			{"actionRecorded", outcome.ActionCode},
		}
		if strings.TrimSpace(outcome.ExceptionCode) != "" {
			events = append(events, struct{ kind, detail string }{"exceptionRecorded", outcome.ExceptionCode})
		}
		events = append(events, struct{ kind, detail string }{"outcomeRecorded", outcome.OutcomeCode})
		for index, event := range events {
			if err := insertWorkflowEvent(ctx, tx, outcome.WorkflowID, sequence+int64(index)+1, event.kind, outcome.Actor.ID, event.detail, now); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `UPDATE safety_workflows SET state='resolved',updated_at=$2 WHERE id=$1`, outcome.WorkflowID, now)
		return err
	})
}

type AuthorityRequest struct {
	ReceivedAt         time.Time
	Actor              Actor
	RequesterReference string
}

type CredibleThreat struct {
	ReceivedAt       time.Time
	Actor            Actor
	PreservationCode string
}

func (service *WorkflowService) OpenCredibleThreat(ctx context.Context, threat CredibleThreat) (uuid.UUID, error) {
	if service == nil || service.pool == nil || threat.ReceivedAt.IsZero() || strings.TrimSpace(threat.PreservationCode) == "" {
		return uuid.Nil, ErrInvalidWorkflowTransition
	}
	if !threat.Actor.HasPermission(PermissionWorkflowManage) {
		return uuid.Nil, ErrUnauthorized
	}
	id := uuid.New()
	now := service.now().UTC()
	err := pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO safety_workflows(
			id,workflow_class,state,urgency,responsible_actor,received_at,created_at,updated_at
		) VALUES($1,'credibleThreat','inProgress','immediate',$2,$3,$4,$4)`, id, threat.Actor.ID, threat.ReceivedAt.UTC(), now); err != nil {
			return err
		}
		if err := insertWorkflowEvent(ctx, tx, id, 1, "threatEscalated", threat.Actor.ID, "emergencyServicesFirst", now); err != nil {
			return err
		}
		return insertWorkflowEvent(ctx, tx, id, 2, "preservationRecorded", threat.Actor.ID, threat.PreservationCode, now)
	})
	return id, err
}

func (service *WorkflowService) OpenAuthorityRequest(ctx context.Context, request AuthorityRequest) (uuid.UUID, error) {
	if service == nil || service.pool == nil || request.ReceivedAt.IsZero() || strings.TrimSpace(request.RequesterReference) == "" {
		return uuid.Nil, ErrInvalidWorkflowTransition
	}
	if !request.Actor.HasPermission(PermissionAuthorityReport) {
		return uuid.Nil, ErrUnauthorized
	}
	id := uuid.New()
	now := service.now().UTC()
	_, err := service.pool.Exec(ctx, `WITH workflow AS (
		INSERT INTO safety_workflows(id,workflow_class,state,urgency,responsible_actor,received_at,created_at,updated_at)
		VALUES($1,'authorityRequest','open','immediate',$2,$3,$4,$4)
	)
	INSERT INTO safety_authority_requests(workflow_id,requester_reference,verification_state)
	VALUES($1,$5,'pending')`, id, request.Actor.ID, request.ReceivedAt.UTC(), now, request.RequesterReference)
	return id, err
}

func (service *WorkflowService) VerifyAuthority(ctx context.Context, workflowID uuid.UUID, actor Actor, verificationReference string, genuine bool) error {
	if workflowID == uuid.Nil || strings.TrimSpace(verificationReference) == "" {
		return ErrInvalidWorkflowTransition
	}
	if !actor.HasPermission(PermissionAuthorityReport) {
		return ErrUnauthorized
	}
	now := service.now().UTC()
	return pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		state := "verified"
		event := "authorityVerified"
		workflowState := "inProgress"
		if !genuine {
			state, event, workflowState = "rejected", "authorityRejected", "refused"
		}
		command, err := tx.Exec(ctx, `UPDATE safety_authority_requests SET verification_state=$2,independent_verification=$3 WHERE workflow_id=$1 AND verification_state='pending'`, workflowID, state, verificationReference)
		if err != nil || command.RowsAffected() != 1 {
			return ErrInvalidWorkflowTransition
		}
		if err := insertWorkflowEvent(ctx, tx, workflowID, 1, event, actor.ID, verificationReference, now); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE safety_workflows SET state=$2,updated_at=$3 WHERE id=$1`, workflowID, workflowState, now)
		return err
	})
}

var allowedDisclosureFields = map[string]struct{}{
	"accountDid": {}, "recordUri": {}, "recordCid": {}, "eventTimestamps": {}, "accountContact": {},
}

func (service *WorkflowService) ApproveAndDisclose(ctx context.Context, workflowID uuid.UUID, actor Actor, legalBasis string, fields []string) error {
	if workflowID == uuid.Nil || strings.TrimSpace(legalBasis) == "" || len(fields) == 0 {
		return ErrLawfulApprovalRequired
	}
	if !actor.HasPermission(PermissionDisclosureApprove) {
		return ErrUnauthorized
	}
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if _, allowed := allowedDisclosureFields[field]; !allowed {
			return ErrLawfulApprovalRequired
		}
		seen[field] = struct{}{}
	}
	minimized := make([]string, 0, len(seen))
	for field := range seen {
		minimized = append(minimized, field)
	}
	slices.Sort(minimized)
	now := service.now().UTC()
	return pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		var verification string
		if err := tx.QueryRow(ctx, `SELECT verification_state FROM safety_authority_requests WHERE workflow_id=$1 FOR UPDATE`, workflowID).Scan(&verification); err != nil {
			return err
		}
		if verification != "verified" {
			return ErrAuthorityNotVerified
		}
		if _, err := tx.Exec(ctx, `UPDATE safety_authority_requests SET legal_basis_reference=$2,approved_by_actor=$3,disclosed_fields=$4 WHERE workflow_id=$1`, workflowID, legalBasis, actor.ID, minimized); err != nil {
			return err
		}
		if err := insertWorkflowEvent(ctx, tx, workflowID, 2, "lawfulProcessApproved", actor.ID, legalBasis, now); err != nil {
			return err
		}
		return insertWorkflowEvent(ctx, tx, workflowID, 3, "disclosureRecorded", actor.ID, "minimized", now)
	})
}

func (service *WorkflowService) CloseAuthorityRequest(ctx context.Context, workflowID uuid.UUID, actor Actor, reviewCode string) error {
	if workflowID == uuid.Nil || strings.TrimSpace(reviewCode) == "" {
		return ErrInvalidWorkflowTransition
	}
	if !actor.HasPermission(PermissionAuthorityReport) {
		return ErrUnauthorized
	}
	now := service.now().UTC()
	return pgx.BeginFunc(ctx, service.pool, func(tx pgx.Tx) error {
		command, err := tx.Exec(ctx, `UPDATE safety_authority_requests SET reviewed_at=$2 WHERE workflow_id=$1 AND verification_state IN ('verified','rejected')`, workflowID, now)
		if err != nil || command.RowsAffected() != 1 {
			return ErrInvalidWorkflowTransition
		}
		var sequence int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(sequence),0)+1 FROM safety_workflow_events WHERE workflow_id=$1`, workflowID).Scan(&sequence); err != nil {
			return err
		}
		if err := insertWorkflowEvent(ctx, tx, workflowID, sequence, "postIncidentReviewed", actor.ID, reviewCode, now); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE safety_workflows SET state='resolved',updated_at=$2 WHERE id=$1`, workflowID, now)
		return err
	})
}

func insertWorkflowEvent(ctx context.Context, tx pgx.Tx, workflowID uuid.UUID, sequence int64, eventType, actorID, detail string, at time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO safety_workflow_events(id,workflow_id,sequence,event_type,actor_id,detail_code,occurred_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.New(), workflowID, sequence, eventType, actorID, nullableString(detail), at)
	return err
}

func nullableUUID(value uuid.UUID) any {
	if value == uuid.Nil {
		return nil
	}
	return value
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
