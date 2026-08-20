package app

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/scheduledposts"
)

// scheduledStorageDependencies owns the private draft repository and object
// transport. Lifecycle-aware services and workers are composed later after the
// shared observer and owner authority exist.
type scheduledStorageDependencies struct {
	store   *scheduledposts.Store
	objects *scheduledposts.S3ObjectStore
}

type scheduledLifecycleDependencies struct {
	media                *scheduledposts.PrivateMediaService
	cleanup              *scheduledposts.CleanupProcessor
	accountDeletion      *scheduledposts.AccountDeletion
	departureParticipant ownerlifecycle.TransitionParticipant
}

type scheduledPublicationDependencies struct {
	manual scheduledposts.ManualPublisher
	worker *scheduledposts.Worker
}

// newScheduledLifecycleDependencies owns private-media admission, durable
// cleanup, and the bounded lifecycle participant. Publication is constructed
// separately after the authenticated effect boundary is available.
func newScheduledLifecycleDependencies(
	pool *pgxpool.Pool,
	storage *scheduledStorageDependencies,
	owners *ownerDependencies,
	observer *observability.Observer,
	cfg Config,
) (*scheduledLifecycleDependencies, error) {
	storage.store.SetOperationalObserver(observer)
	cleanup, err := scheduledposts.NewCleanupProcessor(scheduledposts.CleanupProcessorOptions{
		Store: storage.store, Objects: storage.objects, Now: time.Now,
		OwnerFence: owners.fence, Observer: observer,
	})
	if err != nil {
		return nil, fmt.Errorf("scheduled cleanup processor: %w", err)
	}
	accountDeletion := scheduledposts.NewAccountDeletion(pool, time.Now, owners.fence)
	return &scheduledLifecycleDependencies{
		media: scheduledposts.NewPrivateMediaService(
			storage.store,
			storage.objects,
			scheduledposts.PrivateMediaServiceOptions{
				Lifecycle:  owners.lifecycles,
				PutTimeout: cfg.ScheduledMediaPutTimeout,
				// No finite MinIO/S3 settlement bound has been proven, so exact-key
				// safety work remains reconciling until absence is authoritative.
				TestedSettlementBound: 0,
				SettlementMargin:      0,
			},
		),
		cleanup:              cleanup,
		accountDeletion:      accountDeletion,
		departureParticipant: accountDeletion.DepartureParticipant(),
	}, nil
}

func newScheduledPublicationDependencies(
	pool *pgxpool.Pool,
	storage *scheduledStorageDependencies,
	content *contentDependencies,
	pdsEffects *pdsEffectDependencies,
	observer *observability.Observer,
	cfg Config,
) (*scheduledPublicationDependencies, error) {
	processor, err := scheduledposts.NewPublicationProcessor(scheduledposts.PublicationProcessorOptions{
		Store:         storage.store,
		Sessions:      auth.NewBackgroundSessionSelector(pool),
		NewEffects:    pdsEffects.guarded,
		Objects:       storage.objects,
		Now:           time.Now,
		MaxMediaBytes: cfg.MaxImageUploadBytes,
		Observer:      observer,
		Validate: func(ctx context.Context, owner syntax.DID, payload scheduledposts.Payload) error {
			return api.ValidateScheduledPublication(
				ctx,
				content.posts,
				owner,
				payload,
				api.MediaLimits{
					MaxPostImages:       api.DefaultMaxPostImages,
					MaxImageUploadBytes: cfg.MaxImageUploadBytes,
				},
			)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("scheduled publication processor: %w", err)
	}
	manual, err := scheduledposts.NewManualPublicationService(storage.store, processor)
	if err != nil {
		return nil, fmt.Errorf("manual scheduled publisher: %w", err)
	}
	worker, err := scheduledposts.NewWorker(scheduledposts.WorkerOptions{
		Store: storage.store, Processor: processor, Now: time.Now,
		Observer: observer,
	})
	if err != nil {
		return nil, fmt.Errorf("scheduled publication worker: %w", err)
	}
	return &scheduledPublicationDependencies{manual: manual, worker: worker}, nil
}

func newScheduledStorageDependencies(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg Config,
) (*scheduledStorageDependencies, error) {
	store := scheduledposts.NewStore(pool)
	objects, err := scheduledposts.NewS3ObjectStore(ctx, cfg.ScheduledPostsS3)
	if err != nil {
		return nil, fmt.Errorf("scheduled private object store: %w", err)
	}
	if err := objects.Check(ctx); err != nil {
		return nil, fmt.Errorf("scheduled private object store check: %w", err)
	}
	return &scheduledStorageDependencies{store: store, objects: objects}, nil
}
