package followergrowth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const followerGrowthCaptureLockID int64 = 0x6372616674736b79

// LockCaptureTransaction serializes snapshot capture with terminal snapshot
// cleanup so private rows cannot commit after an owner's purge completes.
func LockCaptureTransaction(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, followerGrowthCaptureLockID); err != nil {
		return fmt.Errorf("lock follower growth capture: %w", err)
	}
	return nil
}

// Store persists and reads private follower-growth observations.
type Store struct {
	pool *pgxpool.Pool
}

// CaptureResult describes one daily snapshot capture attempt.
type CaptureResult struct {
	CapturedProfileCount int64
	AlreadyCompleted     bool
	LatestSuccessfulAge  *time.Duration
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Read returns owner history for a bounded response range.
func (s *Store) Read(ctx context.Context, owner syntax.DID, dateRange DateRange) (History, error) {
	rows, err := s.pool.Query(ctx, `
		WITH owner_history AS (
			SELECT snapshot_date, follower_count, captured_at
			FROM follower_growth_snapshots
			WHERE profile_did = $1
		), metadata AS (
			SELECT MIN(snapshot_date) AS available_from
			FROM owner_history
		), latest AS (
			SELECT snapshot_date, follower_count, captured_at
			FROM owner_history
			ORDER BY snapshot_date DESC
			LIMIT 1
		), ranged AS (
			SELECT snapshot_date, follower_count, captured_at
			FROM owner_history
			WHERE snapshot_date >= $2 AND snapshot_date <= $3
		)
		SELECT
			metadata.available_from,
			latest.snapshot_date,
			latest.follower_count,
			latest.captured_at,
			ranged.snapshot_date,
			ranged.follower_count,
			ranged.captured_at
		FROM metadata
		LEFT JOIN latest ON true
		LEFT JOIN ranged ON true
		ORDER BY ranged.snapshot_date
	`, owner, dateRange.Start, dateRange.End)
	if err != nil {
		return History{}, fmt.Errorf("query follower growth history: %w", err)
	}
	defer rows.Close()

	var history History
	for rows.Next() {
		var (
			availableFrom   sql.NullTime
			latestDate      sql.NullTime
			latestCount     sql.NullInt64
			latestCaptured  sql.NullTime
			pointDate       sql.NullTime
			pointCount      sql.NullInt64
			pointCapturedAt sql.NullTime
		)
		if err := rows.Scan(
			&availableFrom,
			&latestDate,
			&latestCount,
			&latestCaptured,
			&pointDate,
			&pointCount,
			&pointCapturedAt,
		); err != nil {
			return History{}, fmt.Errorf("scan follower growth history: %w", err)
		}
		if history.AvailableFrom == nil && availableFrom.Valid {
			value := utcDate(availableFrom.Time)
			history.AvailableFrom = &value
		}
		if history.Latest == nil && latestDate.Valid && latestCount.Valid && latestCaptured.Valid {
			history.Latest = &Snapshot{
				Date:          utcDate(latestDate.Time),
				FollowerCount: latestCount.Int64,
				CapturedAt:    latestCaptured.Time.UTC(),
			}
		}
		if pointDate.Valid && pointCount.Valid && pointCapturedAt.Valid {
			history.Snapshots = append(history.Snapshots, Snapshot{
				Date:          utcDate(pointDate.Time),
				FollowerCount: pointCount.Int64,
				CapturedAt:    pointCapturedAt.Time.UTC(),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return History{}, fmt.Errorf("iterate follower growth history: %w", err)
	}
	return history, nil
}

func utcDate(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func nonNegativeDuration(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}

// Capture records the canonical follower count for every current member.
func (s *Store) Capture(ctx context.Context, snapshotDate, capturedAt time.Time) (CaptureResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return CaptureResult{}, fmt.Errorf("begin follower growth capture: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := LockCaptureTransaction(ctx, tx); err != nil {
		return CaptureResult{}, err
	}

	var (
		existingCount       sql.NullInt64
		existingCompletedAt sql.NullTime
		latestCompletedAt   sql.NullTime
	)
	if err := tx.QueryRow(ctx, `
		SELECT current_run.captured_profile_count,
		       current_run.completed_at,
		       latest_run.completed_at
		FROM (
			SELECT MAX(completed_at) AS completed_at
			FROM follower_growth_snapshot_runs
		) AS latest_run
		LEFT JOIN follower_growth_snapshot_runs AS current_run
		  ON current_run.snapshot_date = $1
	`, snapshotDate).Scan(&existingCount, &existingCompletedAt, &latestCompletedAt); err != nil {
		return CaptureResult{}, fmt.Errorf("read follower growth run: %w", err)
	}
	var latestSuccessfulAge *time.Duration
	if latestCompletedAt.Valid {
		age := nonNegativeDuration(capturedAt.Sub(latestCompletedAt.Time))
		latestSuccessfulAge = &age
	}
	if existingCount.Valid && existingCompletedAt.Valid {
		if err := tx.Commit(ctx); err != nil {
			return CaptureResult{}, fmt.Errorf("commit completed follower growth capture: %w", err)
		}
		age := nonNegativeDuration(capturedAt.Sub(existingCompletedAt.Time))
		return CaptureResult{
			CapturedProfileCount: existingCount.Int64,
			AlreadyCompleted:     true,
			LatestSuccessfulAge:  &age,
		}, nil
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO follower_growth_snapshots (
			profile_did,
			snapshot_date,
			follower_count,
			captured_at
		)
		SELECT
			profile_did,
			$1,
			follower_count,
			$2
		FROM craftsky_profile_follower_counts
	`, snapshotDate, capturedAt)
	if err != nil {
		return CaptureResult{LatestSuccessfulAge: latestSuccessfulAge}, fmt.Errorf("capture follower growth snapshots: %w", err)
	}
	capturedProfileCount := tag.RowsAffected()
	if _, err := tx.Exec(ctx, `
		INSERT INTO follower_growth_snapshot_runs (
			snapshot_date,
			completed_at,
			captured_profile_count
		) VALUES ($1, $2, $3)
	`, snapshotDate, capturedAt, capturedProfileCount); err != nil {
		return CaptureResult{LatestSuccessfulAge: latestSuccessfulAge}, fmt.Errorf("record follower growth run: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return CaptureResult{}, fmt.Errorf("commit follower growth capture: %w", err)
	}
	age := time.Duration(0)
	return CaptureResult{
		CapturedProfileCount: capturedProfileCount,
		LatestSuccessfulAge:  &age,
	}, nil
}
