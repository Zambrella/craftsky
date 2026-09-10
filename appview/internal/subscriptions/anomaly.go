package subscriptions

import "github.com/bluesky-social/indigo/atproto/syntax"

type Anomaly string

const (
	AnomalyNone              Anomaly = ""
	AnomalySameTierDuplicate Anomaly = "same_tier_duplicate"
	AnomalyCrossTierConflict Anomaly = "cross_tier_conflict"
)

type LicenseCandidate struct {
	SubscriptionID string
	Tier           Tier
	ExistingTier   *Tier
	GivesAccess    bool
	PendingTier    *Tier
	AssignedDID    *syntax.DID
}

type LicenseDecision struct {
	Assignable bool
	Anomaly    Anomaly
}

func ClassifySnapshotAnomalies(candidates []LicenseCandidate) map[string]LicenseDecision {
	decisions := make(map[string]LicenseDecision, len(candidates))
	byTier := make(map[Tier][]LicenseCandidate)
	for _, candidate := range candidates {
		decisions[candidate.SubscriptionID] = LicenseDecision{Assignable: candidate.GivesAccess}
		if candidate.GivesAccess {
			byTier[candidate.Tier] = append(byTier[candidate.Tier], candidate)
		}
	}

	for _, sameTier := range byTier {
		if len(sameTier) < 2 {
			continue
		}
		assigned := ""
		for _, candidate := range sameTier {
			if candidate.AssignedDID != nil {
				if assigned != "" {
					assigned = ""
					break
				}
				assigned = candidate.SubscriptionID
			}
		}
		for _, candidate := range sameTier {
			if candidate.SubscriptionID == assigned {
				continue
			}
			decisions[candidate.SubscriptionID] = LicenseDecision{Anomaly: AnomalySameTierDuplicate}
		}
	}

	for _, candidate := range candidates {
		if !candidate.GivesAccess || candidate.PendingTier == nil || *candidate.PendingTier == candidate.Tier {
			if candidate.ExistingTier != nil && *candidate.ExistingTier != candidate.Tier {
				decisions[candidate.SubscriptionID] = LicenseDecision{Anomaly: AnomalyCrossTierConflict}
			}
			continue
		}
		decisions[candidate.SubscriptionID] = LicenseDecision{Anomaly: AnomalyCrossTierConflict}
		for _, target := range byTier[*candidate.PendingTier] {
			decisions[target.SubscriptionID] = LicenseDecision{Anomaly: AnomalyCrossTierConflict}
		}
	}
	return decisions
}
