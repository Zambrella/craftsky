package relationships

import "testing"

func TestDecideMuteLifecycleDeletesOnlyForOwnerMembershipRemoval(t *testing.T) {
	tests := []struct {
		name  string
		event LifecycleEvent
		role  MuteRole
		want  LifecycleDecision
	}{
		{name: "owner membership removal deletes owned mutes", event: LifecycleMembershipRemoved, role: MuteRoleOwner, want: LifecycleDeleteOwnedMutes},
		{name: "subject membership removal retains mute", event: LifecycleMembershipRemoved, role: MuteRoleSubject, want: LifecycleRetainMutes},
		{name: "owner sign out retains mute", event: LifecycleSignedOut, role: MuteRoleOwner, want: LifecycleRetainMutes},
		{name: "owner device removal retains mute", event: LifecycleDeviceRemoved, role: MuteRoleOwner, want: LifecycleRetainMutes},
		{name: "owner account switch retains mute", event: LifecycleAccountSwitched, role: MuteRoleOwner, want: LifecycleRetainMutes},
		{name: "subject session events retain mute", event: LifecycleSignedOut, role: MuteRoleSubject, want: LifecycleRetainMutes},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecideMuteLifecycle(tt.event, tt.role); got != tt.want {
				t.Fatalf("DecideMuteLifecycle(%v, %v) = %v, want %v", tt.event, tt.role, got, tt.want)
			}
		})
	}
}
