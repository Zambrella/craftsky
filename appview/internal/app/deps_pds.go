package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ingestion"
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
	repositoryJobs repositoryJobEnqueuer,
	cfg Config,
	logger *slog.Logger,
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
	ordinary = withDeleteReconciliation(ordinary, repositoryJobs, logger)
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

type repositoryJobEnqueuer interface {
	EnqueueRepositoryJob(context.Context, syntax.DID, ingestion.RepositoryJobKind) error
}

type deleteReconcilingExecutor struct {
	pdseffects.EffectExecutor
	repositoryJobs repositoryJobEnqueuer
	logger         *slog.Logger
}

func withDeleteReconciliation(
	factory pdseffects.ExecutorFactory,
	repositoryJobs repositoryJobEnqueuer,
	logger *slog.Logger,
) pdseffects.ExecutorFactory {
	return func(ctx context.Context, owner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
		executor, err := factory(ctx, owner, sessionID)
		if err != nil {
			return nil, err
		}
		return &deleteReconcilingExecutor{
			EffectExecutor: executor,
			repositoryJobs: repositoryJobs,
			logger:         logger,
		}, nil
	}
}

func (executor *deleteReconcilingExecutor) DeleteRecord(
	ctx context.Context,
	request pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	result, err := executor.EffectExecutor.DeleteRecord(ctx, request)
	if err != nil {
		return result, err
	}
	if err := executor.repositoryJobs.EnqueueRepositoryJob(
		ctx,
		request.Owner,
		ingestion.RepositoryJobPDSReconcile,
	); err != nil {
		executor.logger.Error("queue PDS reconciliation after record delete",
			slog.String("owner_did", request.Owner.String()),
			slog.String("collection", request.Collection.String()),
			slog.String("rkey", request.Rkey.String()),
			slog.Any("error", err),
		)
	}
	return result, nil
}
