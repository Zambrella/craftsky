package moderation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestThirdStrikeAndSevereBasesTransitionIndependently(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))
	owner := syntax.DID("did:plc:threshold")
	for index := 1; index <= 3; index++ {
		caseRow := seedAdjudicationCase(t, pool, store, "threshold-report-"+string(rune('0'+index)), owner)
		resolveStrike(t, service, caseRow, "threshold-replay-000"+string(rune('0'+index)))
	}
	assertStoredStanding(t, pool, owner, 3, true, false, true)

	severeOwner := syntax.DID("did:plc:severe")
	severeCase := seedAdjudicationCase(t, pool, store, "severe-report", severeOwner)
	result, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: severeCase.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "severe-only-00001",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence",
			SeverityRationale: "immediate risk", Consequences: []EffectType{EffectSevereSuspension}},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertStoredStanding(t, pool, severeOwner, 0, false, true, true)
	strikeCase := seedAdjudicationCase(t, pool, store, "severe-strike-report", severeOwner)
	resolveStrike(t, service, strikeCase, "severe-strike-0001")
	assertStoredStanding(t, pool, severeOwner, 1, false, true, true)
	if _, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: severeCase.ID, ExpectedRevision: result.Revision, SourceSystem: "retool",
		ReplayID: "restore-severe-001", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Negate: []EffectType{EffectSevereSuspension}, Rationale: "appeal evidence accepted",
	}); err != nil {
		t.Fatal(err)
	}
	assertStoredStanding(t, pool, severeOwner, 1, false, false, false)

	var severeStrikes, notices int
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_case_strikes WHERE owner_did=$1),
		(SELECT count(*) FROM moderation_notification_intents WHERE recipient_did=$1)`, severeOwner).Scan(&severeStrikes, &notices); err != nil {
		t.Fatal(err)
	}
	if severeStrikes != 1 || notices != 3 {
		t.Fatalf("independent severe strikes/notices = %d/%d, want 1/3", severeStrikes, notices)
	}

	combinedOwner := syntax.DID("did:plc:combined")
	combinedCase := seedAdjudicationCase(t, pool, store, "combined-report", combinedOwner)
	if _, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: combinedCase.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "combined-effects-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonHarassment, Evidence: "evidence",
			SeverityRationale: "immediate risk", Consequences: []EffectType{EffectStrike, EffectSevereSuspension}},
	}); err != nil {
		t.Fatal(err)
	}
	assertStoredStanding(t, pool, combinedOwner, 1, false, true, true)
}

func TestExplicitSevereRestorationTargetsActiveEffectAndAppendsAuditEvent(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))
	owner := syntax.DID("did:plc:explicit-restoration")
	caseRow := seedAdjudicationCase(t, pool, store, "explicit-restoration-report", owner)
	resolved, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "explicit-restoration-decision",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence",
			SeverityRationale: "immediate risk", Consequences: []EffectType{EffectSevereSuspension}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var effectID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT logical_effect_id FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type='severeSuspension'`, caseRow.ID).Scan(&effectID); err != nil {
		t.Fatal(err)
	}

	if _, err := service.RestoreSevereSuspension(ctx, SevereRestorationCommand{
		CaseID: caseRow.ID, EffectID: uuid.New(), ExpectedRevision: resolved.Revision,
		SourceSystem: "retool", ReplayID: "explicit-restoration-wrong-effect", ActorID: "moderator",
		Rationale: "review completed",
	}); !errors.Is(err, ErrInvalidEffectChange) {
		t.Fatalf("wrong effect restoration error = %v", err)
	}

	now = now.Add(time.Hour)
	command := SevereRestorationCommand{
		CaseID: caseRow.ID, EffectID: effectID, ExpectedRevision: resolved.Revision,
		SourceSystem: "retool", ReplayID: "explicit-restoration-correct", ActorID: "moderator",
		Rationale: "review completed",
	}
	restored, err := service.RestoreSevereSuspension(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.RestoreSevereSuspension(ctx, command)
	if err != nil || !replay.Replayed || replay.EventID != restored.EventID {
		t.Fatalf("restoration replay = %+v, err=%v", replay, err)
	}
	assertStoredStanding(t, pool, owner, 0, false, false, false)

	var eventType, action string
	var storedEffectID uuid.UUID
	var activeEffects, notices int
	if err := pool.QueryRow(ctx, `SELECT event.event_type,effect.action,effect.logical_effect_id,
		(SELECT count(*) FROM moderation_active_case_effects WHERE case_id=$1),
		(SELECT count(*) FROM moderation_notification_intents WHERE case_id=$1)
		FROM moderation_case_events event
		JOIN moderation_effect_events effect ON effect.case_event_id=event.id
		WHERE event.id=$2`, caseRow.ID, restored.EventID).Scan(&eventType, &action, &storedEffectID, &activeEffects, &notices); err != nil {
		t.Fatal(err)
	}
	if eventType != "severeRestored" || action != "restore" || storedEffectID != effectID || activeEffects != 0 || notices != 2 {
		t.Fatalf("restoration event = %s/%s/%s active=%d notices=%d", eventType, action, storedEffectID, activeEffects, notices)
	}
}

func TestThirdStrikeAndExpirySerializeInBothLockOrders(t *testing.T) {
	for _, decisionFirst := range []bool{true, false} {
		name := "expiry-before-decision"
		if decisionFirst {
			name = "decision-before-expiry"
		}
		t.Run(name, func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			store := NewStore(pool)
			issued := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
			seedNow := issued
			seedService := NewService(store, testVisibilityWriter{}, func() time.Time { return seedNow }, WithNotificationWriter(testNotificationWriter{}))
			owner := syntax.DID("did:plc:" + name)
			var expiringCase Case
			for index := 1; index <= 2; index++ {
				caseRow := seedAdjudicationCase(t, pool, store, name+"-active-"+string(rune('0'+index)), owner)
				if index == 1 {
					expiringCase = caseRow
				}
				resolveStrike(t, seedService, caseRow, name+"-existing-000"+string(rune('0'+index)))
				seedNow = seedNow.Add(time.Hour)
			}
			third := seedAdjudicationCase(t, pool, store, name+"-third", owner)
			runAt := StrikeDeadline(issued).Add(59 * time.Minute)
			service := NewService(store, testVisibilityWriter{}, func() time.Time { return runAt }, WithNotificationWriter(testNotificationWriter{}))
			processor := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return runAt })
			command := TrustedCommand{
				CaseID: third.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: name + "-third-00001",
				ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
				Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectStrike}},
			}
			decision := func() standingRaceResult {
				result, err := service.ResolveCase(ctx, command)
				return standingRaceResult{command: result, err: err}
			}
			expiry := func() standingRaceResult {
				processed, err := processor.ProcessDue(ctx, 100)
				return standingRaceResult{processed: processed, err: err}
			}

			baseline := readStandingRaceSnapshot(t, pool, owner)
			var decisionResult, expiryResult standingRaceResult
			if decisionFirst {
				decisionResult, expiryResult = runStandingRace(t, ctx, pool, owner, baseline, decision, expiry)
			} else {
				expiryResult, decisionResult = runStandingRace(t, ctx, pool, owner, baseline, expiry, decision)
			}
			if decisionResult.err != nil {
				t.Fatalf("decision: %v", decisionResult.err)
			}
			if expiryResult.err != nil || expiryResult.processed != 1 {
				t.Fatalf("expiry = %d, %v", expiryResult.processed, expiryResult.err)
			}
			assertStoredStanding(t, pool, owner, 2, false, false, false)
			assertStrikeTerminalState(t, pool, expiringCase.ID, 1, 0, 1, 1, 0, 1, false, true, false, 2)
			assertStrikeTerminalState(t, pool, third.ID, 1, 0, 0, 1, 0, 0, true, false, false, 1)

			replay, err := service.ResolveCase(ctx, command)
			if err != nil || !replay.Replayed || replay.EventID != decisionResult.command.EventID {
				t.Fatalf("decision replay = %+v, err=%v", replay, err)
			}
			if processed, err := processor.ProcessDue(ctx, 100); err != nil || processed != 0 {
				t.Fatalf("expiry retry = %d, %v", processed, err)
			}
			wantNotices := 3
			if decisionFirst {
				wantNotices = 4
			}
			assertStandingRaceSnapshot(t, pool, owner, standingRaceSnapshot{
				events: 4, decisions: 3, effectEvents: 4, activeEffects: 2, strikes: 3,
				notices: wantNotices, caseRevisions: 4, standingRevision: 4,
				activeStrikeCount: 2, thresholdSuspended: false, severeSuspended: false,
			})
		})
	}
}

func TestStrikeReversalAndExpirySerializeInBothLockOrders(t *testing.T) {
	for _, reversalFirst := range []bool{true, false} {
		name := "expiry-before-reversal"
		if reversalFirst {
			name = "reversal-before-expiry"
		}
		t.Run(name, func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			store := NewStore(pool)
			issued := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
			seedNow := issued
			seedService := NewService(store, testVisibilityWriter{}, func() time.Time { return seedNow }, WithNotificationWriter(testNotificationWriter{}))
			owner := syntax.DID("did:plc:" + name)
			var target Case
			for index := 1; index <= 3; index++ {
				caseRow := seedAdjudicationCase(t, pool, store, name+"-active-"+string(rune('0'+index)), owner)
				if index == 1 {
					target = caseRow
				}
				resolveStrike(t, seedService, caseRow, name+"-existing-000"+string(rune('0'+index)))
				seedNow = seedNow.Add(time.Hour)
			}
			runAt := StrikeDeadline(issued).Add(59 * time.Minute)
			service := NewService(store, testVisibilityWriter{}, func() time.Time { return runAt }, WithNotificationWriter(testNotificationWriter{}))
			processor := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return runAt })
			command := EffectChangeCommand{
				CaseID: target.ID, ExpectedRevision: 1, SourceSystem: "retool", ReplayID: name + "-negate-00001",
				ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Negate: []EffectType{EffectStrike}, Rationale: "strike reversed",
			}
			reversal := func() standingRaceResult {
				result, err := service.ChangeEffects(ctx, command)
				return standingRaceResult{command: result, err: err}
			}
			expiry := func() standingRaceResult {
				processed, err := processor.ProcessDue(ctx, 100)
				return standingRaceResult{processed: processed, err: err}
			}

			baseline := readStandingRaceSnapshot(t, pool, owner)
			var reversalResult, expiryResult standingRaceResult
			if reversalFirst {
				reversalResult, expiryResult = runStandingRace(t, ctx, pool, owner, baseline, reversal, expiry)
			} else {
				expiryResult, reversalResult = runStandingRace(t, ctx, pool, owner, baseline, expiry, reversal)
			}
			assertStoredStanding(t, pool, owner, 2, false, false, false)
			if reversalFirst {
				if reversalResult.err != nil {
					t.Fatalf("reversal: %v", reversalResult.err)
				}
				if expiryResult.err != nil || expiryResult.processed != 0 {
					t.Fatalf("losing expiry = %d, %v", expiryResult.processed, expiryResult.err)
				}
				assertStrikeTerminalState(t, pool, target.ID, 1, 1, 0, 1, 1, 0, false, false, true, 2)
				replay, err := service.ChangeEffects(ctx, command)
				if err != nil || !replay.Replayed || replay.EventID != reversalResult.command.EventID {
					t.Fatalf("reversal replay = %+v, err=%v", replay, err)
				}
			} else {
				if expiryResult.err != nil || expiryResult.processed != 1 {
					t.Fatalf("winning expiry = %d, %v", expiryResult.processed, expiryResult.err)
				}
				if !errors.Is(reversalResult.err, ErrRevisionConflict) {
					t.Fatalf("losing reversal error = %v, want revision conflict", reversalResult.err)
				}
				assertStrikeTerminalState(t, pool, target.ID, 1, 0, 1, 1, 0, 1, false, true, false, 2)
				if _, err := service.ChangeEffects(ctx, command); !errors.Is(err, ErrRevisionConflict) {
					t.Fatalf("losing reversal retry error = %v, want revision conflict", err)
				}
			}
			if processed, err := processor.ProcessDue(ctx, 100); err != nil || processed != 0 {
				t.Fatalf("expiry retry = %d, %v", processed, err)
			}
			assertStandingRaceSnapshot(t, pool, owner, standingRaceSnapshot{
				events: 4, decisions: 3, effectEvents: 4, activeEffects: 2, strikes: 3,
				notices: 4, caseRevisions: 4, standingRevision: 4,
				activeStrikeCount: 2, thresholdSuspended: false, severeSuspended: false,
			})
		})
	}
}

func TestExpiryProcessorDoesNotLiftSevereSuspensionOrNotifyWithoutEnforcementChange(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	issued := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
	now := issued
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))
	owner := syntax.DID("did:plc:severe-expiry")
	caseRow := seedAdjudicationCase(t, pool, store, "severe-expiry-report", owner)
	if _, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "severe-expiry-decision",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence",
			SeverityRationale: "immediate risk", Consequences: []EffectType{EffectStrike, EffectSevereSuspension}},
	}); err != nil {
		t.Fatal(err)
	}
	now = StrikeDeadline(issued)
	processor := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return now })
	processed, err := processor.ProcessDue(ctx, 100)
	if err != nil || processed != 1 {
		t.Fatalf("ProcessDue = %d, %v", processed, err)
	}
	assertStoredStanding(t, pool, owner, 0, false, true, true)
	var notices int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM moderation_notification_intents WHERE recipient_did=$1`, owner).Scan(&notices); err != nil {
		t.Fatal(err)
	}
	if notices != 1 {
		t.Fatalf("notices = %d, want initial decision only", notices)
	}
}

func TestReappliedStrikeCanExpireAgainWithDistinctWorkerReplayIdentity(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	owner := syntax.DID("did:plc:repeat-expiry")
	caseRow := seedAdjudicationCase(t, pool, store, "repeat-expiry-report", owner)
	resolveStrike(t, service, caseRow, "repeat-expiry-decision")
	processor := NewExpiryProcessor(store, nil, func() time.Time { return now })
	now = StrikeDeadline(now)
	if processed, err := processor.ProcessDue(ctx, 100); err != nil || processed != 1 {
		t.Fatalf("first expiry = %d, %v", processed, err)
	}
	reapplied, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: 2, SourceSystem: "retool", ReplayID: "repeat-expiry-reapply",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Apply: []EffectType{EffectStrike}, Rationale: "new evidence",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reapplied.Revision != 3 {
		t.Fatalf("reapplied revision = %d", reapplied.Revision)
	}
	now = StrikeDeadline(now)
	if processed, err := processor.ProcessDue(ctx, 100); err != nil || processed != 1 {
		t.Fatalf("second expiry = %d, %v", processed, err)
	}
	var expiries int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM moderation_case_events WHERE case_id=$1 AND event_type='strikeExpired'`, caseRow.ID).Scan(&expiries); err != nil {
		t.Fatal(err)
	}
	if expiries != 2 {
		t.Fatalf("expiry events = %d, want 2", expiries)
	}
}

func TestPartialReversalAndStrikeReapplicationPreserveOneLogicalStrike(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))
	owner := syntax.DID("did:plc:partial")
	caseRow := seedAdjudicationCase(t, pool, store, "partial-report", owner)
	resolved, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "partial-decision-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence",
			SeverityRationale: "immediate risk", Consequences: []EffectType{EffectVisibilityTakedown, EffectStrike, EffectSevereSuspension}},
	})
	if err != nil {
		t.Fatal(err)
	}
	reversed, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: resolved.Revision, SourceSystem: "retool", ReplayID: "partial-reverse-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Negate: []EffectType{EffectStrike}, Rationale: "strike was excessive",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: reversed.Revision, SourceSystem: "retool", ReplayID: "partial-reapply-bad",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Apply: []EffectType{EffectStrike},
	}); !errors.Is(err, ErrInvalidEffectChange) {
		t.Fatalf("missing rationale error = %v", err)
	}
	now = now.Add(24 * time.Hour)
	reapplied, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: reversed.Revision, SourceSystem: "retool", ReplayID: "partial-reapply-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Apply: []EffectType{EffectStrike}, Rationale: "new verified evidence",
	})
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: reversed.Revision, SourceSystem: "retool", ReplayID: "partial-reapply-01",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Apply: []EffectType{EffectStrike}, Rationale: "new verified evidence",
	})
	if err != nil || !replay.Replayed || replay.EventID != reapplied.EventID {
		t.Fatalf("effect replay = %+v, err=%v", replay, err)
	}

	var strikes, activeEffects, outputs, notices int
	var issuedAt, dueAt time.Time
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_case_strikes WHERE case_id=$1),
		(SELECT count(*) FROM moderation_active_case_effects WHERE case_id=$1),
		(SELECT count(*) FROM moderation_outputs),
		(SELECT count(*) FROM moderation_notification_intents WHERE case_id=$1),
		issued_at,due_at FROM moderation_case_strikes WHERE case_id=$1`, caseRow.ID).Scan(&strikes, &activeEffects, &outputs, &notices, &issuedAt, &dueAt); err != nil {
		t.Fatal(err)
	}
	if strikes != 1 || activeEffects != 3 || outputs != 1 || notices != 3 || !issuedAt.Equal(now) || !dueAt.Equal(StrikeDeadline(now)) {
		t.Fatalf("strike/effects/outputs/notices=%d/%d/%d/%d issued=%s due=%s", strikes, activeEffects, outputs, notices, issuedAt, dueAt)
	}
	now = dueAt
	processor := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return now })
	if processed, err := processor.ProcessDue(ctx, 100); err != nil || processed != 1 {
		t.Fatalf("expire reapplied strike = %d, %v", processed, err)
	}
}

func TestExpiryProcessorKeepsProjectionUntilCommitAndPreservesSevereBasis(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	issued := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
	now := issued
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))
	owner := syntax.DID("did:plc:expiry")
	var firstCase Case
	for index := 1; index <= 3; index++ {
		caseRow := seedAdjudicationCase(t, pool, store, "expiry-report-"+string(rune('0'+index)), owner)
		if index == 1 {
			firstCase = caseRow
		}
		resolveStrike(t, service, caseRow, "expiry-decision-00"+string(rune('0'+index)))
		now = now.Add(time.Hour)
	}
	assertStoredStanding(t, pool, owner, 3, true, false, true)
	now = StrikeDeadline(issued)
	assertStoredStanding(t, pool, owner, 3, true, false, true)
	processor := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return now })
	processed, err := processor.ProcessDue(ctx, 100)
	if err != nil || processed != 1 {
		t.Fatalf("ProcessDue = %d, %v", processed, err)
	}
	assertStoredStanding(t, pool, owner, 2, false, false, false)
	processed, err = processor.ProcessDue(ctx, 100)
	if err != nil || processed != 0 {
		t.Fatalf("ProcessDue retry = %d, %v", processed, err)
	}
	var expiredAt *time.Time
	var notices int
	if err := pool.QueryRow(ctx, `SELECT expired_at,(SELECT count(*) FROM moderation_notification_intents WHERE recipient_did=$2) FROM moderation_case_strikes WHERE case_id=$1`, firstCase.ID, owner).Scan(&expiredAt, &notices); err != nil {
		t.Fatal(err)
	}
	if expiredAt == nil || notices != 4 {
		t.Fatalf("expiredAt/notices = %v/%d, want processed and 4", expiredAt, notices)
	}
}

func TestExpiryProcessorWaitsUntilDeadlineAndRecoversAfterRestart(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	issued := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
	now := issued
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))
	owner := syntax.DID("did:plc:expiry-restart")
	var expiringCase Case
	for index := 1; index <= 3; index++ {
		caseRow := seedAdjudicationCase(t, pool, store, "expiry-restart-report-"+string(rune('0'+index)), owner)
		if index == 1 {
			expiringCase = caseRow
		}
		resolveStrike(t, service, caseRow, "expiry-restart-decision-00"+string(rune('0'+index)))
		now = now.Add(time.Hour)
	}
	deadline := StrikeDeadline(issued)
	now = deadline.Add(-time.Microsecond)
	processor := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return now })
	if processed, err := processor.ProcessDue(ctx, 100); err != nil || processed != 0 {
		t.Fatalf("pre-deadline processing = %d, %v", processed, err)
	}
	assertStoredStanding(t, pool, owner, 3, true, false, true)

	// No processor runs at the deadline. A restarted worker observes the stored
	// projection and commits expiry before the one-hour convergence limit.
	now = deadline.Add(59 * time.Minute)
	restarted := NewExpiryProcessor(store, testNotificationWriter{}, func() time.Time { return now })
	if processed, err := restarted.ProcessDue(ctx, 100); err != nil || processed != 1 {
		t.Fatalf("restarted processing = %d, %v", processed, err)
	}
	assertStoredStanding(t, pool, owner, 2, false, false, false)
	if processed, err := restarted.ProcessDue(ctx, 100); err != nil || processed != 0 {
		t.Fatalf("restarted retry = %d, %v", processed, err)
	}
	var expiredAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT expired_at FROM moderation_case_strikes WHERE case_id=$1`, expiringCase.ID).Scan(&expiredAt); err != nil {
		t.Fatal(err)
	}
	if expiredAt == nil || !expiredAt.Equal(now) {
		t.Fatalf("effective expiry = %v, want %v", expiredAt, now)
	}
}

type standingRaceResult struct {
	command   CommandResult
	processed int
	err       error
}

type standingRaceSnapshot struct {
	events, decisions, effectEvents, activeEffects, strikes, notices int
	caseRevisions, standingRevision                                  int64
	activeStrikeCount                                                int
	thresholdSuspended, severeSuspended                              bool
}

func runStandingRace(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	owner syntax.DID,
	baseline standingRaceSnapshot,
	first, second func() standingRaceResult,
) (standingRaceResult, standingRaceResult) {
	t.Helper()
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	committed := false
	t.Cleanup(func() {
		if !committed {
			_ = blocker.Rollback(context.Background())
		}
	})
	if err := blocker.QueryRow(ctx, `SELECT owner_did FROM moderation_account_standings WHERE owner_did=$1 FOR UPDATE`, owner).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	var blockerXID string
	if err := blocker.QueryRow(ctx, `SELECT txid_current()::text`).Scan(&blockerXID); err != nil {
		t.Fatal(err)
	}

	firstDone := make(chan standingRaceResult, 1)
	go func() { firstDone <- first() }()
	waitForStandingLockWaiters(t, ctx, pool, blockerXID, 1)
	assertStandingRaceSnapshot(t, pool, owner, baseline)

	secondDone := make(chan standingRaceResult, 1)
	go func() { secondDone <- second() }()
	waitForStandingLockWaiters(t, ctx, pool, blockerXID, 2)
	assertStandingRaceSnapshot(t, pool, owner, baseline)

	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	committed = true
	return <-firstDone, <-secondDone
}

func waitForStandingLockWaiters(t *testing.T, ctx context.Context, pool *pgxpool.Pool, blockerXID string, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting int
		if err := pool.QueryRow(ctx, `SELECT count(DISTINCT pid) FROM pg_locks
			WHERE NOT granted AND (
				(locktype='transactionid' AND transactionid=$1::xid)
				OR (locktype='tuple' AND relation=to_regclass('moderation_account_standings'))
			)`, blockerXID).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("standing contenders waiting = %d, want at least %d", waiting, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func readStandingRaceSnapshot(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) standingRaceSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var snapshot standingRaceSnapshot
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_case_events event JOIN moderation_cases c ON c.id=event.case_id WHERE c.owner_did=$1),
		(SELECT count(*) FROM moderation_decisions decision JOIN moderation_cases c ON c.id=decision.case_id WHERE c.owner_did=$1),
		(SELECT count(*) FROM moderation_effect_events effect JOIN moderation_cases c ON c.id=effect.case_id WHERE c.owner_did=$1),
		(SELECT count(*) FROM moderation_active_case_effects effect JOIN moderation_cases c ON c.id=effect.case_id WHERE c.owner_did=$1),
		(SELECT count(*) FROM moderation_case_strikes WHERE owner_did=$1),
		(SELECT count(*) FROM moderation_notification_intents WHERE recipient_did=$1),
		(SELECT COALESCE(sum(revision),0) FROM moderation_cases WHERE owner_did=$1),
		standing.revision,standing.active_strike_count,standing.threshold_suspended,standing.severe_suspended
		FROM moderation_account_standings standing WHERE standing.owner_did=$1`, owner).Scan(
		&snapshot.events, &snapshot.decisions, &snapshot.effectEvents, &snapshot.activeEffects,
		&snapshot.strikes, &snapshot.notices, &snapshot.caseRevisions, &snapshot.standingRevision,
		&snapshot.activeStrikeCount, &snapshot.thresholdSuspended, &snapshot.severeSuspended,
	); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func assertStandingRaceSnapshot(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, want standingRaceSnapshot) {
	t.Helper()
	if got := readStandingRaceSnapshot(t, pool, owner); got != want {
		t.Fatalf("moderation write snapshot = %+v, want %+v", got, want)
	}
}

func assertStrikeTerminalState(
	t *testing.T,
	pool *pgxpool.Pool,
	caseID uuid.UUID,
	wantDecisions, wantChanges, wantExpiries, wantApplies, wantNegates, wantExpires int,
	wantActive, wantExpired, wantOverturned bool,
	wantRevision int,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var decisions, changes, expiries, applies, negates, expires, revision int
	var active, expired, overturned bool
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_case_events WHERE case_id=$1 AND event_type='decision'),
		(SELECT count(*) FROM moderation_case_events WHERE case_id=$1 AND event_type='effectsChanged'),
		(SELECT count(*) FROM moderation_case_events WHERE case_id=$1 AND event_type='strikeExpired'),
		(SELECT count(*) FROM moderation_effect_events WHERE case_id=$1 AND effect_type='strike' AND action='apply'),
		(SELECT count(*) FROM moderation_effect_events WHERE case_id=$1 AND effect_type='strike' AND action='negate'),
		(SELECT count(*) FROM moderation_effect_events WHERE case_id=$1 AND effect_type='strike' AND action='expire'),
		EXISTS(SELECT 1 FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type='strike'),
		strike.expired_at IS NOT NULL,strike.overturned_at IS NOT NULL,c.revision
		FROM moderation_case_strikes strike JOIN moderation_cases c ON c.id=strike.case_id WHERE strike.case_id=$1`, caseID).Scan(
		&decisions, &changes, &expiries, &applies, &negates, &expires,
		&active, &expired, &overturned, &revision,
	); err != nil {
		t.Fatal(err)
	}
	if decisions != wantDecisions || changes != wantChanges || expiries != wantExpiries ||
		applies != wantApplies || negates != wantNegates || expires != wantExpires ||
		active != wantActive || expired != wantExpired || overturned != wantOverturned || revision != wantRevision {
		t.Fatalf("case terminal state events=%d/%d/%d effects=%d/%d/%d active=%t expired=%t overturned=%t revision=%d",
			decisions, changes, expiries, applies, negates, expires, active, expired, overturned, revision)
	}
}

func resolveStrike(t *testing.T, service *Service, caseRow Case, replayID string) CommandResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: replayID,
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectStrike}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertStoredStanding(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, count int, threshold, severe, effective bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var gotCount int
	var gotThreshold, gotSevere, gotEffective bool
	if err := pool.QueryRow(ctx, `SELECT active_strike_count,threshold_suspended,severe_suspended,effective_suspended FROM moderation_account_standings WHERE owner_did=$1`, owner).Scan(&gotCount, &gotThreshold, &gotSevere, &gotEffective); err != nil {
		t.Fatal(err)
	}
	if gotCount != count || gotThreshold != threshold || gotSevere != severe || gotEffective != effective {
		t.Fatalf("standing = %d/%t/%t/%t, want %d/%t/%t/%t", gotCount, gotThreshold, gotSevere, gotEffective, count, threshold, severe, effective)
	}
}
