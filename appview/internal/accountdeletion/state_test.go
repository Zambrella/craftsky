package accountdeletion

import (
	"errors"
	"testing"
)

func TestDeletionStateMachine(t *testing.T) {
	t.Parallel()

	t.Run("cancel is legal only before acceptance", func(t *testing.T) {
		got, err := Transition(StateIntent, EventCancel)
		if err != nil || got != StateCanceled {
			t.Fatalf("cancel intent = (%q, %v), want (%q, nil)", got, err, StateCanceled)
		}

		accepted, err := Transition(StateIntent, EventAccept)
		if err != nil || accepted != StateQueued {
			t.Fatalf("accept intent = (%q, %v), want (%q, nil)", accepted, err, StateQueued)
		}
		if _, err := Transition(accepted, EventCancel); !errors.Is(err, ErrPointOfNoReturn) {
			t.Fatalf("cancel accepted error = %v, want ErrPointOfNoReturn", err)
		}
		if CanActivate(accepted) {
			t.Fatal("accepted deletion must not permit ordinary account activation")
		}
	})

	t.Run("accepted jobs progress through coarse phases idempotently", func(t *testing.T) {
		steps := []struct {
			from  State
			event Event
			want  State
		}{
			{StateQueued, EventStart, StateRemovingPrivateData},
			{StateRemovingPrivateData, EventPrivateDataRemoved, StateRemovingCraftskyRecords},
			{StateRemovingCraftskyRecords, EventRecordsRemoved, StateWaitingForIndexerConvergence},
			{StateWaitingForIndexerConvergence, EventConverged, StateFinalizing},
			{StateFinalizing, EventFinalized, StateDeleted},
		}
		for _, step := range steps {
			got, err := Transition(step.from, step.event)
			if err != nil || got != step.want {
				t.Fatalf("Transition(%q, %q) = (%q, %v), want (%q, nil)", step.from, step.event, got, err, step.want)
			}
			duplicate, err := Transition(got, step.event)
			if err != nil || duplicate != got {
				t.Fatalf("duplicate Transition(%q, %q) = (%q, %v), want unchanged", got, step.event, duplicate, err)
			}
		}
	})

	t.Run("retry and attention never restore ordinary access", func(t *testing.T) {
		retrying, err := Transition(StateRemovingCraftskyRecords, EventRetrying)
		if err != nil || retrying != StateRetrying {
			t.Fatalf("retry transition = (%q, %v)", retrying, err)
		}
		attention, err := Transition(retrying, EventNeedsAttention)
		if err != nil || attention != StateNeedsAttention {
			t.Fatalf("attention transition = (%q, %v)", attention, err)
		}
		resumed, err := Transition(attention, EventRetry)
		if err != nil || resumed.Status != StatusActive || resumed.Phase != PhaseRemovingCraftskyRecords {
			t.Fatalf("manual retry = (%v, %v), want active records phase", resumed, err)
		}
		for _, state := range []State{retrying, attention, resumed, StateDeleted} {
			if CanActivate(state) {
				t.Fatalf("state %v must not permit ordinary activation", state)
			}
		}
	})
}
