package sourcevalidation

import (
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/tap"
)

func TestProtocolRejectsInvalidPublishedRecords(t *testing.T) {
	for name, record := range map[string]string{
		"missing_type":                   `{"text":"hello","sponsored":false,"createdAt":"2026-10-12T10:00:00Z"}`,
		"null_text_and_sponsored":        `{"$type":"social.craftsky.feed.post","text":null,"sponsored":null,"createdAt":"2026-10-12T10:00:00Z"}`,
		"image_missing_blob":             `{"$type":"social.craftsky.feed.post","text":"hello","sponsored":false,"createdAt":"2026-10-12T10:00:00Z","images":[{}]}`,
		"invalid_datetime_negative_zero": `{"$type":"social.craftsky.feed.post","text":"hello","sponsored":false,"createdAt":"2026-10-12T10:00:00-00:00"}`,
		"invalid_reply":                  `{"$type":"social.craftsky.feed.post","text":"hello","sponsored":false,"createdAt":"2026-10-12T10:00:00Z","reply":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			r := Validate(tap.Event{Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaaa2", Action: "create", Record: json.RawMessage(record)})
			t.Logf("result: structural=%s semantic=%s reason=%s", r.StructuralStatus, r.SemanticStatus, r.Reason)
			if r.StructuralStatus != Invalid && r.SemanticStatus != Invalid {
				t.Fatalf("invalid published record accepted")
			}
		})
	}
}
