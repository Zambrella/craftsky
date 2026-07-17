package push

import (
	"context"
	"reflect"
	"testing"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/notifications"
)

type captureFirebaseClient struct{ message *messaging.Message }

func (c *captureFirebaseClient) Send(_ context.Context, m *messaging.Message) (string, error) {
	c.message = m
	return "provider-id", nil
}
func TestFirebaseSenderBuildsCombinedMessageWithBoundedTTL(t *testing.T) {
	if _, ok := reflect.TypeFor[SendRequest]().FieldByName("NotificationID"); ok {
		t.Fatal("provider SendRequest still carries notification ID")
	}
	client := &captureFirebaseClient{}
	sender := &FirebaseSender{client: client, now: func() time.Time { return time.Unix(1000, 0) }}
	const subjectURI = "at://did:plc:subject/social.craftsky.feed.post/subject"
	const rootURI = "at://did:plc:root/social.craftsky.feed.post/root"
	result, err := sender.Send(context.Background(), SendRequest{
		Token:                 "token",
		Category:              notifications.Like,
		AccountSubscriptionID: "routing",
		RoutingFacts: RoutingFacts{
			SubjectURI: syntax.ATURI(subjectURI),
			RootURI:    syntax.ATURI(rootURI),
		},
		ActorDisplayName: "Alice",
		TTL:              time.Hour,
	})
	if err != nil || result.Class != ResultSuccess {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	m := client.message
	if m.Token != "token" || m.Android == nil || m.Android.TTL == nil || *m.Android.TTL != time.Hour || m.APNS.Headers["apns-expiration"] != "4600" {
		t.Fatalf("message=%+v", m)
	}
	wantData := map[string]string{
		"payloadVersion":        "1",
		"type":                  "like",
		"accountSubscriptionId": "routing",
		"subjectUri":            subjectURI,
		"rootUri":               rootURI,
	}
	if !reflect.DeepEqual(m.Data, wantData) {
		t.Fatalf("data=%#v, want %#v", m.Data, wantData)
	}
	if m.Notification == nil || m.Notification.Title != "Alice" || m.Notification.Body != "liked your post" {
		t.Fatalf("notification=%+v", m.Notification)
	}
	if m.APNS.Payload == nil || m.APNS.Payload.Aps == nil || m.APNS.Payload.Aps.Sound != "default" {
		t.Fatalf("apns payload=%+v, want default sound", m.APNS.Payload)
	}
}
