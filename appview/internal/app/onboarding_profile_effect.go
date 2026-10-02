package app

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
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
	cid, err := adapter.executor.PutProfile(ctx, client, request.Owner, request.OwnerGeneration, request.Record)
	switch {
	case errors.Is(err, pdscommands.ErrOnboardingProfileChanged):
		err = errors.Join(auth.ErrProfileCreationConflict, err)
	case errors.Is(err, pdscommands.ErrOnboardingProfileUnresolved):
		err = errors.Join(auth.ErrProfileWriteUnresolved, err)
	case errors.Is(err, pdscommands.ErrDispatchRejected), errors.Is(err, pdscommands.ErrAtomicWritesUnsupported):
		err = errors.Join(auth.ErrProfileWriteRejected, err)
	}
	return cid, err
}

var _ auth.OnboardingProfileWriter = onboardingProfileEffectAdapter{}

func newOnboardingReconciliationDependencies(
	owners *ownerDependencies,
	authCapability *authDependencies,
	pdsEffects *pdsEffectDependencies,
	scheduledDeparture ownerlifecycle.TransitionParticipant,
	tapCapability *tapDependencies,
) (*auth.OnboardingReconciler, error) {
	return auth.NewOnboardingReconciler(
		owners.lifecycles, authCapability.sessionLifecycle, pdsEffects.pending,
		composeTransitionParticipants(owners.deletionStore.ProfileDepartureParticipant(), scheduledDeparture),
		tapCapability.removeMissingProfile,
	)
}
