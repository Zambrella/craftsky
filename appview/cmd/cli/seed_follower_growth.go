package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/followergrowth"
)

const fakeFollowerGrowthDays = 367

type followerGrowthSeedArgs struct {
	UserDID string
	Now     time.Time
}

type followerGrowthSeedStats struct {
	Snapshots   int
	RangeStart  time.Time
	RangeEnd    time.Time
	LatestCount int64
}

var followerGrowthSeedFlags followerGrowthSeedArgs

var seedFollowerGrowthCmd = &cobra.Command{
	Use:   "follower-growth --user DID",
	Short: "Seed one year of fake follower-growth history for a user",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		env, err := parseEnvFlag()
		if err != nil {
			return err
		}
		if env != app.EnvDev {
			return fmt.Errorf("seed follower-growth is dev-only; refusing to run with --env %s", env)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		deps, cleanup, err := loadDeps(ctx)
		if err != nil {
			return err
		}
		defer cleanup()

		seedArgs := followerGrowthSeedFlags
		seedArgs.Now = time.Now().UTC()
		stats, err := runFollowerGrowthSeed(ctx, deps.DB, seedArgs)
		if err != nil {
			return err
		}
		printFollowerGrowthSeedStats(cmd.OutOrStdout(), stats)
		return nil
	},
}

func init() {
	seedFollowerGrowthCmd.Flags().StringVar(&followerGrowthSeedFlags.UserDID, "user", "", "DID whose follower-growth history should be populated")
	_ = seedFollowerGrowthCmd.MarkFlagRequired("user")
	seedCmd.AddCommand(seedFollowerGrowthCmd)
}

func runFollowerGrowthSeed(ctx context.Context, pool *pgxpool.Pool, args followerGrowthSeedArgs) (followerGrowthSeedStats, error) {
	if pool == nil {
		return followerGrowthSeedStats{}, fmt.Errorf("db pool is required")
	}
	userDID, err := syntax.ParseDID(args.UserDID)
	if err != nil {
		return followerGrowthSeedStats{}, fmt.Errorf("--user must be a DID: %w", err)
	}
	if args.Now.IsZero() {
		args.Now = time.Now().UTC()
	}

	rangeEnd := utcSeedDate(args.Now)
	rangeStart := rangeEnd.AddDate(0, 0, -(fakeFollowerGrowthDays - 1))
	if _, err := followergrowth.NewStore(pool).Capture(ctx, rangeEnd, args.Now.UTC()); err != nil {
		return followerGrowthSeedStats{}, fmt.Errorf("complete current follower-growth capture before seeding: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return followerGrowthSeedStats{}, fmt.Errorf("begin follower-growth seed transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := followergrowth.LockCaptureTransaction(ctx, tx); err != nil {
		return followerGrowthSeedStats{}, err
	}

	batch := &pgx.Batch{}
	count := int64(24)
	changes := [...]int64{0, 1, 0, 2, -1, 0, 1, 0, 0, 2, -1, 1, 0, 0}
	for day := 0; day < fakeFollowerGrowthDays; day++ {
		if day > 0 {
			count += changes[day%len(changes)]
		}
		date := rangeStart.AddDate(0, 0, day)
		batch.Queue(`
			INSERT INTO follower_growth_snapshots (
				profile_did, snapshot_date, follower_count, captured_at
			) VALUES ($1, $2, $3, $4)
			ON CONFLICT (profile_did, snapshot_date) DO UPDATE SET
				follower_count = EXCLUDED.follower_count,
				captured_at = EXCLUDED.captured_at
		`, userDID, date, count, date.Add(2*time.Second))
	}

	results := tx.SendBatch(ctx, batch)
	for day := 0; day < batch.Len(); day++ {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return followerGrowthSeedStats{}, fmt.Errorf("seed follower-growth snapshot %d: %w", day+1, err)
		}
	}
	if err := results.Close(); err != nil {
		return followerGrowthSeedStats{}, fmt.Errorf("seed follower-growth snapshots: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return followerGrowthSeedStats{}, fmt.Errorf("commit follower-growth seed transaction: %w", err)
	}

	return followerGrowthSeedStats{
		Snapshots:   fakeFollowerGrowthDays,
		RangeStart:  rangeStart,
		RangeEnd:    rangeEnd,
		LatestCount: count,
	}, nil
}

func utcSeedDate(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func printFollowerGrowthSeedStats(out io.Writer, stats followerGrowthSeedStats) {
	fmt.Fprintf(out, "seeded follower growth: snapshots=%d range=%s..%s latest=%d\n",
		stats.Snapshots,
		stats.RangeStart.Format(time.DateOnly),
		stats.RangeEnd.Format(time.DateOnly),
		stats.LatestCount,
	)
}
