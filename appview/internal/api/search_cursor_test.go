package api_test

import (
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

func TestSearchChronologicalCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 6, 19, 12, 0, 0, 123, time.UTC)
	cursor, err := api.EncodeChronologicalSearchCursor(createdAt, "at://did:plc:alice/social.craftsky.feed.post/aaa")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	gotCreatedAt, gotURI, err := api.DecodeChronologicalSearchCursor(cursor)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gotCreatedAt != createdAt || gotURI != "at://did:plc:alice/social.craftsky.feed.post/aaa" {
		t.Fatalf("decoded = %v/%v", gotCreatedAt, gotURI)
	}
}

func TestSearchPopularityCursorRoundTripAndInvalid(t *testing.T) {
	rankedAt := time.Date(2026, 6, 19, 12, 34, 56, 0, time.UTC)
	createdAt := rankedAt.Add(-2 * time.Hour)
	cursor, err := api.EncodePopularityCursor(rankedAt, 1.25, createdAt, "at://did:plc:alice/social.craftsky.feed.post/bbb")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := api.DecodePopularityCursor(cursor)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !decoded.RankedAt.Equal(rankedAt) || decoded.Score != 1.25 || !decoded.CreatedAt.Equal(createdAt) || decoded.URI == "" {
		t.Fatalf("decoded = %+v", decoded)
	}
	if _, _, err := api.DecodeChronologicalSearchCursor(cursor); err != envelope.ErrInvalidCursor {
		t.Fatalf("chronological decode of popularity cursor = %v, want invalid cursor", err)
	}
}
