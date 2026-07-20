package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/push"
	"social.craftsky/appview/internal/relationships"
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
	Config        Config
	Logger        *slog.Logger
	DB            *pgxpool.Pool
	AuthService   auth.AuthService
	RateLimiter   *middleware.LocalRateLimiter
	Observability *observability.Observer

	// OAuth subsystem.
	OAuthApp             *oauth.ClientApp
	OAuthStore           *auth.PostgresAuthStore
	CraftskySessionStore *auth.CraftskySessionStore

	// Identity resolution for /v1/whoami. Typed as the interface
	// (not the concrete struct) so route tests can inject a stub
	// without constructing an identity.Directory.
	HandleResolver api.HandleResolver

	Consumer       tap.Consumer
	Indexer        index.Indexer
	PushDispatcher *push.Dispatcher

	// ProfileStore serves the /v1/profiles endpoints.
	ProfileStore *api.ProfileStore
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
	// NewPDSClient produces a PDSClient bound to an OAuth session. Shared
	// by the OAuth callback's InitializeProfile step and the write-proxy
	// handlers (today PUT /v1/profiles/me).
	NewPDSClient auth.PDSClientFactory
}

// NewDevDeps wires the dev variant: debug-level logger, StackedAuthService
// (real OAuth tokens take precedence; X-Dev-DID header is the fallback so
// the legacy curl-with-header workflow keeps working), full OAuth subsystem.
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
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	// Third-party libs that reach for slog.Default should get our logger.
	slog.SetDefault(logger)

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("db connect: %w", err)
	}

	oauthCfg, err := auth.BuildClientConfig(
		cfg.OAuthHostname,
		cfg.OAuthClientSecretKey,
		cfg.OAuthClientKeyID,
		cfg.OAuthScopes,
	)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("build oauth client config: %w", err)
	}
	oauthStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
		SessionExpiry:     cfg.OAuthSessionExpiry,
		SessionInactivity: cfg.OAuthSessionInactivity,
		AuthRequestExpiry: cfg.OAuthAuthRequestExpiry,
		Logger:            logger,
	})
	oauthApp := oauth.NewClientApp(&oauthCfg, oauthStore)
	craftskyStore := auth.NewCraftskySessionStore(pool, cfg.CraftskySessionLastSeenThrottle)
	rateLimiter := middleware.NewLocalRateLimiter(cfg.RateLimits, time.Now)
	logger.Warn("rate limiter is process-local; run one AppView instance or configure shared/edge enforcement before horizontal scaling")

	// Shared atproto identity directory for DID↔handle lookups.
	// indigo provides an in-process cache via DefaultDirectory.
	identityDir := identity.DefaultDirectory()

	anonPDS := auth.NewAnonymousPDSClient(identityDir, 5*time.Second)
	repositoryTracker, err := tap.NewAdminClient(cfg.TapWSURL, &http.Client{Timeout: 5 * time.Second})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("build Tap admin client: %w", err)
	}
	observer := observability.New(observability.Config{
		Env: string(cfg.Env), Release: cfg.SentryRelease, LogsEnabled: cfg.SentryLogsEnabled, TracingEnabled: cfg.SentryTracingEnabled, TracesSampleRate: cfg.SentryTracesSampleRate, MetricsEnabled: cfg.SentryMetricsEnabled, TapTracingEnabled: cfg.SentryTapTracingEnabled, TapTracesSampleRate: cfg.SentryTapTracesSampleRate, SentryDSN: cfg.SentryDSN, Logger: logger,
	})
	lifecycle := notifications.NewService(observer)
	dispatcher := newIndexerDispatcherWithTracker(pool, anonPDS, logger, repositoryTracker, observer, lifecycle)

	deps := &Deps{
		Config:               cfg,
		Logger:               logger,
		DB:                   pool,
		RateLimiter:          rateLimiter,
		Observability:        observer,
		OAuthApp:             oauthApp,
		OAuthStore:           oauthStore,
		CraftskySessionStore: craftskyStore,
		RepositoryTracker:    repositoryTracker,
		HandleResolver:       api.DirectoryHandleResolver{Directory: identityDir},
		Indexer:              dispatcher,
		Consumer:             tap.NotImplemented{}, // temp, replaced below
	}
	if cfg.PushEnabled {
		sender, err := push.NewFirebaseSender(ctx, cfg.FirebaseProjectID)
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("firebase messaging init: %w", err)
		}
		deps.PushDispatcher = push.NewDispatcher(pool, sender, push.DispatcherOptions{BatchSize: cfg.PushBatchSize, LeaseDuration: cfg.PushLeaseDuration, SendTimeout: cfg.PushSendTimeout, Observer: observer})
	}

	deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:             cfg.TapWSURL,
		Indexer:         dispatcher,
		AckTimeout:      cfg.TapAckTimeout,
		ReconnectMax:    cfg.TapReconnectMax,
		MaxRetries:      cfg.TapMaxRetries,
		Logger:          logger,
		Observer:        deps.Observability,
		IdentityHandler: notifications.NewActorDeletionService(pool),
	})
	if cfg.Env == EnvDev {
		deps.HandleResolver = api.DevHandleResolver{
			Primary: api.DirectoryHandleResolver{Directory: identityDir},
			Pool:    pool,
		}
	}

	deps.ProfileStore = api.NewProfileStore(pool)
	deps.IdentityCacheUpdater = api.NewIdentityCacheService(pool, deps.HandleResolver, time.Now)
	deps.FollowStore = api.NewFollowStore(pool)
	deps.RelationshipStore = relationships.NewStore(pool)
	deps.ReportStore = api.NewReportStore(pool)
	deps.ReportForwarder = api.NewPlaceholderReportForwarder(time.Now)
	deps.ModerationStore = api.NewModerationStore(pool)
	deps.NewPDSClient = deps.Observability.WrapPDSFactory(func(ctx context.Context, did syntax.DID, sid string) (auth.PDSClient, error) {
		sess, err := oauthApp.ResumeSession(ctx, did, sid)
		if err != nil {
			err = auth.TranslatePDSError(err)
			if errors.Is(err, auth.ErrPDSSessionExpired) {
				deps.expirePDSSession(ctx, did, sid)
			}
			return nil, err
		}
		return &auth.IndigoPDSClient{
			Client: sess.APIClient(),
			OnSessionExpired: func(ctx context.Context) {
				deps.expirePDSSession(ctx, did, sid)
			},
		}, nil
	})
	deps.RelationshipMutations = relationships.NewMutationService(deps.RelationshipStore, deps.NewPDSClient, time.Now, deps.Observability)

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
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
	dispatcher := index.NewDispatcher(index.NotImplemented{})
	blueskyIdx := index.NewBlueskyProfile(pool)
	backfiller := index.NewObservedBlueskyBackfiller(anonPDS, blueskyIdx, repositoryTracker, observer)
	actorDeletion := notifications.NewActorDeletionService(pool)
	dispatcher.Register("social.craftsky.actor.profile",
		index.NewCraftskyProfile(pool, backfiller, logger, actorDeletion))
	dispatcher.Register("social.craftsky.feed.post",
		index.NewCraftskyPost(pool, logger, lifecycle))
	dispatcher.Register("social.craftsky.feed.like",
		index.NewCraftskyLike(pool, logger, lifecycle))
	dispatcher.Register("social.craftsky.feed.repost",
		index.NewCraftskyRepost(pool, logger, lifecycle))
	dispatcher.Register("app.bsky.graph.follow",
		index.NewBlueskyFollow(pool, lifecycle))
	dispatcher.Register("app.bsky.graph.block",
		index.NewBlueskyBlock(pool, observer))
	dispatcher.Register("app.bsky.actor.profile", blueskyIdx)
	return dispatcher
}
