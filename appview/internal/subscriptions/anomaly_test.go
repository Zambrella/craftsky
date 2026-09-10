package subscriptions

import (
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestClassifySnapshotAnomaliesFailsClosedWithoutInventingLineage(t *testing.T) {
	beneficiary := syntax.DID("did:plc:beneficiary")

	tests := []struct {
		name       string
		candidates []LicenseCandidate
		want       map[string]LicenseDecision
	}{
		{
			name: "one Plus and one Business are independently assignable",
			candidates: []LicenseCandidate{
				{SubscriptionID: "plus", Tier: TierPlus, GivesAccess: true},
				{SubscriptionID: "business", Tier: TierBusiness, GivesAccess: true},
			},
			want: map[string]LicenseDecision{
				"plus": {Assignable: true}, "business": {Assignable: true},
			},
		},
		{
			name: "safe same-tier assignment is preserved and extra fails closed",
			candidates: []LicenseCandidate{
				{SubscriptionID: "plus-assigned", Tier: TierPlus, GivesAccess: true, AssignedDID: &beneficiary},
				{SubscriptionID: "plus-extra", Tier: TierPlus, GivesAccess: true},
			},
			want: map[string]LicenseDecision{
				"plus-assigned": {Assignable: true},
				"plus-extra":    {Anomaly: AnomalySameTierDuplicate},
			},
		},
		{
			name: "ambiguous same-tier duplicates all fail closed",
			candidates: []LicenseCandidate{
				{SubscriptionID: "plus-a", Tier: TierPlus, GivesAccess: true},
				{SubscriptionID: "plus-b", Tier: TierPlus, GivesAccess: true},
			},
			want: map[string]LicenseDecision{
				"plus-a": {Anomaly: AnomalySameTierDuplicate},
				"plus-b": {Anomaly: AnomalySameTierDuplicate},
			},
		},
		{
			name: "provider-reported paid cross-tier change fails closed",
			candidates: []LicenseCandidate{
				{SubscriptionID: "changing-plus", Tier: TierPlus, GivesAccess: true, PendingTier: tierPointer(TierBusiness), AssignedDID: &beneficiary},
				{SubscriptionID: "new-business", Tier: TierBusiness, GivesAccess: true},
			},
			want: map[string]LicenseDecision{
				"changing-plus": {Anomaly: AnomalyCrossTierConflict},
				"new-business":  {Anomaly: AnomalyCrossTierConflict},
			},
		},
		{
			name: "post-lapse new ID is ordinary and unassigned",
			candidates: []LicenseCandidate{
				{SubscriptionID: "old-plus", Tier: TierPlus, AssignedDID: &beneficiary},
				{SubscriptionID: "new-business", Tier: TierBusiness, GivesAccess: true},
			},
			want: map[string]LicenseDecision{
				"old-plus": {}, "new-business": {Assignable: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ClassifySnapshotAnomalies(test.candidates)
			if len(got) != len(test.want) {
				t.Fatalf("decision count = %d, want %d: %+v", len(got), len(test.want), got)
			}
			for id, want := range test.want {
				if got[id] != want {
					t.Fatalf("decision %q = %+v, want %+v", id, got[id], want)
				}
			}
		})
	}
}
