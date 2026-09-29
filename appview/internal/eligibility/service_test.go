package eligibility_test

import (
	"context"
	"os"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/eligibility"
	"social.craftsky/appview/internal/testdb"
)

func TestReviewRejectsProhibitedAgeInferenceAndSupportsReversal(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000078_age_eligibility.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(migration))
	store := eligibility.NewStore(pool)
	did := syntax.DID("did:plc:alice")

	for _, prohibited := range []eligibility.EvidenceKind{"appearance", "interests", "writing", "location", "report_volume", "facial_age_estimate"} {
		_, err := store.Review(context.Background(), eligibility.Review{
			AccountDID: did, State: eligibility.StateRestricted, EvidenceKind: prohibited,
			EvidenceReference: "restricted-ref", ReviewerID: "moderator-1", Reason: "review", AppealGuidance: "Contact support to appeal.",
		})
		if err == nil {
			t.Fatalf("prohibited evidence %q supported a restriction", prohibited)
		}
	}

	restricted, err := store.Review(context.Background(), eligibility.Review{
		AccountDID: did, State: eligibility.StateRestricted, EvidenceKind: eligibility.EvidenceSelfDisclosure,
		EvidenceReference: "restricted-ref", ReviewerID: "moderator-1", Reason: "reviewed disclosure", AppealGuidance: "Contact support to appeal.",
	})
	if err != nil || !restricted.Appealable {
		t.Fatalf("restricted status=%+v err=%v", restricted, err)
	}
	restored, err := store.Review(context.Background(), eligibility.Review{
		AccountDID: did, State: eligibility.StateEligible, ReviewerID: "moderator-2", Reason: "appeal upheld",
	})
	if err != nil || restored.State != eligibility.StateEligible || restored.Appealable {
		t.Fatalf("restored status=%+v err=%v", restored, err)
	}
	var events int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM account_age_eligibility_events WHERE account_did=$1`, did).Scan(&events); err != nil || events != 2 {
		t.Fatalf("events=%d err=%v", events, err)
	}
}
