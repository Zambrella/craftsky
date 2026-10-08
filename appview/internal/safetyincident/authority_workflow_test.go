package safetyincident

import (
	"context"
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

func TestThreatAndAuthorityWorkflowRejectsFraudAndMinimizesDisclosure(t *testing.T) {
	pool := testdb.WithSchema(t, workflowPreStateDDL)
	applyWorkflowMigration(t, pool)
	ctx := context.Background()
	now := time.Date(2030, 4, 4, 14, 0, 0, 0, time.UTC)
	policy, err := NewDeadlinePolicy(nil)
	if err != nil {
		t.Fatal(err)
	}
	service := NewWorkflowService(pool, policy, func() time.Time { return now })
	if _, err := service.OpenCredibleThreat(ctx, CredibleThreat{
		ReceivedAt: now, Actor: Actor{ID: "moderator", Role: RoleModerator}, PreservationCode: "safeIdentifiersPreserved",
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("moderator opened credible-threat workflow: %v", err)
	}
	if _, err := service.OpenAuthorityRequest(ctx, AuthorityRequest{
		ReceivedAt: now, Actor: Actor{ID: "admin", Role: RoleSafetyAdministrator, Permissions: map[Permission]bool{}}, RequesterReference: "request",
	}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("administrator without explicit permission opened authority workflow: %v", err)
	}

	threatID, err := service.OpenCredibleThreat(ctx, CredibleThreat{
		ReceivedAt: now.Add(-time.Minute), Actor: Actor{ID: "on-call-owner", Role: RoleSafetyAdministrator}, PreservationCode: "safeIdentifiersPreserved",
	})
	if err != nil {
		t.Fatal(err)
	}
	var urgency string
	var threatEvents int
	if err := pool.QueryRow(ctx, `SELECT urgency,(SELECT count(*) FROM safety_workflow_events WHERE workflow_id=$1) FROM safety_workflows WHERE id=$1`, threatID).Scan(&urgency, &threatEvents); err != nil {
		t.Fatal(err)
	}
	if urgency != "immediate" || threatEvents != 2 {
		t.Fatalf("threat urgency/events = %s/%d", urgency, threatEvents)
	}

	fraudID, err := service.OpenAuthorityRequest(ctx, AuthorityRequest{
		ReceivedAt: now, Actor: Actor{ID: "safety-admin", Role: RoleSafetyAdministrator}, RequesterReference: "synthetic-request-fraud",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyAuthority(ctx, fraudID, Actor{ID: "safety-admin", Role: RoleSafetyAdministrator}, "official-channel-failed", false); err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveAndDisclose(ctx, fraudID, Actor{ID: "legal-approver", Role: RoleSafetyAdministrator}, "process-1", []string{"accountDid"}); !errors.Is(err, ErrAuthorityNotVerified) {
		t.Fatalf("fraud disclosure error = %v", err)
	}
	if err := service.CloseAuthorityRequest(ctx, fraudID, Actor{ID: "reviewer", Role: RoleSafetyAdministrator}, "fraudRejectedAndReviewed"); err != nil {
		t.Fatal(err)
	}

	genuineID, err := service.OpenAuthorityRequest(ctx, AuthorityRequest{
		ReceivedAt: now, Actor: Actor{ID: "safety-admin", Role: RoleSafetyAdministrator}, RequesterReference: "synthetic-request-genuine",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyAuthority(ctx, genuineID, Actor{ID: "verification-actor", Role: RoleSafetyAdministrator}, "independent-official-channel", true); err != nil {
		t.Fatal(err)
	}
	if err := service.ApproveAndDisclose(ctx, genuineID, Actor{ID: "legal-approver", Role: RoleSafetyAdministrator}, "lawful-process-reference", []string{"credentials"}); !errors.Is(err, ErrLawfulApprovalRequired) {
		t.Fatalf("credential disclosure error = %v", err)
	}
	if err := service.ApproveAndDisclose(ctx, genuineID, Actor{ID: "legal-approver", Role: RoleSafetyAdministrator}, "lawful-process-reference", []string{"recordUri", "accountDid", "recordUri"}); err != nil {
		t.Fatal(err)
	}
	if err := service.CloseAuthorityRequest(ctx, genuineID, Actor{ID: "independent-reviewer", Role: RoleSafetyAdministrator}, "postIncidentComplete"); err != nil {
		t.Fatal(err)
	}

	var state, verification string
	var fields []string
	var reviewedAt *time.Time
	var events int
	if err := pool.QueryRow(ctx, `SELECT w.state,a.verification_state,a.disclosed_fields,a.reviewed_at,
		(SELECT count(*) FROM safety_workflow_events WHERE workflow_id=w.id)
		FROM safety_workflows w JOIN safety_authority_requests a ON a.workflow_id=w.id WHERE w.id=$1`, genuineID).
		Scan(&state, &verification, &fields, &reviewedAt, &events); err != nil {
		t.Fatal(err)
	}
	if state != "resolved" || verification != "verified" || reviewedAt == nil || events != 4 {
		t.Fatalf("authority state=%s verification=%s reviewed=%v events=%d", state, verification, reviewedAt, events)
	}
	if len(fields) != 2 || fields[0] != "accountDid" || fields[1] != "recordUri" {
		t.Fatalf("disclosed fields = %v", fields)
	}
}
