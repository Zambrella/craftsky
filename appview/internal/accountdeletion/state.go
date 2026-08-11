package accountdeletion

import (
	"errors"
	"fmt"
)

type Status string

const (
	StatusIntent         Status = "intent"
	StatusActive         Status = "active"
	StatusRetrying       Status = "retrying"
	StatusNeedsAttention Status = "needsAttention"
	StatusDeleted        Status = "deleted"
	StatusCanceled       Status = "canceled"
)

type Phase string

const (
	PhaseNone                         Phase = ""
	PhaseQueued                       Phase = "queued"
	PhaseRemovingPrivateData          Phase = "removingPrivateData"
	PhaseRemovingCraftskyRecords      Phase = "removingCraftskyRecords"
	PhaseWaitingForIndexerConvergence Phase = "waitingForIndexerConvergence"
	PhaseFinalizing                   Phase = "finalizing"
)

type State struct {
	Status Status
	Phase  Phase
}

var (
	StateIntent                       = State{Status: StatusIntent}
	StateCanceled                     = State{Status: StatusCanceled}
	StateQueued                       = State{Status: StatusActive, Phase: PhaseQueued}
	StateRemovingPrivateData          = State{Status: StatusActive, Phase: PhaseRemovingPrivateData}
	StateRemovingCraftskyRecords      = State{Status: StatusActive, Phase: PhaseRemovingCraftskyRecords}
	StateRetrying                     = State{Status: StatusRetrying, Phase: PhaseRemovingCraftskyRecords}
	StateNeedsAttention               = State{Status: StatusNeedsAttention, Phase: PhaseRemovingCraftskyRecords}
	StateWaitingForIndexerConvergence = State{Status: StatusActive, Phase: PhaseWaitingForIndexerConvergence}
	StateFinalizing                   = State{Status: StatusActive, Phase: PhaseFinalizing}
	StateDeleted                      = State{Status: StatusDeleted}
)

type Event string

const (
	EventAccept             Event = "accept"
	EventCancel             Event = "cancel"
	EventStart              Event = "start"
	EventPrivateDataRemoved Event = "privateDataRemoved"
	EventRecordsRemoved     Event = "recordsRemoved"
	EventConverged          Event = "converged"
	EventFinalized          Event = "finalized"
	EventRetrying           Event = "retrying"
	EventNeedsAttention     Event = "needsAttention"
	EventRetry              Event = "retry"
)

var (
	ErrPointOfNoReturn   = errors.New("account deletion is past the point of no return")
	ErrInvalidTransition = errors.New("invalid account deletion transition")
)

func CanActivate(state State) bool {
	return state.Status == StatusIntent || state.Status == StatusCanceled
}

func Transition(state State, event Event) (State, error) {
	if event == EventCancel {
		if state == StateIntent || state == StateCanceled {
			return StateCanceled, nil
		}
		return state, ErrPointOfNoReturn
	}

	if target, ok := idempotentTarget(state, event); ok {
		return target, nil
	}

	switch event {
	case EventAccept:
		if state == StateIntent {
			return StateQueued, nil
		}
	case EventStart:
		if state == StateQueued {
			return StateRemovingPrivateData, nil
		}
	case EventPrivateDataRemoved:
		if state == StateRemovingPrivateData {
			return StateRemovingCraftskyRecords, nil
		}
	case EventRecordsRemoved:
		if state == StateRemovingCraftskyRecords {
			return StateWaitingForIndexerConvergence, nil
		}
	case EventConverged:
		if state == StateWaitingForIndexerConvergence {
			return StateFinalizing, nil
		}
	case EventFinalized:
		if state == StateFinalizing {
			return StateDeleted, nil
		}
	case EventRetrying:
		if state.Status == StatusActive && state.Phase != PhaseQueued {
			return State{Status: StatusRetrying, Phase: state.Phase}, nil
		}
	case EventNeedsAttention:
		if (state.Status == StatusActive || state.Status == StatusRetrying) && state.Phase != PhaseNone {
			return State{Status: StatusNeedsAttention, Phase: state.Phase}, nil
		}
	case EventRetry:
		if state.Status == StatusNeedsAttention && state.Phase != PhaseNone {
			return State{Status: StatusActive, Phase: state.Phase}, nil
		}
	}

	return state, fmt.Errorf("%w: %s from %v", ErrInvalidTransition, event, state)
}

func idempotentTarget(state State, event Event) (State, bool) {
	switch event {
	case EventAccept:
		return state, state == StateQueued
	case EventStart:
		return state, state == StateRemovingPrivateData
	case EventPrivateDataRemoved:
		return state, state == StateRemovingCraftskyRecords
	case EventRecordsRemoved:
		return state, state == StateWaitingForIndexerConvergence
	case EventConverged:
		return state, state == StateFinalizing
	case EventFinalized:
		return state, state == StateDeleted
	case EventRetrying:
		return state, state.Status == StatusRetrying
	case EventNeedsAttention:
		return state, state.Status == StatusNeedsAttention
	case EventRetry:
		return state, false
	default:
		return state, false
	}
}
