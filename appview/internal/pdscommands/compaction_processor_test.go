package pdscommands

import (
	"context"
	"errors"
	"testing"
)

type recordingCommandCompactor struct {
	limit int
	count int
	err   error
}

func (compactor *recordingCommandCompactor) CompactExpired(_ context.Context, limit int) (int, error) {
	compactor.limit = limit
	return compactor.count, compactor.err
}

func TestCompactionProcessorRunsBoundedStoreBatch(t *testing.T) {
	store := &recordingCommandCompactor{count: 7}
	processor, err := NewCompactionProcessor(store, 25)
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if processed != 7 || store.limit != 25 {
		t.Fatalf("ProcessBatch() = %d with limit %d, want 7 with limit 25", processed, store.limit)
	}
}

func TestCompactionProcessorPreservesStoreError(t *testing.T) {
	want := errors.New("compact failed")
	processor, err := NewCompactionProcessor(&recordingCommandCompactor{err: want}, 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = processor.ProcessBatch(context.Background())
	if !errors.Is(err, want) {
		t.Fatalf("ProcessBatch() error = %v, want %v", err, want)
	}
}

func TestNewCompactionProcessorRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name  string
		store commandCompactor
		limit int
	}{
		{name: "missing store", limit: 1},
		{name: "zero limit", store: &recordingCommandCompactor{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewCompactionProcessor(test.store, test.limit); err == nil {
				t.Fatal("NewCompactionProcessor() error = nil")
			}
		})
	}
}
