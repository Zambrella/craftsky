package accountdeletion

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ownerlifecycle"
)

type ExpiredIntent struct {
	JobID uuid.UUID
	Owner syntax.DID
}

type ExpiredIntentSource interface {
	ListExpiredIntents(context.Context, int) ([]ExpiredIntent, error)
}

type IntentExpirer interface {
	ExpireIntent(context.Context, uuid.UUID, syntax.DID) error
}

type IntentExpiryProcessorOptions struct {
	Source    ExpiredIntentSource
	Expirer   IntentExpirer
	BatchSize int
}

// IntentExpiryProcessor returns reversible deletion intents to the lifecycle
// state selected by current membership. Claims need no lease: the owner fence,
// generation CAS, and idempotent operation deletion are the linearization
// points, so multiple processes may safely observe the same bounded batch.
type IntentExpiryProcessor struct {
	source    ExpiredIntentSource
	expirer   IntentExpirer
	batchSize int
}

func NewIntentExpiryProcessor(options IntentExpiryProcessorOptions) (*IntentExpiryProcessor, error) {
	if options.Source == nil || options.Expirer == nil || options.BatchSize <= 0 || options.BatchSize > 1000 {
		return nil, errors.New("invalid account deletion intent expiry processor")
	}
	return &IntentExpiryProcessor{
		source: options.Source, expirer: options.Expirer, batchSize: options.BatchSize,
	}, nil
}

func (processor *IntentExpiryProcessor) RunOnce(ctx context.Context) (int, error) {
	if processor == nil || processor.source == nil || processor.expirer == nil {
		return 0, errors.New("account deletion intent expiry processor is unavailable")
	}
	intents, err := processor.source.ListExpiredIntents(ctx, processor.batchSize)
	if err != nil {
		return 0, fmt.Errorf("list expired account deletion intents: %w", err)
	}
	processed := 0
	for _, intent := range intents {
		err := processor.expirer.ExpireIntent(ctx, intent.JobID, intent.Owner)
		switch {
		case err == nil:
			processed++
		case errors.Is(err, ErrOperationNotFound),
			errors.Is(err, ErrPointOfNoReturn),
			errors.Is(err, ownerlifecycle.ErrGenerationChanged),
			errors.Is(err, ownerlifecycle.ErrTerminalOwner):
			// Another transition won after this process read the bounded batch.
			processed++
		default:
			return processed, fmt.Errorf("expire account deletion intent: %w", err)
		}
	}
	return processed, nil
}

// ProcessBatch lets the expiry processor share the process-wide bounded batch
// worker runner used by the other lifecycle cleanup jobs.
func (processor *IntentExpiryProcessor) ProcessBatch(ctx context.Context) (int, error) {
	return processor.RunOnce(ctx)
}
