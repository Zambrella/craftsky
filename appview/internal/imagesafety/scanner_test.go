package imagesafety

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestFixtureScannerIsDeterministicAndFailsClosed(t *testing.T) {
	t.Parallel()

	fixtures := map[syntax.CID]State{
		"bafyreiclear":        StateClear,
		"bafyreicmatch":       StateMatch,
		"bafyreicunavailable": StateUnavailable,
		"bafyreicerror":       StateError,
	}
	scanner := NewFixtureScanner(fixtures)
	var _ Scanner = scanner

	for cid, want := range fixtures {
		result, err := scanner.Scan(context.Background(), ScanInput{
			BlobCID:  cid,
			MIMEType: "image/png",
			Size:     3,
			Content:  bytes.NewReader([]byte("png")),
		})
		if err != nil {
			t.Fatalf("Scan(%q) error = %v", cid, err)
		}
		if result.State != want {
			t.Errorf("Scan(%q) state = %q, want %q", cid, result.State, want)
		}
	}

	result, err := scanner.Scan(context.Background(), ScanInput{
		BlobCID:  "bafyreicunknown",
		MIMEType: "image/png",
		Size:     3,
		Content:  bytes.NewReader([]byte("png")),
	})
	if !errors.Is(err, ErrUnknownFixture) {
		t.Fatalf("unknown fixture error = %v, want ErrUnknownFixture", err)
	}
	if result.State != StateError {
		t.Errorf("unknown fixture state = %q, want %q", result.State, StateError)
	}
}
