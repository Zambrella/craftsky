package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/accountdeletion"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/ownerlifecycle"
)

type sessionRevocationRunner func(context.Context, syntax.DID) error

func newSessionsCmd(revoke sessionRevocationRunner, revokeAll func(context.Context) (int, error)) *cobra.Command {
	cmd := &cobra.Command{Use: "sessions", Short: "Manage server-side user sessions", Args: cobra.NoArgs}
	var all bool
	revokeCmd := &cobra.Command{
		Use:   "revoke [DID]",
		Short: "Sign a user out on all devices and queue OAuth credential cleanup",
		Long: "Revoke all ordinary CraftSky sessions for one DID, or every non-terminal account with --all, through the existing logout-all flow. " +
			"Access is invalidated immediately; the running AppView workers perform OAuth and push cleanup. " +
			"An eligible credential for an accepted account deletion is preserved.",
		Args: func(cmd *cobra.Command, args []string) error {
			if all && len(args) == 0 {
				return nil
			}
			if !all && len(args) == 1 {
				return nil
			}
			return fmt.Errorf("provide exactly one DID or --all, never both")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if all {
				count, err := revokeAll(cmd.Context())
				if err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "sessions revoked for %d accounts; bulk revocation incomplete\n", count)
					return err
				}
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "sessions revoked for all %d accounts; OAuth and push cleanup queued\n", count)
				return err
			}
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
	}
	revokeCmd.Flags().BoolVar(&all, "all", false, "Sign out every non-terminal account on all devices")
	cmd.AddCommand(revokeCmd)
	return cmd
}

func revokeSessionsForDID(ctx context.Context, did syntax.DID) error {
	pool, revoke, err := connectSessionRevoker(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()
	return revoke(ctx, did)
}

func revokeSessionsForAll(ctx context.Context) (int, error) {
	pool, revoke, err := connectSessionRevoker(ctx)
	if err != nil {
		return 0, err
	}
	defer pool.Close()
	rows, err := pool.Query(ctx, `SELECT owner_did FROM owner_lifecycles WHERE state <> 'terminal' ORDER BY owner_did`)
	if err != nil {
		return 0, fmt.Errorf("list session owners: %w", err)
	}
	var owners []syntax.DID
	for rows.Next() {
		var owner syntax.DID
		if err := rows.Scan(&owner); err != nil {
			rows.Close()
			return 0, err
		}
		owners = append(owners, owner)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return revokeSessionOwners(ctx, owners, revoke)
}

// Snapshot membership is fixed before revocation. Each owner keeps the existing
// lifecycle fence and transaction; failures do not prevent trying other owners.
func revokeSessionOwners(ctx context.Context, owners []syntax.DID, revoke sessionRevocationRunner) (int, error) {
	var failures []error
	count := 0
	for _, owner := range owners {
		if err := ctx.Err(); err != nil {
			failures = append(failures, err)
			break
		}
		ownerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := revoke(ownerCtx, owner)
		cancel()
		if err != nil {
			failures = append(failures, fmt.Errorf("revoke sessions for %s: %w", owner, err))
			continue
		}
		count++
	}
	return count, errors.Join(failures...)
}

func connectSessionRevoker(ctx context.Context) (*pgxpool.Pool, sessionRevocationRunner, error) {
	env, err := parseEnvFlag()
	if err != nil {
		return nil, nil, err
	}
	cfg, err := loadCfgLight(env)
	if err != nil {
		return nil, nil, err
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	success := false
	defer func() {
		if !success {
			pool.Close()
		}
	}()
	fencer, err := ownerlifecycle.NewFencer(pool, cfg.OwnerFenceAcquireTimeout)
	if err != nil {
		return nil, nil, err
	}
	owners, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		return nil, nil, err
	}
	children, err := auth.NewCraftskySessionStoreWithConfig(pool, auth.CraftskySessionConfig{
		Inactivity:            cfg.CraftskySessionInactivity,
		ActivityWriteInterval: cfg.CraftskySessionActivityWriteInterval,
	})
	if err != nil {
		return nil, nil, err
	}
	service, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: owners, Sessions: children,
		DeletionExemption: accountdeletion.NewStore(pool, time.Now),
	})
	if err != nil {
		return nil, nil, err
	}
	success = true
	return pool, service.RevokeAllForDID, nil
}

func init() {
	rootCmd.AddCommand(newSessionsCmd(revokeSessionsForDID, revokeSessionsForAll))
}
