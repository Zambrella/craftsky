package schema_test

import (
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/lexicon/schema"
)

// IT-001 / IT-010: old records remain valid; declared edits use protocol datetimes.
func TestPostEditTimestampContract(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		valid       bool
	}{
		{"old record", "", true},
		{"edited record", `,"editedAt":"2026-10-26T12:00:00.123Z"`, true},
		{"invalid time", `,"editedAt":"yesterday"`, false},
		{"wrong type", `,"editedAt":42`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello","sponsored":false,"createdAt":"2026-01-01T00:00:00Z"` + tc.field + `}`)
			if err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post"); (err == nil) != tc.valid {
				t.Fatalf("valid=%v error=%v", tc.valid, err)
			}
		})
	}
}
