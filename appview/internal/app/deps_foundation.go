package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
)

// foundationDependencies owns deployment validation and the process resources
// every higher-level capability consumes. Building it performs no feature
// wiring, which keeps database and outbound-client cleanup deterministic when a
// later capability constructor fails.
type foundationDependencies struct {
	oauthArtifacts    auth.ClientArtifacts
	handoffReceiptKey []byte
	logger            *slog.Logger
	pool              *pgxpool.Pool
	federated         *federatedClients
}

func newFoundationDependencies(
	ctx context.Context,
	cfg Config,
	level slog.Level,
	resources *dependencyCleanup,
) (*foundationDependencies, error) {
	oauthArtifacts, err := buildOAuthArtifacts(cfg.OAuth)
	if err != nil {
		return nil, fmt.Errorf("build oauth client artifacts: %w", err)
	}
	handoffReceiptKey, err := decodeHandoffReceiptKey(cfg.OAuthHandoffReceiptKey)
	if err != nil {
		return nil, err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	// Third-party libraries that reach for slog.Default use the same bounded
	// process logger as AppView-owned capabilities.
	slog.SetDefault(logger)

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}
	resources.add(pool.Close)
	federated, err := newFederatedClients(cfg.FederatedHTTP)
	if err != nil {
		return nil, err
	}
	resources.add(federated.boundary.CloseIdleConnections)

	return &foundationDependencies{
		oauthArtifacts: oauthArtifacts, handoffReceiptKey: handoffReceiptKey,
		logger: logger, pool: pool, federated: federated,
	}, nil
}
