package scheduledposts

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestFreezePublicationIdentityAndBody(t *testing.T) {
	t.Parallel()

	owner, err := syntax.ParseDID("did:plc:ewvi7nxzyoun6zhxrhs64oiz")
	if err != nil {
		t.Fatal(err)
	}
	firstAttemptAt := time.Date(2026, time.July, 31, 12, 0, 0, 123000, time.UTC)
	firstTID := syntax.NewTIDFromTime(firstAttemptAt, 0)
	firstBody := []byte(`{"$type":"social.craftsky.feed.post","text":"first","createdAt":"2026-07-31T12:00:00.000123Z"}`)

	frozen, err := FreezePublication(nil, PublicationFreezeRequest{
		Owner:     owner,
		TID:       firstTID,
		CreatedAt: firstAttemptAt,
		Body:      firstBody,
	})
	if err != nil {
		t.Fatalf("FreezePublication() error = %v", err)
	}
	wantRkey, err := syntax.ParseRecordKey(firstTID.String())
	if err != nil {
		t.Fatal(err)
	}
	if frozen.Rkey != wantRkey {
		t.Fatalf("Rkey = %q, want %q", frozen.Rkey, wantRkey)
	}
	if !frozen.CreatedAt.Equal(firstAttemptAt) {
		t.Fatalf("CreatedAt = %s, want %s", frozen.CreatedAt, firstAttemptAt)
	}
	if !bytes.Equal(frozen.RecordBytes, firstBody) {
		t.Fatalf("RecordBytes = %s, want %s", frozen.RecordBytes, firstBody)
	}
	if wantHash := sha256.Sum256(firstBody); frozen.RecordHash != wantHash {
		t.Fatalf("RecordHash = %x, want %x", frozen.RecordHash, wantHash)
	}
	wantURI := syntax.ATURI("at://did:plc:ewvi7nxzyoun6zhxrhs64oiz/social.craftsky.feed.post/" + wantRkey.String())
	if frozen.IntendedURI != wantURI {
		t.Fatalf("IntendedURI = %q, want %q", frozen.IntendedURI, wantURI)
	}

	recovered, err := FreezePublication(&frozen, PublicationFreezeRequest{
		Owner:     owner,
		TID:       syntax.NewTIDFromTime(firstAttemptAt.Add(time.Hour), 1),
		CreatedAt: firstAttemptAt.Add(time.Hour),
		Body:      []byte(`{"changed":true}`),
	})
	if err != nil {
		t.Fatalf("FreezePublication(existing) error = %v", err)
	}
	if recovered.Rkey != frozen.Rkey ||
		!recovered.CreatedAt.Equal(frozen.CreatedAt) ||
		recovered.IntendedURI != frozen.IntendedURI ||
		recovered.RecordHash != frozen.RecordHash ||
		!bytes.Equal(recovered.RecordBytes, frozen.RecordBytes) {
		t.Fatalf("recovery changed frozen publication: got %#v, want %#v", recovered, frozen)
	}
}
