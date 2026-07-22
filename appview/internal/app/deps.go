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
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/integrations/instagrammeta"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/push"
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

	// Instagram migration is private AppView data. Verification can remain
	// disabled while the membership/store dependencies continue to expose
	// retained local state and privacy controls.
	InstagramMembership              *instagram.MembershipStore
	InstagramRateLimiter             *instagram.PostgresRateLimiter
	InstagramVerification            *instagram.VerificationService
	InstagramWebhook                 http.Handler
	InstagramWebhookWorker           *instagram.WebhookWorker
	InstagramPrivateData             *instagram.PrivateDataService
	InstagramReconciliation          *instagram.ReconciliationWorker
	InstagramRetention               *instagram.RetentionService
	InstagramAccount                 *instagram.AccountStore
	InstagramImports                 *instagram.ImportService
	InstagramSuggestions             *instagram.SuggestionService
	InstagramNotificationEligibility *instagram.NotificationEligibilityService
	InstagramRestoration             instagram.EligibilityRestorationEnqueuer

	// ProfileStore serves the /v1/profiles endpoints.
	ProfileStore *api.ProfileStore
	// IdentityCacheUpdater upserts authenticated users' current handles after profile initialization.
	IdentityCacheUpdater auth.IdentityCacheUpdater
	// FollowStore serves follow graph read/write operations for /v1/profiles/*/follows.
	FollowStore *api.FollowStore
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
		cfg.OAuthCallbackURL,
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
	observer := observability.New(observability.Config{
		Env: string(cfg.Env), Release: cfg.SentryRelease, LogsEnabled: cfg.SentryLogsEnabled, TracingEnabled: cfg.SentryTracingEnabled, TracesSampleRate: cfg.SentryTracesSampleRate, MetricsEnabled: cfg.SentryMetricsEnabled, TapTracingEnabled: cfg.SentryTapTracingEnabled, TapTracesSampleRate: cfg.SentryTapTracesSampleRate, SentryDSN: cfg.SentryDSN, Logger: logger,
	})
	lifecycle, err := notifications.NewServiceWithOptions(notifications.ServiceOptions{
		InstagramCoalescingWindow: cfg.InstagramLimits.NotificationWindow,
		InstagramCountCap:         cfg.InstagramLimits.NotificationCountCap,
	}, observer)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Instagram notification limits: %w", err)
	}
	// Production matching and every later visibility boundary intentionally
	// fail closed until the repository has a real block/mute safety provider.
	suggestionPolicy := instagram.NewPostgresInstagramSuggestionEligibilityPolicy(pool, nil, time.Now)
	instagramNotificationEligibility, err := instagram.NewNotificationEligibilityService(pool, suggestionPolicy, lifecycle)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Instagram notification eligibility: %w", err)
	}
	instagramMembership := instagram.NewMembershipStore(pool)
	var instagramRateLimiter *instagram.PostgresRateLimiter
	if cfg.InstagramData.Available() {
		instagramRateLimiter, err = instagram.NewPostgresRateLimiter(pool, cfg.InstagramData.HMACKey(), time.Now)
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("Instagram persistent rate limiter: %w", err)
		}
	}
	instagramPrivateData := instagram.NewPrivateDataService(pool, instagramRateLimiter, time.Now)
	notificationActorDeletion := notifications.NewActorDeletionService(pool)
	profileDeletion := &profileMembershipDeletion{
		notifications: notificationActorDeletion,
		instagram:     instagramPrivateData,
		now:           time.Now,
	}
	dispatcher := newIndexerDispatcherWithActorDeletion(pool, anonPDS, logger, lifecycle, profileDeletion)
	identityDeletion := &terminalIdentityDeletion{handlers: []tap.IdentityDeletionHandler{
		notificationActorDeletion,
		instagramPrivateData,
	}}

	deps := &Deps{
		Config:                           cfg,
		Logger:                           logger,
		DB:                               pool,
		RateLimiter:                      rateLimiter,
		Observability:                    observer,
		OAuthApp:                         oauthApp,
		OAuthStore:                       oauthStore,
		CraftskySessionStore:             craftskyStore,
		HandleResolver:                   api.DirectoryHandleResolver{Directory: identityDir},
		Indexer:                          dispatcher,
		Consumer:                         tap.NotImplemented{}, // temp, replaced below
		InstagramMembership:              instagramMembership,
		InstagramRateLimiter:             instagramRateLimiter,
		InstagramPrivateData:             instagramPrivateData,
		InstagramRetention:               instagram.NewRetentionService(pool, time.Now),
		InstagramNotificationEligibility: instagramNotificationEligibility,
		InstagramRestoration:             instagram.NewReconciliationTrigger(pool, time.Now),
	}
	if cfg.PushEnabled {
		sender, err := push.NewFirebaseSender(ctx, cfg.FirebaseProjectID)
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("firebase messaging init: %w", err)
		}
		deps.PushDispatcher = push.NewDispatcher(pool, sender, push.DispatcherOptions{
			BatchSize: cfg.PushBatchSize, LeaseDuration: cfg.PushLeaseDuration,
			SendTimeout: cfg.PushSendTimeout, Observer: observer,
			InstagramEligibility: instagramNotificationEligibility,
		})
	}

	deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:             cfg.TapWSURL,
		Indexer:         dispatcher,
		AckTimeout:      cfg.TapAckTimeout,
		ReconnectMax:    cfg.TapReconnectMax,
		MaxRetries:      cfg.TapMaxRetries,
		Logger:          logger,
		Observer:        deps.Observability,
		IdentityHandler: identityDeletion,
	})
	if cfg.Env == EnvDev {
		deps.HandleResolver = api.DevHandleResolver{
			Primary: api.DirectoryHandleResolver{Directory: identityDir},
			Pool:    pool,
		}
	}

	deps.ProfileStore = api.NewProfileStore(pool, anonPDS)
	verificationStore := instagram.NewVerificationStore(pool)
	var challengeCodec *instagram.ChallengeCodec
	if cfg.InstagramData.Available() {
		challengeCodec, err = instagram.NewChallengeCodec(rand.Reader, cfg.InstagramData.HMACKey())
		if err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("Instagram challenge codec: %w", err)
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
		return nil, nil, fmt.Errorf("Instagram verification service: %w", err)
	}
	if cfg.InstagramMeta.Enabled() && cfg.InstagramMeta.Configured() {
		if deps.InstagramRateLimiter == nil {
			pool.Close()
			return nil, nil, errors.New("Instagram Meta integration requires the persistent rate limiter")
		}
		digests, digestErr := instagrammeta.NewDigestCodec(cfg.InstagramData.HMACKey(), instagram.CanonicalizeChallenge)
		if digestErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("Instagram webhook digest codec: %w", digestErr)
		}
		reducer, reducerErr := instagrammeta.NewPayloadReducer(cfg.InstagramMeta.InstagramAccountID(), digests)
		if reducerErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("Instagram webhook reducer: %w", reducerErr)
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
			return nil, nil, fmt.Errorf("Instagram webhook limiter: %w", limiterErr)
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
			return nil, nil, fmt.Errorf("Instagram webhook store: %w", storeErr)
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
			return nil, nil, fmt.Errorf("Instagram webhook handler: %w", err)
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
			return nil, nil, fmt.Errorf("Instagram Meta client: %w", clientErr)
		}
		redeemer, redeemerErr := instagram.NewVerificationWebhookRedeemer(verificationStore)
		if redeemerErr != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("Instagram webhook redeemer: %w", redeemerErr)
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
			return nil, nil, fmt.Errorf("Instagram webhook worker: %w", err)
		}
	}
	suggestionStore := instagram.NewSuggestionStore(pool, lifecycle)
	deps.InstagramImports, err = instagram.NewImportService(instagram.ImportServiceOptions{
		Repository:      instagram.NewImportStore(pool),
		Matcher:         instagram.NewSuggestionMatcher(pool, suggestionStore, suggestionPolicy, time.Now),
		MaxEntries:      cfg.InstagramLimits.ImportMaxEntries,
		DefaultPageSize: cfg.InstagramLimits.PageDefault,
		MaxPageSize:     cfg.InstagramLimits.PageMax,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Instagram import service: %w", err)
	}
	deps.InstagramAccount = instagram.NewAccountStore(pool, time.Now)
	deps.InstagramReconciliation, err = instagram.NewReconciliationWorker(instagram.ReconciliationWorkerOptions{
		Pool:          pool,
		Store:         suggestionStore,
		Policy:        suggestionPolicy,
		Notifications: lifecycle,
		Now:           time.Now,
		LeaseDuration: cfg.InstagramLimits.WorkerLeaseDuration,
		MaxAttempts:   cfg.InstagramLimits.WorkerMaxAttempts,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Instagram reconciliation worker: %w", err)
	}
	deps.InstagramSuggestions, err = instagram.NewSuggestionService(instagram.SuggestionServiceOptions{
		Repository:      suggestionStore,
		Policy:          suggestionPolicy,
		DefaultPageSize: cfg.InstagramLimits.PageDefault,
		MaxPageSize:     cfg.InstagramLimits.PageMax,
	})
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("Instagram suggestion service: %w", err)
	}
	deps.IdentityCacheUpdater = api.NewIdentityCacheService(pool, deps.HandleResolver, time.Now)
	deps.FollowStore = api.NewFollowStore(pool)
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
	lifecycle := notifications.Lifecycle(notifications.NoopLifecycle{})
	if len(lifecycles) > 0 && lifecycles[0] != nil {
		lifecycle = lifecycles[0]
	}
	return newIndexerDispatcherWithActorDeletion(
		pool,
		anonPDS,
		logger,
		lifecycle,
		notifications.NewActorDeletionService(pool),
	)
}

func newIndexerDispatcherWithActorDeletion(
	pool *pgxpool.Pool,
	anonPDS auth.PDSClient,
	logger *slog.Logger,
	lifecycle notifications.Lifecycle,
	actorDeletion notifications.ActorDeletion,
) *index.Dispatcher {
	if lifecycle == nil {
		lifecycle = notifications.NoopLifecycle{}
	}
	if actorDeletion == nil {
		actorDeletion = notifications.NoopActorDeletion{}
	}
	dispatcher := index.NewDispatcher(index.NotImplemented{})
	blueskyIdx := index.NewBlueskyProfile(pool)
	backfiller := index.NewBlueskyBackfiller(anonPDS, blueskyIdx)
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
	dispatcher.Register("app.bsky.actor.profile", blueskyIdx)
	return dispatcher
}
