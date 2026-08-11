package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
)

type ClaimedOperation struct {
	JobID        uuid.UUID
	Owner        syntax.DID
	Status       Status
	Phase        Phase
	AttemptCount int
	LeaseToken   uuid.UUID
}

type DeletionProcessor interface {
	Process(context.Context, ClaimedOperation) error
}

type WorkerStore interface {
	ClaimDue(context.Context, string, time.Duration) (ClaimedOperation, bool, error)
	RecordFailure(context.Context, ClaimedOperation, RetryDecision, ErrorCategory, int) error
	DeferAttempt(context.Context, ClaimedOperation, time.Time) error
	CompleteAttempt(context.Context, ClaimedOperation) error
}

type PhaseFailure struct {
	kind     FailureKind
	category ErrorCategory
	err      error
}

type PhasePending struct {
	At time.Time
}

func (pending *PhasePending) Error() string { return "account deletion phase is awaiting convergence" }

func NewPhaseFailure(kind FailureKind, category ErrorCategory, err error) error {
	if err == nil {
		err = errors.New("account deletion phase failed")
	}
	return &PhaseFailure{kind: kind, category: category, err: err}
}

func (failure *PhaseFailure) Error() string { return failure.err.Error() }
func (failure *PhaseFailure) Unwrap() error { return failure.err }

type WorkerOptions struct {
	Store         WorkerStore
	Processor     DeletionProcessor
	WorkerID      string
	Now           func() time.Time
	LeaseDuration time.Duration
	RetryPolicy   RetryPolicy
	Telemetry     *DeletionTelemetry
}

type Worker struct {
	store         WorkerStore
	processor     DeletionProcessor
	workerID      string
	now           func() time.Time
	leaseDuration time.Duration
	retryPolicy   RetryPolicy
	telemetry     *DeletionTelemetry
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
		store:         options.Store,
		processor:     options.Processor,
		workerID:      options.WorkerID,
		now:           options.Now,
		leaseDuration: options.LeaseDuration,
		retryPolicy:   options.RetryPolicy,
		telemetry:     options.Telemetry,
	}, nil
}

func (worker *Worker) ProcessOne(ctx context.Context) (bool, error) {
	operation, found, err := worker.store.ClaimDue(ctx, worker.workerID, worker.leaseDuration)
	if err != nil || !found {
		return found, err
	}
	if worker.telemetry != nil {
		worker.telemetry.Phase(ctx, operation.Phase)
	}
	if err := worker.processor.Process(ctx, operation); err != nil {
		var pending *PhasePending
		if errors.As(err, &pending) {
			if persistErr := worker.store.DeferAttempt(ctx, operation, pending.At); persistErr != nil {
				return true, fmt.Errorf("defer account deletion phase: %w", persistErr)
			}
			return true, nil
		}
		kind, category := classifyPhaseFailure(err)
		nextAttempt := operation.AttemptCount + 1
		decision := worker.retryPolicy.Decide(worker.now().UTC(), operation.JobID.String(), nextAttempt, kind)
		if persistErr := worker.store.RecordFailure(ctx, operation, decision, category, nextAttempt); persistErr != nil {
			return true, fmt.Errorf("persist account deletion failure: %w", persistErr)
		}
		if worker.telemetry != nil {
			if decision.Action == RetrySchedule {
				worker.telemetry.AutomaticRetry(ctx, operation.Phase, category)
			} else {
				worker.telemetry.NeedsAttention(ctx, operation.Phase, category)
			}
		}
		return true, nil
	}
	if err := worker.store.CompleteAttempt(ctx, operation); err != nil {
		return true, fmt.Errorf("complete account deletion attempt: %w", err)
	}
	return true, nil
}

func classifyPhaseFailure(err error) (FailureKind, ErrorCategory) {
	var failure *PhaseFailure
	if errors.As(err, &failure) {
		return failure.kind, failure.category
	}
	return FailurePermanent, ErrorCategoryTerminal
}
