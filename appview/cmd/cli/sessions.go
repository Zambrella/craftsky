package main

import (
	"context"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type sessionRevocationRunner func(context.Context, syntax.DID) error

func newSessionsCmd(revoke sessionRevocationRunner) *cobra.Command {
	cmd := &cobra.Command{Use: "sessions", Short: "Manage server-side user sessions", Args: cobra.NoArgs}
	cmd.AddCommand(&cobra.Command{
		Use:   "revoke DID",
		Short: "Sign a user out on all devices and queue OAuth credential cleanup",
		Long: "Revoke all ordinary CraftSky sessions for one DID through the existing logout-all flow. " +
			"Access is invalidated immediately; the running AppView workers perform OAuth and push cleanup. " +
			"An eligible credential for an accepted account deletion is preserved.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			did, err := syntax.ParseDID(args[0])
			if err != nil {
				return fmt.Errorf("DID must be a valid AT Protocol DID: %w", err)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			if err := revoke(ctx, did); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "sessions revoked for %s; OAuth and push cleanup queued\n", did)
			return err
		},
	})
	return cmd
}

func revokeSessionsForDID(ctx context.Context, did syntax.DID) error {
	env, err := parseEnvFlag()
	if err != nil {
		return err
	}
	cfg, err := loadCfgLight(env)
	if err != nil {
		return err
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	fencer, err := ownerlifecycle.NewFencer(pool, cfg.OwnerFenceAcquireTimeout)
	if err != nil {
		return err
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		return err
	}
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            cfg.CraftskySessionInactivity,
		ActivityWriteInterval: cfg.CraftskySessionActivityWriteInterval,
	})
	if err != nil {
		return err
	}
	service, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children,
		DeletionExemption: accountdeletion.NewStore(pool, time.Now),
	})
	if err != nil {
		return err
	}
	return service.RevokeAllForDID(ctx, did)
}

func init() {
	rootCmd.AddCommand(newSessionsCmd(revokeSessionsForDID))
}
