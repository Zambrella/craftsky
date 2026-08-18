package scheduledposts

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type recordingClaimStore struct {
	items     []WorkItem
	gotLimit  int
	gotNow    time.Time
	claimRuns int
}

func (store *recordingClaimStore) ScheduledQueueSnapshot(
	_ context.Context,
	_ time.Time,
) (ScheduledQueueSnapshot, error) {
	return ScheduledQueueSnapshot{
		StatusCounts: map[Status]int{
			StatusScheduled: 1,
			StatusRetrying:  1,
		},
		Due: 2, Overdue: 1, OldestDueAge: 90 * time.Second,
	}, nil
}

func (store *recordingClaimStore) ClaimBatch(_ context.Context, limit int, now time.Time) ([]WorkItem, error) {
	store.gotLimit = limit
	store.gotNow = now
	store.claimRuns++
	return append([]WorkItem(nil), store.items...), nil
}

type recordingWorkProcessor struct {
	items []WorkItem
}

type recordingScheduledObserver struct {
	queues     []ScheduledQueueSnapshot
	operations []string
}

func (observer *recordingScheduledObserver) ObserveScheduledQueue(
	status string,
	count int,
	due int,
	overdue int,
	oldestDueAge time.Duration,
) {
	observer.queues = append(observer.queues, ScheduledQueueSnapshot{
		StatusCounts: map[Status]int{Status(status): count},
		Due:          due, Overdue: overdue, OldestDueAge: oldestDueAge,
	})
}

func (observer *recordingScheduledObserver) ObserveScheduledOperation(
	operation string,
	result string,
	errorClass string,
	_ time.Duration,
) {
	observer.operations = append(
		observer.operations,
		operation+":"+result+":"+errorClass,
	)
}

func (*recordingScheduledObserver) ObserveScheduledPublication(int, time.Duration, time.Duration) {
}

func (*recordingScheduledObserver) ObserveScheduledCleanupQueue(int, time.Duration) {}

func (processor *recordingWorkProcessor) Process(_ context.Context, item WorkItem) error {
	processor.items = append(processor.items, item)
	return nil
}

func TestWorkerProcessesBoundedBatchWithoutHTTP(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 31, 12, 0, 0, 0, time.UTC)
	store := &recordingClaimStore{items: []WorkItem{
		{ID: uuid.MustParse("11111111-1111-4111-8111-111111111111"), OwnerGeneration: 1},
		{ID: uuid.MustParse("22222222-2222-4222-8222-222222222222"), OwnerGeneration: 1},
	}}
	processor := &recordingWorkProcessor{}
	observer := &recordingScheduledObserver{}
	worker, err := NewWorker(WorkerOptions{
		Store:     store,
		Processor: processor,
		Now:       func() time.Time { return now },
		BatchSize: 2,
		Observer:  observer,
	})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	processed, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if processed != 2 || len(processor.items) != 2 {
		t.Fatalf("ProcessBatch() processed %d with %d recorded items, want 2", processed, len(processor.items))
	}
	if store.gotLimit != 2 || !store.gotNow.Equal(now) || store.claimRuns != 1 {
		t.Fatalf("ClaimBatch() got limit=%d now=%s runs=%d", store.gotLimit, store.gotNow, store.claimRuns)
	}
	if len(observer.queues) != 4 {
		t.Fatalf("queue observations=%d, want four status gauges", len(observer.queues))
	}
	if len(observer.operations) != 1 || observer.operations[0] != "claim:success:none" {
		t.Fatalf("operation observations=%v", observer.operations)
	}

	if _, err := NewWorker(WorkerOptions{
		Store:     store,
		Processor: processor,
		BatchSize: MaximumWorkerBatch + 1,
	}); err == nil {
		t.Fatal("NewWorker() accepted an unbounded batch")
	}
}
