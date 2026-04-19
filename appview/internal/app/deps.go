package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/jackc/pgx/v5/pgxpool"

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

	Consumer tap.Consumer
	Indexer  index.Indexer
}

// NewDevDeps wires the dev variant: debug-level logger, MockAuthService,
// NotImplemented consumer+indexer.
func NewDevDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	deps, cleanup, err := newDeps(ctx, cfg, slog.LevelDebug)
	if err != nil {
		return nil, nil, err
	}
	deps.AuthService = &auth.MockAuthService{DefaultDID: cfg.DevDID}
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

	indexerImpl := index.NewBlueskyPostsSample(pool)

	deps := &Deps{
		Config:               cfg,
		Logger:               logger,
		DB:                   pool,
		OAuthApp:             oauthApp,
		OAuthStore:           oauthStore,
		CraftskySessionStore: craftskyStore,
		Indexer:              indexerImpl,
		Consumer:             tap.NotImplemented{}, // temp, replaced below
	}

	deps.Consumer = tap.NewWSConsumer(tap.WSConsumerConfig{
		URL:          cfg.TapWSURL,
		Indexer:      indexerImpl,
		AckTimeout:   cfg.TapAckTimeout,
		ReconnectMax: cfg.TapReconnectMax,
		MaxRetries:   cfg.TapMaxRetries,
		Logger:       logger,
	})

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
