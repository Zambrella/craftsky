package index

import (
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/imagesafety"
)

func TestSelectProfileImage(t *testing.T) {
	t.Parallel()

	previous := &profileImage{CID: syntax.CID("bafyprevious"), MIMEType: "image/jpeg"}
	candidate := &profileImage{CID: syntax.CID("bafycandidate"), MIMEType: "image/png"}

	tests := []struct {
		name     string
		state    imagesafety.State
		previous *profileImage
		want     *profileImage
	}{
		{name: "clear promotes candidate", state: imagesafety.StateClear, previous: previous, want: candidate},
		{name: "pending retains previous", state: imagesafety.StatePending, previous: previous, want: previous},
		{name: "match retains previous", state: imagesafety.StateMatch, previous: previous, want: previous},
		{name: "unavailable retains previous", state: imagesafety.StateUnavailable, previous: previous, want: previous},
		{name: "error retains previous", state: imagesafety.StateError, previous: previous, want: previous},
		{name: "unknown retains previous", state: imagesafety.State("unknown"), previous: previous, want: previous},
		{name: "non-clear without history uses placeholder", state: imagesafety.StatePending, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := selectProfileImage(candidate, test.state, test.previous); got != test.want {
				t.Fatalf("selectProfileImage() = %+v, want %+v", got, test.want)
			}
		})
	}
}
