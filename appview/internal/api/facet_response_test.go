package api_test

import (
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestFacetMentionSuggestionJSONOmitsUnknownOptionalFields(t *testing.T) {
	t.Parallel()
	item := api.BuildFacetMentionSuggestion(api.MentionSuggestionRow{
		DID:               "did:plc:alice",
		Handle:            "alice.craftsky.social",
		IsCraftskyProfile: true,
		ViewerIsFollowing: false,
	})

	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	for key, want := range map[string]any{
		"did":               "did:plc:alice",
		"handle":            "alice.craftsky.social",
		"isCraftskyProfile": true,
		"viewerIsFollowing": false,
	} {
		if got := body[key]; got != want {
			t.Fatalf("%s = %v, want %v; raw=%s", key, got, want, raw)
		}
	}
	if _, ok := body["displayName"]; ok {
		t.Fatalf("displayName present in %s, want omitted", raw)
	}
	if _, ok := body["avatar"]; ok {
		t.Fatalf("avatar present in %s, want omitted", raw)
	}
}
