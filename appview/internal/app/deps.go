package app

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/push"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/tap"
)

// Deps is the fully-wired set of dependencies for one Craftsky App View
// process. NewDevDeps and NewProdDeps build it; cmd/appview and cmd/cli
// both consume it.
//
// Deps is passed into NewServer, routes.AddRoutes, and CLI subcommand
// entry points — it is never passed into an individual HTTP handler.
// Handler factories in internal/api take only the specific dependencies
// they use.
type Deps struct {
	Config                      Config
	Logger                      *slog.Logger
	DB                          *pgxpool.Pool
	AuthService                 auth.AuthService
	RateLimiter                 *middleware.LocalRateLimiter
	Observability               *observability.Observer
	AccountDeletion             accountdeletion.Service
	AccountDeletionOAuth        auth.AccountDeletionOAuthCallbacks
	AccountDeletionPendingLogin auth.AccountDeletionPendingLoginPolicy
	AccountDeletionWorker       *accountdeletion.Worker
	AccountDeletionIntentExpiry *accountdeletion.IntentExpiryProcessor

	// OAuth subsystem.
	OAuthApp             *oauth.ClientApp
	OAuthArtifacts       auth.ClientArtifacts
	OAuthStore           *auth.PostgresAuthStore
	OAuthFlow            auth.OAuthFlowCoordinator
	HandoffCoordinator   auth.HandoffCoordinator
	SessionLifecycle     *auth.SessionLifecycleService
	OAuthRevocation      *auth.OAuthRevocationProcessor
	AuthAuxiliaryCleanup *auth.AuxiliaryCleanupProcessor
	SessionExpiry        *auth.SessionExpiryProcessor
	TerminalPurge        *ownerlifecycle.TerminalPurgeProcessor
	IdentityCacheRefresh *api.IdentityCacheRefreshProcessor
	CraftskySessionStore *auth.CraftskySessionStore
	OwnerLifecycles      *ownerlifecycle.Store
	OwnerFence           *ownerlifecycle.Fencer
	NewPendingPDSClient  auth.PendingOnboardingPDSClientFactory
	OnboardingProfile    auth.OnboardingProfileWriter
	LoginCompleteURL     string
	DeletionCompleteURL  string

	// Identity resolution for /v1/whoami. Typed as the interface
	// (not the concrete struct) so route tests can inject a stub
	// without constructing an identity.Directory.
	HandleResolver api.HandleResolver
	// AuthoritativeHandleResolver bypasses Indigo's process cache and is used
	// only where a mutable handle selects a durable/security-sensitive target.
	AuthoritativeHandleResolver api.HandleResolver

	Consumer            tap.Consumer
	TapProjectionWorker *ingestion.ProjectionWorker
	TapRepositoryWorker *ingestion.RepositoryWorker
	TapQuarantineWorker *ingestion.QuarantineReplayWorker
	PushDispatcher      *push.Dispatcher

	// Instagram migration is private AppView data. Verification can remain
	// disabled while the membership/store dependencies continue to expose
	// retained local state and privacy controls.
	InstagramMembership     *instagram.MembershipStore
	InstagramRateLimiter    *instagram.PostgresRateLimiter
	InstagramVerification   *instagram.VerificationService
	InstagramWebhook        http.Handler
	InstagramWebhookWorker  *instagram.WebhookWorker
	InstagramPrivateData    *instagram.PrivateDataService
	InstagramReconciliation *instagram.ReconciliationWorker
	InstagramSuggestions    *instagram.SuggestionService
	InstagramRetention      *instagram.RetentionService
	InstagramAccount        *instagram.AccountStore
	InstagramImports        *instagram.ImportService
	InstagramRestoration    *instagram.ReconciliationTrigger

	// ProfileStore serves the /v1/profiles endpoints.
	ProfileStore *api.ProfileStore
	// ProfileCustomisationStore owns AppView-only public appearance choices.
	ProfileCustomisationStore *api.ProfileCustomisationStore
	// IdentityCacheUpdater upserts authenticated users' current handles after profile initialization.
	IdentityCacheUpdater auth.IdentityCacheUpdater
	// RepositoryTracker requests ordinary Tap tracking/backfill on membership and OAuth initialization.
	RepositoryTracker auth.RepositoryTracker
	// FollowStore serves follow graph read/write operations for /v1/profiles/*/follows.
	FollowStore *api.FollowStore
	// RelationshipStore owns private mutes and reads the Tap-owned block projection.
	RelationshipStore *relationships.Store
	// RelationshipMutations is the narrow handler-facing mutation service.
	RelationshipMutations api.RelationshipMutationService
	// ReportStore persists AppView-private moderation report intake.
	ReportStore *api.ReportStore
	// ReportForwarder prepares future report forwarding metadata without live PDS/Ozone submission.
	ReportForwarder api.ReportForwarder
	// ModerationStore persists dev/test synthetic moderation outputs for enforcement.
	ModerationStore *api.ModerationStore
	// LanguagePreferences owns private per-account posting and content-language preferences.
	LanguagePreferences *languages.Store
	// NewPDSEffects is the only ordinary authenticated PDS mutation
	// capability exposed to request and background-work handlers. It persists
	// deterministic effect intent before crossing the remote boundary.
	NewPDSEffects pdseffects.ExecutorFactory

	// Scheduled posts and their private staged media are AppView-owned data.
	ScheduledPosts           *scheduledposts.Store
	ScheduledMedia           *scheduledposts.PrivateMediaService
	ScheduledCleanup         *scheduledposts.CleanupProcessor
	ScheduledPublisher       *scheduledposts.Worker
	ScheduledManualPublisher scheduledposts.ManualPublisher
}

type instagramRelationshipSafetyProvider struct {
	store *relationships.Store
}

func (p instagramRelationshipSafetyProvider) RelationshipSafety(
	ctx context.Context,
	importerDID syntax.DID,
	targetDID syntax.DID,
) (instagram.RelationshipSafetyFacts, error) {
	state, err := p.store.State(ctx, importerDID, targetDID)
	if err != nil {
		return instagram.RelationshipSafetyFacts{}, err
	}
	return instagram.RelationshipSafetyFacts{
		Available:            true,
		ImporterBlocksTarget: state.Blocking,
		TargetBlocksImporter: state.BlockedBy,
		ImporterMutesTarget:  state.Muted,
	}, nil
}

// NewDevDeps wires the dev variant: debug-level logger and StackedAuthService.
// Real OAuth tokens take precedence; the route middleware applies the explicit
// local/remote dev-authorization policy before making X-Dev-DID available as a
// fallback.
func NewDevDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	deps, cleanup, err := newDeps(ctx, cfg, slog.LevelDebug)
	if err != nil {
		return nil, nil, err
	}
	deps.AuthService = &auth.StackedAuthService{
		Real: &auth.CraftskyAuthService{Store: deps.CraftskySessionStore},
	}
	return deps, cleanup, nil
}

// NewProdDeps wires the prod variant: info-level logger, CraftskyAuthService
// backed by craftsky_sessions, full OAuth subsystem.
func NewProdDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	deps, cleanup, err := newDeps(ctx, cfg, slog.LevelInfo)
	if err != nil {
		return nil, nil, err
	}
	deps.AuthService = &auth.CraftskyAuthService{Store: deps.CraftskySessionStore}
	return deps, cleanup, nil
}

// newDeps is the shared core of NewDevDeps and NewProdDeps. AuthService is
// left nil — the caller assigns it based on env.
func newDeps(ctx context.Context, cfg Config, level slog.Level) (
	result *Deps,
	resultCleanup func(),
	resultErr error,
) {
	resources := &dependencyCleanup{}
	defer func() {
		if resultErr != nil {
			resources.close()
		}
	}()
	foundation, err := newFoundationDependencies(ctx, cfg, level, resources)
	if err != nil {
		return nil, nil, err
	}
	oauthArtifacts := foundation.oauthArtifacts
	handoffReceiptKey := foundation.handoffReceiptKey
	logger := foundation.logger
	pool := foundation.pool
	federated := foundation.federated
	owners, err := newOwnerDependencies(pool, cfg)
	if err != nil {
		return nil, nil, err
	}
	ownerFence := owners.fence
	ownerLifecycles := owners.lifecycles
	onboardingEffects := owners.onboardingEffects
	scheduledStorage, err := newScheduledStorageDependencies(ctx, pool, cfg)
	if err != nil {
		return nil, nil, err
	}
	scheduledStore := scheduledStorage.store

	authCapability, err := newAuthDependencies(
		pool, federated, owners, oauthArtifacts, handoffReceiptKey, cfg, logger,
	)
	if err != nil {
		return nil, nil, err
	}
	oauthStore := authCapability.store
	oauthApp := authCapability.app
	craftskyStore := authCapability.craftskySessions
	oauthFlow := authCapability.flow
	handoffs := authCapability.handoffs
	sessionLifecycle := authCapability.sessionLifecycle
	admission := newAdmissionDependencies(cfg, logger)

	// Identity and every metadata/PDS request share one hardened outbound
	// boundary. There is no process-default HTTP client fallback.
	observer := newObservabilityDependencies(cfg, logger, resources)
	identities := newIdentityResolutionDependencies(
		federated.directory, federated.authoritativeDirectory, pool, cfg.Env, observer,
	)
	content := newContentDependencies(pool, observer)
	notificationStore := content.posts
	authorityWorkers, err := newAuthorityWorkers(
		pool, authCapability, owners, notificationStore, observer, cfg,
	)
	if err != nil {
		return nil, nil, err
	}
	oauthRevocation := authorityWorkers.oauthRevocation
	authAuxiliaryCleanup := authorityWorkers.auxiliaryCleanup
	sessionExpiry := authorityWorkers.sessionExpiry
	terminalPurge := authorityWorkers.terminalPurge
	scheduledLifecycle, err := newScheduledLifecycleDependencies(
		pool, scheduledStorage, owners, observer, cfg,
	)
	if err != nil {
		return nil, nil, err
	}
	relationshipStore := content.relationships
	languagePreferences := content.languages
	instagramStorage, err := newInstagramStorageDependencies(pool, owners, content, cfg)
	if err != nil {
		return nil, nil, err
	}
	instagramMembership := instagramStorage.membership
	instagramRateLimiter := instagramStorage.rateLimiter
	instagramPrivateData := instagramStorage.privateData
	scheduledAccountDeletion := scheduledLifecycle.accountDeletion
	scheduledDepartureParticipant := scheduledLifecycle.departureParticipant
	tapCapability, err := newTapDependencies(
		pool,
		federated,
		authCapability,
		owners,
		content,
		instagramStorage,
		scheduledAccountDeletion,
		observer,
		identities.invalidator,
		cfg,
		logger,
	)
	if err != nil {
		return nil, nil, err
	}
	instagramRestoration := instagramStorage.restoration
	moderationStore := instagramStorage.moderationStore
	loginCompleteURL := resolveOriginPath(cfg.VerifiedLinkOrigin, "/auth/complete")
	deletionCompleteURL := resolveOriginPath(cfg.VerifiedLinkOrigin, "/account-deletion/reauth-complete")
	pdsEffects, err := newPDSEffectDependencies(
		authCapability, federated, owners, observer, cfg,
	)
	if err != nil {
		return nil, nil, err
	}

	deps := &Deps{
		Config:                      cfg,
		Logger:                      logger,
		DB:                          pool,
		RateLimiter:                 admission.rateLimiter,
		Observability:               observer,
		OAuthApp:                    oauthApp,
		OAuthArtifacts:              oauthArtifacts,
		OAuthStore:                  oauthStore,
		OAuthFlow:                   oauthFlow,
		HandoffCoordinator:          handoffs,
		SessionLifecycle:            sessionLifecycle,
		OAuthRevocation:             oauthRevocation,
		AuthAuxiliaryCleanup:        authAuxiliaryCleanup,
		SessionExpiry:               sessionExpiry,
		TerminalPurge:               terminalPurge,
		CraftskySessionStore:        craftskyStore,
		OwnerLifecycles:             ownerLifecycles,
		OwnerFence:                  ownerFence,
		OnboardingProfile:           onboardingProfileEffectAdapter{executor: onboardingEffects},
		NewPendingPDSClient:         pdsEffects.pending,
		NewPDSEffects:               pdsEffects.ordinary,
		LoginCompleteURL:            loginCompleteURL.String(),
		DeletionCompleteURL:         deletionCompleteURL.String(),
		RepositoryTracker:           tapCapability.repositoryTracker,
		HandleResolver:              identities.cached,
		AuthoritativeHandleResolver: identities.authoritative,
		TapProjectionWorker:         tapCapability.projectionWorker,
		TapRepositoryWorker:         tapCapability.repositoryWorker,
		TapQuarantineWorker:         tapCapability.quarantineWorker,
		Consumer:                    tapCapability.consumer,
		RelationshipStore:           relationshipStore,
		LanguagePreferences:         languagePreferences,
		ProfileStore:                content.profiles,
		ProfileCustomisationStore:   content.profileCustomisation,
		FollowStore:                 content.follows,
		ReportStore:                 content.reports,
		ReportForwarder:             content.reportForwarder,
		InstagramMembership:         instagramMembership,
		InstagramRateLimiter:        instagramRateLimiter,
		InstagramPrivateData:        instagramPrivateData,
		InstagramRestoration:        instagramRestoration,
		ModerationStore:             moderationStore,
		ScheduledPosts:              scheduledStore,
		ScheduledMedia:              scheduledLifecycle.media,
		ScheduledCleanup:            scheduledLifecycle.cleanup,
	}
	deps.PushDispatcher, err = newPushDependencies(ctx, pool, owners, observer, cfg)
	if err != nil {
		return nil, nil, err
	}

	instagramRuntime, err := newInstagramRuntimeDependencies(
		ctx, pool, instagramStorage, owners, pdsEffects, cfg, logger,
	)
	if err != nil {
		return nil, nil, err
	}
	deps.InstagramVerification = instagramRuntime.verification
	deps.InstagramWebhook = instagramRuntime.webhook
	deps.InstagramWebhookWorker = instagramRuntime.webhookWorker
	deps.InstagramImports = instagramRuntime.imports
	deps.InstagramAccount = instagramRuntime.account
	deps.InstagramReconciliation = instagramRuntime.reconciliation
	deps.InstagramSuggestions = instagramRuntime.suggestions
	deps.InstagramRetention = instagramRuntime.retention
	contentRuntime, err := newContentRuntimeDependencies(
		pool, deps.AuthoritativeHandleResolver, content, pdsEffects, instagramStorage,
		observer, identities.invalidator, cfg,
	)
	if err != nil {
		return nil, nil, err
	}
	deps.IdentityCacheUpdater = contentRuntime.identityCache
	deps.IdentityCacheRefresh = contentRuntime.identityRefresh
	deps.RelationshipMutations = contentRuntime.relationshipMutations
	deletion, err := newAccountDeletionDependencies(
		pool,
		authCapability,
		owners,
		federated,
		instagramPrivateData,
		scheduledAccountDeletion,
		scheduledDepartureParticipant,
		deps.AuthoritativeHandleResolver,
		observer,
		cfg,
		logger,
	)
	if err != nil {
		return nil, nil, err
	}
	deps.AccountDeletion = deletion.service
	deps.AccountDeletionOAuth = deletion.service
	deps.AccountDeletionPendingLogin = deletion.service
	deps.AccountDeletionWorker = deletion.worker
	deps.AccountDeletionIntentExpiry = deletion.intentExpiry
	scheduledPublication, err := newScheduledPublicationDependencies(
		pool, scheduledStorage, content, pdsEffects, observer, cfg,
	)
	if err != nil {
		return nil, nil, err
	}
	deps.ScheduledManualPublisher = scheduledPublication.manual
	deps.ScheduledPublisher = scheduledPublication.worker
	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			resources.close()
			deps.Logger.Info("shutdown: db pool closed")
		})
	}

	// Startup log lines per spec. AC #1/#2 check for presence/absence.
	if cfg.Env == EnvDev {
		logger.Debug("log level", slog.String("level", "debug"))
	}
	logger.Info("deps initialised", slog.String("env", string(cfg.Env)))

	return deps, cleanup, nil
}

func buildOAuthArtifacts(deployment OAuthDeployment) (auth.ClientArtifacts, error) {
	mode := auth.ClientMode(deployment.Mode)
	return auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:            mode,
		ClientID:        deployment.ClientID,
		CallbackURL:     deployment.CallbackURL,
		JWKSURL:         deployment.JWKSURL,
		ClientSecretKey: deployment.ClientSecretKey.Reveal(),
		ClientKeyID:     deployment.ClientKeyID,
		Scopes:          append([]string(nil), deployment.Scopes...),
	})
}

func newTransactionalIndexerDispatcherWithActorDeletion(
	pool *pgxpool.Pool,
	logger *slog.Logger,
	observer index.RelationshipObserver,
	lifecycle notifications.Lifecycle,
	actorDeletion notifications.ActorDeletion,
) *index.TransactionalDispatcher {
	if lifecycle == nil {
		lifecycle = notifications.NoopLifecycle{}
	}
	if actorDeletion == nil {
		actorDeletion = notifications.NoopActorDeletion{}
	}
	transactional := index.NewTransactionalDispatcher()
	blueskyIdx := index.NewBlueskyProfile(pool)
	profile := index.NewTransactionalCraftskyProfile(pool, logger, actorDeletion)
	post := index.NewCraftskyPost(pool, logger, lifecycle)
	like := index.NewCraftskyLike(pool, logger, lifecycle)
	repost := index.NewCraftskyRepost(pool, logger, lifecycle)
	follow := index.NewBlueskyFollow(pool, lifecycle)
	block := index.NewBlueskyBlock(pool, observer)
	registrations := []struct {
		collection syntax.NSID
		indexer    index.TransactionalIndexer
	}{
		{collection: "social.craftsky.actor.profile", indexer: profile},
		{collection: "social.craftsky.feed.post", indexer: post},
		{collection: "social.craftsky.feed.like", indexer: like},
		{collection: "social.craftsky.feed.repost", indexer: repost},
		{collection: "app.bsky.actor.profile", indexer: blueskyIdx},
		{collection: "app.bsky.graph.follow", indexer: follow},
		{collection: "app.bsky.graph.block", indexer: block},
	}
	for _, registration := range registrations {
		transactional.Register(registration.collection, registration.indexer)
	}
	return transactional
}
