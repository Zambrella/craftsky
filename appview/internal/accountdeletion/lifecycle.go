package accountdeletion

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
)

type DeletionPDSClientFactory func(
	context.Context,
	syntax.DID,
	auth.DeletionSessionAuthority,
) (auth.DeletionPDSClient, error)

type AcceptedPrivateCleanup interface {
	PurgeAccepted(context.Context, uuid.UUID, syntax.DID, int64) error
}

type LifecycleProcessorOptions struct {
	Store           *Store
	Cleaner         *PrivateCleaner
	AcceptedCleanup AcceptedPrivateCleanup
	NewPDSClient    DeletionPDSClientFactory
	BatchSize       int
}

type LifecycleProcessor struct {
	store           *Store
	cleaner         *PrivateCleaner
	acceptedCleanup AcceptedPrivateCleanup
	newPDSClient    DeletionPDSClientFactory
	batchSize       int
}

func NewLifecycleProcessor(options LifecycleProcessorOptions) (*LifecycleProcessor, error) {
	if options.Store == nil || options.Cleaner == nil || options.AcceptedCleanup == nil ||
		options.NewPDSClient == nil {
		return nil, errors.New("account deletion lifecycle dependencies are unavailable")
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 20
	}
	return &LifecycleProcessor{
		store: options.Store, cleaner: options.Cleaner, acceptedCleanup: options.AcceptedCleanup,
		newPDSClient: options.NewPDSClient, batchSize: options.BatchSize,
	}, nil
}

func (processor *LifecycleProcessor) Process(ctx context.Context, operation ClaimedOperation) error {
	if err := processor.store.AdoptUncertainPDSAttempts(ctx, operation); err != nil {
		return NewDeletionFailure(ErrorCategoryPDS, err)
	}
	if err := processor.acceptedCleanup.PurgeAccepted(
		ctx,
		operation.JobID,
		operation.Owner,
		operation.OwnerGeneration,
	); err != nil {
		return NewDeletionFailure(ErrorCategoryPrivateCleanup, err)
	}
	if err := processor.cleaner.Run(ctx, operation.Owner); err != nil {
		return NewDeletionFailure(ErrorCategoryPrivateCleanup, err)
	}
	authority, err := processor.store.BoundDeletionCredential(ctx, operation)
	if err != nil {
		return NewDeletionFailure(ErrorCategoryReauthentication, err)
	}
	client, err := processor.newPDSClient(ctx, operation.Owner, authority)
	if err != nil {
		return NewDeletionFailure(ErrorCategoryReauthentication, err)
	}
	deleter := NewPDSDeleter(client, processor.batchSize)
	claim, found, err := processor.store.ClaimDuePDSSafety(ctx, operation)
	if err != nil {
		return NewDeletionFailure(ErrorCategoryPDS, err)
	}
	if found {
		if err := deleter.DeleteExact(ctx, operation.Owner, claim.URI); err != nil {
			return NewDeletionFailure(ErrorCategoryPDS, err)
		}
		if err := processor.store.RecordPDSSafetyPending(
			ctx,
			claim,
			processor.store.now().UTC(),
			"absenceObservedUnbounded",
		); err != nil {
			return NewDeletionFailure(ErrorCategoryPDS, err)
		}
	}
	if _, err := deleter.DeleteAll(ctx, operation.Owner); err != nil {
		return NewDeletionFailure(ErrorCategoryPDS, err)
	}
	converged, err := processor.store.SafetyConverged(ctx, operation)
	if err != nil {
		return NewDeletionFailure(ErrorCategoryPDS, err)
	}
	if !converged {
		return NewDeletionFailure(ErrorCategoryPDS, ErrSafetyPending)
	}
	return nil
}

var _ DeletionProcessor = (*LifecycleProcessor)(nil)
