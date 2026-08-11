package accountdeletion

import (
	"context"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var (
	ErrReauthenticationRequired   = errors.New("account deletion reauthentication required")
	ErrConfirmationHandleMismatch = errors.New("account deletion confirmation handle mismatch")
	ErrDeletionAlreadyPending     = errors.New("account deletion already pending")
	ErrRecoveryUnauthorized       = errors.New("account deletion recovery unauthorized")
)

type CreateIntentParams struct {
	Owner    syntax.DID
	DeviceID string
}

type IntentResult struct {
	JobID       string    `json:"jobId"`
	StatusToken string    `json:"statusToken"`
	AuthURL     string    `json:"authUrl"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type AcceptParams struct {
	JobID              string
	Owner              syntax.DID
	DeviceID           string
	StatusCapability   string
	ReauthProof        string
	ConfirmationHandle string
}

type AcceptResult struct {
	JobID  string `json:"jobId"`
	Status Status `json:"status"`
	Phase  Phase  `json:"phase"`
}

type RecoveryResult struct {
	JobID       string `json:"jobId"`
	StatusToken string `json:"statusToken"`
	Status      Status `json:"status"`
	Phase       Phase  `json:"phase"`
}

type Service interface {
	CreateIntent(ctx context.Context, params CreateIntentParams) (IntentResult, error)
	CancelIntent(ctx context.Context, jobID string, owner syntax.DID, statusCapability string) error
	Accept(ctx context.Context, params AcceptParams) (AcceptResult, error)
}

type RecoveryService interface {
	Recover(ctx context.Context, formerBearer, deviceID string) (RecoveryResult, error)
}

type ReauthenticationStart struct {
	AuthURL   string    `json:"authUrl"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type StatusRouteService interface {
	AuthorizeStatusRoute(ctx context.Context, token string, jobID uuid.UUID, deviceID string, action StatusAction) (StatusGrant, error)
	GetStatus(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (DeletionStatusView, error)
	Retry(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (DeletionStatusView, error)
	StartReauthentication(ctx context.Context, jobID uuid.UUID, owner syntax.DID) (ReauthenticationStart, error)
}
