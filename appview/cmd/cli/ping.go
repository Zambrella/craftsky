package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the configured Postgres and print pool stats",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		deps, cleanup, err := loadDeps(ctx)
		if err != nil {
			return err
		}
		defer cleanup()

		// loadDeps already pings during db.Connect, but we run our own
		// Ping here so a DB that went down between Connect and this call
		// is caught honestly.
		if err := deps.DB.Ping(ctx); err != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
		s := deps.DB.Stat()
		fmt.Printf("ok: db up — acquired=%d idle=%d total=%d max=%d\n",
			s.AcquiredConns(), s.IdleConns(), s.TotalConns(), s.MaxConns())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pingCmd)
}
