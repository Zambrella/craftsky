package scheduledposts

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestPayloadRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload Payload
	}{
		{
			name: "standard",
			payload: Payload{
				Kind:   PostKindStandard,
				Text:   "A linked #project with a mention",
				Facets: json.RawMessage(`[{"index":{"byteStart":2,"byteEnd":8},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"project"}]}]`),
				Langs:  []string{"en-GB", "fr"},
				Media: []PayloadMedia{
					{ID: "11111111-1111-4111-8111-111111111111", Alt: "first image", Width: 1200, Height: 800},
					{ID: "22222222-2222-4222-8222-222222222222", Alt: "second image", Width: 800, Height: 1200},
				},
				External: &PayloadExternal{
					SourceURI:    "https://source.example/pattern",
					URI:          "https://final.example/pattern#section",
					Title:        "Frozen pattern",
					Description:  "Frozen description",
					ThumbMediaID: "55555555-5555-4555-8555-555555555555",
				},
			},
		},
		{
			name: "project",
			payload: Payload{
				Kind:    PostKindProject,
				Text:    "Finished cardigan",
				Langs:   []string{"en"},
				Project: json.RawMessage(`{"common":{"craftType":"social.craftsky.feed.defs#knitting","title":"Cardigan","materials":[{"text":"Wool","facets":[{"index":{"byteStart":0,"byteEnd":4},"features":[]}]}],"colors":["blue","cream"],"designTags":["cables","winter"]},"details":{"$type":"social.craftsky.project.knitting#details","projectType":"garment"}}`),
				Media: []PayloadMedia{
					{ID: "33333333-3333-4333-8333-333333333333", Alt: "front", Width: 1600, Height: 1200},
					{ID: "44444444-4444-4444-8444-444444444444", Alt: "detail", Width: 1200, Height: 1200},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := EncodePayload(test.payload)
			if err != nil {
				t.Fatalf("EncodePayload() error = %v", err)
			}
			decoded, err := DecodePayload(encoded)
			if err != nil {
				t.Fatalf("DecodePayload() error = %v", err)
			}
			if !reflect.DeepEqual(decoded, test.payload) {
				t.Fatalf("DecodePayload() = %#v, want %#v", decoded, test.payload)
			}
			reencoded, err := EncodePayload(decoded)
			if err != nil {
				t.Fatalf("EncodePayload(decoded) error = %v", err)
			}
			if !bytes.Equal(reencoded, encoded) {
				t.Fatalf("payload bytes changed across round trip\nfirst:  %s\nsecond: %s", encoded, reencoded)
			}
			for index, media := range decoded.Media {
				if media.ID != test.payload.Media[index].ID {
					t.Fatalf("media order changed at %d: got %q, want %q", index, media.ID, test.payload.Media[index].ID)
				}
			}
		})
	}
}

func TestUT018ScheduledExternalEligibility(t *testing.T) {
	t.Parallel()

	if err := ValidateScheduleEligibility(PostShape{Kind: PostKindStandard, HasExternal: true}); err != nil {
		t.Fatalf("standard external eligibility = %v", err)
	}
	if err := ValidateScheduleEligibility(PostShape{Kind: PostKindProject, HasExternal: true}); !errors.Is(err, ErrIneligibleScheduledPost) {
		t.Fatalf("project external eligibility = %v, want %v", err, ErrIneligibleScheduledPost)
	}
}
