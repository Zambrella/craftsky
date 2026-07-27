package instagram

import (
	"errors"
	"testing"
)

func TestAutomaticFollowState_TransitionsAndVerificationLifetimeSuppression(t *testing.T) {
	t.Parallel()

	allowed := [][2]AutomaticFollowState{
		{AutomaticFollowPending, AutomaticFollowWriting},
		{AutomaticFollowPending, AutomaticFollowInvalidated},
		{AutomaticFollowWriting, AutomaticFollowPending},
		{AutomaticFollowWriting, AutomaticFollowFollowed},
		{AutomaticFollowWriting, AutomaticFollowAlreadyFollowing},
		{AutomaticFollowWriting, AutomaticFollowInvalidated},
		{AutomaticFollowInvalidated, AutomaticFollowPending},
	}
	for _, transition := range allowed {
		if err := ValidateAutomaticFollowTransition(transition[0], transition[1]); err != nil {
			t.Errorf("ValidateAutomaticFollowTransition(%q, %q): %v", transition[0], transition[1], err)
		}
	}

	for _, terminal := range []AutomaticFollowState{
		AutomaticFollowFollowed,
		AutomaticFollowAlreadyFollowing,
	} {
		if !terminal.SuppressesReconciliation() {
			t.Errorf("%q must suppress reconciliation for the verification lifetime", terminal)
		}
		if err := ValidateAutomaticFollowTransition(terminal, AutomaticFollowPending); !errors.Is(err, ErrInstagramStateTransition) {
			t.Errorf("terminal %q -> pending error = %v, want ErrInstagramStateTransition", terminal, err)
		}
	}

	if AutomaticFollowInvalidated.SuppressesReconciliation() {
		t.Fatal("invalidated work must be reconsiderable after eligibility is restored")
	}
	if AutomaticFollowWriting.SuppressesReconciliation() {
		t.Fatal("writing work must be recoverable rather than permanently suppressed")
	}
}

func TestAutomaticFollowState_RejectsInvalidValues(t *testing.T) {
	t.Parallel()

	if err := ValidateAutomaticFollowTransition(AutomaticFollowState("unknown"), AutomaticFollowPending); !errors.Is(err, ErrInvalidInstagramState) {
		t.Fatalf("invalid transition error = %v, want ErrInvalidInstagramState", err)
	}
	if AutomaticFollowState("unknown").SuppressesReconciliation() {
		t.Fatal("an unknown state must not suppress reconciliation")
	}
}
