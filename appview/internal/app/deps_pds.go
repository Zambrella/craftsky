package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/pdseffects"
)

// pdsEffectDependencies is the complete authenticated PDS capability surface
// made available to higher-level AppView features. It intentionally exposes no
// raw PDS client factory.
type pdsEffectDependencies struct {
	pending    auth.PendingOnboardingPDSClientFactory
	blobs      api.BlobEffectFactory
	guarded    pdseffects.GuardedCapabilityCoordinatorFactory
	commands   *pdscommands.SetCommandService
	append     *pdscommands.AppendCommandService
	addressed  *pdscommands.AddressedCommandService
	compound   *pdscommands.CompoundCommandService
	compaction *pdscommands.CompactionProcessor
}

func newPDSEffectDependencies(
	authCapability *authDependencies,
	federated *federatedClients,
	owners *ownerDependencies,
	pool *pgxpool.Pool,
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
	blobs := func(ctx context.Context, owner syntax.DID, sessionID string) (api.BlobEffectExecutor, error) {
		return ordinary(ctx, owner, sessionID)
	}
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{
		Pool: pool, Lifecycles: owners.lifecycles,
	})
	if err != nil {
		return nil, fmt.Errorf("PDS command store: %w", err)
	}
	commandCompaction, err := pdscommands.NewCompactionProcessor(commandStore, cfg.PDSCommandCompactionBatchSize)
	if err != nil {
		return nil, fmt.Errorf("PDS command compaction processor: %w", err)
	}
	snapshotFallback, err := ingestion.NewRepositorySnapshotFetcher(federated.authoritativeDirectory, federated.pdsRepository)
	if err != nil {
		return nil, fmt.Errorf("PDS set command snapshot fallback: %w", err)
	}
	newCommandBoundary := func(ctx context.Context, owner syntax.DID, sessionID string) (auth.ActiveEffectPDSBoundary, error) {
		client, err := newPDSClient(ctx, owner, sessionID)
		if err != nil {
			return nil, err
		}
		boundary, ok := client.(auth.ActiveEffectPDSBoundary)
		if !ok {
			return nil, errors.New("PDS command boundary is unavailable")
		}
		return boundary, nil
	}
	commands, err := pdscommands.NewSetCommandService(pdscommands.SetCommandServiceConfig{
		Store: commandStore, Lifecycles: owners.lifecycles,
		SnapshotFallback: snapshotFallback,
		NewBoundary:      newCommandBoundary,
	})
	if err != nil {
		return nil, fmt.Errorf("PDS command service: %w", err)
	}
	appendClock := syntax.NewTIDClock(0)
	appendCommands, err := pdscommands.NewAppendCommandService(pdscommands.AppendCommandServiceConfig{
		Store: commandStore, Lifecycles: owners.lifecycles,
		NewBoundary: newCommandBoundary,
		NewRecordKey: func() (syntax.RecordKey, error) {
			return syntax.ParseRecordKey(appendClock.Next().String())
		},
	})
	if err != nil {
		return nil, fmt.Errorf("PDS append command service: %w", err)
	}
	addressedCommands, err := pdscommands.NewAddressedCommandService(pdscommands.AddressedCommandServiceConfig{
		Store: commandStore, Lifecycles: owners.lifecycles,
		NewBoundary: newCommandBoundary,
	})
	if err != nil {
		return nil, fmt.Errorf("PDS addressed command service: %w", err)
	}
	compoundCommands, err := pdscommands.NewCompoundCommandService(pdscommands.CompoundCommandServiceConfig{
		Store: commandStore, Lifecycles: owners.lifecycles,
		NewBoundary: newCommandBoundary,
	})
	if err != nil {
		return nil, fmt.Errorf("PDS compound command service: %w", err)
	}
	guarded, err := pdseffects.NewGuardedCapabilityCoordinatorFactory(
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
		pending: pending, blobs: blobs, guarded: guarded, commands: commands, append: appendCommands, addressed: addressedCommands, compound: compoundCommands, compaction: commandCompaction,
	}, nil
}
