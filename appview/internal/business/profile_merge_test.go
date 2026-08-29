package business

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
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
	if len(view.Products[0].Image) == 0 {
		t.Fatal("safe product image was omitted")
	}
	for index := 1; index < len(view.Products); index++ {
		if len(view.Products[index].Image) != 0 {
			t.Errorf("unsafe product image %d was exposed: %s", index, view.Products[index].Image)
		}
	}
}
