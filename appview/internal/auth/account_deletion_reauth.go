package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var (
	ErrDeletionReauthenticationRequired   = errors.New("deletion reauthentication required")
	ErrDeletionReauthReplayed             = errors.New("deletion reauthentication replayed")
	ErrDeletionConfirmationHandleMismatch = errors.New("deletion confirmation handle mismatch")
)

type OAuthPurpose string

const (
	LoginOAuthPurpose           OAuthPurpose = "login"
	AccountDeletionOAuthPurpose OAuthPurpose = "accountDeletion"
)

type HandoffMode string

const (
	HandoffVerifiedLink HandoffMode = "verified_link"
	HandoffLoopback     HandoffMode = "loopback"
	HandoffDevScheme    HandoffMode = "dev_scheme"
)

type AuthRequestState string

const (
	AuthRequestReady             AuthRequestState = "ready"
	AuthRequestExchangeStarted   AuthRequestState = "exchange_started"
	AuthRequestExchangeFailed    AuthRequestState = "exchange_failed"
	AuthRequestExchangeAmbiguous AuthRequestState = "exchange_ambiguous"
	AuthRequestConsumed          AuthRequestState = "consumed"
	AuthRequestRevoked           AuthRequestState = "revoked"
)

type AuthRequestMetadata struct {
	Purpose           OAuthPurpose
	Owner             syntax.DID
	OwnerGeneration   int64
	AuthEpoch         int64
	JobID             uuid.UUID
	HandoffMode       HandoffMode
	LoopbackURI       string
	DeviceID          string
	RequestState      AuthRequestState
	ExchangeAttemptID uuid.UUID
}

type authRequestMetadataContextKey struct{}

func WithAccountDeletionAuthRequest(
	ctx context.Context,
	owner syntax.DID,
	jobID uuid.UUID,
) context.Context {
	// Retained only as a compile-time migration aid for callers which have not
	// yet supplied lifecycle authority. SaveAuthRequestInfo rejects this
	// incomplete metadata; production wiring must call the authority-bearing
	// variant below.
	return context.WithValue(ctx, authRequestMetadataContextKey{}, AuthRequestMetadata{
		Purpose: AccountDeletionOAuthPurpose,
		Owner:   owner,
		JobID:   jobID,
	})
}

func WithLoginAuthRequest(
	ctx context.Context,
	owner syntax.DID,
	ownerGeneration int64,
	authEpoch int64,
	mode HandoffMode,
	deviceID string,
	loopbackURI string,
) context.Context {
	return context.WithValue(ctx, authRequestMetadataContextKey{}, AuthRequestMetadata{
		Purpose:         LoginOAuthPurpose,
		Owner:           owner,
		OwnerGeneration: ownerGeneration,
		AuthEpoch:       authEpoch,
		HandoffMode:     mode,
		LoopbackURI:     loopbackURI,
		DeviceID:        deviceID,
	})
}

func WithAccountDeletionAuthRequestAuthority(
	ctx context.Context,
	owner syntax.DID,
	ownerGeneration int64,
	authEpoch int64,
	jobID uuid.UUID,
	deviceID string,
) context.Context {
	return context.WithValue(ctx, authRequestMetadataContextKey{}, AuthRequestMetadata{
		Purpose:         AccountDeletionOAuthPurpose,
		Owner:           owner,
		OwnerGeneration: ownerGeneration,
		AuthEpoch:       authEpoch,
		JobID:           jobID,
		HandoffMode:     HandoffVerifiedLink,
		DeviceID:        deviceID,
	})
}

func AuthRequestMetadataFromContext(ctx context.Context) (AuthRequestMetadata, bool) {
	metadata, ok := ctx.Value(authRequestMetadataContextKey{}).(AuthRequestMetadata)
	return metadata, ok
}

func (metadata AuthRequestMetadata) valid() bool {
	if metadata.Owner == "" || metadata.OwnerGeneration <= 0 || metadata.AuthEpoch <= 0 ||
		strings.TrimSpace(metadata.DeviceID) == "" {
		return false
	}
	if metadata.HandoffMode != HandoffVerifiedLink && metadata.HandoffMode != HandoffLoopback && metadata.HandoffMode != HandoffDevScheme {
		return false
	}
	if (metadata.HandoffMode == HandoffLoopback) != (metadata.LoopbackURI != "") {
		return false
	}
	switch metadata.Purpose {
	case LoginOAuthPurpose:
		return metadata.JobID == uuid.Nil
	case AccountDeletionOAuthPurpose:
		return metadata.JobID != uuid.Nil && metadata.HandoffMode == HandoffVerifiedLink
	default:
		return false
	}
}

type AccountDeletionAuthRequest struct {
	Purpose OAuthPurpose
	JobID   string
	Owner   syntax.DID
}

type AccountDeletionOAuthResult struct {
	JobID string
	Proof string
}

type AccountDeletionPendingLogin struct {
}

type AccountDeletionPendingLoginPolicy interface {
	PendingLogin(context.Context, syntax.DID, string, string) (AccountDeletionPendingLogin, bool, error)
	Reject(context.Context, syntax.DID, string) error
}

// AccountDeletionOAuthCallbacks is consulted before ProcessCallback consumes
// the single-use auth request. Implementations persist purpose/owner/job
// metadata atomically with that request and complete only deletion-purpose
// callbacks here.
type AccountDeletionOAuthCallbacks interface {
	RequestForState(ctx context.Context, state string) (AccountDeletionAuthRequest, bool, error)
	Complete(ctx context.Context, request AccountDeletionAuthRequest, did syntax.DID, sessionID string) (AccountDeletionOAuthResult, error)
	Reject(ctx context.Context, did syntax.DID, sessionID string) error
}

// AccountDeletionOAuthAttemptCallbacks is the fenced callback contract. The
// implementation must bind the exact pending parent through
// PostgresAuthStore.BindDeletionCredential before returning proof material.
type AccountDeletionOAuthAttemptCallbacks interface {
	CompleteAttempt(
		context.Context,
		AccountDeletionAuthRequest,
		CallbackAttempt,
	) (AccountDeletionOAuthResult, error)
}

type AccountDeletionReauthIntent struct {
	JobID          string
	Owner          syntax.DID
	ExpectedHandle string
	IssuedAt       time.Time
	ExpiresAt      time.Time
	Canceled       bool
}

type AccountDeletionReauthCompletion struct {
	JobID          string
	Owner          syntax.DID
	OAuthSessionID string
	ProofHash      [sha256.Size]byte
	CompletedAt    time.Time
	ExpiresAt      time.Time
	Consumed       bool
}

func CompleteAccountDeletionReauth(
	intent AccountDeletionReauthIntent,
	callbackDID syntax.DID,
	oauthSessionID string,
	proof string,
	now time.Time,
) (AccountDeletionReauthCompletion, error) {
	if intent.Canceled || intent.JobID == "" || intent.Owner == "" || callbackDID != intent.Owner ||
		oauthSessionID == "" || proof == "" || now.Before(intent.IssuedAt) || !now.Before(intent.ExpiresAt) {
		return AccountDeletionReauthCompletion{}, ErrDeletionReauthenticationRequired
	}
	return AccountDeletionReauthCompletion{
		JobID:          intent.JobID,
		Owner:          intent.Owner,
		OAuthSessionID: oauthSessionID,
		ProofHash:      sha256.Sum256([]byte(proof)),
		CompletedAt:    now,
		ExpiresAt:      intent.ExpiresAt,
	}, nil
}

func ConsumeAccountDeletionReauth(
	intent AccountDeletionReauthIntent,
	completion *AccountDeletionReauthCompletion,
	proof string,
	confirmationHandle string,
	now time.Time,
) (string, error) {
	if completion == nil || intent.Canceled || completion.JobID != intent.JobID || completion.Owner != intent.Owner ||
		completion.OAuthSessionID == "" || now.Before(completion.CompletedAt) || !now.Before(intent.ExpiresAt) || !now.Before(completion.ExpiresAt) {
		return "", ErrDeletionReauthenticationRequired
	}
	if completion.Consumed {
		return "", ErrDeletionReauthReplayed
	}
	if confirmationHandle != intent.ExpectedHandle {
		return "", ErrDeletionConfirmationHandleMismatch
	}
	want := sha256.Sum256([]byte(proof))
	if subtle.ConstantTimeCompare(completion.ProofHash[:], want[:]) != 1 {
		return "", ErrDeletionReauthenticationRequired
	}
	completion.Consumed = true
	return completion.OAuthSessionID, nil
}
