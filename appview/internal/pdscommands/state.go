package pdscommands

import "errors"

var (
	ErrMalformedCommand    = errors.New("PDS command request is malformed")
	ErrDispatchUnavailable = errors.New("PDS command dispatch is unavailable")
	ErrDispatchRejected    = errors.New("PDS command was rejected")
)

func CanTransition(from, to CommandState) bool {
	switch from {
	case CommandPrepared:
		return to == CommandDispatching || to == CommandAccepted || to == CommandRejected
	case CommandDispatching:
		return to == CommandAccepted || to == CommandAmbiguous || to == CommandRejected
	case CommandAmbiguous:
		return to == CommandDispatching || to == CommandAccepted || to == CommandRejected
	default:
		return false
	}
}
