package api

import (
	"testing"

	"social.craftsky/appview/internal/relationships"
)

func TestSavedPostPolicyAllowsMutedButNotBlocks(t *testing.T) {
	for _, test := range []struct {
		name  string
		state relationships.State
		want  bool
	}{
		{name: "eligible", want: true},
		{name: "muted direct access", state: relationships.State{Muted: true}, want: true},
		{name: "blocking", state: relationships.State{Blocking: true}},
		{name: "blocked by", state: relationships.State{BlockedBy: true}},
		{name: "block wins over mute", state: relationships.State{Muted: true, Blocking: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := savedPostPolicyAllows(test.state); got != test.want {
				t.Fatalf("savedPostPolicyAllows(%+v) = %v, want %v", test.state, got, test.want)
			}
		})
	}
}
