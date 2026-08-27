package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

const followerGrowthSeedDDL = `
CREATE TABLE follower_growth_snapshots (
    profile_did TEXT NOT NULL,
    snapshot_date DATE NOT NULL,
    follower_count BIGINT NOT NULL CHECK (follower_count >= 0),
    captured_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (profile_did, snapshot_date)
);
CREATE TABLE follower_growth_snapshot_runs (
    snapshot_date DATE NOT NULL PRIMARY KEY,
    completed_at TIMESTAMPTZ NOT NULL,
    captured_profile_count BIGINT NOT NULL CHECK (captured_profile_count >= 0)
);
`

func TestRunFollowerGrowthSeedCreatesDeterministicYearAndIsIdempotent(t *testing.T) {
	pool := testdb.WithSchema(t, followerGrowthSeedDDL)
	ctx := context.Background()
	now := time.Date(2026, 8, 27, 12, 30, 0, 0, time.FixedZone("test", -7*60*60))
	args := followerGrowthSeedArgs{UserDID: "did:plc:viewer", Now: now}

	if _, err := pool.Exec(ctx, `
		INSERT INTO follower_growth_snapshots(profile_did, snapshot_date, follower_count, captured_at)
		VALUES
			('did:plc:viewer', '2026-08-27', 999, '2026-08-27T12:00:00Z'),
			('did:plc:other', '2026-08-27', 7, '2026-08-27T00:00:02Z');
		INSERT INTO follower_growth_snapshot_runs(snapshot_date, completed_at, captured_profile_count)
		VALUES ('2026-08-27', '2026-08-27T00:00:02Z', 2)
	`); err != nil {
		t.Fatalf("insert existing snapshots: %v", err)
	}

	stats, err := runFollowerGrowthSeed(ctx, pool, args)
	if err != nil {
		t.Fatalf("runFollowerGrowthSeed: %v", err)
	}
	if stats.Snapshots != fakeFollowerGrowthDays {
		t.Fatalf("snapshots = %d, want %d", stats.Snapshots, fakeFollowerGrowthDays)
	}
	if got, want := stats.RangeStart.Format(time.DateOnly), "2025-08-26"; got != want {
		t.Fatalf("range start = %s, want %s", got, want)
	}
	if got, want := stats.RangeEnd.Format(time.DateOnly), "2026-08-27"; got != want {
		t.Fatalf("range end = %s, want %s", got, want)
	}

	if _, err := runFollowerGrowthSeed(ctx, pool, args); err != nil {
		t.Fatalf("runFollowerGrowthSeed second pass: %v", err)
	}

	var rows int
	var firstDate, lastDate time.Time
	var firstCount, lastCount, otherCount int64
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*), MIN(snapshot_date), MAX(snapshot_date),
		       (ARRAY_AGG(follower_count ORDER BY snapshot_date))[1],
		       (ARRAY_AGG(follower_count ORDER BY snapshot_date DESC))[1]
		FROM follower_growth_snapshots
		WHERE profile_did = 'did:plc:viewer'
	`).Scan(&rows, &firstDate, &lastDate, &firstCount, &lastCount); err != nil {
		t.Fatalf("read seeded range: %v", err)
	}
	if rows != fakeFollowerGrowthDays || firstCount != 24 || lastCount != stats.LatestCount {
		t.Fatalf("seeded range rows=%d first=%d last=%d stats=%+v", rows, firstCount, lastCount, stats)
	}
	if got := firstDate.Format(time.DateOnly); got != stats.RangeStart.Format(time.DateOnly) {
		t.Fatalf("stored first date = %s, want %s", got, stats.RangeStart.Format(time.DateOnly))
	}
	if got := lastDate.Format(time.DateOnly); got != stats.RangeEnd.Format(time.DateOnly) {
		t.Fatalf("stored last date = %s, want %s", got, stats.RangeEnd.Format(time.DateOnly))
	}
	if err := pool.QueryRow(ctx, `
		SELECT follower_count FROM follower_growth_snapshots
		WHERE profile_did = 'did:plc:other' AND snapshot_date = '2026-08-27'
	`).Scan(&otherCount); err != nil {
		t.Fatalf("read other user's snapshot: %v", err)
	}
	if otherCount != 7 {
		t.Fatalf("other user's count = %d, want 7", otherCount)
	}

	var gains, losses int
	if err := pool.QueryRow(ctx, `
		WITH changes AS (
			SELECT follower_count - LAG(follower_count) OVER (ORDER BY snapshot_date) AS change
			FROM follower_growth_snapshots
			WHERE profile_did = 'did:plc:viewer'
		)
		SELECT COUNT(*) FILTER (WHERE change > 0), COUNT(*) FILTER (WHERE change < 0)
		FROM changes
	`).Scan(&gains, &losses); err != nil {
		t.Fatalf("read seeded changes: %v", err)
	}
	if gains == 0 || losses == 0 {
		t.Fatalf("growth changes gains=%d losses=%d, want both", gains, losses)
	}
}

func TestRunFollowerGrowthSeedRejectsInvalidDID(t *testing.T) {
	pool := testdb.WithSchema(t, followerGrowthSeedDDL)
	_, err := runFollowerGrowthSeed(context.Background(), pool, followerGrowthSeedArgs{UserDID: "not-a-did"})
	if err == nil || !strings.Contains(err.Error(), "--user must be a DID") {
		t.Fatalf("error = %v, want invalid DID error", err)
	}
}

func TestPrintFollowerGrowthSeedStats(t *testing.T) {
	var out bytes.Buffer
	printFollowerGrowthSeedStats(&out, followerGrowthSeedStats{
		Snapshots:   367,
		RangeStart:  time.Date(2025, 8, 26, 0, 0, 0, 0, time.UTC),
		RangeEnd:    time.Date(2026, 8, 27, 0, 0, 0, 0, time.UTC),
		LatestCount: 154,
	})
	if got, want := out.String(), "seeded follower growth: snapshots=367 range=2025-08-26..2026-08-27 latest=154\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
