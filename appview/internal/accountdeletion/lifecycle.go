package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

type DeletionPDSClientFactory func(context.Context, syntax.DID, string) (auth.DeletionPDSClient, error)

type LifecycleProcessorOptions struct {
	Store        *Store
	Cleaner      *PrivateCleaner
	Convergence  *ConvergenceVerifier
	NewPDSClient DeletionPDSClientFactory
	PollInterval time.Duration
	BatchSize    int
	Now          func() time.Time
	Telemetry    *DeletionTelemetry
}

type LifecycleProcessor struct {
	store        *Store
	cleaner      *PrivateCleaner
	convergence  *ConvergenceVerifier
	newPDSClient DeletionPDSClientFactory
	pollInterval time.Duration
	batchSize    int
	now          func() time.Time
	telemetry    *DeletionTelemetry
}

func NewLifecycleProcessor(options LifecycleProcessorOptions) (*LifecycleProcessor, error) {
	if options.Store == nil || options.Cleaner == nil || options.Convergence == nil || options.NewPDSClient == nil {
		return nil, errors.New("account deletion lifecycle dependencies are unavailable")
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 2 * time.Second
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &LifecycleProcessor{
		store: options.Store, cleaner: options.Cleaner, convergence: options.Convergence,
		newPDSClient: options.NewPDSClient, pollInterval: options.PollInterval,
		batchSize: options.BatchSize, now: options.Now,
		telemetry: options.Telemetry,
	}, nil
}

func (processor *LifecycleProcessor) Process(ctx context.Context, operation ClaimedOperation) error {
	switch operation.Phase {
	case PhaseQueued:
		return processor.store.AdvancePhase(ctx, operation, PhaseRemovingPrivateData)
	case PhaseRemovingPrivateData:
		if err := processor.cleaner.Run(ctx, operation.JobID, operation.Owner); err != nil {
			return NewPhaseFailure(FailureTransient, ErrorCategoryPrivateCleanup, err)
		}
		return processor.store.AdvancePhase(ctx, operation, PhaseRemovingCraftskyRecords)
	case PhaseRemovingCraftskyRecords:
		if _, err := processor.deletePDSRecords(ctx, operation); err != nil {
			return err
		}
		return processor.store.AdvancePhase(ctx, operation, PhaseWaitingForIndexerConvergence)
	case PhaseWaitingForIndexerConvergence:
		converged, err := processor.convergence.IsConverged(ctx, operation.JobID, operation.Owner)
		if err != nil {
			return NewPhaseFailure(FailureTransient, ErrorCategoryIndexer, err)
		}
		gates, err := processor.store.PrivateFinalizationGates(ctx, operation.JobID, operation.Owner, processor.cleaner.ComponentNames())
		if err != nil {
			return NewPhaseFailure(FailureTransient, ErrorCategoryPrivateCleanup, err)
		}
		if !converged || !gates.ScheduledObjectCleanupComplete {
			if processor.telemetry != nil {
				processor.telemetry.ConvergenceDelay(ctx, processor.pollInterval)
			}
			return &PhasePending{At: processor.now().UTC().Add(processor.pollInterval)}
		}
		return processor.store.AdvancePhase(ctx, operation, PhaseFinalizing)
	case PhaseFinalizing:
		result, err := processor.deletePDSRecords(ctx, operation)
		if err != nil {
			return err
		}
		if result.Listed != 0 {
			return processor.store.AdvancePhase(ctx, operation, PhaseWaitingForIndexerConvergence)
		}
		return processor.finalize(ctx, operation)
	default:
		return NewPhaseFailure(FailurePermanent, ErrorCategoryTerminal, fmt.Errorf("invalid lifecycle phase %q", operation.Phase))
	}
}

func (processor *LifecycleProcessor) deletePDSRecords(ctx context.Context, operation ClaimedOperation) (PDSDeletionResult, error) {
	sessionID, err := processor.store.BoundOAuthSession(ctx, operation.JobID, operation.Owner)
	if err != nil {
		return PDSDeletionResult{}, NewPhaseFailure(FailureOAuthUnusable, ErrorCategoryReauthentication, err)
	}
	client, err := processor.newPDSClient(ctx, operation.Owner, sessionID)
	if err != nil {
		return PDSDeletionResult{}, NewPhaseFailure(FailureOAuthUnusable, ErrorCategoryReauthentication, err)
	}
	result, err := NewPDSDeleter(client, processor.store, processor.batchSize).DeleteAll(
		ctx, operation.JobID.String(), operation.Owner,
	)
	if err != nil {
		return result, NewPhaseFailure(FailureTransient, ErrorCategoryPDS, err)
	}
	return result, nil
}

func (processor *LifecycleProcessor) finalize(ctx context.Context, operation ClaimedOperation) error {
	converged, err := processor.convergence.IsConverged(ctx, operation.JobID, operation.Owner)
	if err != nil {
		return NewPhaseFailure(FailureTransient, ErrorCategoryIndexer, err)
	}
	gates, err := processor.store.PrivateFinalizationGates(ctx, operation.JobID, operation.Owner, processor.cleaner.ComponentNames())
	if err != nil {
		return NewPhaseFailure(FailureTransient, ErrorCategoryPrivateCleanup, err)
	}
	gates.ExpectedRecordReceiptsComplete = converged
	gates.IndexedRecordsAbsent = converged
	gates.DerivedEffectsRetracted = converged
	gates.FinalPDSRescanEmpty = true
	// The bound authority is removed in the same transaction that removes the
	// operational row and inserts the minimized audit.
	gates.BoundOAuthSessionRemoved = true
	if !TerminalSuccessEligible(gates) {
		return &PhasePending{At: processor.now().UTC().Add(processor.pollInterval)}
	}
	if err := processor.store.FinalizeSuccess(ctx, operation.JobID, operation.Owner, processor.now().UTC()); err != nil {
		return NewPhaseFailure(FailureTransient, ErrorCategoryTerminal, err)
	}
	return nil
}

var _ DeletionProcessor = (*LifecycleProcessor)(nil)
