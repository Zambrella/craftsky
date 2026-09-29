package subscriptions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
)

var ErrFeatureAccessRequired = errors.New("subscription feature access required")

// WithPlusAccess holds a shared, per-DID PostgreSQL session fence across the
// final publication check and external effect. Access-loss transactions acquire
// the matching exclusive fence before changing entitlement-bearing rows.
func (s *Store) WithPlusAccess(ctx context.Context, did syntax.DID, effect func(context.Context) error) (result error) {
	if s == nil || s.pool == nil || did == "" || effect == nil {
		return errors.New("subscription effect boundary unavailable")
	}
	acquireCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := s.pool.Acquire(acquireCtx)
	if err != nil {
		return fmt.Errorf("acquire subscription access fence connection: %w", err)
	}
	key := accessFenceKey(did)
	if _, err := conn.Exec(acquireCtx, `SELECT pg_advisory_lock_shared($1)`, key); err != nil {
		discardSubscriptionFence(conn)
		return fmt.Errorf("acquire subscription access fence: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, `SELECT pg_advisory_unlock_shared($1)`, key).Scan(&unlocked); err != nil || !unlocked {
			discardSubscriptionFence(conn)
			result = errors.Join(result, fmt.Errorf("release subscription access fence: %v", err))
			return
		}
		conn.Release()
	}()
	access, err := s.SelfAccess(ctx, did, time.Now())
	if err != nil {
		return fmt.Errorf("read fenced subscription access: %w", err)
	}
	if !access.AllowsPlus() {
		return ErrFeatureAccessRequired
	}
	return effect(ctx)
}

func discardSubscriptionFence(conn interface{ Hijack() *pgx.Conn }) {
	underlying := conn.Hijack()
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = underlying.Close(closeCtx)
}
