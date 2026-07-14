package push

import (
	"context"
	"time"

	"social.craftsky/appview/internal/notifications"
)

type ResultClass string

const (
	ResultSuccess          ResultClass = "success"
	ResultRetryable        ResultClass = "retryable"
	ResultInvalidToken     ResultClass = "invalidToken"
	ResultPermanentFailure ResultClass = "permanentFailure"
)

type ProviderResult struct{ Class ResultClass }
type SendRequest struct {
	Token                 string
	NotificationID        string
	Category              notifications.Category
	AccountSubscriptionID string
	ActorDisplayName      string
	Platform              string
	TTL                   time.Duration
}
type Sender interface {
	Send(context.Context, SendRequest) (ProviderResult, error)
}
