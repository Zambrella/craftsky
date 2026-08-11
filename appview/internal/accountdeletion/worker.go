package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type ClaimedOperation struct {
	JobID        uuid.UUID
	Owner        syntax.DID
	AttemptCount int
	LeaseToken   uuid.UUID
}

type DeletionProcessor interface {
	Process(context.Context, ClaimedOperation) error
}

type WorkerStore interface {
	ClaimDue(context.Context, string, time.Duration) (ClaimedOperation, bool, error)
	RecordFailure(context.Context, ClaimedOperation, time.Time, ErrorCategory, int) error
	CompleteAttempt(context.Context, ClaimedOperation) error
}

type DeletionFailure struct {
	category ErrorCategory
	err      error
}

func NewDeletionFailure(category ErrorCategory, err error) error {
	if err == nil {
		err = errors.New("account deletion failed")
	}
	return &DeletionFailure{category: category, err: err}
}

func (failure *DeletionFailure) Error() string { return failure.err.Error() }
func (failure *DeletionFailure) Unwrap() error { return failure.err }

type WorkerOptions struct {
	Store         WorkerStore
	Processor     DeletionProcessor
	WorkerID      string
	Now           func() time.Time
	LeaseDuration time.Duration
	RetryPolicy   RetryPolicy
	Logger        *slog.Logger
}

type Worker struct {
	store         WorkerStore
	processor     DeletionProcessor
	workerID      string
	now           func() time.Time
	leaseDuration time.Duration
	retryPolicy   RetryPolicy
	logger        *slog.Logger
}

func NewWorker(options WorkerOptions) (*Worker, error) {
	if options.Store == nil || options.Processor == nil || options.WorkerID == "" || options.LeaseDuration <= 0 {
		return nil, errors.New("account deletion worker options are invalid")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if len(options.RetryPolicy.Delays) == 0 {
		options.RetryPolicy = DefaultRetryPolicy()
	}
	return &Worker{
		store: options.Store, processor: options.Processor, workerID: options.WorkerID,
		now: options.Now, leaseDuration: options.LeaseDuration,
		retryPolicy: options.RetryPolicy, logger: options.Logger,
	}, nil
}

func (worker *Worker) ProcessOne(ctx context.Context) (bool, error) {
	operation, found, err := worker.store.ClaimDue(ctx, worker.workerID, worker.leaseDuration)
	if err != nil || !found {
		return found, err
	}
	if err := worker.processor.Process(ctx, operation); err != nil {
		category := ErrorCategoryTerminal
		var failure *DeletionFailure
		if errors.As(err, &failure) {
			category = failure.category
		}
		nextAttempt := operation.AttemptCount + 1
		nextAt := worker.retryPolicy.Next(worker.now().UTC(), operation.JobID.String(), nextAttempt)
		if persistErr := worker.store.RecordFailure(ctx, operation, nextAt, category, nextAttempt); persistErr != nil {
			return true, fmt.Errorf("persist account deletion failure: %w", persistErr)
		}
		if worker.logger != nil {
			worker.logger.WarnContext(ctx, "account deletion attempt scheduled for retry",
				slog.String("jobId", operation.JobID.String()),
				slog.String("errorCategory", string(category)),
				slog.Int("attempt", nextAttempt),
			)
		}
		return true, nil
	}
	if err := worker.store.CompleteAttempt(ctx, operation); err != nil {
		return true, fmt.Errorf("finalize account deletion: %w", err)
	}
	if worker.logger != nil {
		worker.logger.InfoContext(ctx, "account deletion completed", slog.String("jobId", operation.JobID.String()))
	}
	return true, nil
}
