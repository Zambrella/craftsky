package relationships

import "testing"

func TestDecideRelationshipPolicyPrecedence(t *testing.T) {
	tests := []struct {
		name         string
		moderation   ModerationState
		relationship State
		surface      Surface
		want         Decision
	}{
		{name: "ordinary top level content is allowed", moderation: ModerationVisible, surface: SurfaceTopLevel, want: DecisionAllow},
		{name: "mute omits unsolicited top level content", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceTopLevel, want: DecisionOmit},
		{name: "mute collapses a reply branch", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceThread, want: DecisionMutedPlaceholder},
		{name: "mute collapses quoted content", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceQuote, want: DecisionMutedPlaceholder},
		{name: "mute permits direct content", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceDirectPost, want: DecisionAllow},
		{name: "mute permits full profile", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceProfile, want: DecisionAllow},
		{name: "mute suppresses notifications", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceNotification, want: DecisionOmit},
		{name: "mute does not hide graph relationships", moderation: ModerationVisible, relationship: State{Muted: true}, surface: SurfaceGraph, want: DecisionAllow},
		{name: "outbound block produces minimal profile", moderation: ModerationVisible, relationship: State{Blocking: true}, surface: SurfaceProfile, want: DecisionMinimalProfile},
		{name: "inbound block is symmetric on profile", moderation: ModerationVisible, relationship: State{BlockedBy: true}, surface: SurfaceProfile, want: DecisionMinimalProfile},
		{name: "outbound block protects direct content", moderation: ModerationVisible, relationship: State{Blocking: true}, surface: SurfaceDirectPost, want: DecisionBlockedPlaceholder},
		{name: "inbound block protects quote", moderation: ModerationVisible, relationship: State{BlockedBy: true}, surface: SurfaceQuote, want: DecisionBlockedPlaceholder},
		{name: "mutual block omits top level content", moderation: ModerationVisible, relationship: State{Blocking: true, BlockedBy: true}, surface: SurfaceTopLevel, want: DecisionOmit},
		{name: "block wins over mute", moderation: ModerationVisible, relationship: State{Muted: true, Blocking: true}, surface: SurfaceThread, want: DecisionBlockedPlaceholder},
		{name: "platform hide wins over mute reveal", moderation: ModerationHidden, relationship: State{Muted: true}, surface: SurfaceThread, want: DecisionOmit},
		{name: "platform takedown wins over block shell", moderation: ModerationTakenDown, relationship: State{BlockedBy: true}, surface: SurfaceProfile, want: DecisionOmit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Decide(tt.moderation, tt.relationship, tt.surface); got != tt.want {
				t.Fatalf("Decide(%v, %+v, %v) = %v, want %v", tt.moderation, tt.relationship, tt.surface, got, tt.want)
			}
		})
	}
}
