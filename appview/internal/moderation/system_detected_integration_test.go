package moderation

import (
	"context"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/safetyincident"
	"social.craftsky/appview/internal/testdb"
)

func TestHumanConfirmationCreatesSystemDetectedCaseWithoutReportAndUsesExistingEnforcement(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL+`
		CREATE TABLE safety_incidents(id UUID PRIMARY KEY,state TEXT NOT NULL,updated_at TIMESTAMPTZ NOT NULL);
		CREATE TABLE safety_incident_subjects(
			incident_id UUID NOT NULL REFERENCES safety_incidents(id), owner_did TEXT NOT NULL,
			subject_uri TEXT NOT NULL,source_cid TEXT NOT NULL,subject_kind TEXT NOT NULL,image_slot TEXT NOT NULL,
			PRIMARY KEY(incident_id,subject_uri,image_slot));
		CREATE TABLE safety_incident_events(
			id UUID PRIMARY KEY,incident_id UUID NOT NULL REFERENCES safety_incidents(id),event_type TEXT NOT NULL,
			actor_id TEXT,source_system TEXT,replay_id TEXT,created_at TIMESTAMPTZ NOT NULL,
			UNIQUE(source_system,replay_id));`)
	ctx := context.Background()
	now := time.Date(2030, 9, 22, 21, 0, 0, 0, time.UTC)
	incidentID := uuid.New()
	uri := syntax.ATURI("at://did:plc:owner/social.craftsky.feed.post/system-detected")
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id,state,updated_at) VALUES($1,'detected',$2)`, incidentID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incident_subjects(incident_id,owner_did,subject_uri,source_cid,subject_kind,image_slot)
		VALUES($1,'did:plc:owner',$2,'bafy-record','post','images.0')`, incidentID, uri); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	actor := safetyincident.Actor{ID: "safety-moderator", Role: safetyincident.RoleModerator}
	caseRow, err := store.ConfirmSystemDetection(ctx, ConfirmSystemDetectionCommand{
		IncidentID: incidentID, SubjectURI: uri, Actor: actor, SourceSystem: "admin-api",
		ReplayID: "confirm-system-detection-1", ConfirmedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	var origin string
	var linkedIncident uuid.UUID
	var reports int
	if err := pool.QueryRow(ctx, `SELECT origin,incident_id,(SELECT count(*) FROM moderation_case_reports WHERE case_id=$1) FROM moderation_cases WHERE id=$1`, caseRow.ID).Scan(&origin, &linkedIncident, &reports); err != nil {
		t.Fatal(err)
	}
	if origin != "systemDetected" || linkedIncident != incidentID || reports != 0 {
		t.Fatalf("origin=%s incident=%s reports=%d", origin, linkedIncident, reports)
	}
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	decision, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "admin-api", ReplayID: "decide-system-detection-1",
		ActorID: actor.ID, SourceDID: syntax.DID("did:plc:labeler"), Decision: Decision{
			Disposition: DispositionViolation, Reason: ReasonChildSafety, LegalClassification: LegalChildSexualExploitation,
			Evidence: "trusted provider result and chain of custody", SeverityRationale: "confirmed severe violation",
			Consequences: []EffectType{EffectVisibilityTakedown, EffectStrike, EffectSevereSuspension},
		},
	})
	if err != nil || decision.Revision != 1 {
		t.Fatalf("decision=%+v err=%v", decision, err)
	}
	confirmed, err := service.ConfirmAppeal(ctx, AppealCommand{CaseID: caseRow.ID, ExpectedRevision: 1, SourceSystem: "admin-api", ReplayID: "appeal-system-detection-1", ActorID: actor.ID})
	if err != nil || confirmed.Revision != 2 {
		t.Fatalf("appeal=%+v err=%v", confirmed, err)
	}
}
