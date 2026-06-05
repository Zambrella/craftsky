package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/api"
)

type identityCacheBackfillStats struct {
	Candidates int
	Upserted   int
	Failed     int
}

type identityCacheBackfillRunner func(context.Context, int) (identityCacheBackfillStats, error)

func newIdentityCacheCmd(runBackfill identityCacheBackfillRunner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "identity-cache",
		Short: "Identity handle-cache operations",
	}
	limit := 100
	backfillCmd := &cobra.Command{
		Use:   "backfill",
		Short: "Backfill cached handles for existing Craftsky profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("limit must be positive")
			}
			stats, err := runBackfill(cmd.Context(), limit)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "identity-cache backfill: candidates=%d upserted=%d failed=%d\n", stats.Candidates, stats.Upserted, stats.Failed)
			return nil
		},
	}
	backfillCmd.Flags().IntVar(&limit, "limit", 100, "maximum Craftsky profile DIDs to resolve")
	cmd.AddCommand(backfillCmd)
	return cmd
}

func init() {
	rootCmd.AddCommand(newIdentityCacheCmd(func(ctx context.Context, limit int) (identityCacheBackfillStats, error) {
		deps, cleanup, err := loadDeps(ctx)
		if err != nil {
			return identityCacheBackfillStats{}, err
		}
		defer cleanup()
		return runIdentityCacheBackfill(ctx, api.NewIdentityCacheStore(deps.DB), deps.HandleResolver, limit, time.Now().UTC())
	}))
}

func runIdentityCacheBackfill(ctx context.Context, store *api.IdentityCacheStore, resolver api.HandleResolver, limit int, now time.Time) (identityCacheBackfillStats, error) {
	dids, err := store.BackfillCandidateDIDs(ctx, limit, now)
	if err != nil {
		return identityCacheBackfillStats{}, err
	}
	stats := identityCacheBackfillStats{Candidates: len(dids)}
	for _, did := range dids {
		handle, err := resolver.ResolveHandle(ctx, did)
		if err != nil || handle.String() == "" {
			stats.Failed++
			continue
		}
		if err := store.Upsert(ctx, did, handle, now); err != nil {
			return stats, err
		}
		stats.Upserted++
	}
	return stats, nil
}
