package instagram

import (
	"errors"
	"testing"
)

func TestSuggestionStateTransitions(t *testing.T) {
	t.Parallel()

	allowed := [][2]SuggestionState{
		{SuggestionPending, SuggestionAccepting},
		{SuggestionPending, SuggestionDismissed},
		{SuggestionPending, SuggestionInvalidated},
		{SuggestionAccepting, SuggestionPending},
		{SuggestionAccepting, SuggestionFollowed},
		{SuggestionAccepting, SuggestionAlreadyFollowing},
		{SuggestionAccepting, SuggestionInvalidated},
	}
	for _, transition := range allowed {
		if err := ValidateSuggestionTransition(transition[0], transition[1]); err != nil {
			t.Errorf("ValidateSuggestionTransition(%q, %q): %v", transition[0], transition[1], err)
		}
	}

	for _, terminal := range []SuggestionState{
		SuggestionFollowed,
		SuggestionAlreadyFollowing,
		SuggestionDismissed,
		SuggestionInvalidated,
	} {
		if !terminal.Terminal() {
			t.Errorf("%q is not terminal", terminal)
		}
		for _, next := range []SuggestionState{SuggestionPending, SuggestionAccepting} {
			if err := ValidateSuggestionTransition(terminal, next); !errors.Is(err, ErrInstagramStateTransition) {
				t.Errorf("transition %q to %q error = %v, want ErrInstagramStateTransition", terminal, next, err)
			}
		}
	}
}

func TestSuggestionStateRejectsUnknownValuesAndStaleGenerations(t *testing.T) {
	t.Parallel()

	if err := ValidateSuggestionTransition(SuggestionState("unknown"), SuggestionPending); !errors.Is(err, ErrInvalidInstagramState) {
		t.Fatalf("unknown state error = %v, want ErrInvalidInstagramState", err)
	}
	if err := ValidateSuggestionGenerations(4, 7, 4, 7); err != nil {
		t.Fatalf("current generations rejected: %v", err)
	}
	for _, test := range []struct {
		name                           string
		storedImporter, storedTarget   int64
		currentImporter, currentTarget int64
	}{
		{name: "stale importer", storedImporter: 4, storedTarget: 7, currentImporter: 5, currentTarget: 7},
		{name: "stale target", storedImporter: 4, storedTarget: 7, currentImporter: 4, currentTarget: 8},
		{name: "invalid stored", storedImporter: 0, storedTarget: 7, currentImporter: 0, currentTarget: 7},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateSuggestionGenerations(
				test.storedImporter,
				test.storedTarget,
				test.currentImporter,
				test.currentTarget,
			); !errors.Is(err, ErrSuggestionGenerationChanged) {
				t.Fatalf("generation error = %v, want ErrSuggestionGenerationChanged", err)
			}
		})
	}
}
