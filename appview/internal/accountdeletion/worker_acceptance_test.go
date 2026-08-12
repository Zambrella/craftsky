package accountdeletion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data) VALUES($1,'deletion-oauth','{}');
		INSERT INTO account_deletion_operations(
			id,owner_did,state,accepted_at,deletion_oauth_session_id,next_attempt_at
		) VALUES('10000000-0000-4000-8000-000000000003',$1,'active',$2,'deletion-oauth',$2);
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
			Store: store, Cleaner: cleaner, BatchSize: 1,
			NewPDSClient: func(context.Context, syntax.DID, string) (auth.DeletionPDSClient, error) {
				return pds, nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		worker, err := NewWorker(WorkerOptions{
			Store: store, Processor: processor, WorkerID: workerID,
			Now: func() time.Time { return now }, LeaseDuration: time.Minute,
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
