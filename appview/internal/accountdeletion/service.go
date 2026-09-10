package accountdeletion

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/subscriptions"
)

var (
	ErrReauthenticationRequired      = errors.New("account deletion reauthentication required")
	ErrConfirmationDIDMismatch       = errors.New("account deletion confirmation DID mismatch")
	ErrDeletionAlreadyPending        = errors.New("account deletion already pending")
	ErrIdentityUnavailable           = errors.New("account deletion identity unavailable")
	ErrProviderBillingMustBeResolved = subscriptions.ErrProviderBillingMustBeResolved
)

type CreateIntentParams struct {
	Owner    syntax.DID
	DeviceID string
}

type IntentResult struct {
	JobID           string           `json:"jobId"`
	AuthURL         string           `json:"authUrl"`
	ConfirmationDID syntax.DID       `json:"confirmationDid"`
	ExpiresAt       time.Time        `json:"expiresAt"`
	Warning         *DeletionWarning `json:"warning,omitempty"`
}

type DeletionWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BillingDeletionParticipant interface {
	BeginDeletion(context.Context, pgx.Tx, syntax.DID, time.Time) (bool, error)
	ConfirmDeletion(context.Context, pgx.Tx, syntax.DID, time.Time) (bool, error)
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
