package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"firebase.google.com/go/v4/errorutils"
	"firebase.google.com/go/v4/messaging"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/notifications"
)

type captureFirebaseClient struct {
	messages []*messaging.Message
	err      error
}

type firebaseFailureTransport struct {
	status     int
	retryAfter string
}

type firebaseTestTransport func(*http.Request) (*http.Response, error)

func (transport firebaseTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func (transport firebaseFailureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: transport.status,
		Header:     http.Header{"Retry-After": {transport.retryAfter}},
		Body:       io.NopCloser(strings.NewReader("{}")),
		Request:    request,
	}, nil
}

func firebaseRetryTestRequest() SendRequest {
	return SendRequest{
		Platform: "android", Token: "test-token", Category: notifications.Like,
		AccountSubscriptionID: "00000000-0000-4000-8000-000000000001",
		RoutingFacts:          RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000002"},
		Semantics:             DeliveryUniqueEvent, TTL: time.Hour,
	}
}

func TestFirebaseSenderPreservesProviderRetryDelay(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name      string
		transport firebaseFailureTransport
		wantClass ResultClass
		wantDelay time.Duration
	}{
		{"missing quota header", firebaseFailureTransport{status: 429}, ResultRetryable, time.Minute},
		{"quota seconds", firebaseFailureTransport{429, "135"}, ResultRetryable, 135 * time.Second},
		{"gateway HTTP date", firebaseFailureTransport{502, now.Add(30 * time.Second).Format(http.TimeFormat)}, ResultRetryable, 30 * time.Second},
		{"plain internal error", firebaseFailureTransport{status: 500}, ResultRetryable, 10 * time.Second},
		{"configuration failure", firebaseFailureTransport{403, "120"}, ResultPermanentFailure, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sender, err := newFirebaseSender(t.Context(), "test-project", &http.Client{Transport: tt.transport})
			if err != nil {
				t.Fatal(err)
			}
			sender.now = func() time.Time { return now }
			result, err := sender.Send(t.Context(), firebaseRetryTestRequest())
			if err == nil || result.Class != tt.wantClass || result.RetryAfter != tt.wantDelay {
				t.Fatalf("result=%+v err=%v", result, err)
			}
		})
	}
}

func TestFirebaseSenderRetains503RetryDelayWhenSDKWaitIsInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	calls := 0
	transport := firebaseTestTransport(func(request *http.Request) (*http.Response, error) {
		calls++
		// Model cancellation once the transport has received the response.
		// The SDK exits its internal retry wait with just the context cause.
		cancel()
		return (firebaseFailureTransport{503, "60"}).RoundTrip(request)
	})
	sender, err := newFirebaseSender(ctx, "test-project", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	sender.now = func() time.Time { return time.Unix(1000, 0) }
	result, err := sender.Send(ctx, firebaseRetryTestRequest())
	if !errors.Is(err, context.Canceled) || result.Class != ResultRetryable || result.RetryAfter != time.Minute || calls != 1 {
		t.Fatalf("result=%+v err=%v calls=%d", result, err, calls)
	}
}

func TestFirebaseSenderRetriesExhaustedTransportErrors(t *testing.T) {
	t.Parallel()
	transport := firebaseTestTransport(func(*http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})
	sender, err := newFirebaseSender(t.Context(), "test-project", &http.Client{Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	result, err := sender.Send(t.Context(), firebaseRetryTestRequest())
	if !errorutils.IsUnknown(err) || errorutils.HTTPResponse(err) != nil || result.Class != ResultRetryable || result.RetryAfter != 10*time.Second {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func (client *captureFirebaseClient) Send(
	_ context.Context,
	message *messaging.Message,
) (string, error) {
	client.messages = append(client.messages, message)
	return "provider-id", client.err
}

func TestFirebaseSenderIncludesThreadRootForNativeMentionsAndQuotes(t *testing.T) {
	const source = "at://did:plc:source/social.craftsky.feed.post/reply"
	const root = "at://did:plc:author/social.craftsky.feed.post/root"
	for _, category := range []notifications.Category{notifications.Mention, notifications.Quote} {
		for _, platform := range []string{"android", "ios", "web"} {
			t.Run(string(category)+"/"+platform, func(t *testing.T) {
				client := &captureFirebaseClient{}
				sender := FirebaseSender{client: client, now: time.Now}
				result, err := sender.Send(t.Context(), SendRequest{
					Platform:              platform,
					Token:                 "test-token",
					Category:              category,
					AccountSubscriptionID: "00000000-0000-4000-8000-000000000001",
					RoutingFacts: RoutingFacts{
						NotificationID: "00000000-0000-4000-8000-000000000002",
						SourceURI:      syntax.ATURI(source),
						RootURI:        syntax.ATURI(root),
					},
					Semantics: DeliveryUniqueEvent,
					TTL:       time.Hour,
				})
				if err != nil || result.Class != ResultSuccess || len(client.messages) != 1 {
					t.Fatalf("result=%+v err=%v messages=%d", result, err, len(client.messages))
				}
				data := client.messages[0].Data
				if data["sourceUri"] != source {
					t.Fatalf("sourceUri=%q, want %q", data["sourceUri"], source)
				}
				if platform == "web" {
					if _, exists := data["rootUri"]; exists {
						t.Fatal("web payload includes a native thread root")
					}
				} else if data["rootUri"] != root {
					t.Fatalf("rootUri=%q, want %q", data["rootUri"], root)
				}
			})
		}
	}
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
			name:     "Android uses standard notification plus routing data",
			platform: "android",
			assert: func(t *testing.T, message *messaging.Message) {
				t.Helper()
				if message.Notification == nil || message.Notification.Title != "Alice" ||
					message.Notification.Body != "liked your post" {
					t.Fatalf("Android notification = %+v", message.Notification)
				}
				if message.Android == nil || message.Android.Priority != "" || message.Android.TTL == nil ||
					*message.Android.TTL != time.Hour {
					t.Fatalf("Android config = %+v", message.Android)
				}
				if message.Android.CollapseKey != "" {
					t.Fatalf("Android unique event collapse key = %q", message.Android.CollapseKey)
				}
				if message.APNS != nil {
					t.Fatalf("Android message unexpectedly contains APNs config: %+v", message.APNS)
				}
				if _, ok := message.Data["displayTitle"]; ok {
					t.Fatalf("Android data duplicates display title: %#v", message.Data)
				}
				if _, ok := message.Data["displayBody"]; ok {
					t.Fatalf("Android data duplicates display body: %#v", message.Data)
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

func TestFirebaseSenderBuildsActorlessModerationMessage(t *testing.T) {
	client := &captureFirebaseClient{}
	sender := &FirebaseSender{client: client, now: time.Now}
	_, err := sender.Send(context.Background(), SendRequest{
		Token:                 "provider-token",
		Category:              notifications.Moderation,
		AccountSubscriptionID: "opaque-account-binding",
		RoutingFacts: RoutingFacts{
			NotificationID: "00000000-0000-4000-8000-000000000001",
			CaseReference:  "MOD-550e8400-e29b-41d4-a716-446655440000",
		},
		Platform:  "ios",
		Semantics: DeliveryUniqueEvent,
		TTL:       time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.messages) != 1 {
		t.Fatalf("provider messages=%d, want 1", len(client.messages))
	}
	message := client.messages[0]
	if message.Notification == nil || message.Notification.Title != "CraftSky" {
		t.Fatalf("moderation notification=%+v", message.Notification)
	}
	for key, want := range map[string]string{
		"type":                  "moderation",
		"accountSubscriptionId": "opaque-account-binding",
		"notificationId":        "00000000-0000-4000-8000-000000000001",
		"caseReference":         "MOD-550e8400-e29b-41d4-a716-446655440000",
	} {
		if message.Data[key] != want {
			t.Errorf("data[%q]=%q, want %q", key, message.Data[key], want)
		}
	}
	for _, forbidden := range []string{"actorDid", "recipientDid", "eventId", "decisionId"} {
		if _, exists := message.Data[forbidden]; exists {
			t.Errorf("actorless moderation data contains %q: %+v", forbidden, message.Data)
		}
	}
}

func TestFirebaseSenderUsesStandardNotificationsWithoutPerDeliveryCollapseMetadata(t *testing.T) {
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
		if message.Notification == nil || message.Android == nil ||
			message.Android.CollapseKey != "" {
			t.Fatalf("Android notification contract = %+v", message)
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

func TestFirebaseSenderIncludesExcerptOnlyInVisibleNotification(t *testing.T) {
	client := &captureFirebaseClient{}
	sender := &FirebaseSender{client: client, now: time.Now}
	_, err := sender.Send(context.Background(), SendRequest{
		Token: "token", Category: notifications.Reply, AccountSubscriptionID: "routing",
		RoutingFacts: RoutingFacts{NotificationID: "00000000-0000-4000-8000-000000000001"},
		Preview:      &notifications.Preview{Text: "Lovely stitches", ProjectTitle: "Winter scarf"},
		Platform:     "ios", Semantics: DeliveryUniqueEvent, TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	message := client.messages[0]
	if message.Notification.Body != "commented on your post: Winter scarf: Lovely stitches" {
		t.Fatalf("notification=%+v", message.Notification)
	}
	for _, value := range message.Data {
		if strings.Contains(value, "Lovely stitches") || strings.Contains(value, "Winter scarf") {
			t.Fatal("preview entered routing data")
		}
	}
}
