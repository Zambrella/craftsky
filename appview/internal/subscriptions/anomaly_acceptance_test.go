package subscriptions

import (
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestProviderAnomaliesFailClosedWithoutMovingAssignments(t *testing.T) {
	beneficiary := syntax.DID("did:plc:anomaly-beneficiary")
	duplicate := ClassifySnapshotAnomalies([]LicenseCandidate{
		{SubscriptionID: "assigned-plus", Tier: TierPlus, GivesAccess: true, AssignedDID: &beneficiary},
		{SubscriptionID: "extra-plus", Tier: TierPlus, GivesAccess: true},
	})
	if !duplicate["assigned-plus"].Assignable || duplicate["assigned-plus"].Anomaly != AnomalyNone {
		t.Fatalf("safe existing assignment was demoted: %+v", duplicate)
	}
	if duplicate["extra-plus"].Assignable || duplicate["extra-plus"].Anomaly != AnomalySameTierDuplicate {
		t.Fatalf("extra duplicate was assignable: %+v", duplicate)
	}

	crossTier := ClassifySnapshotAnomalies([]LicenseCandidate{
		{SubscriptionID: "changing-plus", Tier: TierPlus, GivesAccess: true, PendingTier: tierPointer(TierBusiness), AssignedDID: &beneficiary},
		{SubscriptionID: "new-business", Tier: TierBusiness, GivesAccess: true},
	})
	for id, decision := range crossTier {
		if decision.Assignable || decision.Anomaly != AnomalyCrossTierConflict {
			t.Fatalf("cross-tier %s decision = %+v", id, decision)
		}
	}

	postLapse := ClassifySnapshotAnomalies([]LicenseCandidate{
		{SubscriptionID: "dormant-plus", Tier: TierPlus, AssignedDID: &beneficiary},
		{SubscriptionID: "replacement-business", Tier: TierBusiness, GivesAccess: true},
	})
	if !postLapse["replacement-business"].Assignable || postLapse["replacement-business"].Anomaly != AnomalyNone {
		t.Fatalf("post-lapse replacement was anomalous: %+v", postLapse)
	}
}
