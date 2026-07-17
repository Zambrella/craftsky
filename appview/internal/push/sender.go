package push

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

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
type RoutingFacts struct {
	ActorDID   syntax.DID
	SourceURI  syntax.ATURI
	SubjectURI syntax.ATURI
}
type SendRequest struct {
	Token                 string
	Category              notifications.Category
	AccountSubscriptionID string
	RoutingFacts          RoutingFacts
	ActorDisplayName      string
	Platform              string
	TTL                   time.Duration
}
type Sender interface {
	Send(context.Context, SendRequest) (ProviderResult, error)
}
