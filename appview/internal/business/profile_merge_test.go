package business

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestProfileReplacementMergeAndSafeHydration(t *testing.T) {
	existing := json.RawMessage(`{
		"$type":"social.craftsky.business.profile",
		"tagline":"old tagline",
		"hoursNote":"old hours",
		"products":[{"title":"Old","uri":"https://example.com/old"}],
		"futureExtension":{"enabled":true}
	}`)
	replacement := json.RawMessage(`{"tagline":"new tagline"}`)

	merged, err := MergeProfileReplacement(existing, replacement)
	if err != nil {
		t.Fatalf("MergeProfileReplacement: %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("decode merged profile: %v", err)
	}
	for _, cleared := range []string{"hoursNote", "products"} {
		if _, ok := got[cleared]; ok {
			t.Errorf("omitted known field %q survived replacement", cleared)
		}
	}
	if string(got["tagline"]) != `"new tagline"` {
		t.Fatalf("merged tagline = %s", got["tagline"])
	}
	if string(got["$type"]) != `"social.craftsky.business.profile"` {
		t.Fatalf("merged $type = %s", got["$type"])
	}
	if !bytes.Equal(got["futureExtension"], json.RawMessage(`{"enabled":true}`)) {
		t.Fatalf("future extension = %s", got["futureExtension"])
	}

	independent := json.RawMessage(`{
		"$type":"social.craftsky.business.profile",
		"tagline":"Safe declaration",
		"primaryAction":{"type":"shop","destination":"http://unsafe.example"},
		"futureExtension":"preserve"
	}`)
	before := append(json.RawMessage(nil), independent...)
	view, err := HydrateProfile(independent)
	if err != nil {
		t.Fatalf("HydrateProfile: %v", err)
	}
	if view.Tagline != "Safe declaration" || view.PrimaryAction != nil {
		t.Fatalf("hydrated profile = %+v", view)
	}
	if !bytes.Equal(independent, before) {
		t.Fatal("safe hydration mutated raw source")
	}
}

func TestProfileReplacementPreservesUnknownExtensionForCompleteKnownReplacements(t *testing.T) {
	const extension = `{"nested":{"enabled":true,"sequence":9007199254740993}}`
	existing := json.RawMessage(`{
		"$type":"social.craftsky.business.profile",
		"businessTypes":["teacher"],
		"offerings":["classes"],
		"tagline":"Original details",
		"hoursNote":"Weekdays",
		"serviceArea":"Remove on replacement",
		"location":{"country":"US","locality":"Portland"},
		"primaryAction":{"type":"shop","destination":"https://example.com/shop"},
		"products":[{"title":"Original product","uri":"https://example.com/original"}],
		"com.example.extension":` + extension + `
	}`)
	tests := []struct {
		name           string
		replacement    json.RawMessage
		changedField   string
		changedValue   string
		preservedField string
		preservedValue string
	}{
		{
			name: "detail-only",
			replacement: json.RawMessage(`{
				"businessTypes":["teacher"],"offerings":["classes"],
				"tagline":"Updated details","hoursNote":"Weekdays",
				"location":{"country":"US","locality":"Portland"},
				"primaryAction":{"type":"shop","destination":"https://example.com/shop"},
				"products":[{"title":"Original product","uri":"https://example.com/original"}]
			}`),
			changedField: "tagline", changedValue: `"Updated details"`,
			preservedField: "products", preservedValue: `[{"title":"Original product","uri":"https://example.com/original"}]`,
		},
		{
			name: "product-only",
			replacement: json.RawMessage(`{
				"businessTypes":["teacher"],"offerings":["classes"],
				"tagline":"Original details","hoursNote":"Weekdays",
				"location":{"country":"US","locality":"Portland"},
				"primaryAction":{"type":"shop","destination":"https://example.com/shop"},
				"products":[{"title":"Updated product","uri":"https://example.com/updated"}]
			}`),
			changedField: "products", changedValue: `[{"title":"Updated product","uri":"https://example.com/updated"}]`,
			preservedField: "tagline", preservedValue: `"Original details"`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			merged, err := MergeProfileReplacement(existing, test.replacement)
			if err != nil {
				t.Fatalf("MergeProfileReplacement: %v", err)
			}
			var got map[string]json.RawMessage
			if err := json.Unmarshal(merged, &got); err != nil {
				t.Fatalf("decode merged profile: %v", err)
			}
			var submitted map[string]json.RawMessage
			if err := json.Unmarshal(test.replacement, &submitted); err != nil {
				t.Fatalf("decode submitted replacement: %v", err)
			}
			for field, want := range submitted {
				if !bytes.Equal(got[field], want) {
					t.Errorf("submitted known field %s = %s, want %s", field, got[field], want)
				}
			}
			if !bytes.Equal(got[test.changedField], json.RawMessage(test.changedValue)) {
				t.Errorf("changed %s = %s", test.changedField, got[test.changedField])
			}
			if !bytes.Equal(got[test.preservedField], json.RawMessage(test.preservedValue)) {
				t.Errorf("preserved %s = %s", test.preservedField, got[test.preservedField])
			}
			if _, ok := got["serviceArea"]; ok {
				t.Error("omitted known serviceArea survived replacement")
			}
			if !bytes.Equal(got["com.example.extension"], json.RawMessage(extension)) {
				t.Fatalf("unknown extension = %s", got["com.example.extension"])
			}
		})
	}
}

func TestHydrateProfileOmitsOnlyUnsafeProductImages(t *testing.T) {
	const blobCID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	images := []any{
		map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": blobCID}, "mimeType": "image/png", "size": 12}, "alt": "Safe"},
		map[string]any{"image": map[string]any{"mimeType": "image/png", "size": 12}, "alt": "Missing blob identity"},
		map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": "not-a-cid"}, "mimeType": "image/png", "size": 12}},
		map[string]any{"unexpected": true},
		map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": blobCID}, "mimeType": "image/gif", "size": 12}},
		map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": blobCID}, "mimeType": "image/jpeg", "size": MaxImageBytes + 1}},
		map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": blobCID}, "mimeType": "image/webp", "size": 12}, "alt": strings.Repeat("a", 1001)},
		map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": blobCID}, "mimeType": "image/webp", "size": 12}, "aspectRatio": map[string]any{"width": 0, "height": 1}},
	}
	products := make([]map[string]any, len(images))
	for index, image := range images {
		products[index] = map[string]any{
			"title": "Product", "uri": "https://example.com/product", "image": image,
		}
	}
	raw, err := json.Marshal(map[string]any{"products": products})
	if err != nil {
		t.Fatalf("marshal profile: %v", err)
	}

	view, err := HydrateProfile(raw)
	if err != nil {
		t.Fatalf("HydrateProfile: %v", err)
	}
	if len(view.Products) != len(products) {
		t.Fatalf("products = %d, want %d", len(view.Products), len(products))
	}
	if view.Products[0].Image == nil {
		t.Fatal("safe product image was omitted")
	}
	if image := view.Products[0].Image; image.CID != syntax.CID(blobCID) || image.MIME != "image/png" || image.Size != 12 || image.Alt != "Safe" {
		t.Errorf("safe product image = %#v, want validated source metadata", image)
	}
	for index := 1; index < len(view.Products); index++ {
		if view.Products[index].Image != nil {
			t.Errorf("unsafe product image %d was exposed: %#v", index, view.Products[index].Image)
		}
	}
}

func TestHydrateProfileDistinguishesAbsentAndMalformedImageAlt(t *testing.T) {
	const blob = `{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/png","size":12}`
	tests := []struct {
		name      string
		alt       string
		wantImage bool
	}{
		{name: "absent", wantImage: true},
		{name: "null", alt: `,"alt":null`},
		{name: "non-string", alt: `,"alt":42`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := json.RawMessage(`{"products":[{"title":"Product","uri":"https://example.com/product","image":{"image":` + blob + test.alt + `}}]}`)
			view, err := HydrateProfile(raw)
			if err != nil {
				t.Fatalf("HydrateProfile: %v", err)
			}
			if len(view.Products) != 1 {
				t.Fatalf("products = %d, want 1", len(view.Products))
			}
			image := view.Products[0].Image
			if test.wantImage {
				if image == nil || image.Alt != "" {
					t.Fatalf("absent alt image = %#v, want valid empty alt", image)
				}
				return
			}
			if image != nil {
				t.Fatalf("malformed present alt image = %#v, want omitted", image)
			}
		})
	}
}
