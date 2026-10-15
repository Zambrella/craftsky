package sourcevalidation

import (
	"encoding/json"
	"fmt"
	"strings"
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

func TestPublishedPostImageCount(t *testing.T) {
	t.Parallel()
	for _, count := range []int{5, 8, 9} {
		for _, sponsored := range []bool{false, true} {
			for _, action := range []string{"create", "update"} {
				t.Run(fmt.Sprintf("%d_images/sponsored_%t/%s", count, sponsored, action), func(t *testing.T) {
					images := make([]string, count)
					for i := range images {
						images[i] = `{"image":{"$type":"blob","ref":{"$link":"bafkreia7kykscvecx5ee2w6uduz4yaphmtgda4cm32rypblshwjgzxj6na"},"mimeType":"image/jpeg","size":1476864},"aspectRatio":{"width":3000,"height":2002}}`
					}
					record := fmt.Sprintf(`{"$type":"social.craftsky.feed.post","text":"Moaarrr images! Sponsored btw.","sponsored":%t,"createdAt":"2026-10-15T14:55:09.76848Z","langs":["en"],"images":[%s]}`, sponsored, strings.Join(images, ","))
					result := Validate(tap.Event{Collection: "social.craftsky.feed.post", Rkey: "3mxwcxyxw5c22", Action: action, Record: json.RawMessage(record)})
					if count <= 8 {
						if result.StructuralStatus != Valid || result.SemanticStatus != Valid || result.Reason != "" {
							t.Fatalf("valid published post rejected: %+v", result)
						}
					} else if result.StructuralStatus != Invalid || result.SemanticStatus != Invalid || result.Reason != "invalid_lexicon" {
						t.Fatalf("post exceeding lexicon image limit accepted: %+v", result)
					}
				})
			}
		}
	}
}
