package push

import (
	"context"
	"errors"
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

type DeliverySemantics string

const (
	// DeliveryUniqueEvent means every logical event remains independently
	// meaningful. It must not be projected into provider collapse metadata.
	DeliveryUniqueEvent DeliverySemantics = "unique_event"
)

var ErrPushPayloadInvalid = errors.New("push provider payload is invalid")

type ContentRole string

const (
	ContentRolePost    ContentRole = "post"
	ContentRoleComment ContentRole = "comment"
	ContentRoleReply   ContentRole = "reply"
)

type RoutingFacts struct {
	ActorDID       syntax.DID
	SourceURI      syntax.ATURI
	SubjectURI     syntax.ATURI
	RootURI        syntax.ATURI
	TargetRole     ContentRole
	NotificationID string
}
type SendRequest struct {
	Token                 string
	Category              notifications.Category
	AccountSubscriptionID string
	RoutingFacts          RoutingFacts
	ActorDisplayName      string
	Platform              string
	Semantics             DeliverySemantics
	TTL                   time.Duration
}
type Sender interface {
	Send(context.Context, SendRequest) (ProviderResult, error)
}
