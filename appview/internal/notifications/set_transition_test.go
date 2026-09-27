package notifications

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

type recordingSetTransitionLifecycle struct {
	activations []Activation
	retractions []Retraction
}

func (lifecycle *recordingSetTransitionLifecycle) Activate(_ context.Context, _ pgx.Tx, activation Activation) error {
	lifecycle.activations = append(lifecycle.activations, activation)
	return nil
}

func (lifecycle *recordingSetTransitionLifecycle) Retract(_ context.Context, _ pgx.Tx, retraction Retraction) error {
	lifecycle.retractions = append(lifecycle.retractions, retraction)
	return nil
}

func TestApplySetTransitionUsesOnlyLogicalActivationEdges(t *testing.T) {
	lifecycle := &recordingSetTransitionLifecycle{}
	activation := Activation{ActorDID: "did:plc:actor", SubjectKey: "at://did:plc:subject/social.craftsky.feed.post/3aaaaaaaaaaa2"}
	retraction := Retraction{ActorDID: activation.ActorDID, Category: Like, SubjectKey: activation.SubjectKey, Reason: "setDeactivated"}

	for _, transition := range []SetTransition{SetTransitionActivated, SetTransitionUnchanged, SetTransitionUnchanged, SetTransitionDeactivated} {
		if err := ApplySetTransition(context.Background(), nil, lifecycle, transition, activation, retraction); err != nil {
			t.Fatalf("apply %s transition: %v", transition, err)
		}
	}

	if len(lifecycle.activations) != 1 || lifecycle.activations[0] != activation {
		t.Fatalf("activations = %+v, want [%+v]", lifecycle.activations, activation)
	}
	if len(lifecycle.retractions) != 1 || lifecycle.retractions[0] != retraction {
		t.Fatalf("retractions = %+v, want [%+v]", lifecycle.retractions, retraction)
	}
}
