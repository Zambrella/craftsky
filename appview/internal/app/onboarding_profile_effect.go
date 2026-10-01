package app

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

// onboardingProfileEffectAdapter exposes only the deterministic CraftSky
// profile command admitted during a fenced OAuth callback.
type onboardingProfileEffectAdapter struct {
	executor onboardingProfilePutter
}

type onboardingProfilePutter interface {
	PutProfile(context.Context, auth.PDSClient, syntax.DID, int64, any) (syntax.CID, error)
}

func (adapter onboardingProfileEffectAdapter) PutOnboardingProfile(
	ctx context.Context,
	client auth.PDSClient,
	request auth.OnboardingProfileWrite,
) (syntax.CID, error) {
	if adapter.executor == nil {
		return "", errors.New("onboarding PDS effect executor is unavailable")
	}
	return adapter.executor.PutProfile(ctx, client, request.Owner, request.OwnerGeneration, request.Record)
}

var _ auth.OnboardingProfileWriter = onboardingProfileEffectAdapter{}
