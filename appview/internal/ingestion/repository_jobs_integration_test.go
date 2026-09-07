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
	"social.craftsky/appview/internal/observability"
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
	remoteCause := errors.New("repository snapshot trust failure")
	remoteErr := repositoryJobReasonError{cause: remoteCause, reason: "signature_invalid"}
	if err := store.RunRepositoryJob(ctx, firstClaims[0], func(context.Context, ingestion.RepositoryClaim) (string, error) {
		return "", remoteErr
	}); !errors.Is(err, remoteCause) {
		t.Fatalf("repository failure error=%v", err)
	}
	pending, err := store.RepositoryJob(ctx, did, ingestion.RepositoryJobTapAddRepo)
	if err != nil || pending.State != "pending" || pending.LastReasonCode != "signature_invalid" {
		t.Fatalf("failed repository job=%+v err=%v, want retryable signature failure", pending, err)
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

func TestRepositorySnapshotTrustFailureRemainsDurablyRetryable(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	now := time.Date(2026, 8, 14, 15, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	did := syntax.DID("did:plc:snapshot-retry-owner")
	ctx := context.Background()
	if err := store.EnqueueRepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue snapshot job: %v", err)
	}
	claims, err := store.ClaimRepositoryJobs(ctx, ingestion.RepositoryClaimRequest{
		Worker: "snapshot-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("snapshot claims=%+v err=%v", claims, err)
	}
	cause := errors.New("invalid repository signature")
	err = store.RunRepositoryJob(ctx, claims[0], func(context.Context, ingestion.RepositoryClaim) (string, error) {
		return "", repositoryJobReasonError{cause: cause, reason: "signature_invalid"}
	})
	if !errors.Is(err, cause) {
		t.Fatalf("snapshot job error=%v, want trust failure", err)
	}
	job, err := store.RepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile)
	if err != nil || job.State != "pending" || job.Attempts != 1 ||
		job.LastReasonCode != "signature_invalid" || !job.NextAttemptAt.After(now) || job.LastSuccessfulAt != nil {
		t.Fatalf("retryable snapshot job=%+v err=%v", job, err)
	}
}

func TestRepositoryWorkerEmitsAlertableSecretFreeRepairHealth(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	if err := store.EnqueueRepositoryJob(context.Background(), syntax.DID("did:plc:metric-canary"), ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue repository job: %v", err)
	}
	now = now.Add(16 * time.Minute)
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{MetricRecorder: recorder})
	worker, err := ingestion.NewRepositoryWorker(ingestion.RepositoryWorkerConfig{
		Store: store,
		Handler: func(context.Context, ingestion.RepositoryClaim) (string, error) {
			return "", repositoryJobReasonError{cause: errors.New("secret oauth-token did:plc:metric-canary"), reason: "signature_invalid"}
		},
		WorkerID: "metric-worker", PollInterval: time.Second, LeaseDuration: time.Minute,
		BatchSize: 1, BackoffMin: time.Second, BackoffMax: time.Minute,
		AlertAge: 15 * time.Minute, AlertAttempts: 5, Observer: observer,
	})
	if err != nil {
		t.Fatalf("new repository worker: %v", err)
	}
	if count, err := worker.RunOnce(context.Background()); err == nil || count != 1 {
		t.Fatalf("run repository worker count=%d err=%v, want one retry", count, err)
	}

	calls := recorder.Calls()
	var sawAlert, sawFailure bool
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("unsafe migration metric %#v: %v", call, err)
		}
		if call.Name == "craftsky_appview_repository_repair_alert" && call.Value == 1 {
			sawAlert = true
		}
		if call.Name == "craftsky_appview_repository_repairs_total" && call.Attributes["reason"] == "signature_invalid" {
			sawFailure = true
		}
	}
	if !sawAlert || !sawFailure {
		t.Fatalf("migration metrics alert=%v failure=%v calls=%#v", sawAlert, sawFailure, calls)
	}
}

func TestRepositoryRepairReenqueueStartsFreshConsecutiveAttemptEpisode(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	now := time.Date(2026, 8, 14, 17, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	did := syntax.DID("did:plc:fresh-repair-episode")
	if err := store.EnqueueRepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue old repair: %v", err)
	}
	for attempt := 1; attempt <= 6; attempt++ {
		claims, claimErr := store.ClaimRepositoryJobs(ctx, ingestion.RepositoryClaimRequest{
			Worker: "episode-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
		})
		if claimErr != nil || len(claims) != 1 {
			t.Fatalf("claim attempt %d: claims=%+v err=%v", attempt, claims, claimErr)
		}
		runErr := store.RunRepositoryJob(ctx, claims[0], func(context.Context, ingestion.RepositoryClaim) (string, error) {
			if attempt < 6 {
				return "", repositoryJobReasonError{cause: errors.New("retry repair"), reason: "remote_unavailable"}
			}
			return "revision-old", nil
		})
		if attempt < 6 && runErr == nil {
			t.Fatalf("repair attempt %d unexpectedly succeeded", attempt)
		}
		if attempt == 6 && runErr != nil {
			t.Fatalf("complete old repair: %v", runErr)
		}
		now = now.Add(time.Second)
	}
	now = now.Add(time.Hour)
	if err := store.EnqueueRepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue fresh repair episode: %v", err)
	}
	health, err := store.RepositoryBacklogHealth(ctx)
	if err != nil {
		t.Fatalf("inspect fresh repair health: %v", err)
	}
	if health.Pending != 1 || health.MaxAttempts != 0 || health.OldestAge != 0 {
		t.Fatalf("fresh repair health=%+v, want one age-zero job with zero consecutive attempts", health)
	}
}

func TestRepositoryWorkerAlertsInCycleThatFifthAttemptFails(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	now := time.Date(2026, 8, 14, 18, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	if err := store.EnqueueRepositoryJob(context.Background(), syntax.DID("did:plc:fifth-repair-attempt"), ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatalf("enqueue repository job: %v", err)
	}
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{MetricRecorder: recorder})
	worker, err := ingestion.NewRepositoryWorker(ingestion.RepositoryWorkerConfig{
		Store: store,
		Handler: func(context.Context, ingestion.RepositoryClaim) (string, error) {
			return "", repositoryJobReasonError{cause: errors.New("retry repair"), reason: "remote_unavailable"}
		},
		WorkerID: "fifth-attempt-worker", PollInterval: time.Second, LeaseDuration: time.Minute,
		BatchSize: 1, BackoffMin: time.Second, BackoffMax: time.Second,
		AlertAge: time.Hour, AlertAttempts: 5, Observer: observer,
	})
	if err != nil {
		t.Fatalf("new repository worker: %v", err)
	}
	for attempt := 1; attempt <= 5; attempt++ {
		if count, runErr := worker.RunOnce(context.Background()); runErr == nil || count != 1 {
			t.Fatalf("run attempt %d count=%d err=%v", attempt, count, runErr)
		}
		now = now.Add(time.Second)
	}
	var alert bool
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_repository_repair_alert" && call.Value == 1 {
			alert = true
		}
	}
	if !alert {
		t.Fatal("fifth consecutive failure did not emit an alert in the same worker cycle")
	}
}

type repositoryJobReasonError struct {
	cause  error
	reason string
}

func (err repositoryJobReasonError) Error() string      { return err.cause.Error() }
func (err repositoryJobReasonError) Unwrap() error      { return err.cause }
func (err repositoryJobReasonError) ReasonCode() string { return err.reason }

func assertRepositoryJobCount(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, did syntax.DID, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_repository_jobs WHERE did=$1`, did).Scan(&count); err != nil || count != want {
		t.Fatalf("repository jobs=%d want=%d err=%v", count, want, err)
	}
}
