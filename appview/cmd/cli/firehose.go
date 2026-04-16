package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/firehose"
)

var firehoseSinceFlag string

var firehoseCmd = &cobra.Command{
	Use:   "firehose",
	Short: "Manage the firehose subscriber",
}

// Day one: stubs call firehose.NotImplemented directly — no DB pool, no
// loadDeps. This guarantees AC #11's "firehose: not yet implemented"
// error surfaces cleanly even if Postgres is down. When the real
// Subscriber impl lands, swap to `loadDeps(ctx)` → `deps.Firehose`.
var firehoseReplayCmd = &cobra.Command{
	Use:   "replay",
	Short: "Re-index firehose events since --since (stub until real subscriber lands)",
	RunE: func(cmd *cobra.Command, args []string) error {
		since := time.Time{}
		if firehoseSinceFlag != "" {
			t, err := time.Parse(time.RFC3339, firehoseSinceFlag)
			if err != nil {
				// Try date-only as a convenience.
				t, err = time.Parse("2006-01-02", firehoseSinceFlag)
				if err != nil {
					return fmt.Errorf("--since %q: want RFC3339 or YYYY-MM-DD", firehoseSinceFlag)
				}
			}
			since = t
		}
		return firehose.NotImplemented{}.Replay(context.Background(), since)
	},
}

func init() {
	firehoseReplayCmd.Flags().StringVar(&firehoseSinceFlag, "since", "", `replay events from this time (RFC3339 or YYYY-MM-DD)`)
	firehoseCmd.AddCommand(firehoseReplayCmd)
	rootCmd.AddCommand(firehoseCmd)
}
