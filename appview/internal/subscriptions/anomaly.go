package subscriptions

type Anomaly string

const (
	AnomalyNone              Anomaly = ""
	AnomalySameTierDuplicate Anomaly = "same_tier_duplicate"
	AnomalyCrossTierConflict Anomaly = "cross_tier_conflict"
)

type LicenseCandidate struct {
	SubscriptionID string
	Tier           Tier
	GivesAccess    bool
}

type LicenseDecision struct {
	Assignable bool
	Anomaly    Anomaly
}

// ClassifySnapshotAnomalies treats Plus and Business as mutually exclusive tiers.
// Provider-confirmed tier changes and scheduled changes are ordinary lifecycle
// events; only simultaneous access-bearing subscriptions are ambiguous.
func ClassifySnapshotAnomalies(candidates []LicenseCandidate) map[string]LicenseDecision {
	decisions := make(map[string]LicenseDecision, len(candidates))
	var active []LicenseCandidate
	for _, candidate := range candidates {
		decisions[candidate.SubscriptionID] = LicenseDecision{}
		if candidate.GivesAccess {
			active = append(active, candidate)
		}
	}
	if len(active) > 1 {
		anomaly := AnomalySameTierDuplicate
		for _, candidate := range active {
			if candidate.Tier != active[0].Tier {
				anomaly = AnomalyCrossTierConflict
			}
		}
		for _, candidate := range active {
			decisions[candidate.SubscriptionID] = LicenseDecision{Anomaly: anomaly}
		}
	}
	return decisions
}
