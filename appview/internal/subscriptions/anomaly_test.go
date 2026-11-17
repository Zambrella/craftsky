package subscriptions

import "testing"

func TestPerDIDTierChangesAndDuplicateSubscriptions(t *testing.T) {
	for _, test := range []struct {
		name       string
		candidates []LicenseCandidate
		anomaly    Anomaly
	}{
		{"scheduled downgrade", []LicenseCandidate{{SubscriptionID: "business", Tier: TierBusiness, GivesAccess: true}}, AnomalyNone},
		{"confirmed upgrade", []LicenseCandidate{{SubscriptionID: "changed", Tier: TierBusiness, GivesAccess: true}}, AnomalyNone},
		{"same tier duplicate", []LicenseCandidate{{SubscriptionID: "a", Tier: TierPlus, GivesAccess: true}, {SubscriptionID: "b", Tier: TierPlus, GivesAccess: true}}, AnomalySameTierDuplicate},
		{"cross tier duplicate", []LicenseCandidate{{SubscriptionID: "a", Tier: TierPlus, GivesAccess: true}, {SubscriptionID: "b", Tier: TierBusiness, GivesAccess: true}}, AnomalyCrossTierConflict},
		{"historical plan replaced", []LicenseCandidate{{SubscriptionID: "a", Tier: TierPlus}, {SubscriptionID: "b", Tier: TierBusiness, GivesAccess: true}}, AnomalyNone},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, decision := range ClassifySnapshotAnomalies(test.candidates) {
				if decision.Assignable || decision.Anomaly != test.anomaly {
					t.Fatalf("decision=%+v want anomaly=%s and no assignment", decision, test.anomaly)
				}
			}
		})
	}
}
