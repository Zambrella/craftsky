package envelope_test

import (
	"testing"

	"social.craftsky/appview/internal/api/envelope"
)

func TestCursor_RoundTrip(t *testing.T) {
	in := map[string]any{
		"after": "2026-04-21T12:00:00Z",
		"id":    float64(42), // json numbers decode as float64
	}
	encoded, err := envelope.EncodeCursor(in)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if encoded == "" {
		t.Fatal("encoded cursor should not be empty")
	}

	out, err := envelope.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["after"] != in["after"] {
		t.Errorf("after = %v, want %v", out["after"], in["after"])
	}
	if out["id"] != in["id"] {
		t.Errorf("id = %v, want %v", out["id"], in["id"])
	}
}

func TestCursor_EmptyStringDecodesToEmptyMap(t *testing.T) {
	out, err := envelope.DecodeCursor("")
	if err != nil {
		t.Fatalf("decode empty: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("empty cursor → non-empty map: %v", out)
	}
}

func TestCursor_MalformedReturnsInvalidCursorError(t *testing.T) {
	_, err := envelope.DecodeCursor("not-valid-base64url!!!")
	if err == nil {
		t.Fatal("expected error for malformed cursor")
	}
	if err != envelope.ErrInvalidCursor {
		t.Errorf("err = %v, want ErrInvalidCursor", err)
	}
}

func TestCursor_NonJSONPayloadReturnsInvalidCursorError(t *testing.T) {
	// A valid base64url that doesn't decode to JSON.
	bad := "bm90LWpzb24" // "not-json"
	_, err := envelope.DecodeCursor(bad)
	if err != envelope.ErrInvalidCursor {
		t.Errorf("err = %v, want ErrInvalidCursor", err)
	}
}
