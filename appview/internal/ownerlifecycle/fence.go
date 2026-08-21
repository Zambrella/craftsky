package ownerlifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fenceContextKey struct{}
type fenceConnectionContextKey struct{}
type parentSessionFenceContextKey struct{}

type fenceConnectionState struct {
	conn    *pgxpool.Conn
	discard bool
}

// Fencer owns the process-local entry point to PostgreSQL's session advisory
// owner locks. A single callback holds all requested locks on one dedicated
// connection, and the connection is discarded if unlock cannot be confirmed.
type Fencer struct {
	pool           *pgxpool.Pool
	acquireTimeout time.Duration
}

func NewFencer(pool *pgxpool.Pool, acquireTimeout time.Duration) (*Fencer, error) {
	if pool == nil {
		return nil, errors.New("owner lifecycle fence requires a database pool")
	}
	if acquireTimeout <= 0 {
		return nil, errors.New("owner lifecycle fence acquisition timeout must be positive")
	}
	return &Fencer{pool: pool, acquireTimeout: acquireTimeout}, nil
}

func (fencer *Fencer) WithShared(
	ctx context.Context,
	owners []syntax.DID,
	callback func(context.Context) error,
) error {
	return fencer.with(ctx, owners, true, callback)
}

func (fencer *Fencer) WithExclusive(
	ctx context.Context,
	owners []syntax.DID,
	callback func(context.Context) error,
) error {
	return fencer.with(ctx, owners, false, callback)
}

func (fencer *Fencer) with(
	ctx context.Context,
	owners []syntax.DID,
	shared bool,
	callback func(context.Context) error,
) (returnErr error) {
	if fencer == nil || fencer.pool == nil || callback == nil {
		return errors.New("invalid owner fence request")
	}
	if ctx.Value(fenceContextKey{}) != nil {
		return ErrFenceReentry
	}
	canonical, err := CanonicalOwners(owners)
	if err != nil {
		return err
	}
	keys := make([]int64, len(canonical))
	for index, owner := range canonical {
		keys[index], err = FenceKey(owner)
		if err != nil {
			return err
		}
	}

	acquireCtx, cancelAcquire := context.WithTimeout(ctx, fencer.acquireTimeout)
	defer cancelAcquire()
	conn, err := fencer.pool.Acquire(acquireCtx)
	if err != nil {
		return fmt.Errorf("acquire owner fence connection: %w", err)
	}
	locked := 0
	state := &fenceConnectionState{conn: conn}
	defer func() {
		var cleanupErr error
		if !state.discard {
			cleanupErr = fencer.unlock(conn, keys[:locked], shared)
		}
		if cleanupErr != nil || state.discard {
			fencer.discard(conn)
		} else {
			conn.Release()
		}
		returnErr = errors.Join(returnErr, cleanupErr)
	}()

	lockSQL := `SELECT pg_advisory_lock($1)`
	if shared {
		lockSQL = `SELECT pg_advisory_lock_shared($1)`
	}
	for index, key := range keys {
		if _, err := conn.Exec(acquireCtx, lockSQL, key); err != nil {
			return fmt.Errorf("acquire owner fence at canonical position %d: %w", index, err)
		}
		locked++
	}

	callbackCtx := context.WithValue(ctx, fenceContextKey{}, struct{}{})
	callbackCtx = context.WithValue(callbackCtx, fenceConnectionContextKey{}, state)
	return callback(callbackCtx)
}

func fencedConnection(ctx context.Context) (*pgxpool.Conn, bool) {
	state, ok := ctx.Value(fenceConnectionContextKey{}).(*fenceConnectionState)
	if !ok || state == nil || state.conn == nil || state.discard {
		return nil, false
	}
	return state.conn, true
}

func (fencer *Fencer) withParentSession(
	ctx context.Context,
	owner syntax.DID,
	sessionID string,
	callback func(context.Context) error,
) (returnErr error) {
	if callback == nil || ctx.Value(parentSessionFenceContextKey{}) != nil {
		return ErrFenceReentry
	}
	state, ok := ctx.Value(fenceConnectionContextKey{}).(*fenceConnectionState)
	if !ok || state == nil || state.conn == nil || state.discard {
		return ErrFenceRequired
	}
	key, err := ParentSessionFenceKey(owner, sessionID)
	if err != nil {
		return err
	}
	acquireCtx, cancelAcquire := context.WithTimeout(ctx, fencer.acquireTimeout)
	defer cancelAcquire()
	if _, err := state.conn.Exec(acquireCtx, `SELECT pg_advisory_lock($1)`, key); err != nil {
		state.discard = true
		return fmt.Errorf("acquire OAuth parent-session fence: %w", err)
	}
	defer func() {
		unlockCtx, cancelUnlock := context.WithTimeout(context.Background(), fencer.acquireTimeout)
		defer cancelUnlock()
		var unlocked bool
		if err := state.conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock($1)`, key).Scan(&unlocked); err != nil {
			state.discard = true
			returnErr = errors.Join(returnErr, fmt.Errorf("release OAuth parent-session fence: %w", err))
			return
		}
		if !unlocked {
			state.discard = true
			returnErr = errors.Join(returnErr, errors.New("release OAuth parent-session fence: lock was not held"))
		}
	}()
	callbackCtx := context.WithValue(ctx, parentSessionFenceContextKey{}, struct{}{})
	return callback(callbackCtx)
}

func (fencer *Fencer) discard(conn *pgxpool.Conn) {
	underlying := conn.Hijack()
	closeCtx, cancelClose := context.WithTimeout(context.Background(), fencer.acquireTimeout)
	_ = underlying.Close(closeCtx)
	cancelClose()
}

func (fencer *Fencer) unlock(conn *pgxpool.Conn, keys []int64, shared bool) error {
	if len(keys) == 0 {
		return nil
	}
	unlockCtx, cancelUnlock := context.WithTimeout(context.Background(), fencer.acquireTimeout)
	defer cancelUnlock()
	unlockSQL := `SELECT pg_advisory_unlock($1)`
	if shared {
		unlockSQL = `SELECT pg_advisory_unlock_shared($1)`
	}
	for index := len(keys) - 1; index >= 0; index-- {
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, unlockSQL, keys[index]).Scan(&unlocked); err != nil {
			return fmt.Errorf("release owner fence at canonical position %d: %w", index, err)
		}
		if !unlocked {
			return fmt.Errorf("release owner fence at canonical position %d: lock was not held", index)
		}
	}
	return nil
}
