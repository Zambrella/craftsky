package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/followwrite"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/integrations/instagrammeta"
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
	CraftskySessionStore *auth.CraftskySessionStore
	OwnerLifecycles      *ownerlifecycle.Store
	OwnerFence           *ownerlifecycle.Fencer
	NewPendingPDSClient  auth.PendingOnboardingPDSClientFactory
	LoginCompleteURL     string
	DeletionCompleteURL  string

	// Identity resolution for /v1/whoami. Typed as the interface
	// (not the concrete struct) so route tests can inject a stub
	// without constructing an identity.Directory.
	HandleResolver api.HandleResolver

	Consumer            tap.Consumer
	Indexer             index.Indexer
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
	// NewPDSClient produces a PDSClient bound to an OAuth session. Shared
	// by the OAuth callback's InitializeProfile step and the write-proxy
	// handlers (today PUT /v1/profiles/me).
	NewPDSClient auth.PDSClientFactory
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
func newDeps(ctx context.Context, cfg Config, level slog.Level) (*Deps, func(), error) {
	oauthArtifacts, err := buildOAuthArtifacts(cfg.OAuth)
	if err != nil {
		return nil, nil, fmt.Errorf("build oauth client artifacts: %w", err)
	}
	handoffReceiptKey, err := decodeHandoffReceiptKey(cfg.OAuthHandoffReceiptKey)
	if err != nil {
		return nil, nil, err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	// Third-party libs that reach for slog.Default should get our logger.
	slog.SetDefault(logger)

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("db connect: %w", err)
	}
	federated, err := newFederatedClients(cfg.FederatedHTTP)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	ownerFence, err := ownerlifecycle.NewFencer(pool, cfg.OwnerFenceAcquireTimeout)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("owner lifecycle fence: %w", err)
	}
	ownerLifecycles, err := ownerlifecycle.NewStore(pool, ownerFence, time.Now)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("owner lifecycle store: %w", err)
	}
	deletionStore := accountdeletion.NewStore(pool, time.Now)
	scheduledStore := scheduledposts.NewStore(pool)
	scheduledObjects, err := scheduledposts.NewS3ObjectStore(ctx, cfg.ScheduledPostsS3)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("scheduled private object store: %w", err)
	}
	if err := scheduledObjects.Check(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("scheduled private object store check: %w", err)
	}
	var scheduledCleanup *scheduledposts.CleanupProcessor

	oauthStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		SessionExpiry:                cfg.OAuthSessionAbsoluteLifetime,
		SessionInactivity:            cfg.CraftskySessionInactivity,
		SessionAbsoluteLifetime:      cfg.OAuthSessionAbsoluteLifetime,
		AuthRequestExpiry:            cfg.OAuthAuthRequestExpiry,
		PendingAuthRequestCapacity:   cfg.OAuthPendingAuthRequestCapacity,
		AuthRequestTerminalRetention: cfg.OAuthAuthRequestTerminalRetention,
		OwnerLifecycles:              ownerLifecycles,
		EndpointValidator:            federated.boundary,
		Logger:                       logger,
	})
	oauthApp := oauth.NewClientApp(&oauthArtifacts.Config, oauthStore)
	oauthApp.Client = federated.oauth
	oauthApp.Resolver.Client = federated.metadata
	oauthApp.Dir = federated.directory
	craftskyStore, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            cfg.CraftskySessionInactivity,
		ActivityWriteInterval: cfg.CraftskySessionActivityWriteInterval,
		RecoveryAuthorization: deletionStore,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("CraftSky session store: %w", err)
	}
	oauthFlow, err := auth.NewOAuthFlowService(auth.OAuthFlowServiceOptions{
		App: oauthApp, Store: oauthStore, Owners: ownerLifecycles,
		StartOperationTimeout:    cfg.OAuthLoginStartTimeout,
		CallbackOperationTimeout: cfg.OAuthCallbackOperationTimeout,
		DeletionRequests:         deletionStore,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("OAuth flow service: %w", err)
	}
	handoffs, err := auth.NewHandoffService(auth.HandoffServiceOptions{
		Pool: pool, Owners: ownerLifecycles, Sessions: craftskyStore,
		ExchangeTTL: cfg.OAuthHandoffExchangeTTL, ConfirmationTTL: cfg.OAuthHandoffConfirmationTTL,
		ReceiptKey: handoffReceiptKey, ReceiptKeyVersion: cfg.OAuthHandoffReceiptKeyVersion,
		Random: rand.Reader, Now: time.Now,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("OAuth handoff service: %w", err)
	}
	sessionLifecycle, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: ownerLifecycles, Sessions: craftskyStore,
		DeletionExemption: deletionStore,
		Now:               time.Now,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("session lifecycle service: %w", err)
	}
	oauthSessionCoordinator, err := auth.NewOAuthSessionCoordinator(auth.OAuthSessionCoordinatorOptions{
		App: oauthApp, Store: oauthStore, Owners: ownerLifecycles,
		OperationTimeout: cfg.OAuthSessionOperationTimeout,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("OAuth session coordinator: %w", err)
	}
	rateLimiter := middleware.NewLocalRateLimiter(cfg.RateLimits, time.Now)
	logger.Warn("rate limiter is process-local; run one AppView instance or configure shared/edge enforcement before horizontal scaling")

	// Identity and every metadata/PDS request share one hardened outbound
	// boundary. There is no process-default HTTP client fallback.
	identityDir := federated.directory
	anonPDS, err := auth.NewAnonymousPDSClient(identityDir, federated.pdsJSON, federated.boundary)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("anonymous PDS client: %w", err)
	}
	repositoryTracker, err := tap.NewAdminClient(cfg.TapWSURL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("build Tap admin client: %w", err)
	}
	observer := observability.New(observability.Config{
		Env: string(cfg.Env), Release: cfg.SentryRelease, LogsEnabled: cfg.SentryLogsEnabled, TracingEnabled: cfg.SentryTracingEnabled, TracesSampleRate: cfg.SentryTracesSampleRate, MetricsEnabled: cfg.SentryMetricsEnabled, TapTracingEnabled: cfg.SentryTapTracingEnabled, TapTracesSampleRate: cfg.SentryTapTracesSampleRate, SentryDSN: cfg.SentryDSN, Logger: logger,
	})
	notificationStore := api.NewPostStore(pool, observer)
	oauthCredentialRevoker, err := auth.NewIndigoOAuthCredentialRevoker(oauthApp, oauthStore)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("OAuth credential revoker: %w", err)
	}
	oauthRevocation, err := auth.NewOAuthRevocationProcessor(auth.OAuthRevocationProcessorOptions{
		Pool: pool, Revoker: oauthCredentialRevoker, Now: time.Now,
		BatchSize: cfg.OAuthRevocationBatchSize, LeaseDuration: cfg.OAuthRevocationLeaseDuration,
		OperationTimeout: cfg.OAuthRevocationOperationTimeout, MaxAttempts: cfg.OAuthRevocationMaxAttempts,
		BaseBackoff: cfg.OAuthRevocationBackoffMin, MaxBackoff: cfg.OAuthRevocationBackoffMax,
		MaxCredentialRetention: cfg.OAuthRevocationMaxCredentialRetention,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("OAuth revocation processor: %w", err)
	}
	authAuxiliaryCleanup, err := auth.NewAuxiliaryCleanupProcessor(auth.AuxiliaryCleanupProcessorOptions{
		Pool: pool, Cleaner: notificationStore, Now: time.Now,
		BatchSize: cfg.AuthAuxiliaryCleanupBatchSize, LeaseDuration: cfg.AuthAuxiliaryCleanupLeaseDuration,
		OperationTimeout: cfg.AuthAuxiliaryCleanupOperationTimeout, MaxAttempts: cfg.AuthAuxiliaryCleanupMaxAttempts,
		BaseBackoff: cfg.AuthAuxiliaryCleanupBackoffMin, MaxBackoff: cfg.AuthAuxiliaryCleanupBackoffMax,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("auth auxiliary cleanup processor: %w", err)
	}
	sessionExpiry, err := auth.NewSessionExpiryProcessor(auth.SessionExpiryProcessorOptions{
		Lifecycle: sessionLifecycle,
		BatchSize: cfg.SessionExpirySweepBatch,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("session expiry processor: %w", err)
	}
	terminalPurge, err := ownerlifecycle.NewTerminalPurgeProcessor(ownerlifecycle.TerminalPurgeProcessorConfig{
		Store:          ownerLifecycles,
		WorkerID:       "appview-terminal-purge",
		PollInterval:   cfg.TerminalPurgePollInterval,
		ComponentLimit: cfg.TerminalPurgeComponentLimit,
		RowBatchSize:   cfg.TerminalPurgeRowBatchSize,
		LeaseDuration:  cfg.TerminalPurgeLeaseDuration,
		RetryDelay:     cfg.TerminalPurgeRetryDelay,
		Observer:       observer,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("terminal purge processor: %w", err)
	}
	scheduledStore.SetOperationalObserver(observer)
	scheduledCleanup, err = scheduledposts.NewCleanupProcessor(scheduledposts.CleanupProcessorOptions{
		Store: scheduledStore, Objects: scheduledObjects, Now: time.Now,
		OwnerFence: ownerFence, Observer: observer,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("scheduled cleanup processor: %w", err)
	}
	notificationLifecycle := notifications.NewService(observer)
	relationshipStore := relationships.NewStore(pool)
	languagePreferences := languages.NewStore(pool)
	suggestionPolicy := instagram.NewPostgresInstagramSuggestionEligibilityPolicy(
		pool,
		instagramRelationshipSafetyProvider{store: relationshipStore},
		time.Now,
	)
	instagramMembership := instagram.NewMembershipStore(pool)
	var instagramRateLimiter *instagram.PostgresRateLimiter
	if cfg.InstagramData.Available() {
		instagramRateLimiter, err = instagram.NewPostgresRateLimiter(pool, cfg.InstagramData.HMACKey(), time.Now)
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram persistent rate limiter: %w", err)
		}
	}
	instagramPrivateData := instagram.NewPrivateDataService(pool, instagramRateLimiter, time.Now)
	notificationActorDeletion := notifications.NewActorDeletionService(pool)
	scheduledAccountDeletion := scheduledposts.NewAccountDeletion(pool, time.Now, ownerFence)
	scheduledDepartureParticipant := scheduledAccountDeletion.DepartureParticipant()
	profileDeletion := &profileMembershipDeletion{
		notifications: notificationActorDeletion,
		scheduled:     scheduledAccountDeletion,
		instagram:     instagramPrivateData,
		now:           time.Now,
	}
	dispatchers := newIndexerDispatcherBundleWithActorDeletion(
		pool,
		anonPDS,
		logger,
		repositoryTracker,
		observer,
		notificationLifecycle,
		profileDeletion,
	)
	dispatcher := dispatchers.legacy
	ingestionStore, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Tap ingestion store: %w", err)
	}
	profileDepartureParticipant := sessionLifecycle.OwnerTransitionParticipant(
		nil,
		composeTransitionParticipants(
			deletionStore.ProfileDepartureParticipant(),
			scheduledDepartureParticipant,
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
	terminalAuthParticipant := sessionLifecycle.OwnerTransitionParticipant(
		nil,
		scheduledDepartureParticipant,
	)
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
		return terminalAuthParticipant(ctx, tx, prior, terminal)
	}
	ingestionService, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: ingestionStore, Lifecycles: ownerLifecycles,
		ProfileParticipant: profileParticipant, TerminalParticipant: terminalParticipant,
		TerminalComponents: terminalPurgeCatalogue(),
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Tap ingestion service: %w", err)
	}
	tapProjectionWorker, err := ingestion.NewProjectionWorker(ingestion.ProjectionWorkerConfig{
		Store: ingestionStore, Projector: dispatchers.transactional.Project,
		WorkerID: "appview-tap-projection", PollInterval: cfg.TapProjectionPollInterval,
		LeaseDuration: cfg.TapProjectionLeaseDuration, BatchSize: cfg.TapProjectionBatchSize,
		BackoffMin: cfg.TapProjectionBackoffMin, BackoffMax: cfg.TapProjectionBackoffMax,
		Logger: logger,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Tap projection worker: %w", err)
	}
	tapRepositoryWorker, err := ingestion.NewRepositoryWorker(ingestion.RepositoryWorkerConfig{
		Store:    ingestionStore,
		Handler:  newTapRepositoryJobHandler(ingestionStore, ingestionService, repositoryTracker, anonPDS),
		WorkerID: "appview-tap-repository", PollInterval: cfg.TapRepositoryPollInterval,
		LeaseDuration: cfg.TapRepositoryLeaseDuration, BatchSize: cfg.TapRepositoryBatchSize,
		BackoffMin: cfg.TapRepositoryBackoffMin, BackoffMax: cfg.TapRepositoryBackoffMax,
		Logger: logger,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Tap repository worker: %w", err)
	}
	tapQuarantineWorker, err := ingestion.NewQuarantineReplayWorker(ingestion.QuarantineReplayWorkerConfig{
		Store: ingestionStore,
		Handler: func(ctx context.Context, envelope []byte) (tap.Outcome, error) {
			return ingestionStore.ReplayEnvelope(ctx, envelope, ingestionService)
		},
		WorkerID: "appview-tap-quarantine", PollInterval: cfg.TapQuarantinePollInterval,
		LeaseDuration:    cfg.TapQuarantineLeaseDuration,
		OperationTimeout: cfg.TapQuarantineOperationTimeout,
		BatchSize:        cfg.TapQuarantineBatchSize, Logger: logger,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Tap quarantine replay worker: %w", err)
	}
	instagramRestoration := instagram.NewReconciliationTrigger(pool, time.Now)
	moderationRestoration, err := instagram.NewModerationRestorationRelay(pool, ownerLifecycles, time.Now)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("moderation restoration relay: %w", err)
	}
	moderationStore, err := api.NewModerationStore(pool, ownerLifecycles)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("moderation store: %w", err)
	}
	loginCompleteURL := resolveOriginPath(cfg.VerifiedLinkOrigin, "/auth/complete")
	deletionCompleteURL := resolveOriginPath(cfg.VerifiedLinkOrigin, "/account-deletion/reauth-complete")

	deps := &Deps{
		Config:               cfg,
		Logger:               logger,
		DB:                   pool,
		RateLimiter:          rateLimiter,
		Observability:        observer,
		OAuthApp:             oauthApp,
		OAuthArtifacts:       oauthArtifacts,
		OAuthStore:           oauthStore,
		OAuthFlow:            oauthFlow,
		HandoffCoordinator:   handoffs,
		SessionLifecycle:     sessionLifecycle,
		OAuthRevocation:      oauthRevocation,
		AuthAuxiliaryCleanup: authAuxiliaryCleanup,
		SessionExpiry:        sessionExpiry,
		TerminalPurge:        terminalPurge,
		CraftskySessionStore: craftskyStore,
		OwnerLifecycles:      ownerLifecycles,
		OwnerFence:           ownerFence,
		LoginCompleteURL:     loginCompleteURL.String(),
		DeletionCompleteURL:  deletionCompleteURL.String(),
		RepositoryTracker:    repositoryTracker,
		HandleResolver:       api.DirectoryHandleResolver{Directory: identityDir},
		Indexer:              dispatcher,
		TapProjectionWorker:  tapProjectionWorker,
		TapRepositoryWorker:  tapRepositoryWorker,
		TapQuarantineWorker:  tapQuarantineWorker,
		Consumer:             tap.NotImplemented{}, // temp, replaced below
		RelationshipStore:    relationshipStore,
		LanguagePreferences:  languagePreferences,
		InstagramMembership:  instagramMembership,
		InstagramRateLimiter: instagramRateLimiter,
		InstagramPrivateData: instagramPrivateData,
		InstagramRetention: instagram.NewRetentionService(
			pool,
			time.Now,
			instagram.RetentionServiceOptions{
				Restoration:        instagramRestoration,
				ModerationReceipts: moderationStore,
			},
		),
		InstagramRestoration: instagramRestoration,
		ModerationStore:      moderationStore,
		ScheduledPosts:       scheduledStore,
		ScheduledMedia: scheduledposts.NewPrivateMediaService(
			scheduledStore,
			scheduledObjects,
			scheduledposts.PrivateMediaServiceOptions{
				Lifecycle:  ownerLifecycles,
				PutTimeout: cfg.ScheduledMediaPutTimeout,
				// No finite MinIO/S3 server-side settlement guarantee is
				// assumed. Exact-key tombstones therefore remain eligible for
				// reconciliation until object absence is independently proven.
				TestedSettlementBound: 0,
				SettlementMargin:      0,
			},
		),
		ScheduledCleanup: scheduledCleanup,
	}
	deps.NewPendingPDSClient = func(ctx context.Context, attempt auth.CallbackAttempt) (auth.PDSClient, error) {
		stored, err := oauthStore.ResumePendingOnboardingSession(ctx, attempt)
		if err != nil {
			return nil, err
		}
		return federated.newPendingPDSClient(ctx, oauthApp.Config, stored.Data)
	}
	if cfg.PushEnabled {
		sender, err := push.NewFirebaseSender(ctx, cfg.FirebaseProjectID)
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("firebase messaging init: %w", err)
		}
		deps.PushDispatcher, err = push.NewDispatcherValidated(pool, sender, push.DispatcherOptions{
			BatchSize: cfg.PushBatchSize, Concurrency: cfg.PushConcurrency,
			LeaseDuration: cfg.PushLeaseDuration, SendTimeout: cfg.PushSendTimeout,
			FinalizationMargin: cfg.PushFinalizationMargin, Observer: observer,
			LifecycleFence: pushOwnerLifecycleFence{store: ownerLifecycles},
		})
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("push dispatcher init: %w", err)
		}
	}

	deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
		URL: cfg.TapWSURL, Ingestor: ingestionService,
		AckTimeout: cfg.TapAckTimeout, ReconnectMax: cfg.TapReconnectMax,
		Logger: logger, Observer: deps.Observability,
	})
	if cfg.Env == EnvDev {
		deps.HandleResolver = api.DevHandleResolver{
			Primary: api.DirectoryHandleResolver{Directory: identityDir},
			Pool:    pool,
		}
	}

	deps.ProfileStore = api.NewProfileStore(pool)
	deps.ProfileCustomisationStore = api.NewProfileCustomisationStore(pool)
	verificationStore := instagram.NewVerificationStore(pool)
	var challengeCodec *instagram.ChallengeCodec
	if cfg.InstagramData.Available() {
		challengeCodec, err = instagram.NewChallengeCodec(rand.Reader, cfg.InstagramData.HMACKey())
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram challenge codec: %w", err)
		}
	}
	deps.InstagramVerification, err = instagram.NewVerificationService(instagram.VerificationServiceOptions{
		Store:     verificationStore,
		Codec:     challengeCodec,
		TTL:       cfg.InstagramLimits.ChallengeTTL,
		DMURL:     cfg.InstagramMeta.DMURL(),
		HMACKey:   cfg.InstagramData.HMACKey(),
		Available: cfg.InstagramMeta.Enabled() && cfg.InstagramMeta.Configured(),
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("instagram verification service: %w", err)
	}
	if cfg.InstagramMeta.Enabled() && cfg.InstagramMeta.Configured() {
		if deps.InstagramRateLimiter == nil {
			pool.Close()
			return nil, nil, errors.New("instagram Meta integration requires the persistent rate limiter")
		}
		digests, digestErr := instagrammeta.NewDigestCodec(cfg.InstagramData.HMACKey(), instagram.CanonicalizeChallenge)
		if digestErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook digest codec: %w", digestErr)
		}
		reducer, reducerErr := instagrammeta.NewPayloadReducer(cfg.InstagramMeta.InstagramAccountID(), digests)
		if reducerErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook reducer: %w", reducerErr)
		}
		webhookRateLimiter, limiterErr := middleware.NewInstagramWebhookRateLimiter(
			deps.InstagramRateLimiter,
			cfg.InstagramDeployment.TrustedProxyCIDRs(),
			cfg.InstagramLimits.WebhookIPPerMinute,
			cfg.InstagramLimits.WebhookGlobalPerMinute,
			cfg.InstagramLimits.InvalidIPPer15Minutes,
		)
		if limiterErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook limiter: %w", limiterErr)
		}
		retryPolicy := instagram.WebhookRetryPolicy{
			MaxAttempts:      cfg.InstagramLimits.WorkerMaxAttempts,
			InitialBackoff:   cfg.InstagramLimits.WorkerBackoffInitial,
			MaxBackoff:       cfg.InstagramLimits.WorkerBackoffMax,
			MaxProcessingAge: cfg.InstagramLimits.WorkerMaxProcessingAge,
		}
		webhookStore, storeErr := instagram.NewWebhookStoreWithOptions(pool, instagram.WebhookStoreOptions{
			LeaseDuration: cfg.InstagramLimits.WorkerLeaseDuration,
			RetryPolicy:   retryPolicy,
		})
		if storeErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook store: %w", storeErr)
		}
		deps.InstagramWebhook, err = instagrammeta.NewWebhookHandler(instagrammeta.WebhookHandlerConfig{
			AppSecret:       []byte(cfg.InstagramMeta.AppSecret()),
			VerifyToken:     cfg.InstagramMeta.VerifyToken(),
			Reducer:         reducer,
			Sink:            webhookStore,
			Limiter:         webhookRateLimiter,
			BodyLimitBytes:  cfg.InstagramLimits.WebhookBodyLimitBytes,
			MaxEvents:       cfg.InstagramLimits.WebhookMaxEvents,
			Now:             time.Now,
			Logger:          deps.Logger,
			UnsafeDebugLogs: cfg.UnsafeLogInstagramWebhookBodies,
		})
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook handler: %w", err)
		}
		baseURL := cfg.InstagramMeta.APIBaseURL()
		metaClient, clientErr := instagrammeta.NewHTTPClient(instagrammeta.HTTPClientConfig{
			HTTPClient:        &http.Client{},
			BaseURL:           baseURL.String(),
			APIVersion:        cfg.InstagramMeta.APIVersion(),
			AccessToken:       cfg.InstagramMeta.AccessToken(),
			OfficialAccountID: cfg.InstagramMeta.InstagramAccountID(),
			RequestTimeout:    cfg.InstagramLimits.MetaHTTPTimeout,
			ResponseLimit:     cfg.InstagramLimits.MetaResponseLimitBytes,
			MaxConcurrent:     cfg.InstagramLimits.MetaLookupConcurrency,
		})
		if clientErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram Meta client: %w", clientErr)
		}
		redeemer, redeemerErr := instagram.NewVerificationWebhookRedeemer(verificationStore)
		if redeemerErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook redeemer: %w", redeemerErr)
		}
		replyText := ""
		if cfg.InstagramMeta.RepliesEnabled() {
			replyText = "CraftSky received your verification message. Return to CraftSky to confirm your Instagram username."
		}
		deps.InstagramWebhookWorker, err = instagram.NewWebhookWorker(
			webhookStore,
			redeemer,
			deps.InstagramMembership,
			metaClient,
			instagram.WebhookWorkerOptions{
				BatchSize:                  1,
				Now:                        time.Now,
				ReplyText:                  replyText,
				ReplyWindow:                cfg.InstagramLimits.DMReplyWindow,
				RateLimiter:                deps.InstagramRateLimiter,
				InvalidIGSIDPer15Minutes:   cfg.InstagramLimits.InvalidIGSIDPer15Minutes,
				MetaLookupsPerIGSIDPerHour: cfg.InstagramLimits.MetaLookupsPerIGSIDPerHour,
				MembershipInactivator:      instagramPrivateData,
				RetryPolicy:                retryPolicy,
			},
		)
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("instagram webhook worker: %w", err)
		}
	}
	privateSuggestions, err := instagram.NewPrivateSuggestionStore(
		pool,
		ownerLifecycles,
		time.Now,
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("instagram private suggestion store: %w", err)
	}
	deps.InstagramImports, err = instagram.NewImportService(instagram.ImportServiceOptions{
		Repository:      instagram.NewImportStore(pool),
		Matcher:         instagram.NewPrivateSuggestionMatcher(pool, privateSuggestions, suggestionPolicy, time.Now),
		MaxEntries:      cfg.InstagramLimits.ImportMaxEntries,
		DefaultPageSize: cfg.InstagramLimits.PageDefault,
		MaxPageSize:     cfg.InstagramLimits.PageMax,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("instagram import service: %w", err)
	}
	deps.InstagramAccount = instagram.NewAccountStore(pool, time.Now)
	deps.InstagramReconciliation, err = instagram.NewReconciliationWorker(instagram.ReconciliationWorkerOptions{
		Pool:                  pool,
		PrivateSuggestions:    privateSuggestions,
		Policy:                suggestionPolicy,
		Membership:            instagramMembership,
		MembershipInactivator: instagramPrivateData,
		Now:                   time.Now,
		LeaseDuration:         cfg.InstagramLimits.WorkerLeaseDuration,
		MaxAttempts:           cfg.InstagramLimits.WorkerMaxAttempts,
		ModerationRestoration: moderationRestoration,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("instagram reconciliation worker: %w", err)
	}
	deps.IdentityCacheUpdater = api.NewIdentityCacheService(pool, deps.HandleResolver, time.Now)
	deps.FollowStore = api.NewFollowStore(pool)
	deps.ReportStore = api.NewReportStore(pool)
	deps.ReportForwarder = api.NewPlaceholderReportForwarder(time.Now)
	deps.NewPDSClient = deps.Observability.WrapPDSFactory(func(_ context.Context, did syntax.DID, sid string) (auth.PDSClient, error) {
		return auth.NewCoordinatedPDSClient(
			oauthSessionCoordinator,
			did,
			sid,
			func(operationCtx context.Context, session *oauth.ClientSession) (auth.PDSClient, error) {
				// The coordinator translates terminal OAuth failures and performs
				// local-first versioned revocation after the operation returns.
				return federated.newPDSClient(operationCtx, session, nil)
			},
		)
	})
	deps.NewPDSEffects, err = pdseffects.NewExecutorFactory(
		ownerLifecycles,
		deps.NewPDSClient,
		cfg.PDSEffectTimeout,
		time.Now,
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ordinary PDS effect executor: %w", err)
	}
	followCoordinator, err := followwrite.NewCoordinator(
		ownerLifecycles,
		followwrite.NewService(deps.NewPDSClient),
		cfg.PDSEffectTimeout,
		time.Now,
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ordinary follow effect coordinator: %w", err)
	}
	deps.InstagramSuggestions, err = instagram.NewSuggestionService(
		privateSuggestions,
		ownerLifecycles,
		suggestionPolicy,
		instagramSuggestionFollowAdapter{coordinator: followCoordinator},
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("instagram suggestion service: %w", err)
	}
	deletionService, err := accountdeletion.NewAppService(accountdeletion.AppServiceOptions{
		Pool: pool, Store: deletionStore, OAuth: oauthFlow,
		Owners: ownerLifecycles, Sessions: sessionLifecycle, OAuthStore: oauthStore,
		DepartureParticipant: scheduledDepartureParticipant,
		Now:                  time.Now, Random: rand.Reader, IntentTTL: cfg.AccountDeletionIntentTTL,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("account deletion service: %w", err)
	}
	instagramDeletion, err := accountdeletion.NewNamedPrivateCleanup(
		"instagramPrivate",
		func(ctx context.Context, owner syntax.DID) error {
			return accountdeletion.PurgeInstagramForAccountDeletion(ctx, instagramPrivateData, owner)
		},
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("account deletion Instagram cleanup: %w", err)
	}
	deletionCleaner, err := accountdeletion.NewPrivateCleaner(
		[]accountdeletion.PrivateCleanupComponent{
			accountdeletion.NewDatabasePrivateCleanup(pool),
			instagramDeletion,
		},
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("account deletion private cleanup: %w", err)
	}
	deletionLifecycle, err := accountdeletion.NewLifecycleProcessor(accountdeletion.LifecycleProcessorOptions{
		Store: deletionStore, Cleaner: deletionCleaner, AcceptedCleanup: scheduledAccountDeletion,
		NewPDSClient: func(_ context.Context, owner syntax.DID, authority auth.DeletionSessionAuthority) (auth.DeletionPDSClient, error) {
			return auth.NewCoordinatedDeletionPDSClient(
				oauthSessionCoordinator,
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
		pool.Close()
		return nil, nil, fmt.Errorf("account deletion lifecycle: %w", err)
	}
	deletionWorker, err := accountdeletion.NewWorker(accountdeletion.WorkerOptions{
		Store: deletionStore, Processor: deletionLifecycle, Finalizer: deletionService,
		WorkerID: "appview", Now: time.Now, LeaseDuration: 2 * time.Minute,
		RetryPolicy: accountdeletion.DefaultRetryPolicy(),
		Logger:      logger,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("account deletion worker: %w", err)
	}
	deletionIntentExpiry, err := accountdeletion.NewIntentExpiryProcessor(accountdeletion.IntentExpiryProcessorOptions{
		Source: deletionStore, Expirer: deletionService, BatchSize: cfg.AccountDeletionIntentSweepBatch,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("account deletion intent expiry processor: %w", err)
	}
	deps.AccountDeletion = deletionService
	deps.AccountDeletionOAuth = deletionService
	deps.AccountDeletionPendingLogin = deletionService
	deps.AccountDeletionWorker = deletionWorker
	deps.AccountDeletionIntentExpiry = deletionIntentExpiry
	scheduledPublisher, err := scheduledposts.NewPublicationProcessor(scheduledposts.PublicationProcessorOptions{
		Store:         scheduledStore,
		Sessions:      auth.NewBackgroundSessionSelector(pool),
		NewPDS:        deps.NewPDSClient,
		Objects:       scheduledObjects,
		Now:           time.Now,
		MaxMediaBytes: cfg.MaxImageUploadBytes,
		Observer:      observer,
		Validate: func(ctx context.Context, owner syntax.DID, payload scheduledposts.Payload) error {
			return api.ValidateScheduledPublication(
				ctx,
				api.NewPostStore(pool, observer),
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
		pool.Close()
		return nil, nil, fmt.Errorf("scheduled publication processor: %w", err)
	}
	deps.ScheduledManualPublisher, err = scheduledposts.NewManualPublicationService(
		scheduledStore,
		scheduledPublisher,
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("manual scheduled publisher: %w", err)
	}
	deps.ScheduledPublisher, err = scheduledposts.NewWorker(scheduledposts.WorkerOptions{
		Store: scheduledStore, Processor: scheduledPublisher, Now: time.Now,
		Observer: observer,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("scheduled publication worker: %w", err)
	}
	deps.RelationshipMutations = relationships.NewMutationServiceWithRestoration(
		deps.RelationshipStore,
		deps.NewPDSClient,
		time.Now,
		deps.InstagramRestoration,
		deps.Observability,
	)

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			federated.boundary.CloseIdleConnections()
			deps.Observability.Flush(2 * time.Second)
			deps.DB.Close()
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

func (d *Deps) expirePDSSession(ctx context.Context, did syntax.DID, sid string) {
	attrs := []any{
		slog.String("component", "pds"),
		slog.String("operation", "oauth.session_resume"),
		slog.String("failure_stage", "session_resume"),
		slog.String("result", "error"),
		slog.String("error_category", "auth"),
	}
	if runID := middleware.GetRunID(ctx); runID != "" {
		attrs = append(attrs, slog.String("run_id", runID))
	}
	d.Logger.Warn("PDS OAuth session expired; revoking Craftsky sessions",
		attrs...)
	if err := d.CraftskySessionStore.RevokeOAuthSession(ctx, did.String(), sid); err != nil {
		d.Logger.Error("revoke Craftsky sessions failed", attrs...)
	}
	if err := d.OAuthStore.DeleteSession(ctx, did, sid); err != nil {
		d.Logger.Error("delete OAuth session failed", attrs...)
	}
}

func newIndexerDispatcher(pool *pgxpool.Pool, anonPDS auth.PDSClient, logger *slog.Logger, lifecycles ...notifications.Lifecycle) *index.Dispatcher {
	return newIndexerDispatcherWithTracker(pool, anonPDS, logger, nil, nil, lifecycles...)
}

func newIndexerDispatcherWithTracker(
	pool *pgxpool.Pool,
	anonPDS auth.PDSClient,
	logger *slog.Logger,
	repositoryTracker tap.RepositoryTracker,
	observer index.RelationshipObserver,
	lifecycles ...notifications.Lifecycle,
) *index.Dispatcher {
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return newIndexerDispatcherWithActorDeletion(
		pool,
		anonPDS,
		logger,
		repositoryTracker,
		observer,
		lifecycle,
		notifications.NewActorDeletionService(pool),
	)
}

func newIndexerDispatcherWithActorDeletion(
	pool *pgxpool.Pool,
	anonPDS auth.PDSClient,
	logger *slog.Logger,
	repositoryTracker tap.RepositoryTracker,
	observer index.RelationshipObserver,
	lifecycle notifications.Lifecycle,
	actorDeletion notifications.ActorDeletion,
) *index.Dispatcher {
	return newIndexerDispatcherBundleWithActorDeletion(
		pool, anonPDS, logger, repositoryTracker, observer, lifecycle, actorDeletion,
	).legacy
}

type indexerDispatcherBundle struct {
	legacy        *index.Dispatcher
	transactional *index.TransactionalDispatcher
}

func newIndexerDispatcherBundleWithActorDeletion(
	pool *pgxpool.Pool,
	anonPDS auth.PDSClient,
	logger *slog.Logger,
	repositoryTracker tap.RepositoryTracker,
	observer index.RelationshipObserver,
	lifecycle notifications.Lifecycle,
	actorDeletion notifications.ActorDeletion,
) indexerDispatcherBundle {
	if lifecycle == nil {
		lifecycle = notifications.NoopLifecycle{}
	}
	if actorDeletion == nil {
		actorDeletion = notifications.NoopActorDeletion{}
	}
	legacy := index.NewDispatcher(index.NotImplemented{})
	transactional := index.NewTransactionalDispatcher()
	blueskyIdx := index.NewBlueskyProfile(pool)
	backfiller := index.NewObservedBlueskyBackfiller(anonPDS, blueskyIdx, repositoryTracker, observer)
	profile := index.NewCraftskyProfile(pool, backfiller, logger, actorDeletion)
	post := index.NewCraftskyPost(pool, logger, lifecycle)
	like := index.NewCraftskyLike(pool, logger, lifecycle)
	repost := index.NewCraftskyRepost(pool, logger, lifecycle)
	follow := index.NewBlueskyFollow(pool, lifecycle)
	block := index.NewBlueskyBlock(pool, observer)
	registrations := []struct {
		collection syntax.NSID
		indexer    interface {
			index.Indexer
			index.TransactionalIndexer
		}
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
		legacy.Register(registration.collection, registration.indexer)
		transactional.Register(registration.collection, registration.indexer)
	}
	return indexerDispatcherBundle{legacy: legacy, transactional: transactional}
}
