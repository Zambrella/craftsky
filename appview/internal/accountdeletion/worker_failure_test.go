package accountdeletion

import (
	"bytes"
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"log/slog"
	"strings"
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

	var local bytes.Buffer
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	operation := ClaimedOperation{
		JobID:        uuid.MustParse("10000000-0000-4000-8000-000000000099"),
		Owner:        syntax.DID("did:plc:alice"),
		AttemptCount: 99,
		LeaseToken:   uuid.MustParse("20000000-0000-4000-8000-000000000099"),
	}
	store := &recordingRetryWorkerStore{operation: operation}
	worker, err := NewWorker(WorkerOptions{
		Store: store, Logger: slog.New(slog.NewJSONHandler(&local, nil)),
		Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error {
			return NewDeletionFailure(ErrorCategoryPDS, &pgconn.PgError{Code: "40001", Message: "private deletion target canary"})
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
	for _, positive := range []string{"40001", "did:plc:alice", operation.JobID.String(), "failure_stage", "retry"} {
		if !strings.Contains(local.String(), positive) {
			t.Fatalf("IT-007 deletion retry lost %s: %s", positive, local.String())
		}
	}
	for _, private := range []string{"private deletion target canary", operation.LeaseToken.String()} {
		if strings.Contains(local.String(), private) {
			t.Fatalf("private retry leaked %s", private)
		}
	}

	if store.completed {
		t.Fatal("failed attempt was finalized")
	}
}

func TestWorkerSchedulesRetryWhenRemoteSafetyAppearsBeforeFinalization(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
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
		Store: store, Logger: slog.New(slog.NewJSONHandler(&output, nil)),
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
	for _, selected := range []string{"did:plc:alice", operation.JobID.String(), "finalization", "retry"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s in %s", selected, output.String())
		}
	}
	if strings.Contains(output.String(), operation.LeaseToken.String()) {
		t.Error("lease capability leaked")
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

func TestIT007DeletionFinalizationFailureKeepsReturnAndCause(t *testing.T) {
	var output bytes.Buffer
	cause := &pgconn.PgError{Code: "40001", Detail: "PRIVATE_DELETION_PAYLOAD"}
	operation := ClaimedOperation{JobID: uuid.New(), Owner: syntax.DID("did:plc:owner"), LeaseToken: uuid.New(), AttemptCount: 2}
	store := &recordingRetryWorkerStore{operation: operation, completeErr: cause}
	worker, err := NewWorker(WorkerOptions{Store: store, Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error { return nil }), Finalizer: deletionFinalizerFunc(store.CompleteAttempt), WorkerID: "worker", Now: time.Now, LeaseDuration: time.Minute, Logger: slog.New(slog.NewJSONHandler(&output, nil))})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.ProcessOne(context.Background())
	if !processed || !errors.Is(err, cause) {
		t.Fatalf("processed=%t err=%v", processed, err)
	}
	for _, selected := range []string{"40001", "did:plc:owner", operation.JobID.String(), "finalization", "error"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s in %s", selected, output.String())
		}
	}
	for _, private := range []string{"PRIVATE_DELETION_PAYLOAD", operation.LeaseToken.String()} {
		if strings.Contains(output.String(), private) {
			t.Errorf("leaked %s", private)
		}
	}
}

func TestIT007DeletionFailurePersistenceRetainsBothCauses(t *testing.T) {
	var output bytes.Buffer
	original := NewDeletionFailure(ErrorCategoryPDS, &pgconn.PgError{Code: "40001", Detail: "PRIVATE_ORIGINAL"})
	persist := &pgconn.PgError{Code: "08006", Detail: "PRIVATE_PERSIST"}
	operation := ClaimedOperation{JobID: uuid.New(), Owner: syntax.DID("did:plc:owner"), LeaseToken: uuid.New(), AttemptCount: 2}
	store := &recordingRetryWorkerStore{operation: operation, persistErr: persist}
	worker, err := NewWorker(WorkerOptions{Store: store, Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error { return original }), Finalizer: deletionFinalizerFunc(store.CompleteAttempt), WorkerID: "worker", Now: time.Now, LeaseDuration: time.Minute, Logger: slog.New(slog.NewJSONHandler(&output, nil))})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.ProcessOne(context.Background())
	if !processed || !errors.Is(err, persist) {
		t.Fatalf("processed=%t err=%v", processed, err)
	}
	for _, selected := range []string{"40001", "08006", "did:plc:owner", operation.JobID.String(), "failure_persistence"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s in %s", selected, output.String())
		}
	}
	if strings.Contains(output.String(), "PRIVATE_") || strings.Contains(output.String(), operation.LeaseToken.String()) {
		t.Error("private data leaked")
	}
}
func TestIT007DeletionSafetyRetryPersistenceRetainsSourceCause(t *testing.T) {
	var output bytes.Buffer
	original := ErrSafetyPending
	persist := &pgconn.PgError{Code: "08006", Detail: "PRIVATE_PERSIST"}
	operation := ClaimedOperation{JobID: uuid.New(), Owner: syntax.DID("did:plc:owner"), LeaseToken: uuid.New(), AttemptCount: 2}
	store := &recordingRetryWorkerStore{operation: operation, persistErr: persist, completeErr: original}
	worker, err := NewWorker(WorkerOptions{Store: store, Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error { return nil }), Finalizer: deletionFinalizerFunc(store.CompleteAttempt), WorkerID: "worker", Now: time.Now, LeaseDuration: time.Minute, Logger: slog.New(slog.NewJSONHandler(&output, nil))})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.ProcessOne(context.Background())
	if !processed || !errors.Is(err, persist) {
		t.Fatalf("processed=%t err=%v", processed, err)
	}
	for _, selected := range []string{"*errors.errorString", "08006", "did:plc:owner", operation.JobID.String(), "failure_persistence"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s in %s", selected, output.String())
		}
	}
	if strings.Contains(output.String(), "PRIVATE_") || strings.Contains(output.String(), operation.LeaseToken.String()) {
		t.Error("private data leaked")
	}
}

type recordingRetryWorkerStore struct {
	operation        ClaimedOperation
	claimed          bool
	nextAt           time.Time
	category         ErrorCategory
	nextAttemptCount int
	completed        bool
	completeErr      error
	persistErr       error
	claimErr         error
}

func (store *recordingRetryWorkerStore) ClaimDue(context.Context, string, time.Duration) (ClaimedOperation, bool, error) {
	if store.claimErr != nil {
		return ClaimedOperation{}, false, store.claimErr
	}
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
	return store.persistErr
}

func (store *recordingRetryWorkerStore) CompleteAttempt(context.Context, ClaimedOperation) error {
	store.completed = true
	return store.completeErr
}

var _ WorkerStore = (*recordingRetryWorkerStore)(nil)

func TestIT007DeletionClaimFailureRetainsCauseWithoutInventingOwner(t *testing.T) {
	var output bytes.Buffer
	cause := &pgconn.PgError{Code: "08006", Detail: "PRIVATE_CLAIM"}
	store := &recordingRetryWorkerStore{claimErr: cause}
	worker, err := NewWorker(WorkerOptions{Store: store, Processor: deletionProcessorFunc(func(context.Context, ClaimedOperation) error { return nil }), Finalizer: deletionFinalizerFunc(store.CompleteAttempt), WorkerID: "worker", Now: time.Now, LeaseDuration: time.Minute, Logger: slog.New(slog.NewJSONHandler(&output, nil))})
	if err != nil {
		t.Fatal(err)
	}
	processed, err := worker.ProcessOne(context.Background())
	if processed || !errors.Is(err, cause) {
		t.Fatal("claim outcome changed")
	}
	for _, selected := range []string{"08006", "claim"} {
		if !strings.Contains(output.String(), selected) {
			t.Errorf("missing %s: %s", selected, output.String())
		}
	}
	for _, excluded := range []string{"PRIVATE_", "operation_account_did", "00000000-0000-0000-0000-000000000000"} {
		if strings.Contains(output.String(), excluded) {
			t.Errorf("invented/protected field %s", excluded)
		}
	}
}
