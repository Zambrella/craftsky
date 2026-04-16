package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/firehose"
	"social.craftsky/appview/internal/index"
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

	// Day-one stubs. Shape is stable so CLI subcommands compile.
	Firehose firehose.Subscriber
	Indexer  index.Indexer
}

// NewDevDeps wires the dev variant: debug-level logger, MockAuthService,
// NotImplemented firehose+indexer.
func NewDevDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	return newDeps(ctx, cfg, slog.LevelDebug, &auth.MockAuthService{DefaultDID: cfg.DevDID})
}

// NewProdDeps wires the prod variant: info-level logger,
// NotImplementedAuthService, NotImplemented firehose+indexer.
func NewProdDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
	return newDeps(ctx, cfg, slog.LevelInfo, auth.NotImplementedAuthService{})
}

// newDeps is the shared core of NewDevDeps and NewProdDeps. Keeping the
// divergence to just the three parameters (log level, auth service, and
// cfg) makes the env-conditional surface easy to audit.
func newDeps(ctx context.Context, cfg Config, level slog.Level, authSvc auth.AuthService) (*Deps, func(), error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	// Third-party libs that reach for slog.Default should get our logger.
	slog.SetDefault(logger)

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("db connect: %w", err)
	}

	deps := &Deps{
		Config:      cfg,
		Logger:      logger,
		DB:          pool,
		AuthService: authSvc,
		Firehose:    firehose.NotImplemented{},
		Indexer:     index.NotImplemented{},
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
