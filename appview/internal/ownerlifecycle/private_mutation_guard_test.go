package ownerlifecycle

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestRequirePrivateMutationAuthorityDecisionTable(t *testing.T) {
	owner := syntax.DID("did:plc:private-mutation-owner")
	activeTarget := syntax.DID("did:plc:private-mutation-active-target")
	departedTarget := syntax.DID("did:plc:private-mutation-departed-target")
	terminalTarget := syntax.DID("did:plc:private-mutation-terminal-target")
	unknownTarget := syntax.DID("did:plc:private-mutation-unknown-target")
	states := map[syntax.DID]Lifecycle{
		owner:          {Owner: owner, State: StateActive, Generation: 7},
		activeTarget:   {Owner: activeTarget, State: StateActive, Generation: 2},
		departedTarget: {Owner: departedTarget, State: StateDeparted, Generation: 4},
		terminalTarget: {Owner: terminalTarget, State: StateTerminal, Generation: 9},
	}

	tests := []struct {
		name     string
		states   map[syntax.DID]Lifecycle
		expected int64
		targets  []syntax.DID
		wantErr  error
	}{
		{name: "active exact generation", states: states, expected: 7},
		{name: "active target", states: states, expected: 7, targets: []syntax.DID{activeTarget}},
		{name: "departed target remains addressable", states: states, expected: 7, targets: []syntax.DID{departedTarget}},
		{name: "unknown external target remains addressable", states: states, expected: 7, targets: []syntax.DID{unknownTarget}},
		{name: "generation is required", states: states, wantErr: ErrGenerationRequired},
		{name: "stale generation", states: states, expected: 6, wantErr: ErrGenerationChanged},
		{name: "owner absent", states: map[syntax.DID]Lifecycle{}, expected: 7, wantErr: ErrOwnerNotActive},
		{name: "owner departed", states: map[syntax.DID]Lifecycle{owner: {Owner: owner, State: StateDeparted, Generation: 7}}, expected: 7, wantErr: ErrOwnerNotActive},
		{name: "terminal target", states: states, expected: 7, targets: []syntax.DID{terminalTarget}, wantErr: ErrTerminalOwner},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := RequirePrivateMutationAuthority(test.states, owner, test.expected, test.targets)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestExpectedGenerationContextRejectsInvalidValues(t *testing.T) {
	if _, ok := ExpectedGeneration(context.Background()); ok {
		t.Fatal("background context unexpectedly carried an owner generation")
	}
	for _, generation := range []int64{-1, 0} {
		if _, ok := ExpectedGeneration(WithExpectedGeneration(context.Background(), generation)); ok {
			t.Fatalf("generation %d was accepted", generation)
		}
	}
	if generation, ok := ExpectedGeneration(WithExpectedGeneration(context.Background(), 3)); !ok || generation != 3 {
		t.Fatalf("generation = %d/%t, want 3/true", generation, ok)
	}
}

func TestRequireNonTerminalTargetsAllowsUnknownAndDepartedButRejectsTerminal(t *testing.T) {
	departed := syntax.DID("did:plc:derived-cache-departed")
	terminal := syntax.DID("did:plc:derived-cache-terminal")
	unknown := syntax.DID("did:plc:derived-cache-unknown")
	states := map[syntax.DID]Lifecycle{
		departed: {Owner: departed, State: StateDeparted, Generation: 2},
		terminal: {Owner: terminal, State: StateTerminal, Generation: 4},
	}
	if err := RequireNonTerminalTargets(states, []syntax.DID{departed, unknown}); err != nil {
		t.Fatalf("non-terminal targets rejected: %v", err)
	}
	if err := RequireNonTerminalTargets(states, []syntax.DID{unknown, terminal}); !errors.Is(err, ErrTerminalOwner) {
		t.Fatalf("terminal target error = %v, want ErrTerminalOwner", err)
	}
}
