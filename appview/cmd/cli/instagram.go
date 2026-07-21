package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/instagram"
)

type instagramCLIBackend interface {
	ListOpenConflicts(context.Context, int, uuid.UUID) ([]instagram.OperatorConflict, uuid.UUID, error)
	ResolveConflict(context.Context, uuid.UUID, instagram.OperatorConflictResolution) (instagram.OperatorConflictResult, error)
	RevokeLink(context.Context, uuid.UUID) (instagram.OperatorLinkResult, error)
	ListJobs(context.Context, instagram.OperatorJobKind, int, uuid.UUID) ([]instagram.OperatorJob, uuid.UUID, error)
	InspectJob(context.Context, instagram.OperatorJobKind, uuid.UUID) (instagram.OperatorJob, error)
	RetryJob(context.Context, instagram.OperatorJobKind, uuid.UUID) (instagram.OperatorJobResult, error)
	PurgeExpiredImports(context.Context, int) (int, error)
}

type instagramCLIBackendLoader func(context.Context) (instagramCLIBackend, func(), error)

type postgresInstagramCLIBackend struct {
	operator  *instagram.OperatorService
	retention *instagram.RetentionService
}

func (b *postgresInstagramCLIBackend) ListOpenConflicts(ctx context.Context, limit int, after uuid.UUID) ([]instagram.OperatorConflict, uuid.UUID, error) {
	return b.operator.ListOpenConflicts(ctx, limit, after)
}

func (b *postgresInstagramCLIBackend) ResolveConflict(ctx context.Context, id uuid.UUID, resolution instagram.OperatorConflictResolution) (instagram.OperatorConflictResult, error) {
	return b.operator.ResolveConflict(ctx, id, resolution)
}

func (b *postgresInstagramCLIBackend) RevokeLink(ctx context.Context, id uuid.UUID) (instagram.OperatorLinkResult, error) {
	return b.operator.RevokeLink(ctx, id)
}

func (b *postgresInstagramCLIBackend) ListJobs(ctx context.Context, kind instagram.OperatorJobKind, limit int, after uuid.UUID) ([]instagram.OperatorJob, uuid.UUID, error) {
	return b.operator.ListJobs(ctx, kind, limit, after)
}

func (b *postgresInstagramCLIBackend) InspectJob(ctx context.Context, kind instagram.OperatorJobKind, id uuid.UUID) (instagram.OperatorJob, error) {
	return b.operator.InspectJob(ctx, kind, id)
}

func (b *postgresInstagramCLIBackend) RetryJob(ctx context.Context, kind instagram.OperatorJobKind, id uuid.UUID) (instagram.OperatorJobResult, error) {
	return b.operator.RetryJob(ctx, kind, id)
}

func (b *postgresInstagramCLIBackend) PurgeExpiredImports(ctx context.Context, limit int) (int, error) {
	return b.retention.PurgeExpiredImports(ctx, limit)
}

func newInstagramCmd(load instagramCLIBackendLoader) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instagram",
		Short: "Bounded private Instagram migration operations",
		Args:  cobra.NoArgs,
	}
	cmd.AddCommand(newInstagramConflictsCmd(load))
	cmd.AddCommand(newInstagramLinksCmd(load))
	cmd.AddCommand(newInstagramJobsCmd(load))
	cmd.AddCommand(newInstagramRetentionCmd(load))
	return cmd
}

func newInstagramConflictsCmd(load instagramCLIBackendLoader) *cobra.Command {
	cmd := &cobra.Command{Use: "conflicts", Short: "Inspect and resolve opaque link conflicts", Args: cobra.NoArgs}

	listLimit := 100
	listAfter := ""
	list := &cobra.Command{
		Use:   "list",
		Short: "List unresolved conflicts without identity evidence",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateInstagramCLILimit(listLimit); err != nil {
				return err
			}
			after, err := parseOptionalOpaqueID("after-conflict-id", listAfter)
			if err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			items, next, err := backend.ListOpenConflicts(cmd.Context(), listLimit, after)
			if err != nil {
				return err
			}
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "conflict id=%s state=%s openedAt=%s expiresAt=%s\n",
					item.ID, item.State, formatOperatorTime(item.OpenedAt), formatOperatorTime(item.ExpiresAt))
			}
			if next != uuid.Nil {
				fmt.Fprintf(cmd.OutOrStdout(), "nextConflictId=%s\n", next)
			}
			return nil
		},
	}
	list.Flags().IntVar(&listLimit, "limit", 100, "maximum rows (1-500)")
	list.Flags().StringVar(&listAfter, "after-conflict-id", "", "opaque conflict cursor")

	resolveID := ""
	resolveValue := ""
	resolve := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve one conflict explicitly without transferring ownership",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, err := parseRequiredOpaqueID("conflict-id", resolveID)
			if err != nil {
				return err
			}
			resolution, err := parseConflictResolution(resolveValue)
			if err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			result, err := backend.ResolveConflict(cmd.Context(), id, resolution)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "conflict id=%s state=%s changed=%s\n", result.ID, result.State, strconv.FormatBool(result.Changed))
			return nil
		},
	}
	resolve.Flags().StringVar(&resolveID, "conflict-id", "", "required opaque conflict UUID")
	resolve.Flags().StringVar(&resolveValue, "resolution", "", `required: "keep-existing" or "revoke-existing"`)

	cmd.AddCommand(list, resolve)
	return cmd
}

func newInstagramLinksCmd(load instagramCLIBackendLoader) *cobra.Command {
	cmd := &cobra.Command{Use: "links", Short: "Operate on opaque account-link records", Args: cobra.NoArgs}
	linkID := ""
	revoke := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke one link and invalidate only pending dependent work",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			id, err := parseRequiredOpaqueID("link-id", linkID)
			if err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			result, err := backend.RevokeLink(cmd.Context(), id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "link id=%s state=%s changed=%s\n", result.ID, result.State, strconv.FormatBool(result.Changed))
			return nil
		},
	}
	revoke.Flags().StringVar(&linkID, "link-id", "", "required opaque link UUID")
	cmd.AddCommand(revoke)
	return cmd
}

func newInstagramJobsCmd(load instagramCLIBackendLoader) *cobra.Command {
	cmd := &cobra.Command{Use: "jobs", Short: "Inspect and retry bounded durable work", Args: cobra.NoArgs}

	listKind := ""
	listLimit := 100
	listAfter := ""
	list := &cobra.Command{
		Use:   "list",
		Short: "List redacted job state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			kind, err := parseJobKind(listKind)
			if err != nil {
				return err
			}
			if err := validateInstagramCLILimit(listLimit); err != nil {
				return err
			}
			after, err := parseOptionalOpaqueID("after-job-id", listAfter)
			if err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			items, next, err := backend.ListJobs(cmd.Context(), kind, listLimit, after)
			if err != nil {
				return err
			}
			for _, item := range items {
				writeOperatorJob(cmd, item)
			}
			if next != uuid.Nil {
				fmt.Fprintf(cmd.OutOrStdout(), "nextJobId=%s\n", next)
			}
			return nil
		},
	}
	list.Flags().StringVar(&listKind, "kind", "", `required: "webhook" or "reconciliation"`)
	list.Flags().IntVar(&listLimit, "limit", 100, "maximum rows (1-500)")
	list.Flags().StringVar(&listAfter, "after-job-id", "", "opaque job cursor")

	inspectKind := ""
	inspectID := ""
	inspect := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect one redacted job",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			kind, err := parseJobKind(inspectKind)
			if err != nil {
				return err
			}
			id, err := parseRequiredOpaqueID("job-id", inspectID)
			if err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			item, err := backend.InspectJob(cmd.Context(), kind, id)
			if err != nil {
				return err
			}
			writeOperatorJob(cmd, item)
			return nil
		},
	}
	inspect.Flags().StringVar(&inspectKind, "kind", "", `required: "webhook" or "reconciliation"`)
	inspect.Flags().StringVar(&inspectID, "job-id", "", "required opaque job UUID")

	retryKind := ""
	retryID := ""
	retry := &cobra.Command{
		Use:   "retry",
		Short: "Retry one safely recoverable job",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			kind, err := parseJobKind(retryKind)
			if err != nil {
				return err
			}
			id, err := parseRequiredOpaqueID("job-id", retryID)
			if err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			result, err := backend.RetryJob(cmd.Context(), kind, id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "job id=%s kind=%s status=%s changed=%s\n", result.ID, result.Kind, result.Status, strconv.FormatBool(result.Changed))
			return nil
		},
	}
	retry.Flags().StringVar(&retryKind, "kind", "", `required: "webhook" or "reconciliation"`)
	retry.Flags().StringVar(&retryID, "job-id", "", "required opaque job UUID")

	cmd.AddCommand(list, inspect, retry)
	return cmd
}

func newInstagramRetentionCmd(load instagramCLIBackendLoader) *cobra.Command {
	cmd := &cobra.Command{Use: "retention", Short: "Run bounded Instagram retention operations", Args: cobra.NoArgs}
	limit := 100
	purgeImports := &cobra.Command{
		Use:   "purge-imports",
		Short: "Purge expired import aggregates in a stable bounded batch",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := validateInstagramCLILimit(limit); err != nil {
				return err
			}
			backend, cleanup, err := load(cmd.Context())
			if err != nil {
				return err
			}
			defer cleanup()
			purged, err := backend.PurgeExpiredImports(cmd.Context(), limit)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "imports purged=%d limit=%d\n", purged, limit)
			return nil
		},
	}
	purgeImports.Flags().IntVar(&limit, "limit", 100, "maximum primary imports (1-500)")
	cmd.AddCommand(purgeImports)
	return cmd
}

func writeOperatorJob(cmd *cobra.Command, item instagram.OperatorJob) {
	terminal := "-"
	if item.TerminalAt != nil {
		terminal = formatOperatorTime(*item.TerminalAt)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "job id=%s kind=%s status=%s attempts=%d nextAttemptAt=%s terminalAt=%s createdAt=%s\n",
		item.ID, item.Kind, item.Status, item.Attempts, formatOperatorTime(item.NextAttemptAt), terminal, formatOperatorTime(item.CreatedAt))
}

func validateInstagramCLILimit(limit int) error {
	if limit < 1 || limit > instagram.MaxOperatorBatch {
		return fmt.Errorf("limit must be between 1 and %d", instagram.MaxOperatorBatch)
	}
	return nil
}

func parseRequiredOpaqueID(name, value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, fmt.Errorf("--%s is required", name)
	}
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return uuid.Nil, fmt.Errorf("--%s must be an opaque UUID", name)
	}
	return id, nil
}

func parseOptionalOpaqueID(name, value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return parseRequiredOpaqueID(name, value)
}

func parseConflictResolution(value string) (instagram.OperatorConflictResolution, error) {
	switch value {
	case "keep-existing":
		return instagram.ResolutionKeepExisting, nil
	case "revoke-existing":
		return instagram.ResolutionRevokeExisting, nil
	default:
		return "", fmt.Errorf("--resolution must be keep-existing or revoke-existing")
	}
}

func parseJobKind(value string) (instagram.OperatorJobKind, error) {
	kind := instagram.OperatorJobKind(value)
	if !kind.Valid() {
		return "", fmt.Errorf("--kind must be webhook or reconciliation")
	}
	return kind, nil
}

func formatOperatorTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func loadInstagramCLIBackend(ctx context.Context) (instagramCLIBackend, func(), error) {
	deps, cleanup, err := loadDeps(ctx)
	if err != nil {
		return nil, nil, err
	}
	operator, err := instagram.NewOperatorService(deps.DB, deps.Config.InstagramData.HMACKey(), time.Now)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return &postgresInstagramCLIBackend{
		operator:  operator,
		retention: instagram.NewRetentionService(deps.DB, time.Now),
	}, cleanup, nil
}

func init() {
	rootCmd.AddCommand(newInstagramCmd(loadInstagramCLIBackend))
}
