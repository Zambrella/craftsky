package instagram

import (
	"strings"
	"testing"
)

func TestInstagramSuggestionEligibilityPolicyRequiresEveryBoundary(t *testing.T) {
	t.Parallel()

	eligible := EligibilitySnapshot{
		ImporterCurrentMember: true,
		TargetCurrentMember:   true,
		LinkState:             LinkActive,
		DMVerified:            true,
		Discoverable:          true,
		ConflictFree:          true,
		ImportedUsername:      "synthetic.crafter",
		CurrentUsername:       "synthetic.crafter",
		SafetyDataAvailable:   true,
	}

	for _, stage := range AllEligibilityStages() {
		decision := EvaluateInstagramSuggestionEligibility(stage, eligible)
		if !decision.Eligible || decision.Reason != EligibilityAllowed {
			t.Errorf("stage %q decision = %+v, want allowed", stage, decision)
		}
	}

	cases := []struct {
		name   string
		mutate func(*EligibilitySnapshot)
		reason EligibilityReason
	}{
		{name: "importer departed", mutate: func(s *EligibilitySnapshot) { s.ImporterCurrentMember = false }, reason: EligibilityMembership},
		{name: "target departed", mutate: func(s *EligibilitySnapshot) { s.TargetCurrentMember = false }, reason: EligibilityMembership},
		{name: "link membership inactive", mutate: func(s *EligibilitySnapshot) { s.LinkState = LinkMembershipInactive }, reason: EligibilityLink},
		{name: "link revoked", mutate: func(s *EligibilitySnapshot) { s.LinkState = LinkRevoked }, reason: EligibilityLink},
		{name: "link superseded", mutate: func(s *EligibilitySnapshot) { s.LinkState = LinkSuperseded }, reason: EligibilityLink},
		{name: "link disputed", mutate: func(s *EligibilitySnapshot) { s.LinkState = LinkDisputed }, reason: EligibilityConflict},
		{name: "not DM verified", mutate: func(s *EligibilitySnapshot) { s.DMVerified = false }, reason: EligibilityLink},
		{name: "discovery disabled", mutate: func(s *EligibilitySnapshot) { s.Discoverable = false }, reason: EligibilityDiscovery},
		{name: "conflict pending", mutate: func(s *EligibilitySnapshot) { s.ConflictFree = false }, reason: EligibilityConflict},
		{name: "stale exact username", mutate: func(s *EligibilitySnapshot) { s.CurrentUsername = "changed.crafter" }, reason: EligibilityUsername},
		{name: "self", mutate: func(s *EligibilitySnapshot) { s.Self = true }, reason: EligibilitySelf},
		{name: "already followed", mutate: func(s *EligibilitySnapshot) { s.AlreadyFollowing = true }, reason: EligibilityAlreadyFollowing},
		{name: "target hidden", mutate: func(s *EligibilitySnapshot) { s.TargetHidden = true }, reason: EligibilityModeration},
		{name: "target taken down", mutate: func(s *EligibilitySnapshot) { s.TargetTakenDown = true }, reason: EligibilityModeration},
		{name: "importer blocks target", mutate: func(s *EligibilitySnapshot) { s.ImporterBlocksTarget = true }, reason: EligibilityRelationshipSafety},
		{name: "target blocks importer", mutate: func(s *EligibilitySnapshot) { s.TargetBlocksImporter = true }, reason: EligibilityRelationshipSafety},
		{name: "importer mutes target", mutate: func(s *EligibilitySnapshot) { s.ImporterMutesTarget = true }, reason: EligibilityRelationshipSafety},
		{name: "safety source unavailable", mutate: func(s *EligibilitySnapshot) { s.SafetyDataAvailable = false }, reason: EligibilitySafetyUnavailable},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := eligible
			tt.mutate(&snapshot)
			for _, stage := range AllEligibilityStages() {
				decision := EvaluateInstagramSuggestionEligibility(stage, snapshot)
				if decision.Eligible || decision.Reason != tt.reason {
					t.Errorf("stage %q decision = %+v, want denied %q", stage, decision, tt.reason)
				}
			}
		})
	}
}

func TestInstagramSuggestionEligibilityUsesExactNormalizedUsername(t *testing.T) {
	t.Parallel()

	base := EligibilitySnapshot{
		ImporterCurrentMember: true,
		TargetCurrentMember:   true,
		LinkState:             LinkActive,
		DMVerified:            true,
		Discoverable:          true,
		ConflictFree:          true,
		ImportedUsername:      "Synthetic.Crafter",
		CurrentUsername:       "@synthetic.crafter",
		SafetyDataAvailable:   true,
	}
	if decision := EvaluateInstagramSuggestionEligibility(EligibilityAtMatch, base); !decision.Eligible {
		t.Fatalf("equivalent normalized usernames denied: %+v", decision)
	}

	for _, username := range []string{"synthetic_crafter", "synthetic.crafter2", "syntheticcrafter"} {
		snapshot := base
		snapshot.CurrentUsername = username
		if decision := EvaluateInstagramSuggestionEligibility(EligibilityAtMatch, snapshot); decision.Eligible || decision.Reason != EligibilityUsername {
			t.Errorf("fuzzy username %q decision = %+v, want exact-match denial", username, decision)
		}
	}
}

func TestInstagramSuggestionEligibilityRejectsUnknownStageAndRedactsSnapshot(t *testing.T) {
	t.Parallel()

	snapshot := EligibilitySnapshot{
		ImporterCurrentMember: true,
		TargetCurrentMember:   true,
		LinkState:             LinkActive,
		DMVerified:            true,
		Discoverable:          true,
		ConflictFree:          true,
		ImportedUsername:      "synthetic-private-canary",
		CurrentUsername:       "synthetic-private-canary",
		SafetyDataAvailable:   true,
	}
	decision := EvaluateInstagramSuggestionEligibility(EligibilityStage("future"), snapshot)
	if decision.Eligible || decision.Reason != EligibilityInvalidInput {
		t.Fatalf("unknown-stage decision = %+v", decision)
	}
	if got := snapshot.String(); got == "" || strings.Contains(got, "synthetic-private-canary") {
		t.Fatalf("snapshot String() was not safely redacted: %q", got)
	}
}
