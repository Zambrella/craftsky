package accountdeletion

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

var (
	ErrReauthenticationRequired = errors.New("account deletion reauthentication required")
	ErrConfirmationDIDMismatch  = errors.New("account deletion confirmation DID mismatch")
	ErrDeletionAlreadyPending   = errors.New("account deletion already pending")
	ErrIdentityUnavailable      = errors.New("account deletion identity unavailable")
)

type CreateIntentParams struct {
	Owner    syntax.DID
	DeviceID string
}

type IntentResult struct {
	JobID           string     `json:"jobId"`
	AuthURL         string     `json:"authUrl"`
	ConfirmationDID syntax.DID `json:"confirmationDid"`
	ExpiresAt       time.Time  `json:"expiresAt"`
}

type AcceptParams struct {
	JobID           string
	Owner           syntax.DID
	ReauthProof     string
	ConfirmationDID syntax.DID
}

type Service interface {
	CreateIntent(context.Context, CreateIntentParams) (IntentResult, error)
	CancelIntent(context.Context, string, syntax.DID) error
	Accept(context.Context, AcceptParams) error
}
