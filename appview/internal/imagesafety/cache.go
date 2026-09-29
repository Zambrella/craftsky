package imagesafety

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type ScanKey struct {
	BlobCID       syntax.CID
	ScannerID     string
	PolicyVersion string
	CorpusVersion string
}

func (key ScanKey) Valid() bool {
	return key.BlobCID != "" &&
		key.ScannerID != "" &&
		key.PolicyVersion != "" &&
		key.CorpusVersion != ""
}

type Result struct {
	Key         ScanKey
	State       State
	CompletedAt time.Time
}

func (result Result) ReusableFor(key ScanKey) bool {
	return result.Key.Valid() && key.Valid() && result.Key == key && result.State.Terminal()
}
