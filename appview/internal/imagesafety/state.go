package imagesafety

type State string

const (
	StatePending     State = "pending"
	StateClear       State = "clear"
	StateMatch       State = "match"
	StateUnavailable State = "unavailable"
	StateError       State = "error"
)

func (state State) Valid() bool {
	switch state {
	case StatePending, StateClear, StateMatch, StateUnavailable, StateError:
		return true
	default:
		return false
	}
}

func (state State) Terminal() bool {
	return state == StateClear || state == StateMatch
}

func (state State) DisplayEligible() bool {
	return state == StateClear
}
