package app

import (
	"context"
	"fmt"
	"log/slog"
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
	Config      Config
	Logger      *slog.Logger
	DB          *pgxpool.Pool
	AuthService auth.AuthService

	// OAuth subsystem.
	OAuthApp             *oauth.ClientApp
	OAuthStore           *auth.PostgresAuthStore
	CraftskySessionStore *auth.CraftskySessionStore

	// Identity resolution for /v1/whoami. Typed as the interface
	// (not the concrete struct) so route tests can inject a stub
	// without constructing an identity.Directory.
	HandleResolver api.HandleResolver

	Consumer tap.Consumer
	Indexer  index.Indexer

	// ProfileStore serves the /v1/profiles endpoints.
	ProfileStore *api.ProfileStore
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

	// Shared atproto identity directory for DID↔handle lookups.
	// indigo provides an in-process cache via DefaultDirectory.
	identityDir := identity.DefaultDirectory()

	anonPDS := auth.NewAnonymousPDSClient(identityDir, 5*time.Second)
	dispatcher := newIndexerDispatcher(pool, anonPDS, logger)

	deps := &Deps{
		Config:               cfg,
		Logger:               logger,
		DB:                   pool,
		OAuthApp:             oauthApp,
		OAuthStore:           oauthStore,
		CraftskySessionStore: craftskyStore,
		HandleResolver:       api.DirectoryHandleResolver{Directory: identityDir},
		Indexer:              dispatcher,
		Consumer:             tap.NotImplemented{}, // temp, replaced below
	}

	deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          cfg.TapWSURL,
		Indexer:      dispatcher,
		AckTimeout:   cfg.TapAckTimeout,
		ReconnectMax: cfg.TapReconnectMax,
		MaxRetries:   cfg.TapMaxRetries,
		Logger:       logger,
	})

	deps.ProfileStore = api.NewProfileStore(pool)
	deps.NewPDSClient = func(ctx context.Context, did syntax.DID, sid string) (auth.PDSClient, error) {
		sess, err := oauthApp.ResumeSession(ctx, did, sid)
		if err != nil {
			return nil, err
		}
		return &auth.IndigoPDSClient{Client: sess.APIClient()}, nil
	}

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
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

func newIndexerDispatcher(pool *pgxpool.Pool, anonPDS auth.PDSClient, logger *slog.Logger) *index.Dispatcher {
	dispatcher := index.NewDispatcher(index.NotImplemented{})
	blueskyIdx := index.NewBlueskyProfile(pool)
	backfiller := index.NewBlueskyBackfiller(anonPDS, blueskyIdx)
	dispatcher.Register("social.craftsky.actor.profile",
		index.NewCraftskyProfile(pool, backfiller, logger))
	dispatcher.Register("social.craftsky.feed.post",
		index.NewCraftskyPost(pool, logger))
	dispatcher.Register("social.craftsky.feed.like",
		index.NewCraftskyLike(pool, logger))
	dispatcher.Register("social.craftsky.feed.repost",
		index.NewCraftskyRepost(pool, logger))
	dispatcher.Register("app.bsky.actor.profile", blueskyIdx)
	return dispatcher
}
