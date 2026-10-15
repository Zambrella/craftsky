package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"log/slog"
	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/operatorconsole"
	"time"
)

type operatorConnector func(context.Context) (*pgxpool.Pool, func(), error)

func connectOperators(ctx context.Context) (*pgxpool.Pool, func(), error) {
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
	return pool, pool.Close, nil
}

type operatorCommandError struct{ cause error }

func (e operatorCommandError) Error() string {
	if errors.Is(e.cause, operatorconsole.ErrInvalid) {
		return "invalid operator command"
	}
	if errors.Is(e.cause, operatorconsole.ErrDenied) {
		return "operator unavailable"
	}
	return "operator command failed; inspect protected diagnostics"
}
func (e operatorCommandError) Unwrap() error { return e.cause }
func protectOperatorError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(cmd.ErrOrStderr(), nil)))
	result := "error"
	if errors.Is(err, operatorconsole.ErrInvalid) || errors.Is(err, operatorconsole.ErrDenied) {
		result = "expected"
	}
	observability.LogDiagnostic(cmd.Context(), logger, observability.DiagnosticInput{Error: err, Context: observability.EventContext{"component": "operator-cli", "operation": "operator.maintenance", "failure_stage": "execution", "result": result}})
	return operatorCommandError{cause: err}
}
func newOperatorsCmd(connect operatorConnector) *cobra.Command {
	root := &cobra.Command{Use: "operators", Short: "Maintain DID-allowlisted console operators using privileged database access", SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs}
	run := func(cmd *cobra.Command, fn func(context.Context, *operatorconsole.Store) error) error {
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		pool, closeDB, err := connect(ctx)
		if err != nil {
			return protectOperatorError(cmd, err)
		}
		defer closeDB()
		return protectOperatorError(cmd, fn(ctx, operatorconsole.NewStore(pool, time.Now)))
	}
	var allowSet string
	allow := &cobra.Command{Use: "allow DID", Short: "Allow a new Atmosphere account; emits identity metadata only", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		did, err := syntax.ParseDID(args[0])
		if err != nil {
			return protectOperatorError(cmd, operatorconsole.ErrInvalid)
		}
		return run(cmd, func(ctx context.Context, s *operatorconsole.Store) error {
			op, err := s.Allow(ctx, did, allowSet)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(op)
		})
	}}
	allow.Flags().StringVar(&allowSet, "permission-set", "", "admin, mod or helper")
	var set string
	change := &cobra.Command{Use: "set OPERATOR_ID", Short: "Change the operator permission set", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := uuid.Parse(args[0])
		if err != nil {
			return protectOperatorError(cmd, operatorconsole.ErrInvalid)
		}
		return run(cmd, func(ctx context.Context, s *operatorconsole.Store) error { return s.SetPermissionSet(ctx, id, set) })
	}}
	change.Flags().StringVar(&set, "permission-set", "", "admin, mod or helper")
	root.AddCommand(allow, change)
	for _, action := range []string{"disable", "revoke-sessions", "audit"} {
		root.AddCommand(&cobra.Command{Use: action + " OPERATOR_ID", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return protectOperatorError(cmd, operatorconsole.ErrInvalid)
			}
			return run(cmd, func(ctx context.Context, s *operatorconsole.Store) error {
				switch action {
				case "disable":
					return s.Disable(ctx, id)
				case "revoke-sessions":
					return s.RevokeSessions(ctx, id)
				default:
					events, err := s.Audit(ctx, id)
					if err != nil {
						return err
					}
					return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"items": events})
				}
			})
		}})
	}
	root.AddCommand(&cobra.Command{Use: "list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		return run(cmd, func(ctx context.Context, s *operatorconsole.Store) error {
			ops, err := s.List(ctx)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(map[string]any{"items": ops})
		})
	}})
	root.RunE = func(cmd *cobra.Command, _ []string) error { return cmd.Help() }
	return root
}
func init() { rootCmd.AddCommand(newOperatorsCmd(connectOperators)) }
