package push

import (
	"context"
	"encoding/json"
	"social.craftsky/appview/internal/notifications"
	"strings"
	"testing"
	"time"
)

func TestFirebaseWebPushHasBoundedTTLPrivateAccountRoutingAndNoCollapse(t *testing.T) {
	const subscription = "00000000-0000-4000-8000-000000000001"
	for _, chat := range []bool{false, true} {
		t.Run(map[bool]string{false: "activity", true: "chat"}[chat], func(t *testing.T) {
			client := &captureFirebaseClient{}
			sender := &FirebaseSender{client: client, now: time.Now}
			request := SendRequest{Platform: "web", Token: "test-token", AccountSubscriptionID: subscription, Category: notifications.Like, RoutingFacts: RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000002"}, Semantics: DeliveryUniqueEvent, TTL: 90 * time.Second}
			if chat {
				request.Category = ""
				request.RoutingFacts = RoutingFacts{}
				request.Chat = &ChatRouting{OpenReference: "00000000-0000-4000-8000-000000000003", AcceptedConversation: true}
			}
			result, err := sender.Send(context.Background(), request)
			if err != nil || result.Class != ResultSuccess {
				t.Fatalf("result=%v err=%v", result, err)
			}
			message := client.messages[0]
			if message.Notification != nil || message.Android != nil || message.APNS != nil || message.Webpush == nil || message.Webpush.Headers["TTL"] != "90" {
				t.Fatalf("invalid web platform configuration")
			}
			wantBody := "liked your post"
			if chat {
				wantBody = "sent you a message"
			}
			if message.Data["webTitle"] != "Someone" || message.Data["webBody"] != wantBody {
				t.Fatalf("incorrect service worker display copy: %+v", message.Data)
			}
			if message.Webpush.FCMOptions != nil || message.Data["accountSubscriptionId"] != subscription {
				t.Fatalf("invalid service worker routing data")
			}
			if chat && message.Data["openReference"] != request.Chat.OpenReference {
				t.Fatalf("chat routing reference missing")
			}
			encoded, err := json.Marshal(message)
			if err != nil {
				t.Fatal(err)
			}
			for _, forbidden := range []string{"collapse_key", "apns-collapse-id", "Topic", "recipientDid", "messageText"} {
				if strings.Contains(string(encoded), forbidden) {
					t.Fatalf("unexpected web delivery field %s", forbidden)
				}
			}
		})
	}
}
func TestFirebaseWebPushRejectsUnboundAccountRouting(t *testing.T) {
	sender := &FirebaseSender{client: &captureFirebaseClient{}, now: time.Now}
	_, err := sender.buildMessage(SendRequest{Platform: "web", Token: "token", AccountSubscriptionID: "not-a-routing-id", Category: notifications.Like, RoutingFacts: RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000002"}, Semantics: DeliveryUniqueEvent, TTL: time.Minute})
	if err != ErrPushPayloadInvalid {
		t.Fatalf("error=%v", err)
	}
}
