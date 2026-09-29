package safetyincident

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInvalidCSEAWorkflow = errors.New("invalid CSEA workflow transition")

type CSEAPriority string

const (
	CSEAPriorityImmediate CSEAPriority = "immediate"
	CSEAPriorityUrgent    CSEAPriority = "urgent"
	CSEAPriorityStandard  CSEAPriority = "standard"
)

type ReportKind string

const (
	ReportInitial    ReportKind = "initial"
	ReportSupplement ReportKind = "supplement"
	ReportDuplicate  ReportKind = "duplicate"
)

type ResolutionOutcome string

const (
	ResolutionReportedAndClosed ResolutionOutcome = "reportedAndClosed"
	ResolutionNoReportRequired  ResolutionOutcome = "noReportRequired"
	ResolutionDuplicateClosed   ResolutionOutcome = "duplicateClosed"
)

type CSEAWorkflow struct{ pool *pgxpool.Pool }

func NewCSEAWorkflow(pool *pgxpool.Pool) *CSEAWorkflow { return &CSEAWorkflow{pool: pool} }

type ClassificationCommand struct {
	IncidentID        uuid.UUID
	Priority          CSEAPriority
	Assignee          string
	ReportingDeadline time.Time
	Actor             Actor
	SourceSystem      string
	ReplayID          string
	At                time.Time
}

func (workflow *CSEAWorkflow) ClassifyAndAssign(ctx context.Context, command ClassificationCommand) error {
	if workflow == nil || workflow.pool == nil || !validOperation(command.IncidentID, command.Actor, command.SourceSystem, command.ReplayID, command.At) ||
		(command.Priority != CSEAPriorityImmediate && command.Priority != CSEAPriorityUrgent && command.Priority != CSEAPriorityStandard) ||
		strings.TrimSpace(command.Assignee) == "" || !command.ReportingDeadline.After(command.At) {
		return ErrInvalidCSEAWorkflow
	}
	return pgx.BeginFunc(ctx, workflow.pool, func(tx pgx.Tx) error {
		var replayed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_incident_events WHERE source_system=$1 AND replay_id=$2 AND incident_id=$3 AND event_type='priorityClassified')`, command.SourceSystem, command.ReplayID+":priority", command.IncidentID).Scan(&replayed); err != nil {
			return err
		}
		if replayed {
			return nil
		}
		tag, err := tx.Exec(ctx, `UPDATE safety_incidents SET priority=$2,assigned_to=$3,reporting_deadline=$4,updated_at=$5
			WHERE id=$1 AND state IN ('detected','confirmed','caseCreated','reporting')`, command.IncidentID, command.Priority,
			command.Assignee, command.ReportingDeadline.UTC(), command.At.UTC())
		if err != nil || tag.RowsAffected() != 1 {
			if err != nil {
				return err
			}
			return ErrInvalidCSEAWorkflow
		}
		if err := appendIncidentEvent(ctx, tx, command.IncidentID, "priorityClassified", command.Actor.ID, command.SourceSystem, command.ReplayID+":priority", string(command.Priority), command.At); err != nil {
			return err
		}
		return appendIncidentEvent(ctx, tx, command.IncidentID, "assigned", command.Actor.ID, command.SourceSystem, command.ReplayID+":assignment", command.Assignee, command.At.Add(time.Microsecond))
	})
}

type ReportCommand struct {
	IncidentID         uuid.UUID
	Kind               ReportKind
	AuthorityReference string
	DuplicateOfID      uuid.UUID
	RetentionUntil     time.Time
	Actor              Actor
	SourceSystem       string
	ReplayID           string
	At                 time.Time
}

type AuthorityReport struct {
	ID          uuid.UUID
	DeadlineMet bool
}

func (workflow *CSEAWorkflow) SubmitReport(ctx context.Context, command ReportCommand) (AuthorityReport, error) {
	if workflow == nil || workflow.pool == nil || !validOperation(command.IncidentID, command.Actor, command.SourceSystem, command.ReplayID, command.At) ||
		(command.Kind != ReportInitial && command.Kind != ReportSupplement && command.Kind != ReportDuplicate) ||
		!validReference(command.AuthorityReference) || !command.RetentionUntil.After(command.At) ||
		(command.Kind == ReportDuplicate) != (command.DuplicateOfID != uuid.Nil) {
		return AuthorityReport{}, ErrInvalidCSEAWorkflow
	}
	var result AuthorityReport
	err := pgx.BeginFunc(ctx, workflow.pool, func(tx pgx.Tx) error {
		var deadline time.Time
		if err := tx.QueryRow(ctx, `SELECT reporting_deadline FROM safety_incidents WHERE id=$1 AND reporting_deadline IS NOT NULL FOR UPDATE`, command.IncidentID).Scan(&deadline); err != nil {
			return ErrInvalidCSEAWorkflow
		}
		if command.Kind != ReportInitial {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_authority_reports WHERE incident_id=$1 AND report_kind='initial')`, command.IncidentID).Scan(&exists); err != nil || !exists {
				return ErrInvalidCSEAWorkflow
			}
		}
		if command.Kind == ReportDuplicate {
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_authority_reports WHERE id=$1 AND incident_id=$2)`, command.DuplicateOfID, command.IncidentID).Scan(&exists); err != nil || !exists {
				return ErrInvalidCSEAWorkflow
			}
		}
		result.ID = uuid.New()
		result.DeadlineMet = !command.At.After(deadline)
		if err := tx.QueryRow(ctx, `INSERT INTO safety_authority_reports(
			id,incident_id,report_kind,authority_reference,duplicate_of_id,submitted_by,submitted_at,
			reporting_deadline,deadline_met,retention_until,source_system,replay_id
		) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT(source_system,replay_id) DO UPDATE SET replay_id=EXCLUDED.replay_id
		RETURNING id,deadline_met`, result.ID, command.IncidentID, command.Kind, command.AuthorityReference,
			nullUUID(command.DuplicateOfID), command.Actor.ID, command.At.UTC(), deadline, result.DeadlineMet,
			command.RetentionUntil.UTC(), command.SourceSystem, command.ReplayID).Scan(&result.ID, &result.DeadlineMet); err != nil {
			return err
		}
		eventType := map[ReportKind]string{ReportInitial: "initialReportSubmitted", ReportSupplement: "supplementSubmitted", ReportDuplicate: "duplicateRecorded"}[command.Kind]
		if err := appendIncidentEvent(ctx, tx, command.IncidentID, eventType, command.Actor.ID, command.SourceSystem, command.ReplayID, command.AuthorityReference, command.At); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `UPDATE safety_incidents SET state='reporting',updated_at=$2 WHERE id=$1 AND state<>'resolved'`, command.IncidentID, command.At.UTC())
		return err
	})
	return result, err
}

type InformationRequestCommand struct {
	IncidentID         uuid.UUID
	AuthorityReference string
	DueAt              time.Time
	Actor              Actor
	SourceSystem       string
	ReplayID           string
	At                 time.Time
}

type InformationRequest struct{ ID uuid.UUID }

func (workflow *CSEAWorkflow) RecordInformationRequest(ctx context.Context, command InformationRequestCommand) (InformationRequest, error) {
	if workflow == nil || workflow.pool == nil || !validOperation(command.IncidentID, command.Actor, command.SourceSystem, command.ReplayID, command.At) ||
		!validReference(command.AuthorityReference) || command.DueAt.Before(command.At) {
		return InformationRequest{}, ErrInvalidCSEAWorkflow
	}
	result := InformationRequest{ID: uuid.New()}
	err := pgx.BeginFunc(ctx, workflow.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO safety_authority_information_requests(
			id,incident_id,authority_reference,state,due_at,received_at,received_by,source_system,replay_id
		) VALUES($1,$2,$3,'received',$4,$5,$6,$7,$8)
		ON CONFLICT(source_system,replay_id) DO UPDATE SET replay_id=EXCLUDED.replay_id RETURNING id`, result.ID,
			command.IncidentID, command.AuthorityReference, command.DueAt.UTC(), command.At.UTC(), command.Actor.ID,
			command.SourceSystem, command.ReplayID).Scan(&result.ID); err != nil {
			return err
		}
		return appendIncidentEvent(ctx, tx, command.IncidentID, "informationRequestReceived", command.Actor.ID, command.SourceSystem, command.ReplayID, command.AuthorityReference, command.At)
	})
	return result, err
}

type InformationResponseCommand struct {
	RequestID    uuid.UUID
	Actor        Actor
	SourceSystem string
	ReplayID     string
	At           time.Time
}

func (workflow *CSEAWorkflow) RespondToInformationRequest(ctx context.Context, command InformationResponseCommand) error {
	if workflow == nil || workflow.pool == nil || command.RequestID == uuid.Nil || strings.TrimSpace(command.Actor.ID) == "" ||
		strings.TrimSpace(command.SourceSystem) == "" || strings.TrimSpace(command.ReplayID) == "" || command.At.IsZero() {
		return ErrInvalidCSEAWorkflow
	}
	return pgx.BeginFunc(ctx, workflow.pool, func(tx pgx.Tx) error {
		var replayed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_incident_events WHERE source_system=$1 AND replay_id=$2 AND event_type='informationRequestResponded')`, command.SourceSystem, command.ReplayID).Scan(&replayed); err != nil {
			return err
		}
		if replayed {
			return nil
		}
		var incidentID uuid.UUID
		var reference string
		if err := tx.QueryRow(ctx, `UPDATE safety_authority_information_requests SET state='responded',responded_at=$2,responded_by=$3
			WHERE id=$1 AND state='received' RETURNING incident_id,authority_reference`, command.RequestID, command.At.UTC(), command.Actor.ID).Scan(&incidentID, &reference); err != nil {
			return ErrInvalidCSEAWorkflow
		}
		if !command.Actor.Allowed(PermissionAuthorityReport, incidentID) {
			return ErrUnauthorized
		}
		return appendIncidentEvent(ctx, tx, incidentID, "informationRequestResponded", command.Actor.ID, command.SourceSystem, command.ReplayID, reference, command.At)
	})
}

type ResolutionCommand struct {
	IncidentID   uuid.UUID
	Outcome      ResolutionOutcome
	Actor        Actor
	SourceSystem string
	ReplayID     string
	At           time.Time
}

func (workflow *CSEAWorkflow) Resolve(ctx context.Context, command ResolutionCommand) error {
	if workflow == nil || workflow.pool == nil || !validOperation(command.IncidentID, command.Actor, command.SourceSystem, command.ReplayID, command.At) ||
		(command.Outcome != ResolutionReportedAndClosed && command.Outcome != ResolutionNoReportRequired && command.Outcome != ResolutionDuplicateClosed) {
		return ErrInvalidCSEAWorkflow
	}
	return pgx.BeginFunc(ctx, workflow.pool, func(tx pgx.Tx) error {
		var replayed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_incident_events WHERE source_system=$1 AND replay_id=$2 AND incident_id=$3 AND event_type='resolved')`, command.SourceSystem, command.ReplayID, command.IncidentID).Scan(&replayed); err != nil {
			return err
		}
		if replayed {
			return nil
		}
		if _, err := tx.Exec(ctx, `INSERT INTO safety_incident_resolutions(id,incident_id,outcome,resolved_by,resolved_at)
			VALUES($1,$2,$3,$4,$5) ON CONFLICT(incident_id) DO NOTHING`, uuid.New(), command.IncidentID, command.Outcome, command.Actor.ID, command.At.UTC()); err != nil {
			return err
		}
		if err := appendIncidentEvent(ctx, tx, command.IncidentID, "resolved", command.Actor.ID, command.SourceSystem, command.ReplayID, string(command.Outcome), command.At); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE safety_incidents SET state='resolved',updated_at=$2 WHERE id=$1 AND state<>'resolved'`, command.IncidentID, command.At.UTC())
		if err != nil || tag.RowsAffected() != 1 {
			return ErrInvalidCSEAWorkflow
		}
		return nil
	})
}

func validOperation(incidentID uuid.UUID, actor Actor, sourceSystem, replayID string, at time.Time) bool {
	return incidentID != uuid.Nil && actor.Allowed(PermissionAuthorityReport, incidentID) &&
		strings.TrimSpace(sourceSystem) != "" && strings.TrimSpace(replayID) != "" && !at.IsZero()
}

func validReference(reference string) bool {
	return strings.TrimSpace(reference) != "" && len(reference) <= 512
}

func appendIncidentEvent(ctx context.Context, tx pgx.Tx, incidentID uuid.UUID, eventType, actorID, sourceSystem, replayID, reference string, at time.Time) error {
	_, err := tx.Exec(ctx, `INSERT INTO safety_incident_events(id,incident_id,event_type,actor_id,source_system,replay_id,reference_id,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(source_system,replay_id) DO NOTHING`, uuid.New(), incidentID,
		eventType, actorID, sourceSystem, replayID, nullString(reference), at.UTC())
	return err
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullUUID(value uuid.UUID) any {
	if value == uuid.Nil {
		return nil
	}
	return value
}
