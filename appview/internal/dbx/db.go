// Package dbx names the shared SQL boundary implemented by pools and transactions.
package dbx

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Normalize preserves the nil-pool semantics of existing optional dependencies
// when a concrete pool crosses the pool-or-transaction interface boundary.
func Normalize(db DB) DB {
	if pool, ok := db.(*pgxpool.Pool); ok && pool == nil {
		return nil
	}
	return db
}
