package app

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/relationships"
)

// contentDependencies groups AppView-owned read models and private mutation
// stores. Remote PDS capabilities and integration clients are deliberately not
// part of this bundle.
type contentDependencies struct {
	posts                 *api.PostStore
	notificationLifecycle *notifications.Service
	relationships         *relationships.Store
	languages             *languages.Store
	profiles              *api.ProfileStore
	profileCustomisation  *api.ProfileCustomisationStore
	follows               *api.FollowStore
	reports               *api.ReportStore
	reportForwarder       api.ReportForwarder
}

type contentRuntimeDependencies struct {
	identityCache         auth.IdentityCacheUpdater
	identityRefresh       *api.IdentityCacheRefreshProcessor
	relationshipMutations api.RelationshipMutationService
}

func newContentDependencies(
	pool *pgxpool.Pool,
	observer *observability.Observer,
) *contentDependencies {
	return &contentDependencies{
		posts:                 api.NewPostStore(pool, observer),
		notificationLifecycle: notifications.NewService(observer),
		relationships:         relationships.NewStore(pool),
		languages:             languages.NewStore(pool),
		profiles:              api.NewProfileStore(pool),
		profileCustomisation:  api.NewProfileCustomisationStore(pool),
		follows:               api.NewFollowStore(pool),
		reports:               api.NewReportStore(pool),
		reportForwarder:       api.NewPlaceholderReportForwarder(time.Now),
	}
}

func newContentRuntimeDependencies(
	pool *pgxpool.Pool,
	handleResolver api.HandleResolver,
	content *contentDependencies,
	pdsEffects *pdsEffectDependencies,
	instagramStorage *instagramStorageDependencies,
	observer *observability.Observer,
	identityInvalidator api.IdentityInvalidator,
	cfg Config,
) (*contentRuntimeDependencies, error) {
	identityStore := api.NewIdentityCacheStore(pool, observer)
	identityRefresh, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
		Store: identityStore, Resolver: handleResolver,
		BatchSize:           cfg.IdentityCacheRefreshBatchSize,
		OperationTimeout:    cfg.IdentityCacheRefreshOperationTimeout,
		RetryDelay:          cfg.IdentityCacheRefreshRetryDelay,
		Now:                 time.Now,
		IdentityInvalidator: identityInvalidator,
	})
	if err != nil {
		return nil, fmt.Errorf("identity cache refresh processor: %w", err)
	}
	return &contentRuntimeDependencies{
		identityCache:   api.NewIdentityCacheService(pool, handleResolver, time.Now, observer),
		identityRefresh: identityRefresh,
		relationshipMutations: relationships.NewMutationServiceWithRestoration(
			content.relationships,
			pdsEffects.ordinary,
			time.Now,
			instagramStorage.restoration,
			observer,
		),
	}, nil
}
