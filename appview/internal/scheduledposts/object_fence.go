package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type privateObjectFence struct {
	conn      *pgxpool.Conn
	objectKey string
	mu        sync.Mutex
	released  bool
}

func acquirePrivateObjectFence(
	ctx context.Context,
	pool *pgxpool.Pool,
	objectKey string,
) (*privateObjectFence, error) {
	if pool == nil || objectKey == "" {
		return nil, errors.New("invalid private object fence request")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire private object fence connection: %w", err)
	}
	if _, err := conn.Exec(ctx, lockCleanupEffectForSessionSQL, objectKey); err != nil {
		underlying := conn.Hijack()
		closeCtx, cancelClose := context.WithTimeout(
			context.Background(), publishingEffectCleanupTimeout,
		)
		defer cancelClose()
		_ = underlying.Close(closeCtx)
		return nil, fmt.Errorf("lock private object key: %w", err)
	}
	return &privateObjectFence{conn: conn, objectKey: objectKey}, nil
}

func (fence *privateObjectFence) Release() error {
	if fence == nil {
		return nil
	}
	fence.mu.Lock()
	defer fence.mu.Unlock()
	if fence.released {
		return nil
	}
	err := releaseCleanupEffectConnection(fence.conn, fence.objectKey)
	fence.conn = nil
	fence.released = true
	return err
}
