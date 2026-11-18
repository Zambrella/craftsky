package moderation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/testdb"
)

func TestChangedAppealCanReverseSevereSuspensionWithoutClearingOtherConsequences(t *testing.T) {
	for _, scenario := range []struct {
		name        string
		otherSevere bool
		strikes     int
		keepWarning bool
	}{
		{name: "severe only"},
		{name: "another case still suspends", otherSevere: true},
		{name: "strike threshold still suspends", strikes: 3},
		{name: "unselected warning remains", keepWarning: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL)
			ctx := context.Background()
			store := NewStore(pool)
			now := time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC)
			service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
			owner := syntax.DID("did:plc:severe-appeal")
			caseRow := seedAdjudicationCase(t, pool, store, "severe-appeal-report", owner)
			effects := []EffectType{EffectSevereSuspension}
			if scenario.keepWarning {
				effects = append(effects, EffectFormalWarning)
			}
			decision, err := service.ResolveCase(ctx, TrustedCommand{
				CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "moderation-console", ReplayID: "severe-appeal-decision",
				ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
				Decision: Decision{Disposition: DispositionViolation, Reason: ReasonChildSafety, Evidence: "Synthetic reviewed facts", SeverityRationale: "Synthetic severe decision", Consequences: effects},
			})
			if err != nil {
				t.Fatal(err)
			}
			if scenario.otherSevere {
				other := seedAdjudicationCase(t, pool, store, "other-severe-report", owner)
				_, err := service.ResolveCase(ctx, TrustedCommand{
					CaseID: other.ID, ExpectedRevision: 0, SourceSystem: "moderation-console", ReplayID: "other-severe-decision",
					ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
					Decision: Decision{Disposition: DispositionViolation, Reason: ReasonChildSafety, Evidence: "Separate synthetic facts", SeverityRationale: "Separate synthetic severe decision", Consequences: []EffectType{EffectSevereSuspension}},
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			for i := range scenario.strikes {
				strike := seedAdjudicationCase(t, pool, store, fmt.Sprintf("separate-strike-%d", i), owner)
				resolveStrike(t, service, strike, fmt.Sprintf("separate-strike-decision-%d", i))
			}
			confirmed, err := service.ConfirmAppeal(ctx, AppealCommand{
				CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "moderation-console", ReplayID: "severe-appeal-confirm", ActorID: "moderator",
			})
			if err != nil {
				t.Fatal(err)
			}
			command := AppealResolutionCommand{
				CaseID: caseRow.ID, ExpectedRevision: confirmed.Revision, SourceSystem: "moderation-console", ReplayID: "severe-appeal-resolve",
				ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Outcome: AppealStatusChanged,
				Negate: []EffectType{EffectSevereSuspension}, Rationale: "Review supports reversing only this severe suspension",
			}
			result, err := service.ResolveAppeal(ctx, command)
			if err != nil {
				t.Fatal(err)
			}
			if result.Revision != 3 {
				t.Fatalf("appeal revision = %d, want 3", result.Revision)
			}
			if replay, err := service.ResolveAppeal(ctx, command); err != nil || !replay.Replayed || replay.EventID != result.EventID {
				t.Fatalf("exact replay = %+v, %v", replay, err)
			}
			assertStoredStanding(t, pool, owner, scenario.strikes, scenario.strikes >= 3, scenario.otherSevere, scenario.otherSevere || scenario.strikes >= 3)
			var status, rationale string
			var activeSevere, activeWarning, appealEvents, negations int
			if err := pool.QueryRow(ctx, `SELECT a.status,a.resolution_rationale,
				(SELECT count(*) FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type='severeSuspension'),
				(SELECT count(*) FROM moderation_active_case_effects WHERE case_id=$1 AND effect_type='formalWarning'),
				(SELECT count(*) FROM moderation_case_events WHERE case_id=$1 AND event_type='appealResolved'),
				(SELECT count(*) FROM moderation_effect_events WHERE case_id=$1 AND effect_type='severeSuspension' AND action='negate')
				FROM moderation_appeals a WHERE a.case_id=$1`, caseRow.ID).Scan(&status, &rationale, &activeSevere, &activeWarning, &appealEvents, &negations); err != nil {
				t.Fatal(err)
			}
			wantWarning := 0
			if scenario.keepWarning {
				wantWarning = 1
			}
			if status != "changed" || rationale != command.Rationale || activeSevere != 0 || activeWarning != wantWarning || appealEvents != 1 || negations != 1 {
				t.Fatalf("appeal status/rationale=%s/%q effects=%d/%d events=%d/%d", status, rationale, activeSevere, activeWarning, appealEvents, negations)
			}
		})
	}
}
