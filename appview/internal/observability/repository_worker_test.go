package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
)

func TestRepositoryWorkerEscalatesPersistentFailureWithoutStoppingRecovery(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	did := syntax.DID("did:plc:public-repository")
	if err := store.EnqueueRepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile); err != nil {
		t.Fatal(err)
	}
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	cause := errors.Join(ingestion.ErrReconciliationSourceChanged, errors.New("password=private-repository-canary"))
	worker, err := ingestion.NewRepositoryWorker(ingestion.RepositoryWorkerConfig{
		Store: store, Logger: logger, Observer: observer, WorkerID: "test", PollInterval: time.Second, LeaseDuration: time.Minute,
		BatchSize: 1, BackoffMin: time.Second, BackoffMax: time.Minute, AlertAttempts: 2,
		Handler: func(context.Context, ingestion.RepositoryClaim) (string, error) { return "", cause },
	})
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= 3; attempt++ {
		local.Reset()
		count, err := worker.RunOnce(ctx)
		if count != 1 || !errors.Is(err, cause) {
			t.Fatalf("attempt %d lost returned cause: %d %v", attempt, count, err)
		}
		if !observer.Flush(time.Second) {
			t.Fatal("flush failed")
		}
		level, result := "WARN", "retry"
		if attempt >= 2 {
			level, result = "ERROR", "exhausted"
		}
		if !strings.Contains(local.String(), `"level":"`+level+`"`) || !strings.Contains(local.String(), `"result":"`+result+`"`) {
			t.Fatalf("wrong severity: %s", local.String())
		}
		issues, logs := 0, 0
		for _, event := range transport.Events() {
			if len(event.Exception) > 0 {
				issues++
				data, _ := json.Marshal(event)
				for _, want := range []string{did.String(), "tap.repository.pds_reconcile", "exhausted", "errorString", "source_changed", "tap source changed during repository reconciliation"} {
					if !strings.Contains(string(data), want) {
						t.Errorf("issue missing %s: %s", want, data)
					}
				}
			}
			for _, log := range event.Logs {
				logs++
				want := sentry.LogLevelWarn
				if logs >= 2 {
					want = sentry.LogLevelError
				}
				if log.Level != want {
					t.Errorf("log %d severity=%s want=%s", logs, log.Level, want)
				}
			}
			data, _ := json.Marshal(event)
			if strings.Contains(string(data), "private-repository-canary") {
				t.Fatal("protected prose leaked into Sentry")
			}
		}
		if issues != attempt-1 || logs != attempt {
			t.Fatalf("attempt %d: issues=%d logs=%d", attempt, issues, logs)
		}
		if strings.Contains(local.String(), "private-repository-canary") {
			t.Fatal("protected prose leaked locally")
		}
		job, err := store.RepositoryJob(ctx, did, ingestion.RepositoryJobPDSReconcile)
		if err != nil || job.State != "pending" || job.Attempts != attempt {
			t.Fatalf("retry changed: %+v %v", job, err)
		}
		now = job.NextAttemptAt.Add(time.Millisecond)
	}
}
