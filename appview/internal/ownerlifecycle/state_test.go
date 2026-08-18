package ownerlifecycle

import (
	"errors"
	"slices"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestValidateTransition(t *testing.T) {
	t.Parallel()

	valid := [][2]State{
		{StateDeparted, StateActive},
		{StateActive, StateDeparted},
		{StateActive, StateDeletionPending},
		{StateDeletionPending, StateActive},
		{StateDeletionPending, StateDeparted},
		{StateDeletionPending, StateDeleting},
		{StateDeleting, StateDeparted},
		{StateActive, StateTerminal},
		{StateDeparted, StateTerminal},
		{StateDeletionPending, StateTerminal},
		{StateDeleting, StateTerminal},
	}
	for _, transition := range valid {
		if err := ValidateTransition(transition[0], transition[1]); err != nil {
			t.Errorf("ValidateTransition(%q, %q): %v", transition[0], transition[1], err)
		}
	}

	invalid := [][2]State{
		{StateActive, StateActive},
		{StateDeparted, StateDeleting},
		{StateDeleting, StateActive},
		{StateTerminal, StateActive},
		{StateTerminal, StateDeparted},
		{"unknown", StateActive},
		{StateActive, "unknown"},
	}
	for _, transition := range invalid {
		if err := ValidateTransition(transition[0], transition[1]); !errors.Is(err, ErrInvalidTransition) {
			t.Errorf("ValidateTransition(%q, %q) error = %v, want ErrInvalidTransition", transition[0], transition[1], err)
		}
	}
}

func TestTransitionAdvancesAuthEpoch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		from State
		to   State
		want bool
	}{
		{StateDeparted, StateActive, false},
		{StateActive, StateDeparted, true},
		{StateActive, StateDeletionPending, false},
		{StateDeletionPending, StateActive, false},
		{StateDeletionPending, StateDeleting, true},
		{StateDeleting, StateDeparted, true},
		{StateActive, StateTerminal, true},
		{StateDeparted, StateTerminal, true},
	}
	for _, test := range tests {
		if got := transitionAdvancesAuthEpoch(test.from, test.to); got != test.want {
			t.Errorf("transitionAdvancesAuthEpoch(%q, %q) = %t, want %t", test.from, test.to, got, test.want)
		}
	}
}

func TestCanonicalOwners(t *testing.T) {
	t.Parallel()

	a := syntax.DID("did:plc:aaa")
	b := syntax.DID("did:plc:bbb")
	got, err := CanonicalOwners([]syntax.DID{b, a, b})
	if err != nil {
		t.Fatalf("CanonicalOwners: %v", err)
	}
	if want := []syntax.DID{a, b}; !slices.Equal(got, want) {
		t.Fatalf("CanonicalOwners = %v, want %v", got, want)
	}
	if _, err := CanonicalOwners([]syntax.DID{a, ""}); !errors.Is(err, ErrInvalidOwner) {
		t.Fatalf("empty owner error = %v, want ErrInvalidOwner", err)
	}
}

func TestNormalizeExpectedOwnersSupportsFenceOnlyMissingExpectation(t *testing.T) {
	active := syntax.DID("did:plc:active-expected-owner")
	missing := syntax.DID("did:plc:missing-expected-owner")
	scope, owners, err := normalizeExpectedOwners([]ExpectedOwner{
		{Owner: missing, AllowMissing: true},
		{Owner: active, Generation: 4},
		{Owner: missing, AllowMissing: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if scope[active] != 4 || scope[missing] != 0 {
		t.Fatalf("normalized scope = %+v", scope)
	}
	if want := []syntax.DID{missing, active, missing}; !slices.Equal(owners, want) {
		t.Fatalf("normalized owners = %v, want %v", owners, want)
	}
	for _, invalid := range [][]ExpectedOwner{
		{{Owner: missing}},
		{{Owner: missing, Generation: 2, AllowMissing: true}},
		{{Owner: missing, AllowMissing: true}, {Owner: missing, Generation: 2}},
	} {
		if _, _, err := normalizeExpectedOwners(invalid); !errors.Is(
			err, ErrGenerationChanged,
		) {
			t.Fatalf("invalid missing expectation %+v error = %v", invalid, err)
		}
	}
}

func TestFenceKeyStableAndNamespaced(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:aaa")
	one, err := FenceKey(owner)
	if err != nil {
		t.Fatal(err)
	}
	two, err := FenceKey(owner)
	if err != nil {
		t.Fatal(err)
	}
	other, err := FenceKey(syntax.DID("did:plc:bbb"))
	if err != nil {
		t.Fatal(err)
	}
	if one != two {
		t.Fatalf("FenceKey is unstable: %d != %d", one, two)
	}
	if one == other {
		t.Fatalf("test owners unexpectedly share fence key %d", one)
	}
}
