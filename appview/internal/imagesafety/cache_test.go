package imagesafety

import (
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestResultReusableFor(t *testing.T) {
	t.Parallel()

	key := ScanKey{
		BlobCID:       syntax.CID("bafyreiblob"),
		ScannerID:     "fixture-scanner",
		PolicyVersion: "policy-v1",
		CorpusVersion: "corpus-v1",
	}

	tests := []struct {
		name   string
		result Result
		key    ScanKey
		want   bool
	}{
		{name: "same clear result", result: Result{Key: key, State: StateClear}, key: key, want: true},
		{name: "same match result", result: Result{Key: key, State: StateMatch}, key: key, want: true},
		{name: "pending result", result: Result{Key: key, State: StatePending}, key: key},
		{name: "changed cid", result: Result{Key: key, State: StateClear}, key: withBlobCID(key, "bafyreinew")},
		{name: "changed scanner", result: Result{Key: key, State: StateClear}, key: withScannerID(key, "other-scanner")},
		{name: "changed policy", result: Result{Key: key, State: StateClear}, key: withPolicyVersion(key, "policy-v2")},
		{name: "changed corpus", result: Result{Key: key, State: StateClear}, key: withCorpusVersion(key, "corpus-v2")},
		{name: "incomplete stored key", result: Result{Key: ScanKey{BlobCID: key.BlobCID}, State: StateClear}, key: key},
		{name: "incomplete requested key", result: Result{Key: key, State: StateClear}, key: ScanKey{BlobCID: key.BlobCID}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.result.ReusableFor(test.key); got != test.want {
				t.Errorf("Result.ReusableFor() = %t, want %t", got, test.want)
			}
		})
	}
}

func withBlobCID(key ScanKey, cid syntax.CID) ScanKey {
	key.BlobCID = cid
	return key
}

func withScannerID(key ScanKey, scannerID string) ScanKey {
	key.ScannerID = scannerID
	return key
}

func withPolicyVersion(key ScanKey, policyVersion string) ScanKey {
	key.PolicyVersion = policyVersion
	return key
}

func withCorpusVersion(key ScanKey, corpusVersion string) ScanKey {
	key.CorpusVersion = corpusVersion
	return key
}
