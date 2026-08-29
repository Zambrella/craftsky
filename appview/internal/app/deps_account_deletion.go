package app

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/scheduledposts"
)

type accountDeletionDependencies struct {
	service      *accountdeletion.AppService
	worker       *accountdeletion.Worker
	intentExpiry *accountdeletion.IntentExpiryProcessor
}

// newAccountDeletionDependencies constructs the complete deletion-only
// capability. Its PDS factory cannot be reused for ordinary member effects.
func newAccountDeletionDependencies(
	pool *pgxpool.Pool,
	authCapability *authDependencies,
	owners *ownerDependencies,
	federated *federatedClients,
	accountTypes accountdeletion.AccountTypeDeleter,
	instagramPrivateData *instagram.PrivateDataService,
	scheduledAccountDeletion *scheduledposts.AccountDeletion,
	departureParticipant ownerlifecycle.TransitionParticipant,
	identityResolver api.HandleResolver,
	observer *observability.Observer,
	cfg Config,
	logger *slog.Logger,
) (*accountDeletionDependencies, error) {
	service, err := accountdeletion.NewAppService(accountdeletion.AppServiceOptions{
		Pool: pool, Store: owners.deletionStore, OAuth: authCapability.flow,
		Owners: owners.lifecycles, Sessions: authCapability.sessionLifecycle,
		OAuthStore: authCapability.store, DepartureParticipant: departureParticipant,
		IdentityResolver: identityResolver,
		IdentityIndex:    api.NewIdentityCacheStore(pool, observer),
		Now:              time.Now, Random: rand.Reader, IntentTTL: cfg.AccountDeletionIntentTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("account deletion service: %w", err)
	}
	instagramDeletion, err := accountdeletion.NewNamedPrivateCleanup(
		"instagramPrivate",
		func(ctx context.Context, owner syntax.DID) error {
			return accountdeletion.PurgeInstagramForAccountDeletion(ctx, instagramPrivateData, owner)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("account deletion Instagram cleanup: %w", err)
	}
	cleaner, err := accountdeletion.NewPrivateCleaner(
		[]accountdeletion.PrivateCleanupComponent{
			accountdeletion.NewDatabasePrivateCleanup(pool),
			instagramDeletion,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("account deletion private cleanup: %w", err)
	}
	lifecycle, err := accountdeletion.NewLifecycleProcessor(accountdeletion.LifecycleProcessorOptions{
		Store: owners.deletionStore, Cleaner: cleaner,
		AcceptedCleanup: scheduledAccountDeletion,
		AccountTypes:    accountTypes,
		NewPDSClient: func(
			_ context.Context,
			owner syntax.DID,
			authority auth.DeletionSessionAuthority,
		) (auth.DeletionPDSClient, error) {
			return auth.NewCoordinatedDeletionPDSClient(
				authCapability.sessionCoordinator,
				owner,
				authority,
				func(operationCtx context.Context, session *oauth.ClientSession) (auth.PDSClient, error) {
					return federated.newPDSClient(operationCtx, session, nil)
				},
			)
		},
		BatchSize: 20,
	})
	if err != nil {
		return nil, fmt.Errorf("account deletion lifecycle: %w", err)
	}
	worker, err := accountdeletion.NewWorker(accountdeletion.WorkerOptions{
		Store: owners.deletionStore, Processor: lifecycle, Finalizer: service,
		WorkerID: "appview", Now: time.Now, LeaseDuration: 2 * time.Minute,
		RetryPolicy: accountdeletion.DefaultRetryPolicy(), Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("account deletion worker: %w", err)
	}
	intentExpiry, err := accountdeletion.NewIntentExpiryProcessor(accountdeletion.IntentExpiryProcessorOptions{
		Source: owners.deletionStore, Expirer: service,
		BatchSize: cfg.AccountDeletionIntentSweepBatch,
	})
	if err != nil {
		return nil, fmt.Errorf("account deletion intent expiry processor: %w", err)
	}
	return &accountDeletionDependencies{
		service: service, worker: worker, intentExpiry: intentExpiry,
	}, nil
}
