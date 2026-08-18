package ingestion_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/testdb"
)

func TestRepositoryReconciliationSurvivesFailureRestartAndLeaseReclaim(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	now := time.Date(2026, 8, 14, 14, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	store, err := ingestion.NewStore(pool, clock)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	did := syntax.DID("did:plc:repository-owner")
	for attempt := 0; attempt < 2; attempt++ {
		if err := store.EnqueueRepositoryJob(ctx, did, ingestion.RepositoryJobTapAddRepo); err != nil {
			t.Fatalf("enqueue repository attempt %d: %v", attempt, err)
		}
	}
	assertRepositoryJobCount(t, pool, did, 1)

	firstClaims, err := store.ClaimRepositoryJobs(ctx, ingestion.RepositoryClaimRequest{
		Worker: "repository-worker-a", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(firstClaims) != 1 {
		t.Fatalf("first repository claims=%+v err=%v", firstClaims, err)
	}
	remoteErr := errors.New("Tap AddRepo unavailable")
	if err := store.RunRepositoryJob(ctx, firstClaims[0], func(context.Context, ingestion.RepositoryClaim) (string, error) {
		return "", remoteErr
	}); !errors.Is(err, remoteErr) {
		t.Fatalf("repository failure error=%v", err)
	}

	// Recreating the store represents an AppView restart. The failed job is
	// persisted with backoff and can be reclaimed later.
	now = now.Add(2 * time.Second)
	restarted, err := ingestion.NewStore(pool, clock)
	if err != nil {
		t.Fatalf("restart ingestion store: %v", err)
	}
	secondClaims, err := restarted.ClaimRepositoryJobs(ctx, ingestion.RepositoryClaimRequest{
		Worker: "repository-worker-b", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(secondClaims) != 1 {
		t.Fatalf("second repository claims=%+v err=%v", secondClaims, err)
	}
	if secondClaims[0].Attempts != 2 {
		t.Fatalf("repository attempts=%d, want 2", secondClaims[0].Attempts)
	}
	if err := restarted.RunRepositoryJob(ctx, secondClaims[0], func(_ context.Context, claim ingestion.RepositoryClaim) (string, error) {
		if claim.DID != did || claim.Kind != ingestion.RepositoryJobTapAddRepo {
			t.Fatalf("repository claim=%+v", claim)
		}
		return "3m-authoritative", nil
	}); err != nil {
		t.Fatalf("complete repository job: %v", err)
	}
	job, err := restarted.RepositoryJob(ctx, did, ingestion.RepositoryJobTapAddRepo)
	if err != nil || job.State != "complete" || job.AuthoritativeRevision != "3m-authoritative" {
		t.Fatalf("completed repository job=%+v err=%v", job, err)
	}
}

func assertRepositoryJobCount(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, did syntax.DID, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_repository_jobs WHERE did=$1`, did).Scan(&count); err != nil || count != want {
		t.Fatalf("repository jobs=%d want=%d err=%v", count, want, err)
	}
}
