package pdscommands

import "testing"

func TestCommandStateTransitionsAreExplicitAndTerminalOutcomesImmutable(t *testing.T) {
	states := []CommandState{CommandPrepared, CommandDispatching, CommandAccepted, CommandAmbiguous, CommandRejected}
	allowed := map[[2]CommandState]bool{
		{CommandPrepared, CommandAccepted}:     true,
		{CommandPrepared, CommandDispatching}:  true,
		{CommandPrepared, CommandRejected}:     true,
		{CommandDispatching, CommandAccepted}:  true,
		{CommandDispatching, CommandAmbiguous}: true,
		{CommandDispatching, CommandRejected}:  true,
		{CommandAmbiguous, CommandDispatching}: true,
		{CommandAmbiguous, CommandAccepted}:    true,
		{CommandAmbiguous, CommandRejected}:    true,
	}
	for _, from := range states {
		for _, to := range states {
			want := allowed[[2]CommandState{from, to}]
			if got := CanTransition(from, to); got != want {
				t.Fatalf("CanTransition(%s, %s) = %t, want %t", from, to, got, want)
			}
		}
	}
	for _, terminal := range []CommandState{CommandAccepted, CommandRejected} {
		for _, next := range states {
			if CanTransition(terminal, next) {
				t.Fatalf("terminal %s transitioned to %s", terminal, next)
			}
		}
	}
}
