package relationships

import "github.com/bluesky-social/indigo/atproto/syntax"

// State is the authenticated viewer's relationship to one subject. Muted is
// intentionally one-way; either block direction activates symmetric policy.
type State struct {
	Muted     bool
	Blocking  bool
	BlockedBy bool
}

func (s State) HasBlock() bool {
	return s.Blocking || s.BlockedBy
}

type BlockMutationResult struct {
	State State
	URI   syntax.ATURI
	CID   syntax.CID
	Rkey  syntax.RecordKey
}
