package scheduledposts

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func waitForScheduledSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	waitForScheduledResult(t, signal, description)
}

func waitForScheduledResult[T any](t *testing.T, result <-chan T, description string) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
		var zero T
		return zero
	}
}

func waitForScheduledAdvisoryWaiter(t *testing.T, pool *pgxpool.Pool, premature <-chan error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_locks
				WHERE locktype='advisory' AND NOT granted
			)
		`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case err := <-premature:
			t.Fatalf("operation completed before advisory lock contention was observed: %v", err)
		case <-ctx.Done():
			t.Fatal("timed out waiting for advisory lock contention")
		case <-time.After(5 * time.Millisecond):
		}
	}
}
