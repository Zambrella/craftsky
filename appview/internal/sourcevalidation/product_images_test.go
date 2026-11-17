package sourcevalidation

import (
	"encoding/json"
	"social.craftsky/appview/internal/tap"
	"testing"
)

func TestProductPhotoArrayLexiconBound(t *testing.T) {
	for _, count := range []int{0, 4, 5} {
		images := make([]any, count)
		for i := range images {
			images[i] = map[string]any{"image": map[string]any{"$type": "blob", "ref": map[string]any{"$link": "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"}, "mimeType": "image/png", "size": 12}, "alt": "Wool"}
		}
		raw, _ := json.Marshal(map[string]any{"$type": "social.craftsky.business.profile", "products": []any{map[string]any{"title": "Wool", "uri": "https://shop.example/wool", "images": images}}})
		result := Validate(tap.Event{Collection: "social.craftsky.business.profile", Rkey: "self", Action: "create", Record: raw})
		if count <= 4 && (result.StructuralStatus != Valid || result.SemanticStatus != Valid) {
			t.Fatalf("count %d: %+v", count, result)
		}
		if count > 4 && result.StructuralStatus != Invalid {
			t.Fatalf("five photos accepted: %+v", result)
		}
	}
}
