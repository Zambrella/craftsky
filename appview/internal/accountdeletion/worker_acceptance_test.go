package accountdeletion

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestWorkerAcceptanceReplaysWholeCleanupAndFinishesWithoutIndexerState(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	seedAccountDeletionOwner(t, pool, owner, "deleting", 1, 2, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,next_attempt_at
		) VALUES('10000000-0000-4000-8000-000000000003',$1,1,'active',$2,'deletion-oauth',1,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}

	cleanupCalls := 0
	failFirst := true
	cleaner, err := NewPrivateCleaner([]PrivateCleanupComponent{
		fakePrivateCleanupComponent{name: "private", run: func(got syntax.DID) error {
			if got != owner {
				t.Fatalf("cleanup owner = %s", got)
			}
			cleanupCalls++
			if failFirst {
				failFirst = false
				return errors.New("synthetic interruption")
			}
			return nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	pds := &leanAcceptancePDS{
		owner:   owner,
		records: []auth.PDSRecord{{URI: "at://did:plc:alice/social.craftsky.feed.post/post1"}},
	}
	newWorker := func(workerID string) *Worker {
		store := NewStore(pool, func() time.Time { return now })
		processor, err := NewLifecycleProcessor(LifecycleProcessorOptions{
			Store: store, Cleaner: cleaner, AcceptedCleanup: noOpAcceptedPrivateCleanup{}, BatchSize: 1,
			AccountTypes: accountTypeDeleterFunc(func(context.Context, syntax.DID) error { return nil }),
			NewPDSClient: func(context.Context, syntax.DID, auth.DeletionSessionAuthority) (auth.DeletionPDSClient, error) {
				return pds, nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		worker, err := NewWorker(WorkerOptions{
			Store: store, Processor: processor,
			Finalizer: deletionFinalizerFunc(func(ctx context.Context, operation ClaimedOperation) error {
				if _, err := pool.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1`, operation.JobID); err != nil {
					return err
				}
				_, err := pool.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did=$1`, operation.Owner)
				return err
			}),
			WorkerID: workerID,
			Now:      func() time.Time { return now }, LeaseDuration: time.Minute,
			RetryPolicy: RetryPolicy{Delays: []time.Duration{0}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return worker
	}

	if processed, err := newWorker("before-restart").ProcessOne(ctx); err != nil || !processed {
		t.Fatalf("first attempt = (%t, %v)", processed, err)
	}
	if processed, err := newWorker("after-restart").ProcessOne(ctx); err != nil || !processed {
		t.Fatalf("restarted attempt = (%t, %v)", processed, err)
	}
	if cleanupCalls != 2 || len(pds.records) != 0 || pds.listCalls < len(CraftskyRecordCollections())*2 {
		t.Fatalf("cleanupCalls=%d records=%v listCalls=%d", cleanupCalls, pds.records, pds.listCalls)
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE owner_did=$1`, owner, 0)
	assertCount(t, pool, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner, 0)
}

func TestDelayedPDSCommitCannotFinalizeDeletionAfterFirstEmptyScan(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 13, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000035")
	store := NewStore(pool, func() time.Time { return now })
	seedDeletionSafetyOwner(t, pool, owner, 7, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,next_attempt_at
		) VALUES($1,$2,7,'active',$3,'deletion-oauth',1,$3)
	`, jobID, owner, now); err != nil {
		t.Fatal(err)
	}
	const delayedURI = "at://did:plc:alice/social.craftsky.feed.post/delayed"
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,mutation_key,deterministic_key,
			request_fingerprint,record_fingerprint,remote_outcome,projection_disposition,
			repeat_forbidden,remote_deadline,dispatched_at,created_at,updated_at
		) VALUES(
			'delayed-write',$1,7,'pds_record','put_record','delayed-write',$2,
			decode(repeat('02',32),'hex'),decode(repeat('02',32),'hex'),'outcome_unknown_pre_transition',
			'hidden_non_active',true,$3,$4,$4,$4
		)
	`, owner, delayedURI, now.Add(time.Minute), now); err != nil {
		t.Fatal(err)
	}
	operation := ClaimedOperation{JobID: jobID, Owner: owner, OwnerGeneration: 7}
	if err := store.AdoptUncertainPDSAttempts(ctx, operation); err != nil {
		t.Fatal(err)
	}

	cleaner, err := NewPrivateCleaner([]PrivateCleanupComponent{
		fakePrivateCleanupComponent{name: "private", run: func(syntax.DID) error { return nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	pds := &leanAcceptancePDS{owner: owner}
	processor, err := NewLifecycleProcessor(LifecycleProcessorOptions{
		Store: store, Cleaner: cleaner, AcceptedCleanup: noOpAcceptedPrivateCleanup{}, BatchSize: 1,
		AccountTypes: accountTypeDeleterFunc(func(context.Context, syntax.DID) error { return nil }),
		NewPDSClient: func(context.Context, syntax.DID, auth.DeletionSessionAuthority) (auth.DeletionPDSClient, error) {
			return pds, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := NewWorker(WorkerOptions{
		Store: store, Processor: processor,
		Finalizer: deletionFinalizerFunc(func(context.Context, ClaimedOperation) error {
			return errors.New("unexpected finalization before safety convergence")
		}),
		WorkerID: "delayed-pds",
		Now:      func() time.Time { return now }, LeaseDuration: time.Minute,
		RetryPolicy: RetryPolicy{Delays: []time.Duration{0}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if processed, err := worker.ProcessOne(ctx); err != nil || !processed {
		t.Fatalf("first empty-scan attempt = (%t, %v)", processed, err)
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID, 1)
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_safety_tombstones WHERE operation_id=$1 AND state<>'settled'`, jobID, 1)

	pds.records = []auth.PDSRecord{{URI: syntax.ATURI(delayedURI)}}
	if processed, err := worker.ProcessOne(ctx); err != nil || !processed {
		t.Fatalf("delayed-commit reconciliation = (%t, %v)", processed, err)
	}
	if len(pds.records) != 0 {
		t.Fatalf("delayed PDS record survived exact-key reconciliation: %v", pds.records)
	}
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID, 1)
	assertCount(t, pool, `SELECT count(*) FROM account_deletion_safety_tombstones WHERE operation_id=$1 AND state<>'settled'`, jobID, 1)
}

func TestLifecycleProcessorRunsGenerationBoundAcceptedCleanupBeforeGenericCleanup(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-4000-8000-000000000036")
	leaseToken := uuid.MustParse("20000000-0000-4000-8000-000000000036")
	seedDeletionSafetyOwner(t, pool, owner, 7, now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,
			deletion_oauth_session_id,deletion_credential_generation,next_attempt_at,lease_owner,
			lease_token,lease_expires_at
		) VALUES($1,$2,7,'active',$3,'deletion-oauth',1,$3,'worker',$4,$5)
	`, jobID, owner, now, leaseToken, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	var calls []string
	acceptedCleanup := acceptedPrivateCleanupFunc(func(
		_ context.Context,
		gotJob uuid.UUID,
		gotOwner syntax.DID,
		gotGeneration int64,
	) error {
		if gotJob != jobID || gotOwner != owner || gotGeneration != 7 {
			t.Fatalf("accepted cleanup scope = (%s, %s, %d)", gotJob, gotOwner, gotGeneration)
		}
		calls = append(calls, "accepted")
		return nil
	})
	cleaner, err := NewPrivateCleaner([]PrivateCleanupComponent{
		fakePrivateCleanupComponent{name: "private", run: func(syntax.DID) error {
			calls = append(calls, "generic")
			return nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool, func() time.Time { return now })
	processor, err := NewLifecycleProcessor(LifecycleProcessorOptions{
		Store: store, Cleaner: cleaner, AcceptedCleanup: acceptedCleanup,
		AccountTypes: accountTypeDeleterFunc(func(_ context.Context, gotOwner syntax.DID) error {
			if gotOwner != owner {
				t.Fatalf("account type cleanup owner = %s", gotOwner)
			}
			calls = append(calls, "accountType")
			return nil
		}),
		NewPDSClient: func(context.Context, syntax.DID, auth.DeletionSessionAuthority) (auth.DeletionPDSClient, error) {
			return &leanAcceptancePDS{owner: owner}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	operation := ClaimedOperation{
		JobID: jobID, Owner: owner, OwnerGeneration: 7,
		LeaseToken: leaseToken, LeaseExpiresAt: now.Add(time.Minute),
	}
	if err := processor.Process(ctx, operation); err != nil {
		t.Fatal(err)
	}
	if got, want := calls, []string{"accepted", "generic", "accountType"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanup order = %v, want %v", got, want)
	}
}

type accountTypeDeleterFunc func(context.Context, syntax.DID) error

func (deleter accountTypeDeleterFunc) DeleteAccountType(ctx context.Context, owner syntax.DID) error {
	return deleter(ctx, owner)
}

type acceptedPrivateCleanupFunc func(context.Context, uuid.UUID, syntax.DID, int64) error

type noOpAcceptedPrivateCleanup struct{}

func (noOpAcceptedPrivateCleanup) PurgeAccepted(context.Context, uuid.UUID, syntax.DID, int64) error {
	return nil
}

func (cleanup acceptedPrivateCleanupFunc) PurgeAccepted(
	ctx context.Context,
	jobID uuid.UUID,
	owner syntax.DID,
	generation int64,
) error {
	return cleanup(ctx, jobID, owner, generation)
}

type leanAcceptancePDS struct {
	owner     syntax.DID
	records   []auth.PDSRecord
	listCalls int
}

func (pds *leanAcceptancePDS) ListRecords(_ context.Context, owner syntax.DID, collection, _ string, _ int) ([]auth.PDSRecord, string, error) {
	if owner != pds.owner {
		return nil, "", errors.New("wrong owner")
	}
	pds.listCalls++
	for _, record := range pds.records {
		if record.URI.Collection().String() == collection {
			return []auth.PDSRecord{record}, "", nil
		}
	}
	return nil, "", nil
}

func (pds *leanAcceptancePDS) DeleteRecord(_ context.Context, owner syntax.DID, collection, rkey string) error {
	if owner != pds.owner {
		return errors.New("wrong owner")
	}
	for index, record := range pds.records {
		if record.URI.Collection().String() == collection && record.URI.RecordKey().String() == rkey {
			pds.records = append(pds.records[:index], pds.records[index+1:]...)
			return nil
		}
	}
	return auth.ErrRecordNotFound
}
