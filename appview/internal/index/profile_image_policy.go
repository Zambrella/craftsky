package index

import (
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/imagesafety"
)

type profileImage struct {
	CID      syntax.CID
	MIMEType string
}

func selectProfileImage(candidate *profileImage, state imagesafety.State, previous *profileImage) *profileImage {
	if candidate != nil && state.DisplayEligible() {
		return candidate
	}
	return previous
}
