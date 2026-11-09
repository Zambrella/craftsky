package safetyincident

import (
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
	"testing"
	"time"
)

func TestIntimateImageLinkedIntakeKeepsOriginalReceiptAndOverdueDeadline(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	received := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	now := received.Add(26 * time.Hour)
	intakeID := uuid.New()
	if _, err := pool.Exec(t.Context(), `INSERT INTO external_safety_intakes(id,reference,provider_message_ref,intake_kind,urgency,canonical_subject,owner_actor_id,attachment_status,received_at,created_at) VALUES($1,'synthetic-intake','synthetic-message','complaint','priority','at://did:plc:fixture/social.craftsky.feed.post/test','operator','none',$2,$3)`, intakeID, received, now); err != nil {
		t.Fatal(err)
	}
	policy, err := NewDeadlinePolicy(map[WorkflowClass]DeadlineTarget{WorkflowIntimateImage: {ResponseWithin: 24 * time.Hour}})
	if err != nil {
		t.Fatal(err)
	}
	service := NewWorkflowService(pool, policy, func() time.Time { return now })
	id, deadline, err := service.OpenIntimateImage(t.Context(), IntimateImageCommand{IntakeID: intakeID, ReceivedAt: now, Actor: Actor{ID: "operator:fixture", Role: RoleSafetyAdministrator}, StandingCode: "synthetic-depicted-person", DeclarationsCode: "synthetic-good-faith"})
	if err != nil {
		t.Fatal(err)
	}
	if !deadline.DueAt.Equal(received.Add(24*time.Hour)) || !deadline.DueAt.Before(now) {
		t.Fatalf("late intake reset deadline: %v", deadline.DueAt)
	}
	var saved time.Time
	if err := pool.QueryRow(t.Context(), `SELECT received_at FROM safety_workflows WHERE id=$1`, id).Scan(&saved); err != nil || !saved.Equal(received) {
		t.Fatalf("receipt: %v %v", saved, err)
	}
}
