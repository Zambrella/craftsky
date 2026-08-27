package followergrowth

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const storeIntegrationBaseDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT        NOT NULL PRIMARY KEY,
    record_cid TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_follows (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);
`

func TestStoreCaptureCanonicalCounts(t *testing.T) {
	pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
	ctx := context.Background()
	applyFollowerGrowthMigration(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid'),
			('did:plc:carol', 'carol-cid');
		INSERT INTO atproto_follows (
			uri, did, rkey, cid, subject_did, record, created_at
		) VALUES
			('at://did:plc:alice/app.bsky.graph.follow/bob', 'did:plc:alice', 'bob', 'follow-1', 'did:plc:bob', '{}', '2026-08-20T00:00:00Z'),
			('at://did:plc:dana/app.bsky.graph.follow/bob', 'did:plc:dana', 'bob', 'follow-2', 'did:plc:bob', '{}', '2026-08-20T00:00:00Z'),
			('at://did:plc:alice/app.bsky.graph.follow/dana', 'did:plc:alice', 'dana', 'follow-3', 'did:plc:dana', '{}', '2026-08-20T00:00:00Z');
	`); err != nil {
		t.Fatalf("seed membership graph: %v", err)
	}

	snapshotDate := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	capturedAt := snapshotDate.Add(2 * time.Second)
	result, err := NewStore(pool).Capture(ctx, snapshotDate, capturedAt)
	if err != nil {
		t.Fatalf("capture follower growth: %v", err)
	}
	if result.CapturedProfileCount != 3 {
		t.Fatalf("captured profile count = %d, want 3", result.CapturedProfileCount)
	}

	rows, err := pool.Query(ctx, `
		SELECT profile_did, follower_count
		FROM follower_growth_snapshots
		WHERE snapshot_date = $1
		ORDER BY profile_did
	`, snapshotDate)
	if err != nil {
		t.Fatalf("query snapshots: %v", err)
	}
	defer rows.Close()

	got := make(map[string]int64)
	for rows.Next() {
		var (
			did   string
			count int64
		)
		if err := rows.Scan(&did, &count); err != nil {
			t.Fatalf("scan snapshot: %v", err)
		}
		got[did] = count
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate snapshots: %v", err)
	}
	want := map[string]int64{
		"did:plc:alice": 0,
		"did:plc:bob":   1,
		"did:plc:carol": 0,
	}
	if len(got) != len(want) {
		t.Fatalf("snapshots = %v, want %v", got, want)
	}
	for did, wantCount := range want {
		if got[did] != wantCount {
			t.Errorf("snapshot count for %s = %d, want %d", did, got[did], wantCount)
		}
	}
	if _, exists := got["did:plc:dana"]; exists {
		t.Fatal("captured snapshot for non-member did:plc:dana")
	}
}

func TestStoreCaptureIsAtomicAndConcurrent(t *testing.T) {
	t.Run("failed run retains latest successful age", func(t *testing.T) {
		pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
		ctx := context.Background()
		applyFollowerGrowthMigration(t, pool)
		seedCaptureMembers(t, pool)
		previousDate := growthDate(2026, time.August, 24)
		previousCompletedAt := previousDate.Add(2 * time.Second)
		if _, err := pool.Exec(ctx, `
			INSERT INTO follower_growth_snapshot_runs (
				snapshot_date, completed_at, captured_profile_count
			) VALUES ($1, $2, 3)
		`, previousDate, previousCompletedAt); err != nil {
			t.Fatalf("seed previous completed run: %v", err)
		}
		if _, err := pool.Exec(ctx, `
			CREATE FUNCTION reject_current_follower_growth_run()
			RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				RAISE EXCEPTION 'injected current run failure';
			END;
			$$;
			CREATE TRIGGER reject_current_follower_growth_run
			BEFORE INSERT ON follower_growth_snapshot_runs
			FOR EACH STATEMENT EXECUTE FUNCTION reject_current_follower_growth_run();
		`); err != nil {
			t.Fatalf("install current-run failure trigger: %v", err)
		}

		currentDate := previousDate.AddDate(0, 0, 1)
		capturedAt := currentDate.Add(5*time.Minute + 2*time.Second)
		result, err := NewStore(pool).Capture(ctx, currentDate, capturedAt)
		if err == nil {
			t.Fatal("capture succeeded despite run ledger failure")
		}
		if result.LatestSuccessfulAge == nil || *result.LatestSuccessfulAge != 24*time.Hour+5*time.Minute {
			t.Fatalf("latest successful age = %v, want 24h5m", result.LatestSuccessfulAge)
		}
		assertGrowthRowsForDate(t, pool, currentDate, 0)
	})

	t.Run("run ledger failure rolls back snapshots", func(t *testing.T) {
		pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
		ctx := context.Background()
		applyFollowerGrowthMigration(t, pool)
		seedCaptureMembers(t, pool)
		if _, err := pool.Exec(ctx, `
			CREATE FUNCTION reject_follower_growth_run()
			RETURNS trigger LANGUAGE plpgsql AS $$
			BEGIN
				RAISE EXCEPTION 'injected run ledger failure';
			END;
			$$;
			CREATE TRIGGER reject_follower_growth_run
			BEFORE INSERT ON follower_growth_snapshot_runs
			FOR EACH STATEMENT EXECUTE FUNCTION reject_follower_growth_run();
		`); err != nil {
			t.Fatalf("install failure trigger: %v", err)
		}

		date := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
		if _, err := NewStore(pool).Capture(ctx, date, date.Add(2*time.Second)); err == nil {
			t.Fatal("capture succeeded despite run ledger failure")
		}
		assertGrowthRowCount(t, pool, "follower_growth_snapshots", 0)
		assertGrowthRowCount(t, pool, "follower_growth_snapshot_runs", 0)
	})

	t.Run("concurrent attempts produce one complete logical run", func(t *testing.T) {
		pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
		ctx := context.Background()
		applyFollowerGrowthMigration(t, pool)
		seedCaptureMembers(t, pool)

		date := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
		start := make(chan struct{})
		results := make(chan CaptureResult, 2)
		errs := make(chan error, 2)
		var ready sync.WaitGroup
		ready.Add(2)
		for range 2 {
			go func() {
				ready.Done()
				<-start
				result, err := NewStore(pool).Capture(ctx, date, date.Add(2*time.Second))
				results <- result
				errs <- err
			}()
		}
		ready.Wait()
		close(start)

		var alreadyCompleted int
		for range 2 {
			if err := <-errs; err != nil {
				t.Fatalf("concurrent capture: %v", err)
			}
			if result := <-results; result.AlreadyCompleted {
				alreadyCompleted++
			}
		}
		if alreadyCompleted != 1 {
			t.Fatalf("already-completed results = %d, want 1", alreadyCompleted)
		}
		assertGrowthRowCount(t, pool, "follower_growth_snapshots", 3)
		assertGrowthRowCount(t, pool, "follower_growth_snapshot_runs", 1)
	})
}

func TestStoreCaptureObservesProjectionChangesOnlyOnLaterDates(t *testing.T) {
	pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
	ctx := context.Background()
	applyFollowerGrowthMigration(t, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid');
	`); err != nil {
		t.Fatalf("seed initial members: %v", err)
	}

	store := NewStore(pool)
	firstDate := growthDate(2026, time.August, 25)
	if _, err := store.Capture(ctx, firstDate, firstDate.Add(2*time.Second)); err != nil {
		t.Fatalf("capture first date: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:carol', 'carol-cid');
		INSERT INTO atproto_follows (
			uri, did, rkey, cid, subject_did, record, created_at
		) VALUES (
			'at://did:plc:alice/app.bsky.graph.follow/bob',
			'did:plc:alice', 'bob', 'follow-1', 'did:plc:bob', '{}', now()
		);
	`); err != nil {
		t.Fatalf("mutate membership and follows: %v", err)
	}
	assertSnapshotCount(t, pool, firstDate, "did:plc:bob", 0)
	assertGrowthRowsForDate(t, pool, firstDate, 2)

	result, err := store.Capture(ctx, firstDate, firstDate.Add(time.Hour))
	if err != nil {
		t.Fatalf("retry completed date: %v", err)
	}
	if !result.AlreadyCompleted {
		t.Fatal("same-date retry was not reported already completed")
	}
	assertGrowthRowsForDate(t, pool, firstDate, 2)

	secondDate := firstDate.AddDate(0, 0, 1)
	if _, err := store.Capture(ctx, secondDate, secondDate.Add(2*time.Second)); err != nil {
		t.Fatalf("capture second date: %v", err)
	}
	assertSnapshotCount(t, pool, secondDate, "did:plc:bob", 1)
	assertGrowthRowsForDate(t, pool, secondDate, 3)

	if _, err := pool.Exec(ctx, `
		DELETE FROM atproto_follows WHERE did = 'did:plc:alice' AND subject_did = 'did:plc:bob';
		DELETE FROM craftsky_profiles WHERE did = 'did:plc:alice';
	`); err != nil {
		t.Fatalf("depart and unfollow: %v", err)
	}
	assertSnapshotCount(t, pool, secondDate, "did:plc:bob", 1)

	thirdDate := secondDate.AddDate(0, 0, 1)
	if _, err := store.Capture(ctx, thirdDate, thirdDate.Add(2*time.Second)); err != nil {
		t.Fatalf("capture third date: %v", err)
	}
	assertSnapshotCount(t, pool, thirdDate, "did:plc:bob", 0)
	assertGrowthRowsForDate(t, pool, thirdDate, 2)

	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-rejoined-cid');
		INSERT INTO atproto_follows (
			uri, did, rkey, cid, subject_did, record, created_at
		) VALUES (
			'at://did:plc:alice/app.bsky.graph.follow/bob-rejoined',
			'did:plc:alice', 'bob-rejoined', 'follow-rejoined', 'did:plc:bob', '{}', now()
		);
	`); err != nil {
		t.Fatalf("rejoin and refollow: %v", err)
	}
	assertSnapshotCount(t, pool, thirdDate, "did:plc:bob", 0)
	fourthDate := thirdDate.AddDate(0, 0, 1)
	if _, err := store.Capture(ctx, fourthDate, fourthDate.Add(2*time.Second)); err != nil {
		t.Fatalf("capture fourth date after rejoin: %v", err)
	}
	assertSnapshotCount(t, pool, fourthDate, "did:plc:bob", 1)
	assertGrowthRowsForDate(t, pool, fourthDate, 3)
}

func TestStoreReadReturnsGlobalMetadataAndBoundedSparseHistory(t *testing.T) {
	pool := testdb.WithSchema(t, storeIntegrationBaseDDL)
	ctx := context.Background()
	applyFollowerGrowthMigration(t, pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did, snapshot_date, follower_count, captured_at
		) VALUES
			('did:plc:alice', '2026-06-01', 5, '2026-06-01T00:00:02Z'),
			('did:plc:alice', '2026-08-20', 10, '2026-08-20T00:00:02Z'),
			('did:plc:alice', '2026-08-22', 12, '2026-08-22T00:00:02Z'),
			('did:plc:alice', '2026-08-24', 11, '2026-08-24T00:00:02Z');
	`); err != nil {
		t.Fatalf("seed sparse history: %v", err)
	}

	dateRange := DateRange{
		Start: growthDate(2026, time.August, 19),
		End:   growthDate(2026, time.August, 25),
	}
	history, err := NewStore(pool).Read(ctx, syntax.DID("did:plc:alice"), dateRange)
	if err != nil {
		t.Fatalf("read owner history: %v", err)
	}
	if history.AvailableFrom == nil || !history.AvailableFrom.Equal(growthDate(2026, time.June, 1)) {
		t.Fatalf("availableFrom = %v, want 2026-06-01", history.AvailableFrom)
	}
	if history.Latest == nil || !history.Latest.Date.Equal(growthDate(2026, time.August, 24)) || history.Latest.FollowerCount != 11 {
		t.Fatalf("latest = %+v, want 2026-08-24 count 11", history.Latest)
	}
	if len(history.Snapshots) != 3 {
		t.Fatalf("in-range snapshots = %+v, want three", history.Snapshots)
	}
	wantDates := []time.Time{
		growthDate(2026, time.August, 20),
		growthDate(2026, time.August, 22),
		growthDate(2026, time.August, 24),
	}
	for i, want := range wantDates {
		if !history.Snapshots[i].Date.Equal(want) {
			t.Errorf("snapshot %d date = %s, want %s", i, history.Snapshots[i].Date, want)
		}
	}

	empty, err := NewStore(pool).Read(ctx, syntax.DID("did:plc:bob"), dateRange)
	if err != nil {
		t.Fatalf("read no-history owner: %v", err)
	}
	if empty.AvailableFrom != nil || empty.Latest != nil || len(empty.Snapshots) != 0 {
		t.Fatalf("no-history result = %+v, want empty", empty)
	}

	leapRange := PeriodOneYear.Range(time.Date(2028, time.February, 29, 12, 0, 0, 0, time.UTC))
	leapHistory, err := NewStore(pool).Read(ctx, syntax.DID("did:plc:alice"), leapRange)
	if err != nil {
		t.Fatalf("read leap range: %v", err)
	}
	leapGrowth := BuildSeries(leapHistory, leapRange)
	if len(leapGrowth.Points) != 367 || !leapGrowth.Points[0].Date.Equal(growthDate(2027, time.February, 28)) {
		t.Fatalf("leap series has %d points from %s, want 367 from 2027-02-28", len(leapGrowth.Points), leapGrowth.Points[0].Date)
	}
}

func seedCaptureMembers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid'),
			('did:plc:carol', 'carol-cid');
	`); err != nil {
		t.Fatalf("seed capture members: %v", err)
	}
}

func assertGrowthRowCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}

func assertGrowthRowsForDate(t *testing.T, pool *pgxpool.Pool, date time.Time, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM follower_growth_snapshots WHERE snapshot_date = $1
	`, date).Scan(&got); err != nil {
		t.Fatalf("count snapshots for %s: %v", date, err)
	}
	if got != want {
		t.Fatalf("snapshot count for %s = %d, want %d", date, got, want)
	}
}

func assertSnapshotCount(t *testing.T, pool *pgxpool.Pool, date time.Time, did string, want int64) {
	t.Helper()
	var got int64
	if err := pool.QueryRow(context.Background(), `
		SELECT follower_count
		FROM follower_growth_snapshots
		WHERE snapshot_date = $1 AND profile_did = $2
	`, date, did).Scan(&got); err != nil {
		t.Fatalf("read snapshot for %s on %s: %v", did, date, err)
	}
	if got != want {
		t.Fatalf("snapshot for %s on %s = %d, want %d", did, date, got, want)
	}
}

func applyFollowerGrowthMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	up, err := os.ReadFile("../../migrations/000060_follower_growth_snapshots.up.sql")
	if err != nil {
		t.Fatalf("read follower growth migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(up)); err != nil {
		t.Fatalf("apply follower growth migration: %v", err)
	}
}
