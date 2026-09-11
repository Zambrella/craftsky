package moderation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestAppealCorrespondenceRequiresConfirmationAndUsesOneLifecycle(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	owner := syntax.DID("did:plc:appeal")
	caseRow := seedAdjudicationCase(t, pool, store, "appeal-report", owner)
	decision, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "appeal-decision-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reference, _ := FormatCaseReference(caseRow.ID)
	correspondence := AppealCorrespondence{
		CaseReference: reference, SourceSystem: "email", ReplayID: "message-00000001",
		SenderReference: "arbitrary-sender@example.invalid", ReceivedAt: now.Add(time.Minute),
	}
	firstCorrespondence, err := store.RecordAppealCorrespondence(ctx, correspondence)
	if err != nil {
		t.Fatal(err)
	}
	if replay, err := store.RecordAppealCorrespondence(ctx, correspondence); err != nil || !replay.Replayed {
		t.Fatalf("correspondence replay = %+v, %v", replay, err)
	}
	correspondence.ReplayID = "message-00000002"
	correspondence.SenderReference = "different-sender@example.invalid"
	if _, err := store.RecordAppealCorrespondence(ctx, correspondence); err != nil {
		t.Fatal(err)
	}
	var appeals, correspondenceRows int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM moderation_appeals),(SELECT count(*) FROM moderation_appeal_correspondence)`).Scan(&appeals, &correspondenceRows); err != nil {
		t.Fatal(err)
	}
	if appeals != 0 || correspondenceRows != 2 {
		t.Fatalf("appeals/correspondence before confirmation = %d/%d", appeals, correspondenceRows)
	}

	confirmed, err := service.ConfirmAppeal(ctx, AppealCommand{
		CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "retool", ReplayID: "appeal-confirm-001",
		ActorID: "moderator", CorrespondenceID: firstCorrespondence.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmAppeal(ctx, AppealCommand{
		CaseID: caseRow.ID, ExpectedRevision: confirmed.Revision, SourceSystem: "retool", ReplayID: "appeal-confirm-002",
		ActorID: "moderator",
	}); !errors.Is(err, ErrInvalidAppealTransition) {
		t.Fatalf("second appeal confirmation error = %v", err)
	}
	resolved, err := service.ResolveAppeal(ctx, AppealResolutionCommand{
		CaseID: caseRow.ID, ExpectedRevision: confirmed.Revision, SourceSystem: "retool", ReplayID: "appeal-uphold-0001",
		ActorID: "moderator", Outcome: AppealStatusUpheld, Rationale: "decision remains supported",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Revision != 3 {
		t.Fatalf("appeal result revision = %d, want 3", resolved.Revision)
	}
	var status, caseState string
	var standingCount, notices int
	if err := pool.QueryRow(ctx, `SELECT a.status,c.state,s.active_strike_count,
		(SELECT count(*) FROM moderation_notification_intents WHERE recipient_did=$2)
		FROM moderation_appeals a JOIN moderation_cases c ON c.id=a.case_id
		JOIN moderation_account_standings s ON s.owner_did=c.owner_did WHERE a.case_id=$1`, caseRow.ID, owner).Scan(&status, &caseState, &standingCount, &notices); err != nil {
		t.Fatal(err)
	}
	if status != "upheld" || caseState != "resolved" || standingCount != 0 || notices != 0 {
		t.Fatalf("appeal state=%s case=%s standing=%d notices=%d", status, caseState, standingCount, notices)
	}
}

func TestChangedAppealRecordsEffectChangeWithoutReopening(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	service := NewService(store, testVisibilityWriter{}, time.Now)
	owner := syntax.DID("did:plc:changed")
	caseRow := seedAdjudicationCase(t, pool, store, "changed-appeal-report", owner)
	decision := resolveStrike(t, service, caseRow, "changed-decision-01")
	confirmed, err := service.ConfirmAppeal(ctx, AppealCommand{
		CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "retool", ReplayID: "changed-confirm-01", ActorID: "moderator",
	})
	if err != nil {
		t.Fatal(err)
	}
	appeal, err := service.ResolveAppeal(ctx, AppealResolutionCommand{
		CaseID: caseRow.ID, ExpectedRevision: confirmed.Revision, SourceSystem: "retool", ReplayID: "changed-resolve-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Outcome: AppealStatusChanged,
		Negate: []EffectType{EffectStrike}, Rationale: "appeal changed decision",
	})
	if err != nil {
		t.Fatal(err)
	}
	if appeal.Revision != 3 {
		t.Fatalf("changed appeal revision = %d, want 3", appeal.Revision)
	}
	assertStoredStanding(t, pool, owner, 0, false, false, false)
	var state string
	var appealEvents, effectEvents int
	if err := pool.QueryRow(ctx, `SELECT state,
		(SELECT count(*) FROM moderation_case_events WHERE case_id=$1 AND event_type IN ('appealConfirmed','appealResolved')),
		(SELECT count(*) FROM moderation_effect_events WHERE case_id=$1 AND action='negate')
		FROM moderation_cases WHERE id=$1`, caseRow.ID).Scan(&state, &appealEvents, &effectEvents); err != nil {
		t.Fatal(err)
	}
	if state != "resolved" || appealEvents != 2 || effectEvents != 1 {
		t.Fatalf("state/appeal events/effect events = %s/%d/%d", state, appealEvents, effectEvents)
	}
}

func TestNoActionCaseCannotCreateAppealLifecycle(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	service := NewService(store, testVisibilityWriter{}, time.Now)
	caseRow := seedAdjudicationCase(t, pool, store, "no-action-appeal-report", syntax.DID("did:plc:no-action-appeal"))
	decision, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "no-action-appeal-decision",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionNoAction},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmAppeal(ctx, AppealCommand{
		CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "retool",
		ReplayID: "no-action-appeal-confirm", ActorID: "moderator",
	}); !errors.Is(err, ErrInvalidAppealTransition) {
		t.Fatalf("no-action appeal confirmation error = %v", err)
	}
	var appeals, events int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_appeals WHERE case_id=$1),
		(SELECT count(*) FROM moderation_case_events WHERE case_id=$1)`, caseRow.ID).Scan(&appeals, &events); err != nil {
		t.Fatal(err)
	}
	if appeals != 0 || events != 1 {
		t.Fatalf("no-action appeals/events = %d/%d, want 0/1 decision event only", appeals, events)
	}
}
