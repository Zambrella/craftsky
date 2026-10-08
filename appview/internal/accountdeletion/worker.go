package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
)

type ClaimedOperation struct {
	JobID           uuid.UUID
	Owner           syntax.DID
	OwnerGeneration int64
	AttemptCount    int
	LeaseToken      uuid.UUID
	LeaseExpiresAt  time.Time
}

type DeletionProcessor interface {
	Process(context.Context, ClaimedOperation) error
}

type WorkerStore interface {
	ClaimDue(context.Context, string, time.Duration) (ClaimedOperation, bool, error)
	RecordFailure(context.Context, ClaimedOperation, time.Time, ErrorCategory, int) error
}

type DeletionFinalizer interface {
	CompleteAccepted(context.Context, ClaimedOperation) error
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
	Finalizer     DeletionFinalizer
	WorkerID      string
	Now           func() time.Time
	LeaseDuration time.Duration
	RetryPolicy   RetryPolicy
	Logger        *slog.Logger
	Observer      *observability.Observer
}

type Worker struct {
	store         WorkerStore
	processor     DeletionProcessor
	finalizer     DeletionFinalizer
	workerID      string
	now           func() time.Time
	leaseDuration time.Duration
	retryPolicy   RetryPolicy
	logger        *slog.Logger
	observer      *observability.Observer
}

func NewWorker(options WorkerOptions) (*Worker, error) {
	if options.Store == nil || options.Processor == nil || options.Finalizer == nil ||
		options.WorkerID == "" || options.LeaseDuration <= 0 {
		return nil, errors.New("account deletion worker options are invalid")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if len(options.RetryPolicy.Delays) == 0 {
		options.RetryPolicy = DefaultRetryPolicy()
	}
	return &Worker{
		store: options.Store, processor: options.Processor, finalizer: options.Finalizer,
		workerID: options.WorkerID,
		now:      options.Now, leaseDuration: options.LeaseDuration,
		retryPolicy: options.RetryPolicy, logger: options.Logger, observer: options.Observer,
	}, nil
}

func (worker *Worker) ProcessOne(ctx context.Context) (bool, error) {
	operation, found, err := worker.store.ClaimDue(ctx, worker.workerID, worker.leaseDuration)
	if err != nil {
		input := observability.DiagnosticInput{Error: err, Context: observability.EventContext{"component": "worker", "operation": "account.delete", "failure_stage": "claim", "result": "error"}}
		observability.LogDiagnostic(ctx, worker.logger, input)
		worker.observer.CaptureDiagnostic(ctx, input)
		return found, err
	}
	if !found {
		return found, nil
	}
	if err := worker.processor.Process(ctx, operation); err != nil {
		// The session coordinator has already atomically removed the stale
		// deletion binding and moved the job to reauth_required. Retrying the
		// old lease would either overwrite that state or reuse revoked authority.
		if errors.Is(err, auth.ErrDeletionReauthenticationRequired) {
			return true, nil
		}
		category := ErrorCategoryTerminal
		var failure *DeletionFailure
		if errors.As(err, &failure) {
			category = failure.category
		}
		nextAttempt := operation.AttemptCount + 1
		nextAt := worker.retryPolicy.Next(worker.now().UTC(), operation.JobID.String(), nextAttempt)
		if persistErr := worker.store.RecordFailure(ctx, operation, nextAt, category, nextAttempt); persistErr != nil {
			worker.reportFailure(ctx, operation, errors.Join(err, persistErr), "failure_persistence", "error", nextAttempt)
			return true, fmt.Errorf("persist account deletion failure: %w", persistErr)
		}
		worker.reportFailure(ctx, operation, err, "processing", "retry", nextAttempt)

		return true, nil
	}
	if err := worker.finalizer.CompleteAccepted(ctx, operation); err != nil {
		if errors.Is(err, ErrSafetyPending) {
			nextAttempt := operation.AttemptCount + 1
			nextAt := worker.retryPolicy.Next(worker.now().UTC(), operation.JobID.String(), nextAttempt)
			if persistErr := worker.store.RecordFailure(
				ctx,
				operation,
				nextAt,
				ErrorCategoryPDS,
				nextAttempt,
			); persistErr != nil {
				worker.reportFailure(ctx, operation, errors.Join(err, persistErr), "failure_persistence", "error", nextAttempt)
				return true, fmt.Errorf("persist account deletion safety retry: %w", persistErr)
			}
			worker.reportFailure(ctx, operation, err, "finalization", "retry", nextAttempt)
			return true, nil
		}
		worker.reportFailure(ctx, operation, err, "finalization", "error", operation.AttemptCount+1)
		return true, fmt.Errorf("finalize account deletion: %w", err)
	}
	if worker.logger != nil {
		worker.logger.InfoContext(ctx, "account deletion completed", slog.String("jobId", operation.JobID.String()))
	}
	return true, nil
}

func (worker *Worker) reportFailure(ctx context.Context, operation ClaimedOperation, cause error, stage, outcome string, attempt int) {
	input := observability.DiagnosticInput{Error: cause, Context: observability.EventContext{"component": "worker", "operation": "account.delete", "failure_stage": stage, "result": outcome, "attempt": attempt}, Workflow: observability.PrivateFailureContext(cause, operation.Owner, operation.JobID.String())}
	observability.LogDiagnostic(ctx, worker.logger, input)
	worker.observer.CaptureDiagnostic(ctx, input)
}
