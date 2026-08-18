package accountdeletion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
)

func TestPDSDeletionRestartConvergesAfterUncertainSideEffect(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:alice")
	pds := &uncertainDeletePDS{
		owner:               owner,
		record:              auth.PDSRecord{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/uncertain")},
		failAfterSideEffect: true,
	}
	deleter := NewPDSDeleter(pds, 20)
	if _, err := deleter.DeleteAll(context.Background(), owner); err == nil {
		t.Fatal("uncertain PDS failure unexpectedly succeeded")
	}
	if pds.record.URI != "" {
		t.Fatal("injected failure did not occur after the PDS side effect")
	}

	result, err := deleter.DeleteAll(context.Background(), owner)
	if err != nil || result.Listed != 0 {
		t.Fatalf("restart convergence = (%+v, %v)", result, err)
	}
}

type uncertainDeletePDS struct {
	owner               syntax.DID
	record              auth.PDSRecord
	failAfterSideEffect bool
}

func (pds *uncertainDeletePDS) ListRecords(_ context.Context, repo syntax.DID, collection string, _ string, _ int) ([]auth.PDSRecord, string, error) {
	if repo != pds.owner {
		return nil, "", errors.New("wrong owner")
	}
	if pds.record.URI != "" && pds.record.URI.Collection() == syntax.NSID(collection) {
		return []auth.PDSRecord{pds.record}, "", nil
	}
	return nil, "", nil
}

func (pds *uncertainDeletePDS) DeleteRecord(_ context.Context, _ syntax.DID, _ string, _ string) error {
	pds.record = auth.PDSRecord{}
	if pds.failAfterSideEffect {
		pds.failAfterSideEffect = false
		return errors.New("synthetic connection loss after side effect")
	}
	return nil
}

var _ auth.DeletionPDSClient = (*uncertainDeletePDS)(nil)

func TestWorkerSchedulesCappedRetryThroughProductionBoundary(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	operation := ClaimedOperation{
		JobID:        uuid.MustParse("10000000-0000-4000-8000-000000000099"),
		Owner:        syntax.DID("did:plc:alice"),
		AttemptCount: 99,
		LeaseToken:   uuid.MustParse("20000000-0000-4000-8000-000000000099"),
	}
	store := &recordingRetryWorkerStore{operation: operation}
	worker, err := NewWorker(WorkerOptions{
		Store: store,
		Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error {
			return NewDeletionFailure(ErrorCategoryPDS, errors.New("synthetic PDS outage"))
		}),
		Finalizer:     deletionFinalizerFunc(store.CompleteAttempt),
		WorkerID:      "worker",
		Now:           func() time.Time { return now },
		LeaseDuration: time.Minute,
		RetryPolicy: RetryPolicy{
			Delays: []time.Duration{0, time.Minute, 6 * time.Hour},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	processed, err := worker.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("process = (%t, %v)", processed, err)
	}
	if store.nextAttemptCount != 100 || store.category != ErrorCategoryPDS || !store.nextAt.Equal(now.Add(6*time.Hour)) {
		t.Fatalf("failure update attempt=%d category=%q next=%s", store.nextAttemptCount, store.category, store.nextAt)
	}
	if store.completed {
		t.Fatal("failed attempt was finalized")
	}
}

func TestWorkerSchedulesRetryWhenRemoteSafetyAppearsBeforeFinalization(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 14, 15, 0, 0, 0, time.UTC)
	operation := ClaimedOperation{
		JobID:           uuid.MustParse("10000000-0000-4000-8000-000000000037"),
		Owner:           syntax.DID("did:plc:alice"),
		OwnerGeneration: 7,
		AttemptCount:    2,
		LeaseToken:      uuid.MustParse("20000000-0000-4000-8000-000000000037"),
		LeaseExpiresAt:  now.Add(time.Minute),
	}
	store := &recordingRetryWorkerStore{operation: operation, completeErr: ErrSafetyPending}
	worker, err := NewWorker(WorkerOptions{
		Store: store,
		Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error {
			return nil
		}),
		Finalizer: deletionFinalizerFunc(store.CompleteAttempt),
		WorkerID:  "worker", Now: func() time.Time { return now },
		LeaseDuration: time.Minute,
		RetryPolicy:   RetryPolicy{Delays: []time.Duration{0, time.Second, time.Minute}},
	})
	if err != nil {
		t.Fatal(err)
	}

	processed, err := worker.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("process = (%t, %v)", processed, err)
	}
	if store.nextAttemptCount != 3 || store.category != ErrorCategoryPDS ||
		!store.nextAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("safety retry attempt=%d category=%q next=%s", store.nextAttemptCount, store.category, store.nextAt)
	}
}

func TestWorkerStopsRetryingAfterDeletionCredentialRequiresReauthentication(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 14, 20, 0, 0, 0, time.UTC)
	operation := ClaimedOperation{
		JobID: uuid.New(), Owner: syntax.DID("did:plc:reauth-required"),
		OwnerGeneration: 3, LeaseToken: uuid.New(), LeaseExpiresAt: now.Add(time.Minute),
	}
	store := &recordingRetryWorkerStore{operation: operation}
	worker, err := NewWorker(WorkerOptions{
		Store: store,
		Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error {
			return NewDeletionFailure(
				ErrorCategoryReauthentication,
				errors.Join(auth.ErrDeletionReauthenticationRequired, errors.New("terminal refresh")),
			)
		}),
		Finalizer: deletionFinalizerFunc(store.CompleteAttempt),
		WorkerID:  "worker", Now: func() time.Time { return now }, LeaseDuration: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.ProcessOne(context.Background())
	if err != nil || !processed {
		t.Fatalf("process reauthentication boundary = (%t, %v)", processed, err)
	}
	if store.nextAttemptCount != 0 || store.completed {
		t.Fatalf("reauth-required job was retried/finalized: retry=%d complete=%t", store.nextAttemptCount, store.completed)
	}
}

type deletionProcessorFunc func(context.Context, ClaimedOperation) error

type deletionFinalizerFunc func(context.Context, ClaimedOperation) error

func (process deletionProcessorFunc) Process(ctx context.Context, operation ClaimedOperation) error {
	return process(ctx, operation)
}

func (finalize deletionFinalizerFunc) CompleteAccepted(ctx context.Context, operation ClaimedOperation) error {
	return finalize(ctx, operation)
}

type recordingRetryWorkerStore struct {
	operation        ClaimedOperation
	claimed          bool
	nextAt           time.Time
	category         ErrorCategory
	nextAttemptCount int
	completed        bool
	completeErr      error
}

func (store *recordingRetryWorkerStore) ClaimDue(context.Context, string, time.Duration) (ClaimedOperation, bool, error) {
	if store.claimed {
		return ClaimedOperation{}, false, nil
	}
	store.claimed = true
	return store.operation, true, nil
}

func (store *recordingRetryWorkerStore) RecordFailure(
	_ context.Context,
	_ ClaimedOperation,
	nextAt time.Time,
	category ErrorCategory,
	nextAttemptCount int,
) error {
	store.nextAt = nextAt
	store.category = category
	store.nextAttemptCount = nextAttemptCount
	return nil
}

func (store *recordingRetryWorkerStore) CompleteAttempt(context.Context, ClaimedOperation) error {
	store.completed = true
	return store.completeErr
}

var _ WorkerStore = (*recordingRetryWorkerStore)(nil)
