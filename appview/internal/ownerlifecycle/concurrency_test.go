package ownerlifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func waitForTestSignal(t *testing.T, signal <-chan struct{}, description string) {
	t.Helper()
	waitForTestResult(t, signal, description)
}

func waitForTestResult[T any](t *testing.T, result <-chan T, description string) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
		var zero T
		return zero
	}
}

func waitForTransactionWaiter(t *testing.T, pool *pgxpool.Pool, xid string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	for {
		var waiting bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_locks
				WHERE locktype='transactionid'
				  AND transactionid=$1::xid
				  AND NOT granted
			)
		`, xid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for transaction %s contention", xid)
		case <-time.After(5 * time.Millisecond):
		}
	}
}
