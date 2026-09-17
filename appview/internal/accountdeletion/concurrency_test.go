package accountdeletion

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func waitForDeletionResult[T any](t *testing.T, result <-chan T, description string) T {
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

func waitForDeletionAdvisoryWaiter(t *testing.T, pool *pgxpool.Pool, key int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_locks
				WHERE locktype='advisory'
				  AND classid=(($1::bigint >> 32) & 4294967295)::oid
				  AND objid=($1::bigint & 4294967295)::oid
				  AND NOT granted
			)
		`, key).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for advisory lock %d", key)
		case <-time.After(5 * time.Millisecond):
		}
	}
}
