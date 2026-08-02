package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	DefaultCleanupBatch         = 20
	MaximumCleanupBatch         = 100
	DefaultCleanupLeaseDuration = time.Minute
	DefaultCleanupRetryDelay    = time.Minute
)

type CleanupOperationStore interface {
	ClaimCleanup(context.Context, int, time.Time, time.Duration) ([]CleanupClaim, error)
	AcquireCleanupEffect(context.Context, CleanupClaim) (CleanupEffectGuard, error)
	PrepareCleanupDelete(context.Context, CleanupClaim) (bool, error)
	CompleteCleanup(context.Context, CleanupClaim) error
	RetryCleanup(context.Context, CleanupClaim, time.Time, string, time.Time) error
}

type CleanupEffectGuard interface {
	Release(context.Context) error
}

type lifecycleSweeper interface {
	SweepExpiredLifecycle(context.Context, time.Time) error
}

type CleanupProcessorOptions struct {
	Store         CleanupOperationStore
	Objects       PrivateObjectStore
	Now           func() time.Time
	BatchSize     int
	LeaseDuration time.Duration
	RetryDelay    time.Duration
	Observer      OperationalObserver
}

type CleanupProcessor struct {
	store         CleanupOperationStore
	objects       PrivateObjectStore
	now           func() time.Time
	batchSize     int
	leaseDuration time.Duration
	retryDelay    time.Duration
	observer      OperationalObserver
}

func NewCleanupProcessor(options CleanupProcessorOptions) (*CleanupProcessor, error) {
	if options.Store == nil || options.Objects == nil {
		return nil, errors.New("scheduled cleanup processor dependencies are required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.BatchSize == 0 {
		options.BatchSize = DefaultCleanupBatch
	}
	if options.LeaseDuration == 0 {
		options.LeaseDuration = DefaultCleanupLeaseDuration
	}
	if options.RetryDelay == 0 {
		options.RetryDelay = DefaultCleanupRetryDelay
	}
	if options.BatchSize < 1 || options.BatchSize > MaximumCleanupBatch ||
		options.LeaseDuration <= 0 || options.RetryDelay <= 0 {
		return nil, errors.New("invalid scheduled cleanup processor options")
	}
	return &CleanupProcessor{
		store: options.Store, objects: options.Objects, now: options.Now,
		batchSize: options.BatchSize, leaseDuration: options.LeaseDuration,
		retryDelay: options.RetryDelay,
		observer:   options.Observer,
	}, nil
}

func (processor *CleanupProcessor) ProcessBatch(ctx context.Context) (int, error) {
	if processor == nil {
		return 0, errors.New("scheduled cleanup processor is nil")
	}
	now := processor.now().UTC()
	if snapshotter, ok := processor.store.(cleanupQueueSnapshotter); ok {
		snapshot, err := snapshotter.CleanupQueueSnapshot(ctx, now)
		if err != nil {
			if processor.observer != nil {
				processor.observer.ObserveScheduledOperation(
					"cleanup", "failure", "dependency_unavailable", 0,
				)
			}
			return 0, err
		}
		if processor.observer != nil {
			processor.observer.ObserveScheduledCleanupQueue(
				snapshot.Pending,
				snapshot.OldestAge,
			)
		}
	}
	if sweeper, ok := processor.store.(lifecycleSweeper); ok {
		if err := sweeper.SweepExpiredLifecycle(ctx, now); err != nil {
			if processor.observer != nil {
				processor.observer.ObserveScheduledOperation(
					"cleanup_sweep", "failure", "dependency_unavailable", 0,
				)
			}
			return 0, fmt.Errorf("sweep scheduled lifecycle: %w", err)
		}
		if processor.observer != nil {
			processor.observer.ObserveScheduledOperation(
				"cleanup_sweep", "success", "none", 0,
			)
		}
	}
	claims, err := processor.store.ClaimCleanup(
		ctx,
		processor.batchSize,
		now,
		processor.leaseDuration,
	)
	if err != nil {
		return 0, fmt.Errorf("claim scheduled cleanup: %w", err)
	}
	for index, claim := range claims {
		if err := processor.processClaim(ctx, claim, now); err != nil {
			return index, err
		}
	}
	return len(claims), nil
}

func (processor *CleanupProcessor) processClaim(
	ctx context.Context,
	claim CleanupClaim,
	now time.Time,
) (processErr error) {
	started := time.Now()
	outcomeErrorClass := "none"
	defer func() {
		if processor.observer == nil {
			return
		}
		result := "success"
		errorClass := outcomeErrorClass
		if processErr != nil {
			result = "failure"
			errorClass = "dependency_unavailable"
		} else if outcomeErrorClass != "none" {
			result = "failure"
		}
		processor.observer.ObserveScheduledOperation(
			"cleanup", result, errorClass, time.Since(started),
		)
	}()
	guard, err := processor.store.AcquireCleanupEffect(ctx, claim)
	if err != nil {
		return fmt.Errorf("acquire scheduled cleanup effect: %w", err)
	}
	defer func() {
		if releaseErr := guard.Release(ctx); releaseErr != nil && processErr == nil {
			processErr = fmt.Errorf("release scheduled cleanup effect: %w", releaseErr)
		}
	}()

	unreferenced, err := processor.store.PrepareCleanupDelete(ctx, claim)
	if err != nil {
		return fmt.Errorf("prepare scheduled cleanup: %w", err)
	}
	if !unreferenced {
		return nil
	}
	if err := processor.objects.Delete(ctx, claim.ObjectKey); err != nil {
		outcomeErrorClass = CleanupErrorObjectDeleteFailed
		if retryErr := processor.store.RetryCleanup(
			ctx,
			claim,
			now.Add(processor.retryDelay),
			CleanupErrorObjectDeleteFailed,
			now,
		); retryErr != nil {
			return fmt.Errorf("retry scheduled cleanup: %w", retryErr)
		}
		return nil
	}
	if err := processor.store.CompleteCleanup(ctx, claim); err != nil {
		return fmt.Errorf("complete scheduled cleanup: %w", err)
	}
	return nil
}
