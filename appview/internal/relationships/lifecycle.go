package relationships

// LifecycleEvent is limited to events that could otherwise be confused with
// permanent membership deletion.
type LifecycleEvent uint8

const (
	LifecycleMembershipRemoved LifecycleEvent = iota
	LifecycleSignedOut
	LifecycleDeviceRemoved
	LifecycleAccountSwitched
)

type MuteRole uint8

const (
	MuteRoleOwner MuteRole = iota
	MuteRoleSubject
)

type LifecycleDecision uint8

const (
	LifecycleRetainMutes LifecycleDecision = iota
	LifecycleDeleteOwnedMutes
)

// DecideMuteLifecycle mirrors the persistence contract: only permanent
// removal of the owning Craftsky membership deletes private mute rows.
func DecideMuteLifecycle(event LifecycleEvent, role MuteRole) LifecycleDecision {
	if event == LifecycleMembershipRemoved && role == MuteRoleOwner {
		return LifecycleDeleteOwnedMutes
	}
	return LifecycleRetainMutes
}
