package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ownerlifecycle"
)

// OnboardingReconciler repairs missed profile departures only after the current
// PDS authority and the newly exchanged OAuth credential have been verified.
type OnboardingReconciler struct {
	owners        *ownerlifecycle.Store
	sessions      *SessionLifecycleService
	newClient     PendingOnboardingPDSClientFactory
	departure     ownerlifecycle.TransitionParticipant
	removeProfile ownerlifecycle.TransitionParticipant
}

func NewOnboardingReconciler(
	owners *ownerlifecycle.Store,
	sessions *SessionLifecycleService,
	newClient PendingOnboardingPDSClientFactory,
	departure, removeProfile ownerlifecycle.TransitionParticipant,
) (*OnboardingReconciler, error) {
	if owners == nil || sessions == nil || sessions.owners != owners || newClient == nil || departure == nil || removeProfile == nil {
		return nil, errors.New("onboarding reconciliation dependencies are unavailable")
	}
	return &OnboardingReconciler{owners: owners, sessions: sessions, newClient: newClient, departure: departure, removeProfile: removeProfile}, nil
}

func (service *OnboardingReconciler) Reconcile(
	ctx context.Context, result OAuthCallbackResult,
) (context.Context, OAuthCallbackResult, error) {
	attempt := result.Attempt
	bound, ok := callbackAttemptFromContext(ctx)
	if !ok || bound != attempt || !attempt.permitsOrdinaryOnboarding() || !attempt.validFor(attempt.Owner, attempt.State) {
		return ctx, result, ErrCallbackAttemptInvalid
	}
	authority, err := service.owners.Get(ctx, attempt.Owner)
	if err != nil {
		return ctx, result, err
	}
	if authority.Generation != attempt.OwnerGeneration || authority.AuthEpoch != attempt.AuthEpoch {
		return ctx, result, ErrCallbackAttemptInvalid
	}
	if authority.State == ownerlifecycle.StateDeparted {
		return ctx, result, nil
	}
	if authority.State != ownerlifecycle.StateActive {
		return ctx, result, ErrOAuthOwnerIneligible
	}
	client, err := service.newClient(ctx, attempt)
	if err != nil {
		return ctx, result, &profileInitStageError{"onboarding_client", err}
	}
	if client == nil {
		return ctx, result, &profileInitStageError{"onboarding_client", errors.New("pending onboarding client unavailable")}
	}
	var profile map[string]any
	_, err = client.GetRecord(ctx, attempt.Owner, craftskyProfileNSID, profileRecordKey, &profile)
	if err == nil {
		return ctx, result, nil
	}
	if !errors.Is(err, ErrRecordNotFound) {
		return ctx, result, &profileInitStageError{"craftsky_profile_read", err}
	}
	participant := service.sessions.profileRecoveryParticipant(attempt, service.departure)
	err = service.owners.WithReconciledProfileDeparture(ctx, authority,
		func(txCtx context.Context, tx pgx.Tx, before, after ownerlifecycle.Lifecycle) error {
			if err := participant(txCtx, tx, before, after); err != nil {
				return err
			}
			return service.removeProfile(txCtx, tx, before, after)
		},
		func(reconciledCtx context.Context, after ownerlifecycle.Lifecycle) error {
			result.Attempt.OwnerGeneration = after.Generation
			result.Attempt.AuthEpoch = after.AuthEpoch
			result.Metadata.OwnerGeneration = after.Generation
			result.Metadata.AuthEpoch = after.AuthEpoch
			ctx = WithCallbackAttempt(reconciledCtx, result.Attempt)
			return nil
		})
	if err != nil {
		return ctx, result, &profileInitStageError{"profile_reconciliation", err}
	}
	return ctx, result, nil
}
