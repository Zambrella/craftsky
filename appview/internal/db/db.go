// Package db owns the Postgres connection pool. It's a thin wrapper around
// pgxpool — other packages receive the resulting *pgxpool.Pool via
// app.Deps and don't import pgx directly.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect parses the given Postgres URL, builds a pool, and verifies the
// connection with a Ping. Returns the pool + nil on success, or nil + a
// wrapping error on parse/connect failure.
//
// Callers own the returned pool and must call pool.Close() when done.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
