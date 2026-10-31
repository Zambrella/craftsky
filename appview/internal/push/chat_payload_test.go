package push

import (
	"context"
	"encoding/json"
	"errors"
	"social.craftsky/appview/internal/notifications"
	"strings"
	"testing"
	"time"
)

// UT-007 / FR-013, RULE-003: the real provider serializer emits sender
// copy and a closed opaque routing contract, with no social/private targets.
func TestFirebaseChatPayloadHasOnlyOpaqueRoutingAndSenderCopy(t *testing.T) {
	const reference = "00000000-0000-4000-8000-000000000123"
	const subscription = "00000000-0000-4000-8000-000000000456"
	for _, platform := range []string{"ios", "android"} {
		client := &captureFirebaseClient{}
		sender := &FirebaseSender{client: client, now: time.Now}
		request := SendRequest{Token: "provider-token", AccountSubscriptionID: subscription, Chat: &ChatRouting{OpenReference: reference}, ActorDisplayName: "Alex", Platform: platform, Semantics: DeliveryUniqueEvent, TTL: time.Hour}
		if _, err := sender.Send(t.Context(), request); err != nil {
			t.Fatal(err)
		}
		message := client.messages[0]
		if message.Notification.Title != "Alex" || message.Notification.Body != "sent you a message" {
			t.Fatalf("visible copy=%+v", message.Notification)
		}
		expected := map[string]string{"payloadVersion": "1", "type": "chat", "accountSubscriptionId": subscription, "openReference": reference}
		if len(message.Data) != len(expected) {
			t.Fatal("provider data contains extra routing fields")
		}
		for key, value := range expected {
			if message.Data[key] != value {
				t.Fatal("wrong opaque routing contract")
			}
		}
		serialized, _ := json.Marshal(message)
		for _, private := range []string{"did:plc:", "convoId", "messageId", "cursor", "notificationId", "private-message-canary"} {
			if strings.Contains(string(serialized), private) {
				t.Fatal("private routing in actual provider message")
			}
		}
		request.Category = notifications.Reply
		if _, err := sender.Send(context.Background(), request); !errors.Is(err, ErrPushPayloadInvalid) {
			t.Fatal("chat accepted social category controls")
		}
		request.Category = ""
		request.RoutingFacts.ActorDID = "did:plc:private-routing-canary"
		if _, err := sender.Send(context.Background(), request); !errors.Is(err, ErrPushPayloadInvalid) {
			t.Fatal("chat accepted raw participant routing")
		}
		request.RoutingFacts = RoutingFacts{}
		request.ActorDisplayName = ""
		if _, err := sender.Send(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		if client.messages[1].Notification.Title != "Someone" {
			t.Fatal("missing sender did not use generic copy")
		}
	}
}
