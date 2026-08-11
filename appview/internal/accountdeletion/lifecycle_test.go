package accountdeletion

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestDurableLifecycleWaitsForReceiptThenFinalizesMinimalAudit(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000940")
	owner := syntax.DID("did:plc:alice")
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/post1")
	current := time.Date(2026, 8, 10, 16, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data) VALUES($1,'deletion-oauth','{}')
	`, owner); err != nil {
		t.Fatalf("seed lifecycle OAuth: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(
			id,owner_did,state,phase,accepted_at,deletion_oauth_session_id,next_attempt_at
		) VALUES($2,$1,'active','queued',$3,'deletion-oauth',$3)
	`, owner, jobID, current); err != nil {
		t.Fatalf("seed lifecycle operation: %v", err)
	}

	store := NewStore(pool, func() time.Time { return current })
	metrics := &recordingDeletionMetrics{}
	telemetry := NewDeletionTelemetry(nil, metrics)
	store.SetTelemetry(telemetry)
	cleaner, err := NewPrivateCleaner(store, []PrivateCleanupComponent{
		fakePrivateCleanupComponent{name: "databasePrivate", run: func(uuid.UUID, syntax.DID) error { return nil }},
		fakePrivateCleanupComponent{name: "scheduledPosts", run: func(uuid.UUID, syntax.DID) error { return nil }},
		fakePrivateCleanupComponent{name: "instagram", run: func(uuid.UUID, syntax.DID) error { return nil }},
	})
	if err != nil {
		t.Fatalf("construct lifecycle cleaner: %v", err)
	}
	pds := &lifecycleDeletionPDS{owner: owner, records: []auth.PDSRecord{{URI: uri}}}
	processor, err := NewLifecycleProcessor(LifecycleProcessorOptions{
		Store:       store,
		Cleaner:     cleaner,
		Convergence: NewConvergenceVerifier(pool),
		NewPDSClient: func(_ context.Context, gotOwner syntax.DID, sessionID string) (auth.DeletionPDSClient, error) {
			if gotOwner != owner || sessionID != "deletion-oauth" {
				return nil, errors.New("wrong bound OAuth scope")
			}
			return pds, nil
		},
		PollInterval: 2 * time.Second,
		Now:          func() time.Time { return current },
		Telemetry:    telemetry,
	})
	if err != nil {
		t.Fatalf("construct lifecycle processor: %v", err)
	}
	worker, err := NewWorker(WorkerOptions{
		Store: store, Processor: processor, WorkerID: "lifecycle-worker",
		Now: func() time.Time { return current }, LeaseDuration: 2 * time.Minute,
		RetryPolicy: DefaultRetryPolicy(),
		Telemetry:   telemetry,
	})
	if err != nil {
		t.Fatalf("construct lifecycle worker: %v", err)
	}

	// queued -> private -> PDS -> waiting; the fourth pass is a normal
	// convergence poll and must not consume the failure retry budget.
	for pass := 0; pass < 4; pass++ {
		processed, err := worker.ProcessOne(ctx)
		if err != nil || !processed {
			t.Fatalf("lifecycle pass %d processed=%t err=%v", pass, processed, err)
		}
	}
	var status Status
	var phase Phase
	var attempt int
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state,phase,attempt_count,next_attempt_at
		FROM account_deletion_operations WHERE id=$1
	`, jobID).Scan(&status, &phase, &attempt, &nextAttempt); err != nil {
		t.Fatalf("read waiting lifecycle: %v", err)
	}
	if status != StatusActive || phase != PhaseWaitingForIndexerConvergence || attempt != 0 || !nextAttempt.Equal(current.Add(2*time.Second)) {
		t.Fatalf("waiting lifecycle status=%s phase=%s attempt=%d next=%s", status, phase, attempt, nextAttempt)
	}

	observer := NewReceiptObserver(pool, func() time.Time { return current })
	if err := observer.ObserveHandled(ctx, tap.Event{
		URI: uri, DID: owner, Collection: "social.craftsky.feed.post",
		Action: "delete", ID: 61, Rev: "rev-delete",
	}); err != nil {
		t.Fatalf("persist lifecycle receipt: %v", err)
	}
	current = current.Add(2 * time.Second)
	for pass := 0; pass < 2; pass++ {
		processed, err := worker.ProcessOne(ctx)
		if err != nil || !processed {
			t.Fatalf("terminal lifecycle pass %d processed=%t err=%v", pass, processed, err)
		}
	}

	var operations, oauthSessions, audits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM account_deletion_operations WHERE id=$1`, jobID).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_sessions WHERE account_did=$1`, owner).Scan(&oauthSessions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM account_deletion_audits WHERE job_id=$1`, jobID).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if operations != 0 || oauthSessions != 0 || audits != 1 {
		t.Fatalf("terminal rows operations=%d oauth=%d audits=%d", operations, oauthSessions, audits)
	}
	if len(pds.records) != 0 || pds.listCalls < 2 {
		t.Fatalf("PDS final rescan records=%v listCalls=%d", pds.records, pds.listCalls)
	}
	events := deletionEventNames(metrics.events)
	if countDeletionEvent(events, "convergenceDelay") != 1 ||
		countDeletionEvent(events, "terminalSuccess") != 1 {
		t.Fatalf("production lifecycle telemetry=%v", events)
	}
}

type lifecycleDeletionPDS struct {
	owner     syntax.DID
	records   []auth.PDSRecord
	listCalls int
}

func (pds *lifecycleDeletionPDS) ListRecords(_ context.Context, owner syntax.DID, collection, _ string, _ int) ([]auth.PDSRecord, string, error) {
	if owner != pds.owner {
		return nil, "", errors.New("wrong PDS owner")
	}
	pds.listCalls++
	for _, record := range pds.records {
		if record.URI.Collection().String() == collection {
			return []auth.PDSRecord{record}, "", nil
		}
	}
	return nil, "", nil
}

func (pds *lifecycleDeletionPDS) DeleteRecord(_ context.Context, owner syntax.DID, collection, rkey string) error {
	if owner != pds.owner {
		return errors.New("wrong PDS owner")
	}
	for index, record := range pds.records {
		if record.URI.Collection().String() == collection && record.URI.RecordKey().String() == rkey {
			pds.records = append(pds.records[:index], pds.records[index+1:]...)
			return nil
		}
	}
	return auth.ErrRecordNotFound
}
