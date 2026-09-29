package safetyincident

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestCSEAWorkflowPersistsCompleteAppendOnlyChronologyAndDeadlineState(t *testing.T) {
	pool := testdb.WithSchema(t, cseaWorkflowDDL)
	ctx := context.Background()
	detectedAt := time.Date(2030, 9, 22, 9, 0, 0, 0, time.UTC)
	deadline := detectedAt.Add(24 * time.Hour)
	incidentID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id,state,detected_at,updated_at) VALUES($1,'detected',$2,$2)`, incidentID, detectedAt); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incident_events(id,incident_id,event_type,created_at) VALUES($1,$2,'detected',$3)`, uuid.New(), incidentID, detectedAt); err != nil {
		t.Fatal(err)
	}
	workflow := NewCSEAWorkflow(pool)
	admin := Actor{ID: "safety-admin", Role: RoleSafetyAdministrator}
	if err := workflow.ClassifyAndAssign(ctx, ClassificationCommand{IncidentID: incidentID, Priority: CSEAPriorityImmediate, Assignee: "trained-responder", ReportingDeadline: deadline, Actor: admin, SourceSystem: "tabletop", ReplayID: "classify-1", At: detectedAt.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	initial, err := workflow.SubmitReport(ctx, ReportCommand{IncidentID: incidentID, Kind: ReportInitial, AuthorityReference: "synthetic-authority-ref-1", RetentionUntil: detectedAt.AddDate(1, 0, 0), Actor: admin, SourceSystem: "tabletop", ReplayID: "report-1", At: deadline.Add(-time.Hour)})
	if err != nil || !initial.DeadlineMet {
		t.Fatalf("initial=%+v err=%v", initial, err)
	}
	if _, err := workflow.SubmitReport(ctx, ReportCommand{IncidentID: incidentID, Kind: ReportSupplement, AuthorityReference: "synthetic-supplement-ref-1", RetentionUntil: detectedAt.AddDate(1, 0, 0), Actor: admin, SourceSystem: "tabletop", ReplayID: "supplement-1", At: deadline.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := workflow.SubmitReport(ctx, ReportCommand{IncidentID: incidentID, Kind: ReportDuplicate, AuthorityReference: "synthetic-duplicate-ref-1", DuplicateOfID: initial.ID, RetentionUntil: detectedAt.AddDate(1, 0, 0), Actor: admin, SourceSystem: "tabletop", ReplayID: "duplicate-1", At: deadline.Add(2 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	request, err := workflow.RecordInformationRequest(ctx, InformationRequestCommand{IncidentID: incidentID, AuthorityReference: "synthetic-follow-up-ref-1", DueAt: deadline.Add(48 * time.Hour), Actor: admin, SourceSystem: "tabletop", ReplayID: "follow-up-1", At: deadline.Add(3 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if err := workflow.RespondToInformationRequest(ctx, InformationResponseCommand{RequestID: request.ID, Actor: admin, SourceSystem: "tabletop", ReplayID: "response-1", At: deadline.Add(4 * time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if err := workflow.Resolve(ctx, ResolutionCommand{IncidentID: incidentID, Outcome: ResolutionReportedAndClosed, Actor: admin, SourceSystem: "tabletop", ReplayID: "resolve-1", At: deadline.Add(5 * time.Hour)}); err != nil {
		t.Fatal(err)
	}

	rows, err := pool.Query(ctx, `SELECT event_type FROM safety_incident_events WHERE incident_id=$1 ORDER BY created_at,id`, incidentID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var event string
		if err := rows.Scan(&event); err != nil {
			t.Fatal(err)
		}
		got = append(got, event)
	}
	want := []string{"detected", "priorityClassified", "assigned", "initialReportSubmitted", "supplementSubmitted", "duplicateRecorded", "informationRequestReceived", "informationRequestResponded", "resolved"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("chronology=%v, want %v", got, want)
	}
	var state, priority, assignee string
	if err := pool.QueryRow(ctx, `SELECT state,priority,assigned_to FROM safety_incidents WHERE id=$1`, incidentID).Scan(&state, &priority, &assignee); err != nil {
		t.Fatal(err)
	}
	if state != "resolved" || priority != "immediate" || assignee != "trained-responder" {
		t.Fatalf("state=%q priority=%q assignee=%q", state, priority, assignee)
	}
	var reports, deadlineMet, retained, linkedDuplicate int
	if err := pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE deadline_met),
		count(*) FILTER (WHERE retention_until=$2),count(*) FILTER (WHERE report_kind='duplicate' AND duplicate_of_id=$3)
		FROM safety_authority_reports WHERE incident_id=$1`, incidentID, detectedAt.AddDate(1, 0, 0), initial.ID).Scan(&reports, &deadlineMet, &retained, &linkedDuplicate); err != nil {
		t.Fatal(err)
	}
	if reports != 3 || deadlineMet != 1 || retained != 3 || linkedDuplicate != 1 {
		t.Fatalf("reports=%d deadlineMet=%d retained=%d linkedDuplicate=%d", reports, deadlineMet, retained, linkedDuplicate)
	}
	var requestState string
	var respondedBeforeDue bool
	if err := pool.QueryRow(ctx, `SELECT state,responded_at<=due_at FROM safety_authority_information_requests WHERE id=$1`, request.ID).Scan(&requestState, &respondedBeforeDue); err != nil {
		t.Fatal(err)
	}
	if requestState != "responded" || !respondedBeforeDue {
		t.Fatalf("request state=%q respondedBeforeDue=%t", requestState, respondedBeforeDue)
	}
}

const cseaWorkflowDDL = `
CREATE TABLE safety_incidents(id UUID PRIMARY KEY,state TEXT NOT NULL,priority TEXT,assigned_to TEXT,reporting_deadline TIMESTAMPTZ,detected_at TIMESTAMPTZ NOT NULL,updated_at TIMESTAMPTZ NOT NULL);
CREATE TABLE safety_incident_events(id UUID PRIMARY KEY,incident_id UUID NOT NULL REFERENCES safety_incidents(id),event_type TEXT NOT NULL,actor_id TEXT,source_system TEXT,replay_id TEXT,reference_id TEXT,created_at TIMESTAMPTZ NOT NULL,UNIQUE(source_system,replay_id));
CREATE TABLE safety_authority_reports(id UUID PRIMARY KEY,incident_id UUID NOT NULL REFERENCES safety_incidents(id),report_kind TEXT NOT NULL,authority_reference TEXT NOT NULL,duplicate_of_id UUID REFERENCES safety_authority_reports(id),submitted_by TEXT NOT NULL,submitted_at TIMESTAMPTZ NOT NULL,reporting_deadline TIMESTAMPTZ NOT NULL,deadline_met BOOLEAN NOT NULL,retention_until TIMESTAMPTZ NOT NULL,source_system TEXT NOT NULL,replay_id TEXT NOT NULL,UNIQUE(source_system,replay_id));
CREATE TABLE safety_authority_information_requests(id UUID PRIMARY KEY,incident_id UUID NOT NULL REFERENCES safety_incidents(id),authority_reference TEXT NOT NULL,state TEXT NOT NULL,due_at TIMESTAMPTZ NOT NULL,received_at TIMESTAMPTZ NOT NULL,responded_at TIMESTAMPTZ,received_by TEXT NOT NULL,responded_by TEXT,source_system TEXT NOT NULL,replay_id TEXT NOT NULL,UNIQUE(source_system,replay_id));
CREATE TABLE safety_incident_resolutions(id UUID PRIMARY KEY,incident_id UUID NOT NULL UNIQUE REFERENCES safety_incidents(id),outcome TEXT NOT NULL,resolved_by TEXT NOT NULL,resolved_at TIMESTAMPTZ NOT NULL);`
