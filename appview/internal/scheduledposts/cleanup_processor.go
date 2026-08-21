package scheduledposts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
	RecordCleanupAbsence(context.Context, CleanupClaim, time.Time) (ObjectCleanupSettlement, error)
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
	OwnerFence    ownerSharedFence
	Now           func() time.Time
	BatchSize     int
	LeaseDuration time.Duration
	RetryDelay    time.Duration
	Observer      OperationalObserver
}

type ownerSharedFence interface {
	WithShared(context.Context, []syntax.DID, func(context.Context) error) error
}

type CleanupProcessor struct {
	store         CleanupOperationStore
	objects       PrivateObjectStore
	ownerFence    ownerSharedFence
	now           func() time.Time
	batchSize     int
	leaseDuration time.Duration
	retryDelay    time.Duration
	observer      OperationalObserver
}

func NewCleanupProcessor(options CleanupProcessorOptions) (*CleanupProcessor, error) {
	if options.Store == nil || options.Objects == nil || options.OwnerFence == nil {
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
		store: options.Store, objects: options.Objects, ownerFence: options.OwnerFence,
		now:       options.Now,
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
	claimCtx, cancelClaim := context.WithTimeout(ctx, processor.leaseDuration)
	defer cancelClaim()
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
	return processor.ownerFence.WithShared(
		claimCtx,
		[]syntax.DID{claim.OwnerDID},
		func(fenceCtx context.Context) error {
			var err error
			outcomeErrorClass, err = processor.processClaimFenced(fenceCtx, claim, now)
			return err
		},
	)
}

func (processor *CleanupProcessor) processClaimFenced(
	ctx context.Context,
	claim CleanupClaim,
	now time.Time,
) (errorClass string, processErr error) {
	errorClass = "none"
	guard, err := processor.store.AcquireCleanupEffect(ctx, claim)
	if err != nil {
		return errorClass, fmt.Errorf("acquire scheduled cleanup effect: %w", err)
	}
	defer func() {
		if releaseErr := guard.Release(ctx); releaseErr != nil && processErr == nil {
			processErr = fmt.Errorf("release scheduled cleanup effect: %w", releaseErr)
		}
	}()

	unreferenced, err := processor.store.PrepareCleanupDelete(ctx, claim)
	if err != nil {
		return errorClass, fmt.Errorf("prepare scheduled cleanup: %w", err)
	}
	if !unreferenced {
		return errorClass, nil
	}
	if err := processor.objects.Delete(ctx, claim.ObjectKey); err != nil {
		errorClass = CleanupErrorObjectDeleteFailed
		if retryErr := processor.store.RetryCleanup(
			ctx,
			claim,
			now.Add(processor.retryDelay),
			CleanupErrorObjectDeleteFailed,
			now,
		); retryErr != nil {
			return errorClass, fmt.Errorf("retry scheduled cleanup: %w", retryErr)
		}
		return errorClass, nil
	}
	present, err := processor.objects.Exists(ctx, claim.ObjectKey)
	if err != nil {
		if retryErr := processor.store.RetryCleanup(
			ctx,
			claim,
			now.Add(processor.retryDelay),
			CleanupErrorObjectDeleteFailed,
			now,
		); retryErr != nil {
			return errorClass, fmt.Errorf("retry scheduled cleanup existence check: %w", retryErr)
		}
		return errorClass, nil
	}
	if present {
		if retryErr := processor.store.RetryCleanup(
			ctx,
			claim,
			now.Add(processor.retryDelay),
			CleanupErrorObjectDeleteFailed,
			now,
		); retryErr != nil {
			return errorClass, fmt.Errorf("retry scheduled cleanup present object: %w", retryErr)
		}
		return errorClass, nil
	}
	settlement, err := processor.store.RecordCleanupAbsence(ctx, claim, now)
	if err != nil {
		return errorClass, fmt.Errorf("record scheduled cleanup absence: %w", err)
	}
	if !settlement.ProvesSettlement() {
		errorClass = CleanupErrorSettlementPending
		if retryErr := processor.store.RetryCleanup(
			ctx,
			claim,
			now.Add(processor.retryDelay),
			CleanupErrorSettlementPending,
			now,
		); retryErr != nil {
			return errorClass, fmt.Errorf("retain scheduled cleanup settlement: %w", retryErr)
		}
		return errorClass, nil
	}
	if err := processor.store.CompleteCleanup(ctx, claim); err != nil {
		return errorClass, fmt.Errorf("complete scheduled cleanup: %w", err)
	}
	return errorClass, nil
}
