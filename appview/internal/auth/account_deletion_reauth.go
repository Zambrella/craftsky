package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
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

const AccountDeletionOAuthPurpose OAuthPurpose = "accountDeletion"

type AuthRequestMetadata struct {
	Purpose OAuthPurpose
	Owner   syntax.DID
	JobID   uuid.UUID
}

type authRequestMetadataContextKey struct{}

func WithAccountDeletionAuthRequest(
	ctx context.Context,
	owner syntax.DID,
	jobID uuid.UUID,
) context.Context {
	return context.WithValue(ctx, authRequestMetadataContextKey{}, AuthRequestMetadata{
		Purpose: AccountDeletionOAuthPurpose,
		Owner:   owner,
		JobID:   jobID,
	})
}

func AuthRequestMetadataFromContext(ctx context.Context) (AuthRequestMetadata, bool) {
	metadata, ok := ctx.Value(authRequestMetadataContextKey{}).(AuthRequestMetadata)
	return metadata, ok
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
