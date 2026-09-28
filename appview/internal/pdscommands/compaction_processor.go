package pdscommands

import (
	"context"
	"errors"
)

type commandCompactor interface {
	CompactExpired(context.Context, int) (int, error)
}

// CompactionProcessor removes bounded batches of expired terminal command
// payloads while leaving the minimal key-reuse tombstone behind.
type CompactionProcessor struct {
	store commandCompactor
	limit int
}

func NewCompactionProcessor(store commandCompactor, limit int) (*CompactionProcessor, error) {
	if store == nil {
		return nil, errors.New("command compaction store is required")
	}
	if limit < 1 {
		return nil, errors.New("command compaction batch size must be positive")
	}
	return &CompactionProcessor{store: store, limit: limit}, nil
}

func (processor *CompactionProcessor) ProcessBatch(ctx context.Context) (int, error) {
	return processor.store.CompactExpired(ctx, processor.limit)
}
