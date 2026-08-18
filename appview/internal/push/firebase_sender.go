package push

import (
	"context"
	"errors"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
)

type firebaseClient interface {
	Send(context.Context, *messaging.Message) (string, error)
}
type FirebaseSender struct {
	client firebaseClient
	now    func() time.Time
}

func NewFirebaseSender(ctx context.Context, projectID string) (*FirebaseSender, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	return &FirebaseSender{client: client, now: time.Now}, nil
}
func (s *FirebaseSender) Send(ctx context.Context, request SendRequest) (ProviderResult, error) {
	message, err := s.buildMessage(request)
	if err != nil {
		return ProviderResult{Class: ResultPermanentFailure}, err
	}
	_, err = s.client.Send(ctx, message)
	if err == nil {
		return ProviderResult{Class: ResultSuccess}, nil
	}
	switch {
	case messaging.IsUnregistered(err):
		return ProviderResult{Class: ResultInvalidToken}, err
	case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) || messaging.IsUnavailable(err) || messaging.IsInternal(err) || messaging.IsQuotaExceeded(err):
		return ProviderResult{Class: ResultRetryable}, err
	default:
		return ProviderResult{Class: ResultPermanentFailure}, err
	}
}

func (s *FirebaseSender) buildMessage(request SendRequest) (*messaging.Message, error) {
	if s == nil || s.client == nil || s.now == nil || request.Token == "" ||
		!request.Category.Valid() || request.TTL <= 0 ||
		request.Semantics != DeliveryUniqueEvent {
		return nil, ErrPushPayloadInvalid
	}
	if _, err := uuid.Parse(request.RoutingFacts.NotificationID); err != nil {
		return nil, ErrPushPayloadInvalid
	}
	payload := BuildPayload(
		request.Category,
		request.AccountSubscriptionID,
		request.ActorDisplayName,
		request.RoutingFacts,
	)
	data := make(map[string]string, len(payload.Data)+3)
	for key, value := range payload.Data {
		data[key] = value
	}
	data["notificationId"] = request.RoutingFacts.NotificationID
	message := &messaging.Message{Token: request.Token, Data: data}
	deadline := s.now().Add(request.TTL)

	switch request.Platform {
	case "android":
		data["displayTitle"] = safePushDisplayText(payload.Title, "CraftSky")
		data["displayBody"] = safePushDisplayText(
			payload.Body,
			"You have a new notification",
		)
		message.Android = &messaging.AndroidConfig{TTL: &request.TTL}
	case "ios":
		message.Notification = &messaging.Notification{
			Title: safePushDisplayText(payload.Title, "CraftSky"),
			Body: safePushDisplayText(
				payload.Body,
				"You have a new notification",
			),
		}
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
