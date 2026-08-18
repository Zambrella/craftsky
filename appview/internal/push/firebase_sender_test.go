package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/notifications"
)

type captureFirebaseClient struct {
	messages []*messaging.Message
	err      error
}

func (client *captureFirebaseClient) Send(
	_ context.Context,
	message *messaging.Message,
) (string, error) {
	client.messages = append(client.messages, message)
	return "provider-id", client.err
}

func TestFirebaseSenderBuildsPlatformSpecificUniqueEventMessages(t *testing.T) {
	if _, ok := reflect.TypeFor[SendRequest]().FieldByName("NotificationID"); ok {
		t.Fatal("provider SendRequest still carries a second notification ID")
	}
	const (
		notificationID = "00000000-0000-4000-8000-000000000654"
		subjectURI     = "at://did:plc:subject/social.craftsky.feed.post/subject"
		rootURI        = "at://did:plc:root/social.craftsky.feed.post/root"
	)

	tests := []struct {
		name     string
		platform string
		assert   func(*testing.T, *messaging.Message)
	}{
		{
			name:     "Android is data only and non-collapsing",
			platform: "android",
			assert: func(t *testing.T, message *messaging.Message) {
				t.Helper()
				if message.Notification != nil {
					t.Fatalf("Android unique event has notification payload: %+v", message.Notification)
				}
				if message.Android == nil || message.Android.TTL == nil ||
					*message.Android.TTL != time.Hour {
					t.Fatalf("Android config = %+v", message.Android)
				}
				if message.Android.CollapseKey != "" {
					t.Fatalf("Android unique event collapse key = %q", message.Android.CollapseKey)
				}
				if message.APNS != nil {
					t.Fatalf("Android message unexpectedly contains APNs config: %+v", message.APNS)
				}
				if message.Data["displayTitle"] != "Alice" ||
					message.Data["displayBody"] != "liked your post" {
					t.Fatalf("Android display data = %#v", message.Data)
				}
			},
		},
		{
			name:     "iOS alert omits collapse ID",
			platform: "ios",
			assert: func(t *testing.T, message *messaging.Message) {
				t.Helper()
				if message.Notification == nil || message.Notification.Title != "Alice" ||
					message.Notification.Body != "liked your post" {
					t.Fatalf("iOS notification = %+v", message.Notification)
				}
				if message.Android != nil {
					t.Fatalf("iOS message unexpectedly contains Android config: %+v", message.Android)
				}
				if message.APNS == nil || message.APNS.Headers["apns-expiration"] != "4600" {
					t.Fatalf("APNs config = %+v", message.APNS)
				}
				if collapseID := message.APNS.Headers["apns-collapse-id"]; collapseID != "" {
					t.Fatalf("APNs unique event collapse ID = %q", collapseID)
				}
				if message.APNS.Payload == nil || message.APNS.Payload.Aps == nil ||
					message.APNS.Payload.Aps.Sound != "default" {
					t.Fatalf("APNs payload = %+v", message.APNS.Payload)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &captureFirebaseClient{}
			sender := &FirebaseSender{
				client: client,
				now:    func() time.Time { return time.Unix(1000, 0) },
			}
			result, err := sender.Send(context.Background(), SendRequest{
				Token:                 "token",
				Category:              notifications.Like,
				AccountSubscriptionID: "routing",
				RoutingFacts: RoutingFacts{
					SubjectURI:     syntax.ATURI(subjectURI),
					RootURI:        syntax.ATURI(rootURI),
					NotificationID: notificationID,
				},
				ActorDisplayName: "Alice",
				Platform:         test.platform,
				Semantics:        DeliveryUniqueEvent,
				TTL:              time.Hour,
			})
			if err != nil || result.Class != ResultSuccess {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if len(client.messages) != 1 {
				t.Fatalf("provider calls = %d, want 1", len(client.messages))
			}
			message := client.messages[0]
			var target struct {
				Token string `json:"token"`
			}
			encoded, err := json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &target); err != nil {
				t.Fatal(err)
			}
			if target.Token != "token" {
				t.Fatalf("message token = %q", target.Token)
			}
			for key, want := range map[string]string{
				"payloadVersion":        "1",
				"type":                  "like",
				"accountSubscriptionId": "routing",
				"notificationId":        notificationID,
				"subjectUri":            subjectURI,
				"rootUri":               rootURI,
			} {
				if got := message.Data[key]; got != want {
					t.Errorf("data[%q] = %q, want %q", key, got, want)
				}
			}
			test.assert(t, message)
		})
	}
}

func TestFirebaseSenderNeverUsesPerDeliveryCollapseMetadata(t *testing.T) {
	client := &captureFirebaseClient{}
	sender := &FirebaseSender{client: client, now: time.Now}
	for index := 1; index <= 5; index++ {
		notificationID := fmt.Sprintf(
			"00000000-0000-4000-8000-%012d",
			index,
		)
		_, err := sender.Send(context.Background(), SendRequest{
			Token:                 "one-registration-token",
			Category:              notifications.Like,
			AccountSubscriptionID: "routing",
			RoutingFacts: RoutingFacts{
				NotificationID: notificationID,
			},
			Platform:  "android",
			Semantics: DeliveryUniqueEvent,
			TTL:       time.Hour,
		})
		if err != nil {
			t.Fatalf("send distinct event %d: %v", index, err)
		}
	}
	if len(client.messages) != 5 {
		t.Fatalf("provider messages = %d, want 5", len(client.messages))
	}
	seen := map[string]bool{}
	for _, message := range client.messages {
		if message.Notification != nil || message.Android == nil ||
			message.Android.CollapseKey != "" {
			t.Fatalf("unique Android event used collapsing payload: %+v", message)
		}
		id := message.Data["notificationId"]
		if id == "" || seen[id] {
			t.Fatalf("notification ID missing or duplicated: %q", id)
		}
		seen[id] = true
	}
}

func TestFirebaseSenderRejectsUnknownPlatformOrSemanticsBeforeProvider(t *testing.T) {
	tests := []SendRequest{
		{
			Token: "token", Category: notifications.Category("instagramMatch"),
			Platform: "android", Semantics: DeliveryUniqueEvent,
			RoutingFacts: RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000001"},
			TTL:          time.Minute,
		},
		{
			Token: "token", Platform: "web", Semantics: DeliveryUniqueEvent,
			RoutingFacts: RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000001"},
			TTL:          time.Minute,
		},
		{
			Token: "token", Platform: "android", Semantics: DeliverySemantics("future"),
			RoutingFacts: RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000001"},
			TTL:          time.Minute,
		},
	}
	for _, request := range tests {
		client := &captureFirebaseClient{}
		sender := &FirebaseSender{client: client, now: time.Now}
		result, err := sender.Send(context.Background(), request)
		if !errors.Is(err, ErrPushPayloadInvalid) || result.Class != ResultPermanentFailure {
			t.Fatalf("result=%+v err=%v, want permanent invalid payload", result, err)
		}
		if len(client.messages) != 0 {
			t.Fatalf("provider called for invalid request: %+v", request)
		}
	}
}
