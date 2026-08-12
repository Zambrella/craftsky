package accountdeletion

import (
	"context"
	"errors"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
)

type DeletionPDSClientFactory func(context.Context, syntax.DID, string) (auth.DeletionPDSClient, error)

type LifecycleProcessorOptions struct {
	Store        *Store
	Cleaner      *PrivateCleaner
	NewPDSClient DeletionPDSClientFactory
	BatchSize    int
}

type LifecycleProcessor struct {
	store        *Store
	cleaner      *PrivateCleaner
	newPDSClient DeletionPDSClientFactory
	batchSize    int
}

func NewLifecycleProcessor(options LifecycleProcessorOptions) (*LifecycleProcessor, error) {
	if options.Store == nil || options.Cleaner == nil || options.NewPDSClient == nil {
		return nil, errors.New("account deletion lifecycle dependencies are unavailable")
	}
	if options.BatchSize <= 0 {
		options.BatchSize = 20
	}
	return &LifecycleProcessor{
		store: options.Store, cleaner: options.Cleaner,
		newPDSClient: options.NewPDSClient, batchSize: options.BatchSize,
	}, nil
}

func (processor *LifecycleProcessor) Process(ctx context.Context, operation ClaimedOperation) error {
	if err := processor.cleaner.Run(ctx, operation.Owner); err != nil {
		return NewDeletionFailure(ErrorCategoryPrivateCleanup, err)
	}
	sessionID, err := processor.store.BoundOAuthSession(ctx, operation.JobID, operation.Owner)
	if err != nil {
		return NewDeletionFailure(ErrorCategoryReauthentication, err)
	}
	client, err := processor.newPDSClient(ctx, operation.Owner, sessionID)
	if err != nil {
		return NewDeletionFailure(ErrorCategoryReauthentication, err)
	}
	if _, err := NewPDSDeleter(client, processor.batchSize).DeleteAll(ctx, operation.Owner); err != nil {
		return NewDeletionFailure(ErrorCategoryPDS, err)
	}
	return nil
}

var _ DeletionProcessor = (*LifecycleProcessor)(nil)
