package index

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// transactionalDatabase is implemented by both pgxpool.Pool and pgx.Tx.
// When backed by pgx.Tx, Begin creates a savepoint; committing that savepoint
// never commits the worker-owned outer transaction.
type transactionalDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
