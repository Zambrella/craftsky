package app

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
)

// onboardingProfileEffectAdapter exposes only the single deterministic
// CraftSky profile Put permitted during a fenced OAuth callback. It cannot be
// reused as an ordinary departed-owner PDS writer.
type onboardingProfileEffectAdapter struct {
	executor onboardingProfilePutter
}

type onboardingProfilePutter interface {
	PutProfile(
		context.Context,
		auth.PDSClient,
		pdseffects.OnboardingProfileRequest,
	) (pdseffects.RecordResult, error)
}

func (adapter onboardingProfileEffectAdapter) PutOnboardingProfile(
	ctx context.Context,
	client auth.PDSClient,
	request auth.OnboardingProfileWrite,
) (syntax.CID, error) {
	if adapter.executor == nil {
		return "", errors.New("onboarding PDS effect executor is unavailable")
	}
	result, err := adapter.executor.PutProfile(ctx, client, pdseffects.OnboardingProfileRequest{
		OperationID: request.OperationID, MutationKey: request.MutationKey,
		Owner: request.Owner, OwnerGeneration: request.OwnerGeneration,
		Record: request.Record,
	})
	return result.CID, err
}

var _ auth.OnboardingProfileWriter = onboardingProfileEffectAdapter{}
