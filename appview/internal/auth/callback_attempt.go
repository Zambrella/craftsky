package auth

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

var ErrCallbackAttemptInvalid = errors.New("OAuth callback attempt invalid")

type CallbackAttempt struct {
	State           string
	AttemptID       uuid.UUID
	Owner           syntax.DID
	OwnerGeneration int64
	AuthEpoch       int64
	Purpose         OAuthPurpose
}

type callbackAttemptContextKey struct{}

func WithCallbackAttempt(ctx context.Context, attempt CallbackAttempt) context.Context {
	return context.WithValue(ctx, callbackAttemptContextKey{}, attempt)
}

func callbackAttemptFromContext(ctx context.Context) (CallbackAttempt, bool) {
	attempt, ok := ctx.Value(callbackAttemptContextKey{}).(CallbackAttempt)
	return attempt, ok
}

func (attempt CallbackAttempt) validFor(owner syntax.DID, sessionID string) bool {
	return attempt.State != "" && attempt.State == sessionID && attempt.AttemptID != uuid.Nil &&
		attempt.Owner != "" && attempt.Owner == owner && attempt.OwnerGeneration > 0 &&
		attempt.AuthEpoch > 0 &&
		(attempt.Purpose == LoginOAuthPurpose || attempt.Purpose == AccountDeletionOAuthPurpose)
}
