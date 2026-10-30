package imagesafety

import (
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

type ScanKey struct {
	BlobCID                syntax.CID
	ScannerID              string
	PolicyVersion          string
	CorpusVersion          string
	developmentAutoApprove bool
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
	return result.Key.Valid() && key.Valid() && result.Key.BlobCID == key.BlobCID && result.Key.ScannerID == key.ScannerID &&
		result.Key.PolicyVersion == key.PolicyVersion && result.Key.CorpusVersion == key.CorpusVersion && result.State.Terminal()
}

// AutoApprovesDevelopmentImages is set only by a validated development config.
func (key ScanKey) AutoApprovesDevelopmentImages() bool {
	return key.developmentAutoApprove && key.ScannerID == DevelopmentApprovalScannerID &&
		key.PolicyVersion == DevelopmentApprovalPolicyVersion && key.CorpusVersion == DevelopmentApprovalCorpusVersion
}
