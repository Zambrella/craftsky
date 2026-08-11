package accountdeletion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/testdb"
)

func TestWorkerUsesBoundedRetryThenManualRetryResetsSameJob(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000910")
	owner := syntax.DID("did:plc:retry-owner")
	current := time.Date(2026, 8, 10, 13, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,state,phase,accepted_at,next_attempt_at
		) VALUES($1,$2,'active','removingPrivateData',$3,$3)
	`, jobID, owner, current); err != nil {
		t.Fatalf("seed retry operation: %v", err)
	}

	store := NewStore(pool, func() time.Time { return current })
	metrics := &recordingDeletionMetrics{}
	telemetry := NewDeletionTelemetry(nil, metrics)
	store.SetTelemetry(telemetry)
	processor := &failingDeletionProcessor{err: NewPhaseFailure(FailureTransient, ErrorCategoryPrivateCleanup, errors.New("temporary private store failure"))}
	worker, err := NewWorker(WorkerOptions{
		Store:         store,
		Processor:     processor,
		WorkerID:      "worker-a",
		Now:           func() time.Time { return current },
		LeaseDuration: 2 * time.Minute,
		RetryPolicy:   DefaultRetryPolicy(),
		Telemetry:     telemetry,
	})
	if err != nil {
		t.Fatalf("construct deletion worker: %v", err)
	}

	policy := DefaultRetryPolicy()
	for failure := 1; failure <= len(policy.Delays); failure++ {
		processed, err := worker.ProcessOne(ctx)
		if err != nil || !processed {
			t.Fatalf("process failure %d: processed=%t err=%v", failure, processed, err)
		}
		var (
			status      Status
			attempt     int
			nextAttempt *time.Time
			category    *string
		)
		if err := pool.QueryRow(ctx, `
			SELECT state,attempt_count,next_attempt_at,error_category
			FROM account_deletion_operations WHERE id=$1
		`, jobID).Scan(&status, &attempt, &nextAttempt, &category); err != nil {
			t.Fatalf("read retry operation %d: %v", failure, err)
		}
		if failure < len(policy.Delays) {
			wantAt := policy.Decide(current, jobID.String(), failure, FailureTransient).At
			if status != StatusRetrying || attempt != failure || nextAttempt == nil || !nextAttempt.Equal(wantAt.Truncate(time.Microsecond)) {
				t.Fatalf("failure %d status=%s attempt=%d next=%v want retry at %s", failure, status, attempt, nextAttempt, wantAt)
			}
			current = wantAt
		} else if status != StatusNeedsAttention || attempt != failure || nextAttempt != nil {
			t.Fatalf("exhausted status=%s attempt=%d next=%v", status, attempt, nextAttempt)
		}
		if category == nil || *category != string(ErrorCategoryPrivateCleanup) {
			t.Fatalf("failure %d error category=%v", failure, category)
		}
	}

	manualAt := current.Add(time.Hour)
	current = manualAt
	if err := store.ManualRetry(ctx, jobID, owner); err != nil {
		t.Fatalf("manual retry: %v", err)
	}
	var status Status
	var attempt int
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state,attempt_count,next_attempt_at
		FROM account_deletion_operations WHERE id=$1 AND owner_did=$2
	`, jobID, owner).Scan(&status, &attempt, &nextAttempt); err != nil {
		t.Fatalf("read manual retry operation: %v", err)
	}
	if status != StatusActive || attempt != 0 || !nextAttempt.Equal(manualAt.Truncate(time.Microsecond)) {
		t.Fatalf("manual retry status=%s attempt=%d next=%s", status, attempt, nextAttempt)
	}
	if processor.calls != len(policy.Delays) {
		t.Fatalf("processor calls=%d want=%d", processor.calls, len(policy.Delays))
	}
	events := deletionEventNames(metrics.events)
	if countDeletionEvent(events, "automaticRetry") != len(policy.Delays)-1 ||
		countDeletionEvent(events, "needsAttention") != 1 ||
		countDeletionEvent(events, "manualRetry") != 1 ||
		countDeletionEvent(events, "phase") != len(policy.Delays) {
		t.Fatalf("production retry telemetry=%v", events)
	}
}

func deletionEventNames(events []DeletionMetricEvent) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Event)
	}
	return names
}

func countDeletionEvent(events []string, target string) int {
	count := 0
	for _, event := range events {
		if event == target {
			count++
		}
	}
	return count
}

type failingDeletionProcessor struct {
	calls int
	err   error
}

func (processor *failingDeletionProcessor) Process(context.Context, ClaimedOperation) error {
	processor.calls++
	return processor.err
}
