package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/spf13/cobra"

	"social.craftsky/appview/internal/db"
	"social.craftsky/appview/internal/ingestion"
)

type tapIngestionOperations interface {
	ListProjectionBacklog(context.Context, int) ([]ingestion.ProjectionBacklogItem, error)
	ListRepositoryBacklog(context.Context, int) ([]ingestion.RepositoryJob, error)
	ListQuarantine(context.Context, int) ([]ingestion.QuarantinedEvent, error)
	RequestQuarantineReplay(context.Context, [32]byte) error
	EnqueueRepositoryJob(context.Context, syntax.DID, ingestion.RepositoryJobKind) error
}

type projectionBacklogReport struct {
	SourceURI      string     `json:"sourceUri"`
	Collection     string     `json:"collection"`
	State          string     `json:"state"`
	DependencyKind string     `json:"dependencyKind,omitempty"`
	DependencyKey  string     `json:"dependencyKey,omitempty"`
	Attempts       int        `json:"attempts"`
	NextAttemptAt  time.Time  `json:"nextAttemptAt"`
	LastReason     string     `json:"lastReason,omitempty"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt,omitempty"`
}

type repositoryBacklogReport struct {
	ID             string     `json:"id"`
	DID            string     `json:"did"`
	Kind           string     `json:"kind"`
	State          string     `json:"state"`
	Attempts       int        `json:"attempts"`
	NextAttemptAt  time.Time  `json:"nextAttemptAt"`
	LastReason     string     `json:"lastReason,omitempty"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt,omitempty"`
}

type quarantineReport struct {
	Fingerprint string          `json:"fingerprint"`
	TapEventID  uint64          `json:"tapEventId"`
	EventType   string          `json:"eventType"`
	Reason      string          `json:"reason"`
	Evidence    json.RawMessage `json:"evidence"`
	Occurrences int64           `json:"occurrences"`
	ReplayState string          `json:"replayState"`
	FirstSeenAt time.Time       `json:"firstSeenAt"`
	LastSeenAt  time.Time       `json:"lastSeenAt"`
	ResolvedAt  *time.Time      `json:"resolvedAt,omitempty"`
}

func writeTapIngestionBacklog(ctx context.Context, operations tapIngestionOperations, limit int, out io.Writer) error {
	if operations == nil || out == nil {
		return errors.New("Tap backlog operations and output are required")
	}
	projection, err := operations.ListProjectionBacklog(ctx, limit)
	if err != nil {
		return err
	}
	repositories, err := operations.ListRepositoryBacklog(ctx, limit)
	if err != nil {
		return err
	}
	projectionReport := make([]projectionBacklogReport, 0, len(projection))
	for _, item := range projection {
		projectionReport = append(projectionReport, projectionBacklogReport{
			SourceURI: item.SourceURI.String(), Collection: item.Collection.String(), State: item.State,
			DependencyKind: item.Dependency.Kind, DependencyKey: item.Dependency.Key,
			Attempts: item.Attempts, NextAttemptAt: item.NextAttempt, LastReason: string(item.LastReason),
			UpdatedAt: item.UpdatedAt, LeaseExpiresAt: item.LeaseExpires,
		})
	}
	repositoryReport := make([]repositoryBacklogReport, 0, len(repositories))
	for _, item := range repositories {
		var leaseExpires *time.Time
		if !item.LeaseExpiresAt.IsZero() {
			value := item.LeaseExpiresAt
			leaseExpires = &value
		}
		repositoryReport = append(repositoryReport, repositoryBacklogReport{
			ID: item.ID.String(), DID: item.DID.String(), Kind: string(item.Kind), State: item.State,
			Attempts: item.Attempts, NextAttemptAt: item.NextAttemptAt,
			LastReason: item.LastReasonCode, LeaseExpiresAt: leaseExpires,
		})
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(struct {
		Projection []projectionBacklogReport `json:"projection"`
		Repository []repositoryBacklogReport `json:"repository"`
	}{Projection: projectionReport, Repository: repositoryReport})
}

func writeTapQuarantine(ctx context.Context, operations tapIngestionOperations, limit int, out io.Writer) error {
	if operations == nil || out == nil {
		return errors.New("Tap quarantine operations and output are required")
	}
	items, err := operations.ListQuarantine(ctx, limit)
	if err != nil {
		return err
	}
	report := make([]quarantineReport, 0, len(items))
	for _, item := range items {
		report = append(report, quarantineReport{
			Fingerprint: hex.EncodeToString(item.Fingerprint[:]), TapEventID: item.TapEventID,
			EventType: item.EventType, Reason: string(item.Reason), Occurrences: item.OccurrenceCount,
			Evidence:    append(json.RawMessage(nil), item.Envelope...),
			ReplayState: item.ReplayState, FirstSeenAt: item.FirstSeenAt,
			LastSeenAt: item.LastSeenAt, ResolvedAt: item.ResolvedAt,
		})
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func requestTapQuarantineReplay(ctx context.Context, operations tapIngestionOperations, rawFingerprint string) error {
	if operations == nil {
		return errors.New("Tap quarantine operations are required")
	}
	decoded, err := hex.DecodeString(rawFingerprint)
	if err != nil || len(decoded) != 32 {
		return errors.New("quarantine fingerprint must be exactly 64 hexadecimal characters")
	}
	var fingerprint [32]byte
	copy(fingerprint[:], decoded)
	return operations.RequestQuarantineReplay(ctx, fingerprint)
}

func enqueueTapPDSReconciliation(ctx context.Context, operations tapIngestionOperations, rawDID string) error {
	if operations == nil {
		return errors.New("Tap reconciliation operations are required")
	}
	did, err := syntax.ParseDID(rawDID)
	if err != nil {
		return fmt.Errorf("invalid reconciliation DID: %w", err)
	}
	return operations.EnqueueRepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile)
}

func withTapIngestionOperations(ctx context.Context, run func(tapIngestionOperations) error) error {
	if run == nil {
		return errors.New("Tap ingestion operation is required")
	}
	env, err := parseEnvFlag()
	if err != nil {
		return err
	}
	config, err := loadCfgLight(env)
	if err != nil {
		return err
	}
	pool, err := db.Connect(ctx, config.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		return err
	}
	return run(store)
}

func init() {
	var backlogLimit int
	backlog := &cobra.Command{
		Use:   "backlog",
		Short: "List unfinished durable Tap projection and repository work",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withTapIngestionOperations(cmd.Context(), func(operations tapIngestionOperations) error {
				return writeTapIngestionBacklog(cmd.Context(), operations, backlogLimit, cmd.OutOrStdout())
			})
		},
	}
	backlog.Flags().IntVar(&backlogLimit, "limit", 100, "maximum rows from each backlog (1-1000)")

	quarantine := &cobra.Command{Use: "quarantine", Short: "Inspect and replay durable Tap quarantine rows"}
	var quarantineLimit int
	quarantineList := &cobra.Command{
		Use:   "list",
		Short: "List bounded Tap quarantine metadata",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withTapIngestionOperations(cmd.Context(), func(operations tapIngestionOperations) error {
				return writeTapQuarantine(cmd.Context(), operations, quarantineLimit, cmd.OutOrStdout())
			})
		},
	}
	quarantineList.Flags().IntVar(&quarantineLimit, "limit", 100, "maximum quarantine rows (1-1000)")
	quarantineReplay := &cobra.Command{
		Use:   "replay FINGERPRINT",
		Short: "Queue one quarantined Tap event for normal reclassification",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTapIngestionOperations(cmd.Context(), func(operations tapIngestionOperations) error {
				if err := requestTapQuarantineReplay(cmd.Context(), operations, args[0]); err != nil {
					return err
				}
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "queued")
				return err
			})
		},
	}
	quarantine.AddCommand(quarantineList, quarantineReplay)

	reconcile := &cobra.Command{
		Use:   "reconcile DID",
		Short: "Queue read-only PDS reconciliation for one repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withTapIngestionOperations(cmd.Context(), func(operations tapIngestionOperations) error {
				if err := enqueueTapPDSReconciliation(cmd.Context(), operations, args[0]); err != nil {
					return err
				}
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "queued")
				return err
			})
		},
	}

	tapCmd.AddCommand(backlog, quarantine, reconcile)
}
