package schema_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"social.craftsky/appview/internal/lexicon/schema"
	"social.craftsky/appview/internal/observability"
)

func TestOversizedPublishedBlobRetainsSafeCause(t *testing.T) {
	raw := json.RawMessage(`{"$type":"social.craftsky.business.profile","products":[{"title":"private-canary","uri":"https://shop.example/yarn","image":{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":11189477}}}]}`)
	err := schema.ValidateBusinessRecord(raw, "social.craftsky.business.profile")
	var failure *schema.ValidationError
	if !errors.As(err, &failure) || failure.BlobBytes != 11189477 || errors.Unwrap(failure) == nil {
		t.Fatalf("missing typed original validator cause: %+v", err)
	}
	encoded, _ := json.Marshal(observability.DescribeError(err, nil))
	if !strings.Contains(string(encoded), "11189477") || !strings.Contains(string(encoded), "lexicon size limit") || strings.Contains(string(encoded), "private-canary") {
		t.Fatalf("unsafe/unhelpful cause: %s", encoded)
	}
}

func TestPostImageCountBoundary(t *testing.T) {
	t.Parallel()
	for _, count := range []int{8, 9} {
		t.Run(fmt.Sprintf("%d images", count), func(t *testing.T) {
			images := make([]string, count)
			for i := range images {
				images[i] = `{"image":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":1}}`
			}
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hi","sponsored":false,"createdAt":"2026-10-14T00:00:00Z","images":[` + strings.Join(images, ",") + `]}`)
			err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post")
			if count == 8 && err != nil {
				t.Fatalf("eight images must validate: %v", err)
			}
			if count == 9 && err == nil {
				t.Fatal("nine images must fail validation")
			}
		})
	}
}

func TestSewingFitNotesBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		valid      bool
	}{
		{"600 graphemes", strings.Repeat("é", 600), true},
		{"601 graphemes", strings.Repeat("a", 601), false},
		{"6000 bytes", strings.Repeat("👨‍👩‍👧‍👦", 240), true},
		{"over 6000 bytes", strings.Repeat("👨‍👩‍👧‍👦", 241), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			notes, _ := json.Marshal(tc.text)
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-10-16T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing"},"details":{"$type":"social.craftsky.project.sewing#details","fitNotes":` + string(notes) + `}}}`)
			if err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post"); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}
