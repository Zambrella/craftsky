package push

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/errorutils"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"

	"social.craftsky/appview/internal/notifications"
)

type firebaseClient interface {
	Send(context.Context, *messaging.Message) (string, error)
}
type FirebaseSender struct {
	client firebaseClient
	now    func() time.Time
}

func NewFirebaseSender(ctx context.Context, projectID string) (*FirebaseSender, error) {
	httpClient, _, err := transport.NewHTTPClient(ctx, option.WithScopes("https://www.googleapis.com/auth/firebase.messaging"))
	if err != nil {
		return nil, err
	}
	return newFirebaseSender(ctx, projectID, httpClient)
}

func newFirebaseSender(ctx context.Context, projectID string, httpClient *http.Client) (*FirebaseSender, error) {
	clientWithRetryHint := *httpClient
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	clientWithRetryHint.Transport = firebaseRetryTransport{base}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, option.WithHTTPClient(&clientWithRetryHint))
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	return &FirebaseSender{client: client, now: time.Now}, nil
}

type firebaseRetryHintKey struct{}

type firebaseRetryHint struct {
	now       func() time.Time
	notBefore time.Time
}

type firebaseRetryTransport struct{ http.RoundTripper }

func (transport firebaseRetryTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := transport.RoundTripper.RoundTrip(request)
	// When the send context ends during its internal 503 backoff, the SDK
	// returns only the context error. Keep the parsed retry time for this send.
	if err != nil || response == nil || response.StatusCode != http.StatusServiceUnavailable {
		return response, err
	}
	hint, ok := request.Context().Value(firebaseRetryHintKey{}).(*firebaseRetryHint)
	if !ok {
		return response, nil
	}
	now := hint.now()
	delay := firebaseRetryAfter(response.Header.Get("Retry-After"), now)
	if retryAt := now.Add(delay); delay > 0 && retryAt.After(hint.notBefore) {
		hint.notBefore = retryAt
	}
	return response, nil
}

func (s *FirebaseSender) Send(ctx context.Context, request SendRequest) (ProviderResult, error) {
	message, err := s.buildMessage(request)
	if err != nil {
		return ProviderResult{Class: ResultPermanentFailure}, err
	}
	hint := firebaseRetryHint{now: s.now}
	_, err = s.client.Send(context.WithValue(ctx, firebaseRetryHintKey{}, &hint), message)
	if err == nil {
		return ProviderResult{Class: ResultSuccess}, nil
	}
	switch {
	case messaging.IsUnregistered(err):
		return ProviderResult{Class: ResultInvalidToken}, err
	default:
		now := s.now()
		result := classifyFirebaseFailure(ctx, err, now)
		if result.Class == ResultRetryable {
			result.RetryAfter = max(result.RetryAfter, hint.notBefore.Sub(now))
		}
		return result, err
	}
}

func classifyFirebaseFailure(ctx context.Context, err error, now time.Time) ProviderResult {
	response := errorutils.HTTPResponse(err)
	retryable := ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) || errorutils.IsDeadlineExceeded(err) ||
		errorutils.IsUnavailable(err) || messaging.IsUnavailable(err) || messaging.IsInternal(err)
	quota := messaging.IsQuotaExceeded(err)
	if response != nil {
		retryable = retryable || response.StatusCode >= http.StatusInternalServerError
		quota = quota || response.StatusCode == http.StatusTooManyRequests
	} else {
		// The SDK maps remaining transport failures (such as a reset or
		// truncated response) to UNKNOWN without an HTTP response.
		retryable = retryable || errorutils.IsUnknown(err)
	}
	if !retryable && !quota {
		return ProviderResult{Class: ResultPermanentFailure}
	}
	minimum := 10 * time.Second
	if quota {
		minimum = time.Minute
	}
	if response != nil {
		minimum = max(minimum, firebaseRetryAfter(response.Header.Get("Retry-After"), now))
	}
	return ProviderResult{Class: ResultRetryable, RetryAfter: minimum}
}

func firebaseRetryAfter(header string, now time.Time) time.Duration {
	header = strings.TrimSpace(header)
	if seconds, err := strconv.ParseUint(header, 10, 32); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if until, err := http.ParseTime(header); err == nil {
		return until.Sub(now)
	}
	return 0
}

func (s *FirebaseSender) buildMessage(request SendRequest) (*messaging.Message, error) {
	if s == nil || s.client == nil || s.now == nil || request.Token == "" ||
		request.TTL <= 0 ||
		request.Semantics != DeliveryUniqueEvent {
		return nil, ErrPushPayloadInvalid
	}
	var payload Payload
	if request.Chat != nil {
		reference, err := uuid.Parse(request.Chat.OpenReference)
		subscription, subscriptionErr := uuid.Parse(request.AccountSubscriptionID)
		if err != nil || reference == uuid.Nil || subscriptionErr != nil || subscription == uuid.Nil || request.Category != "" || request.RoutingFacts != (RoutingFacts{}) {
			return nil, ErrPushPayloadInvalid
		}
		payload = BuildChatPayload(request.AccountSubscriptionID, request.ActorDisplayName, request.Chat.OpenReference)
	} else {
		if !request.Category.Valid() {
			return nil, ErrPushPayloadInvalid
		}
		if _, err := uuid.Parse(request.RoutingFacts.NotificationID); err != nil {
			return nil, ErrPushPayloadInvalid
		}
		payload = BuildPayload(request.Category, request.AccountSubscriptionID, request.ActorDisplayName, request.RoutingFacts)
	}
	data := make(map[string]string, len(payload.Data)+3)
	for key, value := range payload.Data {
		data[key] = value
	}
	if request.Chat == nil {
		data["notificationId"] = request.RoutingFacts.NotificationID
	}
	message := &messaging.Message{
		//lint:ignore SA1019 request.Token is an FCM registration token; Fid requires a different client registration contract.
		Token: request.Token,
		Data:  data,
		Notification: &messaging.Notification{
			Title: safePushDisplayText(payload.Title, "CraftSky"),
			Body: safePushDisplayText(
				previewBody(request.Category, payload.Body, request.Preview),
				"You have a new notification",
			),
		},
	}
	deadline := s.now().Add(request.TTL)

	switch request.Platform {
	case "web":
		routingID, err := uuid.Parse(request.AccountSubscriptionID)
		if err != nil || routingID == uuid.Nil {
			return nil, ErrPushPayloadInvalid
		}
		// The CraftSky service worker owns display and click handling on the
		// web. An FCM notification would bypass that handler and can focus an
		// existing tab without navigating it to the destination.
		if request.Category == notifications.Mention || request.Category == notifications.Quote {
			delete(message.Data, "rootUri")
		}
		message.Data["webTitle"] = message.Notification.Title
		message.Data["webBody"] = message.Notification.Body
		message.Notification = nil
		message.Webpush = &messaging.WebpushConfig{
			Headers: map[string]string{"TTL": strconv.FormatInt(int64(request.TTL/time.Second), 10)},
		}
	case "android":
		message.Android = &messaging.AndroidConfig{TTL: &request.TTL}
		if request.Chat != nil && request.Chat.AcceptedConversation {
			message.Android.Priority = "high"
		}
	case "ios":
		message.APNS = &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-expiration": strconv.FormatInt(deadline.Unix(), 10),
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{Sound: "default"},
			},
		}
	default:
		return nil, ErrPushPayloadInvalid
	}
	return message, nil
}

func safePushDisplayText(value, fallback string) string {
	if value == "" || len(value) > 256 || !utf8.ValidString(value) {
		return fallback
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return fallback
		}
	}
	return value
}
