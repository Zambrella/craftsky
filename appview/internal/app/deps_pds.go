package app

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdseffects"
)

// pdsEffectDependencies is the complete authenticated PDS capability surface
// made available to higher-level AppView features. It intentionally exposes no
// raw PDS client factory.
type pdsEffectDependencies struct {
	pending  auth.PendingOnboardingPDSClientFactory
	ordinary pdseffects.ExecutorFactory
	guarded  pdseffects.GuardedExecutorFactory
}

func newPDSEffectDependencies(
	authCapability *authDependencies,
	federated *federatedClients,
	owners *ownerDependencies,
	observer *observability.Observer,
	cfg Config,
) (*pdsEffectDependencies, error) {
	newPDSClient := observer.WrapPDSFactory(func(
		_ context.Context,
		did syntax.DID,
		sessionID string,
	) (auth.PDSClient, error) {
		return auth.NewCoordinatedPDSClient(
			authCapability.sessionCoordinator,
			did,
			sessionID,
			func(operationCtx context.Context, session *oauth.ClientSession) (auth.PDSClient, error) {
				return federated.newPDSClient(operationCtx, session, nil)
			},
		)
	})
	ordinary, err := pdseffects.NewExecutorFactory(
		owners.lifecycles,
		newPDSClient,
		cfg.PDSEffectTimeout,
		time.Now,
	)
	if err != nil {
		return nil, fmt.Errorf("ordinary PDS effect executor: %w", err)
	}
	guarded, err := pdseffects.NewGuardedExecutorFactory(
		owners.lifecycles,
		newPDSClient,
		cfg.PDSEffectTimeout,
		time.Now,
	)
	if err != nil {
		return nil, fmt.Errorf("guarded PDS effect executor: %w", err)
	}
	pending := func(ctx context.Context, attempt auth.CallbackAttempt) (auth.PDSClient, error) {
		stored, err := authCapability.store.ResumePendingOnboardingSession(ctx, attempt)
		if err != nil {
			return nil, err
		}
		return federated.newPendingPDSClient(ctx, authCapability.app.Config, stored.Data)
	}
	return &pdsEffectDependencies{
		pending: pending, ordinary: ordinary, guarded: guarded,
	}, nil
}
