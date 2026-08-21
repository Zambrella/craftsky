package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/tap"
)

// tapDependencies is the complete durable ingestion pipeline. The anonymous
// PDS client is retained only inside the repository-reconciliation handler.
type tapDependencies struct {
	repositoryTracker *tap.AdminClient
	projectionWorker  *ingestion.ProjectionWorker
	repositoryWorker  *ingestion.RepositoryWorker
	quarantineWorker  *ingestion.QuarantineReplayWorker
	consumer          tap.Consumer
}

func newTapDependencies(
	pool *pgxpool.Pool,
	federated *federatedClients,
	authCapability *authDependencies,
	owners *ownerDependencies,
	content *contentDependencies,
	instagramStorage *instagramStorageDependencies,
	scheduledAccountDeletion *scheduledposts.AccountDeletion,
	observer *observability.Observer,
	identityInvalidator ingestion.IdentityInvalidator,
	cfg Config,
	logger *slog.Logger,
) (*tapDependencies, error) {
	anonPDS, err := auth.NewAnonymousPDSClient(
		federated.directory,
		federated.pdsJSON,
		federated.boundary,
	)
	if err != nil {
		return nil, fmt.Errorf("anonymous PDS client: %w", err)
	}
	repositoryTracker, err := tap.NewAdminClient(
		cfg.TapWSURL,
		&http.Client{Timeout: 5 * time.Second},
	)
	if err != nil {
		return nil, fmt.Errorf("build Tap admin client: %w", err)
	}
	notificationActorDeletion := notifications.NewActorDeletionService(pool)
	profileDeletion := &profileMembershipDeletion{
		notifications: notificationActorDeletion,
		scheduled:     scheduledAccountDeletion,
		instagram:     instagramStorage.privateData,
		now:           time.Now,
	}
	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(
		pool,
		logger,
		observer,
		content.notificationLifecycle,
		profileDeletion,
	)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		return nil, fmt.Errorf("tap ingestion store: %w", err)
	}
	departureParticipant := scheduledAccountDeletion.DepartureParticipant()
	profileDepartureParticipant := authCapability.sessionLifecycle.OwnerTransitionParticipant(
		nil,
		composeTransitionParticipants(
			owners.deletionStore.ProfileDepartureParticipant(),
			departureParticipant,
			store.PDSAttemptDepartureParticipant(),
		),
	)
	profileParticipant := func(
		ctx context.Context,
		tx pgx.Tx,
		before ownerlifecycle.Lifecycle,
		after ownerlifecycle.Lifecycle,
	) error {
		if after.State == ownerlifecycle.StateActive {
			return nil
		}
		return profileDepartureParticipant(ctx, tx, before, after)
	}
	terminalAuthParticipant := authCapability.sessionLifecycle.OwnerTransitionParticipant(
		nil,
		departureParticipant,
	)
	terminalPDSAttemptParticipant := store.PDSAttemptTerminalParticipant()
	terminalParticipant := func(
		ctx context.Context,
		tx pgx.Tx,
		before *ownerlifecycle.Lifecycle,
		terminal ownerlifecycle.Lifecycle,
	) error {
		prior := ownerlifecycle.Lifecycle{Owner: terminal.Owner}
		if before != nil {
			prior = *before
		}
		if err := terminalAuthParticipant(ctx, tx, prior, terminal); err != nil {
			return err
		}
		return terminalPDSAttemptParticipant(ctx, tx, before, terminal)
	}
	service, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: store, Lifecycles: owners.lifecycles,
		ProfileParticipant: profileParticipant, TerminalParticipant: terminalParticipant,
		TerminalCommitTimeout: cfg.tapTerminalCommitTimeout(),
		IdentityInvalidator:   identityInvalidator,
	})
	if err != nil {
		return nil, fmt.Errorf("tap ingestion service: %w", err)
	}
	projectionWorker, err := ingestion.NewProjectionWorker(ingestion.ProjectionWorkerConfig{
		Store: store, Projector: dispatcher.Project,
		WorkerID: "appview-tap-projection", PollInterval: cfg.TapProjectionPollInterval,
		LeaseDuration: cfg.TapProjectionLeaseDuration, BatchSize: cfg.TapProjectionBatchSize,
		BackoffMin: cfg.TapProjectionBackoffMin, BackoffMax: cfg.TapProjectionBackoffMax,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("tap projection worker: %w", err)
	}
	repositoryWorker, err := ingestion.NewRepositoryWorker(ingestion.RepositoryWorkerConfig{
		Store:    store,
		Handler:  newTapRepositoryJobHandler(store, service, repositoryTracker, anonPDS),
		WorkerID: "appview-tap-repository", PollInterval: cfg.TapRepositoryPollInterval,
		LeaseDuration: cfg.TapRepositoryLeaseDuration, BatchSize: cfg.TapRepositoryBatchSize,
		BackoffMin: cfg.TapRepositoryBackoffMin, BackoffMax: cfg.TapRepositoryBackoffMax,
		Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("tap repository worker: %w", err)
	}
	quarantineWorker, err := ingestion.NewQuarantineReplayWorker(ingestion.QuarantineReplayWorkerConfig{
		Store: store,
		Handler: func(ctx context.Context, envelope []byte) (tap.Outcome, error) {
			return store.ReplayEnvelope(ctx, envelope, service)
		},
		WorkerID: "appview-tap-quarantine", PollInterval: cfg.TapQuarantinePollInterval,
		LeaseDuration:    cfg.TapQuarantineLeaseDuration,
		OperationTimeout: cfg.TapQuarantineOperationTimeout,
		BatchSize:        cfg.TapQuarantineBatchSize, Logger: logger,
	})
	if err != nil {
		return nil, fmt.Errorf("tap quarantine replay worker: %w", err)
	}
	consumer := tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: cfg.TapWSURL, Ingestor: service,
		AckTimeout: cfg.tapIngestionTimeout(), ReconnectMax: cfg.TapReconnectMax,
		Logger: logger, Observer: observer,
	})
	return &tapDependencies{
		repositoryTracker: repositoryTracker,
		projectionWorker:  projectionWorker,
		repositoryWorker:  repositoryWorker,
		quarantineWorker:  quarantineWorker,
		consumer:          consumer,
	}, nil
}
