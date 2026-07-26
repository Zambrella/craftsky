package notifications

import "fmt"

import "social.craftsky/appview/internal/relationships"

type EligibilityInput struct {
	Preference            Preference
	IsSelf                bool
	RecipientFollowsActor bool
	Relationship          relationships.State
}

type EligibilityDecision struct {
	Accepted    bool
	Eligible    bool
	PushEnabled bool
}

// EvaluateEligibility snapshots whether a source event is accepted for the
// in-app feed and, independently, whether the accepted event should fan out to
// push. Callers persist this decision at ingestion time.
func EvaluateEligibility(input EligibilityInput) (EligibilityDecision, error) {
	if !input.Preference.Scope.Valid() {
		return EligibilityDecision{}, fmt.Errorf("invalid notification scope %q", input.Preference.Scope)
	}
	if input.IsSelf {
		return EligibilityDecision{}, nil
	}
	if input.Preference.Scope == PeopleIFollow && !input.RecipientFollowsActor {
		return EligibilityDecision{}, nil
	}
	if input.Relationship.Muted || input.Relationship.HasBlock() {
		return EligibilityDecision{Accepted: true}, nil
	}
	return EligibilityDecision{Accepted: true, Eligible: true, PushEnabled: input.Preference.PushEnabled}, nil
}
