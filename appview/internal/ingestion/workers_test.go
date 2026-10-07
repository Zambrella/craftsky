package ingestion_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestProjectionWorkerRejectsUnsafeLeaseAndBackoffConfiguration(t *testing.T) {
	base := ingestion.ProjectionWorkerConfig{
		WorkerID: "projection-worker", PollInterval: time.Second,
		LeaseDuration: 30 * time.Second, BatchSize: 16,
		BackoffMin: time.Second, BackoffMax: time.Minute,
		Projector: func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
			return tap.Applied(), nil
		},
	}
	for name, mutate := range map[string]func(*ingestion.ProjectionWorkerConfig){
		"missing store":   func(config *ingestion.ProjectionWorkerConfig) { config.Store = nil },
		"empty worker":    func(config *ingestion.ProjectionWorkerConfig) { config.WorkerID = "" },
		"zero poll":       func(config *ingestion.ProjectionWorkerConfig) { config.PollInterval = 0 },
		"lease too short": func(config *ingestion.ProjectionWorkerConfig) { config.LeaseDuration = config.PollInterval },
		"zero batch":      func(config *ingestion.ProjectionWorkerConfig) { config.BatchSize = 0 },
		"inverted backoff": func(config *ingestion.ProjectionWorkerConfig) {
			config.BackoffMin = time.Minute
			config.BackoffMax = time.Second
		},
	} {
		t.Run(name, func(t *testing.T) {
			config := base
			config.Store = &ingestion.Store{}
			mutate(&config)
			if _, err := ingestion.NewProjectionWorker(config); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestProjectionWorkerCompletesDurableSourceAndServingMutation(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	ctx := context.Background()
	uri := syntax.ATURI("at://did:plc:worker/social.craftsky.actor.profile/self")
	if _, err := store.IngestRecord(ctx, tap.Event{
		ID: 101, URI: uri,
		DID: "did:plc:worker", Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3aaaaaaaaaaa2", CID: "bafy-worker", Action: "create",
		Record: json.RawMessage(`{"crafts":["sewing"]}`),
	}); err != nil {
		t.Fatalf("ingest source: %v", err)
	}
	worker, err := ingestion.NewProjectionWorker(ingestion.ProjectionWorkerConfig{
		Store: store, WorkerID: "projection-worker", PollInterval: time.Second,
		LeaseDuration: 30 * time.Second, BatchSize: 16,
		BackoffMin: time.Second, BackoffMax: time.Minute,
		Projector: func(ctx context.Context, tx pgx.Tx, source ingestion.SourceRecord) (tap.Outcome, error) {
			_, err := tx.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,$2)`, source.DID, source.CID)
			return tap.Applied(), err
		},
	})
	if err != nil {
		t.Fatalf("new projection worker: %v", err)
	}
	processed, err := worker.RunOnce(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}
	job, err := store.ProjectionJob(ctx, uri)
	if err != nil || job.State != "complete" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
}

func TestRepositoryWorkerRejectsUnsafeConfiguration(t *testing.T) {
	config := ingestion.RepositoryWorkerConfig{
		Store: &ingestion.Store{}, WorkerID: "repository-worker",
		PollInterval: time.Second, LeaseDuration: 30 * time.Second,
		BatchSize: 8, BackoffMin: time.Second, BackoffMax: time.Minute,
		Handler: func(context.Context, ingestion.RepositoryClaim) (string, error) { return "", nil },
	}
	if _, err := ingestion.NewRepositoryWorker(config); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	config.LeaseDuration = config.PollInterval
	if _, err := ingestion.NewRepositoryWorker(config); err == nil {
		t.Fatal("lease must exceed the poll interval")
	}
}

func TestQuarantineReplayWorkerRejectsUnsafeConfiguration(t *testing.T) {
	base := ingestion.QuarantineReplayWorkerConfig{
		Store: &ingestion.Store{}, WorkerID: "quarantine-worker",
		PollInterval: time.Second, LeaseDuration: 30 * time.Second,
		OperationTimeout: 10 * time.Second, BatchSize: 8,
		Handler: func(context.Context, []byte) (tap.Outcome, error) {
			return tap.Applied(), nil
		},
	}
	if _, err := ingestion.NewQuarantineReplayWorker(base); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	for name, mutate := range map[string]func(*ingestion.QuarantineReplayWorkerConfig){
		"missing store":      func(config *ingestion.QuarantineReplayWorkerConfig) { config.Store = nil },
		"missing handler":    func(config *ingestion.QuarantineReplayWorkerConfig) { config.Handler = nil },
		"empty worker":       func(config *ingestion.QuarantineReplayWorkerConfig) { config.WorkerID = "" },
		"zero poll":          func(config *ingestion.QuarantineReplayWorkerConfig) { config.PollInterval = 0 },
		"lease too short":    func(config *ingestion.QuarantineReplayWorkerConfig) { config.LeaseDuration = config.PollInterval },
		"zero operation":     func(config *ingestion.QuarantineReplayWorkerConfig) { config.OperationTimeout = 0 },
		"operation at lease": func(config *ingestion.QuarantineReplayWorkerConfig) { config.OperationTimeout = config.LeaseDuration },
		"zero batch":         func(config *ingestion.QuarantineReplayWorkerConfig) { config.BatchSize = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			config := base
			mutate(&config)
			if _, err := ingestion.NewQuarantineReplayWorker(config); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestProjectionWorkerDiagnosticRetainsSourceAndTerminalOutcome(t *testing.T) {
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	applyTapDurabilityMigration(t, pool)
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	source := tap.Event{ID: 101, URI: "at://did:plc:worker/social.craftsky.actor.profile/self", DID: "did:plc:worker", Collection: "social.craftsky.actor.profile", Rkey: "self", Rev: "3aaaaaaaaaaa2", CID: "bafy-public", Action: "create", Record: json.RawMessage(`{"crafts":["sewing"]}`)}
	if _, err := store.IngestRecord(context.Background(), source); err != nil {
		t.Fatal(err)
	}
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	worker, err := ingestion.NewProjectionWorker(ingestion.ProjectionWorkerConfig{Store: store, WorkerID: "test-projection", PollInterval: time.Second, LeaseDuration: 30 * time.Second, BatchSize: 1, BackoffMin: time.Second, BackoffMax: time.Minute, Logger: logger, Projector: func(context.Context, pgx.Tx, ingestion.SourceRecord) (tap.Outcome, error) {
		return tap.PermanentInvalid(tap.ReasonMalformedRecord), nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	count, err := worker.RunOnce(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	job, err := store.ProjectionJob(context.Background(), source.URI)
	if err != nil || job.State != "permanent_denied" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	for _, want := range []string{"did:plc:worker", "bafy-public", "quarantine", "durably_committed", "101"} {
		if !strings.Contains(local.String(), want) {
			t.Fatalf("missing%s: %s", want, local.String())
		}
	}
}
