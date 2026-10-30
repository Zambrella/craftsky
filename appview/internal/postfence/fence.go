// Package postfence serializes current publication eligibility with the final
// bounded provider attempt. Owner lifecycle fences must be acquired first.
package postfence

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Key(uri syntax.ATURI) int64 {
	sum := sha256.Sum256([]byte("craftsky:post-publication:v1:" + uri.String()))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

func LockTx(ctx context.Context, tx pgx.Tx, uri syntax.ATURI) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, Key(uri))
	if err != nil {
		return fmt.Errorf("acquire post publication transaction fence: %w", err)
	}
	return nil
}

// WithConnection returns discard=true whenever session lock release cannot be
// confirmed. Its owner must close, rather than pool, that connection.
func WithConnection(ctx context.Context, conn *pgx.Conn, uri syntax.ATURI, timeout time.Duration, callback func(context.Context) error) (discard bool, resultErr error) {
	acquireCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if _, err := conn.Exec(acquireCtx, `SELECT pg_advisory_lock($1)`, Key(uri)); err != nil {
		return true, fmt.Errorf("acquire post publication session fence: %w", err)
	}
	defer func() {
		unlockCtx, cancelUnlock := context.WithTimeout(context.Background(), timeout)
		defer cancelUnlock()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock($1)`, Key(uri)).Scan(&unlocked); err != nil {
			discard = true
			resultErr = errors.Join(resultErr, fmt.Errorf("release post publication fence: %w", err))
		} else if !unlocked {
			discard = true
			resultErr = errors.Join(resultErr, errors.New("post publication fence was not held"))
		}
	}()
	return false, callback(ctx)
}

func WithPool(ctx context.Context, pool *pgxpool.Pool, uri syntax.ATURI, callback func(context.Context) error) (resultErr error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	// Until the inner call returns, a panic or uncertain unlock must discard
	// the session rather than returning a possibly locked connection to the pool.
	discard := true
	defer func() {
		if discard {
			raw := conn.Hijack()
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			resultErr = errors.Join(resultErr, raw.Close(closeCtx))
		} else {
			conn.Release()
		}
	}()
	discard, resultErr = WithConnection(ctx, conn.Conn(), uri, time.Second, callback)
	return resultErr
}
