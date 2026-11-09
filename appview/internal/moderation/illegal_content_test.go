package moderation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func illegalContentDecision() Decision {
	return Decision{Disposition: DispositionViolation, Reason: ReasonIllegalContent, Evidence: "private-evidence-canary", UserSafeDetail: "Other illegal content: reviewed finding", LegalClassification: LegalOtherIllegalContent, IllegalContentRationale: "private-legal-basis-canary", SeverityRationale: "private-severity-canary", Consequences: []EffectType{EffectStrike, EffectSevereSuspension}}
}

func TestIllegalContentRequiresSeparateSupportedHumanAssessment(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Decision)
	}{
		{"evidence", func(d *Decision) { d.Evidence = " \t" }},
		{"legal rationale", func(d *Decision) { d.IllegalContentRationale = " \t" }},
		{"safe explanation", func(d *Decision) { d.UserSafeDetail = " \t" }},
		{"classification absent", func(d *Decision) { d.LegalClassification = "" }},
		{"classification unassessed", func(d *Decision) { d.LegalClassification = LegalUnassessed }},
		{"classification incompatible", func(d *Decision) { d.LegalClassification = LegalCredibleThreat }},
		{"severe rationale", func(d *Decision) { d.SeverityRationale = " \t" }},
		{"generic other", func(d *Decision) { d.Reason = ReasonOther }},
		{"unrelated reason", func(d *Decision) { d.Reason = ReasonHate }},
		{"no action", func(d *Decision) {
			*d = Decision{Disposition: DispositionNoAction, IllegalContentRationale: d.IllegalContentRationale}
		}},
	}
	if err := ValidateDecision(illegalContentDecision()); err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := illegalContentDecision()
			test.change(&d)
			if err := ValidateDecision(d); !errors.Is(err, ErrInvalidDecision) {
				t.Fatalf("error=%v", err)
			}
		})
	}
	for _, reason := range []Reason{ReasonFraudOrScam, ReasonSelfHarm} {
		d := Decision{Disposition: DispositionViolation, Reason: reason, Evidence: "human-reviewed evidence", Consequences: []EffectType{EffectSevereSuspension}}
		if err := ValidateDecision(d); !errors.Is(err, ErrInvalidDecision) {
			t.Fatalf("%s missing severity accepted: %v", reason, err)
		}
		d.SeverityRationale = "serious deliberate harm; immediate restriction proportionate"
		if err := ValidateDecision(d); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIllegalContentPersistsOnProductionSchemaWithPrivateRationaleReplayAndAppeal(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	store := NewStore(pool)
	id := uuid.New()
	owner := syntax.DID("did:plc:owner")
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot) VALUES($1,$2,'account',$2,$2,'{"type":"account","did":"did:plc:owner"}')`, id, owner); err != nil {
		t.Fatal(err)
	}
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return time.Date(2026, 11, 9, 12, 0, 0, 0, time.UTC) })
	command := TrustedCommand{CaseID: id, SourceSystem: "moderation-console", ReplayID: "synthetic-illegal-1", ActorID: "operator:fixture", SourceDID: syntax.DID("did:plc:labeler"), Decision: illegalContentDecision()}
	first, err := service.ResolveCase(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.ResolveCase(ctx, command)
	if err != nil || !replay.Replayed || replay.EventID != first.EventID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	changed := command
	changed.Decision.IllegalContentRationale = "changed legal basis"
	if _, err := service.ResolveCase(ctx, changed); !errors.Is(err, ErrReplayConflict) {
		t.Fatalf("changed rationale replay=%v", err)
	}
	changed = command
	changed.ReplayID = "synthetic-illegal-stale"
	if _, err := service.ResolveCase(ctx, changed); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision=%v", err)
	}
	ref, _ := FormatCaseReference(id)
	admin, err := store.AdminDetail(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(admin.Decisions) != 1 || admin.Decisions[0].IllegalContentRationale != command.Decision.IllegalContentRationale || admin.Decisions[0].Reason != ReasonIllegalContent {
		t.Fatalf("admin=%+v", admin.Decisions)
	}
	history, err := store.OwnerHistory(ctx, owner, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(history)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"reason":"illegal_content"`) || !strings.Contains(string(wire), command.Decision.UserSafeDetail) {
		t.Fatalf("owner missing safe fields: %s", wire)
	}
	for _, secret := range []string{"private-evidence-canary", "private-legal-basis-canary", "private-severity-canary", "illegalContentRationale", "legalClassification"} {
		if strings.Contains(string(wire), secret) {
			t.Fatalf("owner leaked %s", secret)
		}
	}
	// Bypass application validation to verify the new production constraint too.
	for _, assignment := range []string{"illegal_content_rationale=NULL", "legal_classification=NULL", "user_safe_detail=' '", "internal_evidence_notes=' '", "reason='other'"} {
		if _, err := pool.Exec(ctx, `UPDATE moderation_decisions SET `+assignment+` WHERE case_id=$1`, id); err == nil {
			t.Fatalf("database accepted %s", assignment)
		}
	}
	confirmed, err := service.ConfirmAppeal(ctx, AppealCommand{CaseID: id, ExpectedRevision: 1, SourceSystem: command.SourceSystem, ReplayID: "synthetic-confirm-appeal", ActorID: command.ActorID})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ResolveAppeal(ctx, AppealResolutionCommand{CaseID: id, ExpectedRevision: confirmed.Revision, SourceSystem: command.SourceSystem, ReplayID: "synthetic-change-appeal", ActorID: command.ActorID, SourceDID: command.SourceDID, Outcome: AppealStatusChanged, Negate: []EffectType{EffectStrike, EffectSevereSuspension}, Rationale: "review overturned finding"})
	if err != nil {
		t.Fatal(err)
	}
	standing, err := store.Standing(ctx, owner)
	if err != nil || standing.ActiveStrikeCount != 0 || standing.Suspended {
		t.Fatalf("standing after appeal=%+v err=%v", standing, err)
	}
}
