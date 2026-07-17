package push

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/notifications"
)

func TestBuildPayloadUT011ExactNotificationFactMatrix(t *testing.T) {
	facts := RoutingFacts{
		ActorDID:   syntax.DID("did:plc:actor"),
		SubjectURI: syntax.ATURI("at://did:plc:subject/social.craftsky.feed.post/subject"),
		SourceURI:  syntax.ATURI("at://did:plc:source/social.craftsky.feed.post/source"),
	}
	tests := map[notifications.Category]map[string]string{
		notifications.Follow:         {"actorDid": facts.ActorDID.String()},
		notifications.Like:           {"subjectUri": facts.SubjectURI.String()},
		notifications.Repost:         {"subjectUri": facts.SubjectURI.String()},
		notifications.Mention:        {"sourceUri": facts.SourceURI.String()},
		notifications.Quote:          {"sourceUri": facts.SourceURI.String()},
		notifications.Reply:          {"subjectUri": facts.SubjectURI.String(), "sourceUri": facts.SourceURI.String()},
		notifications.EverythingElse: {},
	}

	for category, categoryFacts := range tests {
		t.Run(string(category), func(t *testing.T) {
			want := map[string]string{
				"payloadVersion":        "1",
				"type":                  string(category),
				"accountSubscriptionId": "routing-id",
			}
			for key, value := range categoryFacts {
				want[key] = value
			}

			payload := BuildPayload(category, "routing-id", "Alice", facts)
			if !reflect.DeepEqual(payload.Data, want) {
				t.Fatalf("data = %#v, want %#v", payload.Data, want)
			}
			if payload.Title != "Alice" || payload.Body == "" {
				t.Fatalf("visible copy = %q / %q", payload.Title, payload.Body)
			}
			for _, forbidden := range []string{"notificationId", "routeKind", "route", "path", "handle", "postText", "imageUrl", "token"} {
				if _, ok := payload.Data[forbidden]; ok {
					t.Fatalf("data contains forbidden key %q", forbidden)
				}
			}
		})
	}

	unnamed := BuildPayload(notifications.Like, "routing-id", "", facts)
	if unnamed.Title != "Someone" {
		t.Fatalf("unnamed title = %q", unnamed.Title)
	}
}

func TestBuildPayloadUT012BoundsLargestReplyData(t *testing.T) {
	const maxFactBytes = 1024
	uriPrefix := "at://did:plc:"
	uriSuffix := "/social.craftsky.feed.post/post"
	maxURI := uriPrefix + strings.Repeat("a", maxFactBytes-len(uriPrefix)-len(uriSuffix)) + uriSuffix
	payload := BuildPayload(
		notifications.Reply,
		strings.Repeat("r", 128),
		"Alice",
		RoutingFacts{SubjectURI: syntax.ATURI(maxURI), SourceURI: syntax.ATURI(maxURI)},
	)

	for _, key := range []string{"subjectUri", "sourceUri"} {
		if got := len(payload.Data[key]); got != maxFactBytes {
			t.Fatalf("%s bytes = %d, want %d", key, got, maxFactBytes)
		}
	}
	raw, err := json.Marshal(payload.Data)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) >= 4096 {
		t.Fatalf("reply data bytes = %d, want below 4096", len(raw))
	}

	overBound := BuildPayload(
		notifications.Like,
		"routing-id",
		"Alice",
		RoutingFacts{SubjectURI: syntax.ATURI(maxURI + "x")},
	)
	if _, ok := overBound.Data["subjectUri"]; ok {
		t.Fatal("over-bound subjectUri entered provider data")
	}
}
