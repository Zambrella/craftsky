// appview/internal/scheduledposts/store_publication_effect.go
package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PublishingEffectGuard struct {
	conn     *pgxpool.Conn
	ownerDID syntax.DID
	id       uuid.UUID
	mu       sync.Mutex
	released bool
}

type publishingEffectConnectionKey struct{}

type publicationDatabase interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Store) publicationDB(ctx context.Context) publicationDatabase {
	if conn, ok := ctx.Value(publishingEffectConnectionKey{}).(*pgxpool.Conn); ok && conn != nil {
		return conn
	}
	return s.pool
}

func (s *Store) AcquirePublishingEffect(
	ctx context.Context,
	claim PublishingClaim,
) (*PublishingEffectGuard, error) {
	if s == nil || s.pool == nil || claim.ID == uuid.Nil || claim.OwnerDID == "" ||
		claim.OwnerGeneration <= 0 || claim.LeaseToken == uuid.Nil || claim.PayloadVersion < 1 {
		return nil, errors.New("invalid publishing effect claim")
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	handedOff := false
	lockHeld := false
	defer func() {
		if !handedOff {
			if lockHeld {
				_, _ = conn.Exec(
					context.Background(),
					unlockScheduleEffectForSessionSQL,
					claim.OwnerDID,
					claim.ID,
				)
			}
			conn.Release()
		}
	}()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var activeGeneration int64
	if err := tx.QueryRow(ctx, lockActiveScheduledOwnerGenerationSQL, claim.OwnerDID).
		Scan(&activeGeneration); err != nil || activeGeneration != claim.OwnerGeneration {
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWorkerLeaseLost
		}
		return nil, fmt.Errorf("recheck publishing owner lifecycle: %w", err)
	}
	if _, err := tx.Exec(
		ctx,
		lockScheduleEffectForSessionSQL,
		claim.OwnerDID,
		claim.ID,
	); err != nil {
		return nil, fmt.Errorf("lock publishing effect: %w", err)
	}
	lockHeld = true
	var status Status
	var leaseToken *uuid.UUID
	var currentVersion int64
	if err := tx.QueryRow(
		ctx, selectWorkerFenceSQL, claim.OwnerDID, claim.ID, claim.OwnerGeneration,
	).
		Scan(&status, &leaseToken, &currentVersion); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("recheck publishing effect: %w", err)
	}
	if status != StatusPublishing || leaseToken == nil || *leaseToken != claim.LeaseToken {
		return nil, ErrWorkerLeaseLost
	}
	if err := ValidateWorkerVersion(currentVersion, claim.PayloadVersion); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	handedOff = true
	return &PublishingEffectGuard{
		conn: conn, ownerDID: claim.OwnerDID, id: claim.ID,
	}, nil
}

func (guard *PublishingEffectGuard) Release(_ context.Context) error {
	if guard == nil {
		return nil
	}
	guard.mu.Lock()
	defer guard.mu.Unlock()
	if guard.released {
		return nil
	}
	cleanupCtx, cancel := context.WithTimeout(
		context.Background(),
		publishingEffectCleanupTimeout,
	)
	defer cancel()
	var unlocked bool
	err := guard.conn.QueryRow(cleanupCtx, unlockScheduleEffectForSessionSQL, guard.ownerDID, guard.id).
		Scan(&unlocked)
	if err == nil && unlocked {
		guard.conn.Release()
		guard.conn = nil
		guard.released = true
		return nil
	}

	hijacked := guard.conn.Hijack()
	guard.conn = nil
	guard.released = true
	closeCtx, closeCancel := context.WithTimeout(
		context.Background(),
		publishingEffectCleanupTimeout,
	)
	defer closeCancel()
	closeErr := hijacked.Close(closeCtx)
	if err != nil {
		if closeErr != nil {
			return fmt.Errorf("unlock publishing effect: %w; discard connection: %v", err, closeErr)
		}
		return fmt.Errorf("unlock publishing effect: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("publishing effect lock was not held; discard connection: %w", closeErr)
	}
	return errors.New("publishing effect lock was not held")
}

// bind keeps publication completion on the same pool connection that holds
// the session advisory lock. The real OAuth effect boundary already occupies
// its own fenced connection, so consulting the general pool here can deadlock
// at small but valid pool capacities.
func (guard *PublishingEffectGuard) bind(ctx context.Context) context.Context {
	if guard == nil {
		return ctx
	}
	guard.mu.Lock()
	conn := guard.conn
	released := guard.released
	guard.mu.Unlock()
	if released || conn == nil {
		return ctx
	}
	return context.WithValue(ctx, publishingEffectConnectionKey{}, conn)
}
