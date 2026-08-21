package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

const (
	DefaultWorkerBatch = 20
	MaximumWorkerBatch = 100
)

type WorkItem struct {
	ID              uuid.UUID
	OwnerDID        syntax.DID
	OwnerGeneration int64
	LeaseToken      uuid.UUID
	PayloadVersion  int64
	Rkey            syntax.RecordKey
	CreatedAt       time.Time
	Manual          bool
}

type ClaimStore interface {
	ClaimBatch(context.Context, int, time.Time) ([]WorkItem, error)
}

type WorkProcessor interface {
	Process(context.Context, WorkItem) error
}

type WorkerOptions struct {
	Store     ClaimStore
	Processor WorkProcessor
	Now       func() time.Time
	BatchSize int
	Observer  OperationalObserver
}

type Worker struct {
	store     ClaimStore
	processor WorkProcessor
	now       func() time.Time
	batchSize int
	observer  OperationalObserver
}

func NewWorker(options WorkerOptions) (*Worker, error) {
	if options.Store == nil || options.Processor == nil {
		return nil, errors.New("scheduled worker dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.BatchSize == 0 {
		options.BatchSize = DefaultWorkerBatch
	}
	if options.BatchSize < 1 || options.BatchSize > MaximumWorkerBatch {
		return nil, errors.New("invalid scheduled worker batch size")
	}
	return &Worker{
		store:     options.Store,
		processor: options.Processor,
		now:       options.Now,
		batchSize: options.BatchSize,
		observer:  options.Observer,
	}, nil
}

func (worker *Worker) ProcessBatch(ctx context.Context) (int, error) {
	if worker == nil {
		return 0, errors.New("scheduled worker is nil")
	}
	now := worker.now().UTC()
	started := time.Now()
	if snapshotter, ok := worker.store.(scheduledQueueSnapshotter); ok {
		snapshot, err := snapshotter.ScheduledQueueSnapshot(ctx, now)
		if err != nil {
			if worker.observer != nil {
				worker.observer.ObserveScheduledOperation(
					"claim", "failure", "dependency_unavailable", time.Since(started),
				)
			}
			return 0, fmt.Errorf("observe scheduled queue: %w", err)
		}
		observeScheduledQueue(worker.observer, snapshot)
	}
	items, err := worker.store.ClaimBatch(ctx, worker.batchSize, now)
	if err != nil {
		if worker.observer != nil {
			worker.observer.ObserveScheduledOperation(
				"claim", "failure", "dependency_unavailable", time.Since(started),
			)
		}
		return 0, fmt.Errorf("claim scheduled posts: %w", err)
	}
	if worker.observer != nil {
		worker.observer.ObserveScheduledOperation(
			"claim", "success", "none", time.Since(started),
		)
	}
	for index, item := range items {
		if err := worker.processor.Process(ctx, item); err != nil {
			return index, fmt.Errorf("process scheduled post: %w", err)
		}
	}
	return len(items), nil
}
