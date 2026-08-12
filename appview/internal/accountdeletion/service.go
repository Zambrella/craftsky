package accountdeletion

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var (
	ErrReauthenticationRequired   = errors.New("account deletion reauthentication required")
	ErrConfirmationHandleMismatch = errors.New("account deletion confirmation handle mismatch")
	ErrDeletionAlreadyPending     = errors.New("account deletion already pending")
)

type CreateIntentParams struct {
	Owner syntax.DID
}

type IntentResult struct {
	JobID     string    `json:"jobId"`
	AuthURL   string    `json:"authUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type AcceptParams struct {
	JobID              string
	Owner              syntax.DID
	ReauthProof        string
	ConfirmationHandle string
}

type Service interface {
	CreateIntent(context.Context, CreateIntentParams) (IntentResult, error)
	CancelIntent(context.Context, string, syntax.DID) error
	Accept(context.Context, AcceptParams) error
}
