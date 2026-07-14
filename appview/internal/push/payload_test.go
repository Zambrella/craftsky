package push

import (
	"encoding/json"
	"strings"
	"testing"

	"social.craftsky/appview/internal/notifications"
)

func TestBuildPayloadContainsOnlyMinimalRoutingAndGenericCopy(t *testing.T) {
	for _, category := range notifications.Categories() {
		payload := BuildPayload("notification-id", category, "routing-id", "Alice")
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		for _, required := range []string{"Alice", "notification-id", string(category), "routing-id"} {
			if !strings.Contains(text, required) {
				t.Errorf("%s payload missing %q: %s", category, required, text)
			}
		}
		for _, forbidden := range []string{"did:plc:", "at://", "handle", "imageUrl", "postText", "token"} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s payload leaked %q: %s", category, forbidden, text)
			}
		}
	}
	unnamed := BuildPayload("id", notifications.Like, "route", "")
	if unnamed.Title != "Someone" {
		t.Fatalf("unnamed title=%q", unnamed.Title)
	}
}
