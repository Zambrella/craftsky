package accountdeletion

import (
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type Status string

const (
	StatusIntent         Status = "intent"
	StatusActive         Status = "active"
	StatusRetrying       Status = "retrying"
	StatusReauthRequired Status = "reauth_required"
)

var (
	ErrPointOfNoReturn        = errors.New("account deletion is past the point of no return")
	ErrBoundOAuthUnauthorized = errors.New("bound OAuth session unauthorized")
	ErrSafetyLeaseLost        = errors.New("account deletion safety lease lost")
	ErrSafetyPending          = errors.New("account deletion remote safety reconciliation is pending")
)

type PDSSafetyClaim struct {
	ID              uuid.UUID
	OperationID     uuid.UUID
	Owner           syntax.DID
	OwnerGeneration int64
	URI             syntax.ATURI
	SourceAttemptID string
	Attempts        int
	LeaseToken      uuid.UUID
	LeaseExpiresAt  time.Time
}
