package safetyincident

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const workflowPreStateDDL = `
CREATE TABLE moderation_cases (id UUID PRIMARY KEY, owner_did TEXT NOT NULL);
CREATE TABLE moderation_appeal_correspondence (
 id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id), source_system TEXT NOT NULL,
 replay_id TEXT NOT NULL, sender_reference_hash BYTEA, received_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL,
 UNIQUE(source_system,replay_id)
);
`

func TestIntimateImageWorkflowRecordsCompleteAttributedChronology(t *testing.T) {
	pool := testdb.WithSchema(t, workflowPreStateDDL)
	applyWorkflowMigration(t, pool)
	ctx := context.Background()
	received := time.Date(2030, 4, 3, 9, 30, 0, 0, time.UTC)
	clock := func() time.Time { return received.Add(time.Minute) }
	policy, err := NewDeadlinePolicy(map[WorkflowClass]DeadlineTarget{
		WorkflowIntimateImage: {ResponseWithin: 30 * time.Hour, AlertBefore: []time.Duration{6 * time.Hour}},
	})
	if err != nil {
		t.Fatal(err)
	}
	service := NewWorkflowService(pool, policy, clock)
	workflowID, deadline, err := service.OpenIntimateImage(ctx, IntimateImageCommand{
		ReceivedAt: received, Actor: Actor{ID: "safety-admin", Role: RoleSafetyAdministrator}, StandingCode: "depictedPerson", DeclarationsCode: "goodFaithConfirmed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !deadline.DueAt.Equal(received.Add(30 * time.Hour)) {
		t.Fatalf("deadline = %s", deadline.DueAt)
	}
	if err := service.ResolveIntimateImage(ctx, IntimateImageOutcome{
		WorkflowID: workflowID, Actor: Actor{ID: "reviewer-2", Role: RoleSafetyAdministrator}, JudgmentCode: "qualifying",
		SameImageSearchCode: "completed", SubstantiallySameSearchCode: "completed",
		ActionCode: "craftskyTakedown", ExceptionCode: "none", OutcomeCode: "resolvedWithinTarget",
	}); err != nil {
		t.Fatal(err)
	}

	var state string
	var targetSeconds int64
	var eventCount, actorCount int
	if err := pool.QueryRow(ctx, `SELECT state,target_seconds,
		(SELECT count(*) FROM safety_workflow_events WHERE workflow_id=$1),
		(SELECT count(DISTINCT actor_id) FROM safety_workflow_events WHERE workflow_id=$1)
		FROM safety_workflows WHERE id=$1`, workflowID).Scan(&state, &targetSeconds, &eventCount, &actorCount); err != nil {
		t.Fatal(err)
	}
	if state != "resolved" || targetSeconds != 108000 || eventCount != 9 || actorCount != 2 {
		t.Fatalf("state=%s target=%d events=%d actors=%d", state, targetSeconds, eventCount, actorCount)
	}
	rows, err := pool.Query(ctx, `SELECT event_type FROM safety_workflow_events WHERE workflow_id=$1 ORDER BY sequence`, workflowID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := []string{"received", "standingRecorded", "declarationsRecorded", "judgmentRecorded", "sameImageSearched", "substantiallySameSearched", "actionRecorded", "exceptionRecorded", "outcomeRecorded"}
	var got []string
	for rows.Next() {
		var event string
		if err := rows.Scan(&event); err != nil {
			t.Fatal(err)
		}
		got = append(got, event)
	}
	if len(got) != len(want) {
		t.Fatalf("chronology = %v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("chronology = %v", got)
		}
	}
}

func applyWorkflowMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), string(readWorkflowMigration(t))); err != nil {
		t.Fatalf("apply workflow migration: %v", err)
	}
}

func readWorkflowMigration(t *testing.T) []byte {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000077_external_safety_intake.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	return migration
}
