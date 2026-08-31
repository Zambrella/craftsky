package api

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
)

func TestBusinessEventCursorAndLimits(t *testing.T) {
	codec, err := NewEventCursorCodec(bytes.Repeat([]byte{0x42}, 32))
	if err != nil {
		t.Fatalf("NewEventCursorCodec: %v", err)
	}
	scope := syntax.DID("did:plc:alice")
	want := EventCursor{
		Kind:     EventCursorUpcoming,
		AsOf:     time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC),
		StartsAt: time.Date(2026, time.September, 2, 10, 0, 0, 0, time.UTC),
		URI:      syntax.ATURI("at://did:plc:alice/social.craftsky.business.event/3mexample"),
	}
	encoded, err := codec.Encode(want, scope)
	if err != nil {
		t.Fatalf("EncodeEventCursor: %v", err)
	}
	if encoded == "" || strings.Contains(encoded, string(want.URI)) {
		t.Fatalf("cursor = %q, want nonempty opaque value", encoded)
	}
	got, err := codec.Decode(encoded, EventCursorUpcoming, scope)
	if err != nil {
		t.Fatalf("DecodeEventCursor: %v", err)
	}
	if got.Kind != want.Kind || !got.AsOf.Equal(want.AsOf) || !got.StartsAt.Equal(want.StartsAt) || got.URI != want.URI {
		t.Fatalf("decoded cursor = %+v, want %+v", got, want)
	}

	wrongKind, err := codec.Encode(EventCursor{
		Kind: EventCursorManagement, StartsAt: want.StartsAt, URI: want.URI,
	}, scope)
	if err != nil {
		t.Fatalf("encode wrong-kind cursor: %v", err)
	}
	tampered := encoded[:len(encoded)-1] + "A"
	for _, invalid := range []struct {
		value string
		scope syntax.DID
	}{
		{value: "bad@@", scope: scope},
		{value: wrongKind, scope: scope},
		{value: tampered, scope: scope},
		{value: encoded, scope: "did:plc:bob"},
	} {
		if _, err := codec.Decode(invalid.value, EventCursorUpcoming, invalid.scope); !errors.Is(err, envelope.ErrInvalidCursor) {
			t.Errorf("Decode(%q) error = %v, want ErrInvalidCursor", invalid.value, err)
		}
	}

	for _, tt := range []struct {
		raw          string
		defaultLimit int
		want         int
	}{
		{raw: "", defaultLimit: 10, want: 10},
		{raw: "10", defaultLimit: 10, want: 10},
		{raw: "50", defaultLimit: 20, want: 50},
		{raw: "51", defaultLimit: 20, want: 50},
	} {
		got, err := NormalizeEventLimit(tt.raw, tt.defaultLimit)
		if err != nil || got != tt.want {
			t.Errorf("NormalizeEventLimit(%q, %d) = (%d, %v), want (%d, nil)", tt.raw, tt.defaultLimit, got, err, tt.want)
		}
	}
	for _, invalid := range []string{"0", "-1", "not-a-number"} {
		if _, err := NormalizeEventLimit(invalid, 10); err == nil {
			t.Errorf("NormalizeEventLimit(%q) succeeded", invalid)
		}
	}
}

func TestOwnerBusinessEventCursorKindsFreezeCutoff(t *testing.T) {
	codec, err := NewEventCursorCodec(bytes.Repeat([]byte{0x24}, 32))
	if err != nil {
		t.Fatalf("NewEventCursorCodec: %v", err)
	}
	scope := syntax.DID("did:plc:owner")
	cutoff := time.Date(2026, time.August, 30, 12, 0, 0, 0, time.UTC)
	startsAt := cutoff.Add(time.Hour)
	uri := syntax.ATURI("at://did:plc:owner/social.craftsky.business.event/3mexample")

	for _, kind := range []EventCursorKind{EventCursorOwnerUpcoming, EventCursorOwnerHistory} {
		encoded, err := codec.Encode(EventCursor{Kind: kind, AsOf: cutoff, StartsAt: startsAt, URI: uri}, scope)
		if err != nil {
			t.Fatalf("Encode(%s): %v", kind, err)
		}
		decoded, err := codec.Decode(encoded, kind, scope)
		if err != nil {
			t.Fatalf("Decode(%s): %v", kind, err)
		}
		if !decoded.AsOf.Equal(cutoff) || !decoded.StartsAt.Equal(startsAt) || decoded.URI != uri {
			t.Fatalf("decoded %s cursor = %+v", kind, decoded)
		}
		other := EventCursorOwnerUpcoming
		if kind == EventCursorOwnerUpcoming {
			other = EventCursorOwnerHistory
		}
		if _, err := codec.Decode(encoded, other, scope); !errors.Is(err, envelope.ErrInvalidCursor) {
			t.Errorf("Decode(%s as %s) error = %v, want ErrInvalidCursor", kind, other, err)
		}
	}
}
