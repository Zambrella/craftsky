package scheduledposts

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

func TestNewPublicationTombstoneDropsPrivateContent(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:ewvi7nxzyoun6zhxrhs64oiz")
	privatePayload := []byte(`{"text":"PRIVATE-PAYLOAD-CANARY","alt":"PRIVATE-ALT-CANARY"}`)
	publishedAt := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	completed := CompletedPublication{
		ScheduleID:             uuid.MustParse("11111111-1111-4111-8111-111111111111"),
		Owner:                  owner,
		OperationID:            uuid.MustParse("22222222-2222-4222-8222-222222222222"),
		RequestHash:            sha256.Sum256([]byte("canonical request")),
		URI:                    syntax.ATURI("at://did:plc:ewvi7nxzyoun6zhxrhs64oiz/social.craftsky.feed.post/3m123"),
		CID:                    syntax.CID("bafyrei-publication-cid"),
		PublishedAt:            publishedAt,
		PrivatePayload:         privatePayload,
		PrivateMediaObjectKeys: []string{"scheduled-media/PRIVATE-OBJECT-CANARY"},
	}

	tombstone, err := NewPublicationTombstone(completed)
	if err != nil {
		t.Fatalf("NewPublicationTombstone() error = %v", err)
	}
	if tombstone.ScheduleID != completed.ScheduleID ||
		tombstone.Owner != completed.Owner ||
		tombstone.OperationID != completed.OperationID ||
		tombstone.RequestHash != completed.RequestHash ||
		tombstone.URI != completed.URI ||
		tombstone.CID != completed.CID ||
		!tombstone.PublishedAt.Equal(publishedAt) ||
		!tombstone.ExpiresAt.Equal(publishedAt.Add(30*24*time.Hour)) {
		t.Fatalf("tombstone identity fields = %#v", tombstone)
	}

	encoded, err := json.Marshal(tombstone)
	if err != nil {
		t.Fatalf("json.Marshal(tombstone) error = %v", err)
	}
	for _, canary := range []string{"PRIVATE-PAYLOAD-CANARY", "PRIVATE-ALT-CANARY", "PRIVATE-OBJECT-CANARY"} {
		if strings.Contains(string(encoded), canary) || strings.Contains(fmt.Sprint(tombstone), canary) {
			t.Fatalf("private canary %q survived tombstone projection", canary)
		}
	}
}
