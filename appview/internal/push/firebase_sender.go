package push

import (
	"context"
	"errors"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
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
	payload := BuildPayload(request.NotificationID, request.Category, request.AccountSubscriptionID, request.ActorDisplayName)
	deadline := s.now().Add(request.TTL)
	message := &messaging.Message{Token: request.Token, Notification: &messaging.Notification{Title: payload.Title, Body: payload.Body}, Data: payload.Data, Android: &messaging.AndroidConfig{TTL: &request.TTL}, APNS: &messaging.APNSConfig{Headers: map[string]string{"apns-expiration": strconv.FormatInt(deadline.Unix(), 10)}, Payload: &messaging.APNSPayload{Aps: &messaging.Aps{Sound: "default"}}}}
	_, err := s.client.Send(ctx, message)
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
