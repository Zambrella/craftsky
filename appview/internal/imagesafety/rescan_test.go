package imagesafety

import (
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestNeedsRescanIgnoresElapsedTime(t *testing.T) {
	t.Parallel()

	key := ScanKey{
		BlobCID:       syntax.CID("bafyreiblob"),
		ScannerID:     "fixture-scanner",
		PolicyVersion: "policy-v1",
		CorpusVersion: "corpus-v1",
	}
	completed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result := Result{Key: key, State: StateClear, CompletedAt: completed}

	if NeedsRescan(result, key, false, completed.AddDate(10, 0, 0)) {
		t.Fatal("elapsed time alone must not require a rescan")
	}
	if !NeedsRescan(result, key, true, completed) {
		t.Fatal("authorized targeted request must require a rescan")
	}
	if !NeedsRescan(result, withPolicyVersion(key, "policy-v2"), false, completed) {
		t.Fatal("changed approved policy key must require a rescan")
	}
	if !NeedsRescan(result, withCorpusVersion(key, "corpus-v2"), false, completed) {
		t.Fatal("changed approved corpus key must require a rescan")
	}
	if !NeedsRescan(Result{Key: key, State: StateError}, key, false, completed) {
		t.Fatal("non-terminal result must require scanning")
	}
}
