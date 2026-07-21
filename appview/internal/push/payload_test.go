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
		RootURI:    syntax.ATURI("at://did:plc:root/social.craftsky.feed.post/root"),
		SourceURI:  syntax.ATURI("at://did:plc:source/social.craftsky.feed.post/source"),
	}
	tests := map[notifications.Category]map[string]string{
		notifications.Follow:         {"actorDid": facts.ActorDID.String()},
		notifications.Like:           {"subjectUri": facts.SubjectURI.String(), "rootUri": facts.RootURI.String()},
		notifications.Repost:         {"subjectUri": facts.SubjectURI.String(), "rootUri": facts.RootURI.String()},
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

func TestBuildPayloadUT017UsesConversationRoleInVisibleCopy(t *testing.T) {
	tests := []struct {
		name     string
		category notifications.Category
		role     ContentRole
		want     string
	}{
		{"like post", notifications.Like, ContentRolePost, "liked your post"},
		{"like comment", notifications.Like, ContentRoleComment, "liked your comment"},
		{"like reply", notifications.Like, ContentRoleReply, "liked your reply"},
		{"repost post", notifications.Repost, ContentRolePost, "reposted your post"},
		{"repost comment", notifications.Repost, ContentRoleComment, "reposted your comment"},
		{"repost reply", notifications.Repost, ContentRoleReply, "reposted your reply"},
		{"comment on post", notifications.Reply, ContentRolePost, "commented on your post"},
		{"reply to comment", notifications.Reply, ContentRoleComment, "replied to your comment"},
		{"reply to reply", notifications.Reply, ContentRoleReply, "replied to your reply"},
		{"quote post", notifications.Quote, ContentRolePost, "quoted your post"},
		{"quote comment", notifications.Quote, ContentRoleComment, "quoted your comment"},
		{"quote reply", notifications.Quote, ContentRoleReply, "quoted your reply"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseline := BuildPayload(test.category, "routing-id", "Alice", RoutingFacts{})
			payload := BuildPayload(test.category, "routing-id", "Alice", RoutingFacts{TargetRole: test.role})
			if payload.Body != test.want {
				t.Fatalf("body = %q, want %q", payload.Body, test.want)
			}
			if !reflect.DeepEqual(payload.Data, baseline.Data) {
				t.Fatalf("role changed provider data: got %#v, want %#v", payload.Data, baseline.Data)
			}
		})
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

func TestBuildInstagramMatchPayloadIsActorlessAndBounded(t *testing.T) {
	const privateCanary = "private-instagram-identity-canary"
	payload := BuildPayload(
		notifications.InstagramMatch,
		"opaque-account-subscription",
		privateCanary,
		RoutingFacts{
			ActorDID:          syntax.DID("did:plc:" + privateCanary),
			SourceURI:         syntax.ATURI("at://" + privateCanary + "/source"),
			SubjectURI:        syntax.ATURI("at://" + privateCanary + "/subject"),
			RootURI:           syntax.ATURI("at://" + privateCanary + "/root"),
			NotificationID:    "00000000-0000-0000-0000-000000000654",
			SystemCount:       99,
			SystemCountCapped: true,
			SystemDestination: "instagramMigration",
		},
	)

	want := map[string]string{
		"payloadVersion":        "1",
		"type":                  "instagramMatch",
		"accountSubscriptionId": "opaque-account-subscription",
		"notificationId":        "00000000-0000-0000-0000-000000000654",
		"count":                 "99",
		"countCapped":           "true",
		"destination":           "instagramMigration",
	}
	if !reflect.DeepEqual(payload.Data, want) {
		t.Fatalf("data=%#v, want %#v", payload.Data, want)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), privateCanary) || strings.Contains(string(encoded), "did:plc:") || strings.Contains(string(encoded), "at://") {
		t.Fatalf("actorless payload leaked social/private facts: %s", encoded)
	}
	if payload.Title != "CraftSky" || payload.Body == "" {
		t.Fatalf("visible copy=%q / %q", payload.Title, payload.Body)
	}
}
