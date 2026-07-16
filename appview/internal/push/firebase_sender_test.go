package push

import (
	"context"
	"firebase.google.com/go/v4/messaging"
	"social.craftsky/appview/internal/notifications"
	"testing"
	"time"
)

type captureFirebaseClient struct{ message *messaging.Message }

func (c *captureFirebaseClient) Send(_ context.Context, m *messaging.Message) (string, error) {
	c.message = m
	return "provider-id", nil
}
func TestFirebaseSenderBuildsCombinedMessageWithBoundedTTL(t *testing.T) {
	client := &captureFirebaseClient{}
	sender := &FirebaseSender{client: client, now: func() time.Time { return time.Unix(1000, 0) }}
	result, err := sender.Send(context.Background(), SendRequest{Token: "token", NotificationID: "notification", Category: notifications.Like, AccountSubscriptionID: "routing", ActorDisplayName: "Alice", TTL: time.Hour})
	if err != nil || result.Class != ResultSuccess {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	m := client.message
	if m.Token != "token" || m.Android == nil || m.Android.TTL == nil || *m.Android.TTL != time.Hour || m.APNS.Headers["apns-expiration"] != "4600" || m.Data["notificationId"] != "notification" {
		t.Fatalf("message=%+v", m)
	}
	if m.APNS.Payload == nil || m.APNS.Payload.Aps == nil || m.APNS.Payload.Aps.Sound != "default" {
		t.Fatalf("apns payload=%+v, want default sound", m.APNS.Payload)
	}
}
