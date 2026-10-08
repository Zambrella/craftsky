package moderation

import "testing"

func TestApprovedAllegationMappingsKeepUserInputSeparateFromJudgment(t *testing.T) {
	t.Parallel()
	mappings := ApprovedAllegationMappings()
	if len(mappings) != 30 {
		t.Fatalf("approved allegation mappings=%d, want 30", len(mappings))
	}
	groups := map[ReportGroup]bool{}
	for allegation, mapping := range mappings {
		if allegation == "" || mapping.Group == "" || mapping.Destination == "" || mapping.UserSafeLabel == "" {
			t.Fatalf("incomplete mapping for %q: %+v", allegation, mapping)
		}
		groups[mapping.Group] = true
		if mapping.LegalClassification != LegalUnassessed {
			t.Fatalf("allegation %q asserted legal classification %q", allegation, mapping.LegalClassification)
		}
		if mapping.Destination == DestinationInApp && mapping.DecisionReason == "" {
			t.Fatalf("in-app allegation %q has no internal routing reason", allegation)
		}
		if mapping.Destination == DestinationSpecialistEmail && mapping.DecisionReason != "" {
			t.Fatalf("specialist allegation %q became a generic decision reason", allegation)
		}
	}
	for _, group := range []ReportGroup{GroupChildSafety, GroupSexualIntimate, GroupThreatsViolenceSelfHarm, GroupHarassmentHateStalkingPrivacy, GroupFraudScamsImpersonation, GroupSpamCommercialOffTopic, GroupIntellectualProperty, GroupOther} {
		if !groups[group] {
			t.Errorf("missing report group %q", group)
		}
	}
	if mapping := mappings[AllegationIntellectualProperty]; mapping.Destination != DestinationSpecialistEmail {
		t.Fatalf("intellectual property destination=%q", mapping.Destination)
	}
	for _, allegation := range []Allegation{
		AllegationChildSexualExploitation, AllegationGrooming, AllegationChildAbuseMaterial,
		AllegationIntimateImageAbuse, AllegationSexualExploitation, AllegationAdultContent, AllegationGraphicContent,
		AllegationImmediateDanger, AllegationCredibleThreat, AllegationTerrorism, AllegationViolence, AllegationSelfHarm,
		AllegationHarassment, AllegationHate, AllegationStalking, AllegationDoxxing, AllegationBlockEvasion, AllegationPrivacy,
		AllegationFraud, AllegationScam, AllegationPhishing, AllegationImpersonation, AllegationCounterfeit,
		AllegationSpam, AllegationPlatformManipulation, AllegationMisleading, AllegationSuspectedAIGenerated, AllegationOffTopic,
		AllegationIntellectualProperty, AllegationOther,
	} {
		if _, ok := mappings[allegation]; !ok {
			t.Errorf("missing allegation mapping %q", allegation)
		}
	}
}
