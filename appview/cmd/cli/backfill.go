package main

import (
	"context"

	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/index"
)

// Same rationale as firehose.go: calls NotImplemented directly so AC #12
// ("indexer: not yet implemented") surfaces without needing the DB.
var backfillCmd = &cobra.Command{
	Use:   "backfill DID",
	Short: "Re-index all records for a DID (stub until indexer lands)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return index.NotImplemented{}.Backfill(context.Background(), args[0])
	},
}

func init() {
	rootCmd.AddCommand(backfillCmd)
}
