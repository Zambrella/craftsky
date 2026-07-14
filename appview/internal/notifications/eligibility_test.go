package notifications

import "testing"

func TestEvaluateEligibilityAtEventTime(t *testing.T) {
	tests := []struct {
		name     string
		input    EligibilityInput
		accepted bool
		push     bool
	}{
		{"everyone and push enabled", EligibilityInput{Preference: Preference{Scope: Everyone, PushEnabled: true}}, true, true},
		{"everyone and push disabled", EligibilityInput{Preference: Preference{Scope: Everyone, PushEnabled: false}}, true, false},
		{"followed actor under restricted scope", EligibilityInput{Preference: Preference{Scope: PeopleIFollow, PushEnabled: true}, RecipientFollowsActor: true}, true, true},
		{"unfollowed actor under restricted scope", EligibilityInput{Preference: Preference{Scope: PeopleIFollow, PushEnabled: true}}, false, false},
		{"self activity", EligibilityInput{Preference: Preference{Scope: Everyone, PushEnabled: true}, IsSelf: true}, false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := EvaluateEligibility(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.Accepted != test.accepted || got.PushEnabled != test.push {
				t.Fatalf("decision = %+v, want accepted=%v push=%v", got, test.accepted, test.push)
			}
		})
	}
}

func TestEvaluateEligibilityRejectsInvalidScope(t *testing.T) {
	_, err := EvaluateEligibility(EligibilityInput{Preference: Preference{Scope: Scope("invalid"), PushEnabled: true}})
	if err == nil {
		t.Fatal("expected invalid scope error")
	}
}
