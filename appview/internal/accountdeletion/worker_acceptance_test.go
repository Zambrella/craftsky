package accountdeletion

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/scheduledposts"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestWorkerAcceptanceRunsCompleteDeletionLifecycleAcrossRestart(t *testing.T) {
	t.Parallel()

	pool := testdb.WithSchema(t, "")
	applyAllAccountDeletionTestMigrations(t, pool)
	ctx := context.Background()
	current := time.Date(2026, 8, 10, 18, 0, 0, 0, time.UTC)
	now := func() time.Time { return current }
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	jobID := uuid.MustParse("00000000-0000-4000-8000-000000000927")
	seedPrivateManifestExecutionFixture(t, pool, alice, bob, jobID, current)
	if _, err := pool.Exec(ctx, `DELETE FROM account_deletion_operations WHERE id=$1`, jobID); err != nil {
		t.Fatalf("replace fixture operation with acceptance intent: %v", err)
	}

	store := NewStore(pool, now)
	if err := store.CreateIntent(ctx, IntentRecord{
		JobID: jobID, Owner: alice, DeviceID: "alice-phone",
		StatusCapabilityHash:   HashSecret("status-token"),
		ConfirmationHandleHash: HashSecret("@alice.test"),
		ExpiresAt:              current.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("create deletion intent: %v", err)
	}
	if err := store.CompleteReauthentication(ctx, jobID, alice, "alice-deletion-oauth", HashSecret("fresh-proof")); err != nil {
		t.Fatalf("bind fresh reauthentication: %v", err)
	}
	accepted, err := store.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: alice, StatusCapability: "status-token",
		ReauthProof: "fresh-proof", ConfirmationHandle: "@alice.test",
	})
	if err != nil {
		t.Fatalf("accept account deletion: %v", err)
	}
	if accepted.DeletionOAuthSessionID != "alice-deletion-oauth" || accepted.Phase != PhaseQueued {
		t.Fatalf("accepted operation=%+v", accepted)
	}
	assertPrivateCleanupCount(t, pool, "craftsky_sessions", "account_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "oauth_sessions", "session_id", "alice-ordinary-oauth", 0)
	assertPrivateCleanupCount(t, pool, "oauth_sessions", "session_id", "alice-deletion-oauth", 1)

	limiter, err := instagram.NewPostgresRateLimiter(pool, bytes.Repeat([]byte{0x3e}, 32), now)
	if err != nil {
		t.Fatalf("construct Instagram limiter: %v", err)
	}
	instagramPrivate := instagram.NewPrivateDataService(pool, limiter, now)
	instagramCleanup, err := NewNamedPrivateCleanup(
		"instagramPrivate",
		func(ctx context.Context, _ uuid.UUID, owner syntax.DID) error {
			return PurgeInstagramForAccountDeletion(ctx, instagramPrivate, owner)
		},
	)
	if err != nil {
		t.Fatalf("construct Instagram cleanup: %v", err)
	}
	scheduledStore := scheduledposts.NewStore(pool)
	scheduledDeletion := scheduledposts.NewAccountDeletion(pool, now)
	newCleaner := func(store *Store) *PrivateCleaner {
		cleaner, err := NewPrivateCleaner(store, []PrivateCleanupComponent{
			NewDatabasePrivateCleanup(pool), instagramCleanup, scheduledDeletion,
		})
		if err != nil {
			t.Fatalf("construct private cleaner: %v", err)
		}
		return cleaner
	}

	craftskyRecords := []auth.PDSRecord{
		{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/a")},
		{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/b")},
		{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/l")},
		{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.repost/r")},
		{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.actor.profile/self")},
	}
	pds := &acceptanceDeletionPDS{
		owner: alice, records: append([]auth.PDSRecord(nil), craftskyRecords...),
		preserved: []syntax.ATURI{
			"at://did:plc:alice/app.bsky.feed.post/keep",
			"at://did:plc:alice/com.example.private/keep",
		},
	}
	newProcessor := func(store *Store, cleaner *PrivateCleaner) *LifecycleProcessor {
		processor, err := NewLifecycleProcessor(LifecycleProcessorOptions{
			Store: store, Cleaner: cleaner, Convergence: NewConvergenceVerifier(pool),
			NewPDSClient: func(_ context.Context, owner syntax.DID, sessionID string) (auth.DeletionPDSClient, error) {
				if owner != alice || sessionID != "alice-deletion-oauth" {
					return nil, errors.New("worker requested an unbound OAuth scope")
				}
				pds.workerOAuthCalls++
				return pds, nil
			},
			PollInterval: time.Second, BatchSize: 1, Now: now,
		})
		if err != nil {
			t.Fatalf("construct lifecycle processor: %v", err)
		}
		return processor
	}
	newWorker := func(store *Store, cleaner *PrivateCleaner, workerID string) *Worker {
		worker, err := NewWorker(WorkerOptions{
			Store: store, Processor: newProcessor(store, cleaner), WorkerID: workerID,
			Now: now, LeaseDuration: time.Minute, RetryPolicy: DefaultRetryPolicy(),
		})
		if err != nil {
			t.Fatalf("construct deletion worker: %v", err)
		}
		return worker
	}

	worker := newWorker(store, newCleaner(store), "acceptance-before-restart")
	processAcceptancePass(t, worker, ctx, "queued phase")
	processAcceptancePass(t, worker, ctx, "private cleanup phase")

	// Simulate a process restart after durable private checkpoints. The new
	// store/cleaner/worker must continue the same operation without reacceptance.
	restartedStore := NewStore(pool, now)
	restartedCleaner := newCleaner(restartedStore)
	worker = newWorker(restartedStore, restartedCleaner, "acceptance-after-restart")
	processAcceptancePass(t, worker, ctx, "PDS deletion phase")
	processAcceptancePass(t, worker, ctx, "pre-receipt convergence poll")

	if len(pds.records) != 0 || pds.listCalls < len(CraftskyRecordCollections())*2 {
		t.Fatalf("PDS CraftSky records=%v listCalls=%d", pds.records, pds.listCalls)
	}
	if len(pds.preserved) != 2 {
		t.Fatalf("non-CraftSky PDS controls changed: %v", pds.preserved)
	}

	objects := &manifestMemoryObjectStore{objects: map[string][]byte{
		"scheduled-media/alice": []byte("alice-private"),
		"scheduled-media/bob":   []byte("bob-private"),
	}}
	objectCleaner, err := scheduledposts.NewCleanupProcessor(scheduledposts.CleanupProcessorOptions{
		Store: scheduledStore, Objects: objects, Now: now,
	})
	if err != nil {
		t.Fatalf("construct scheduled object cleaner: %v", err)
	}
	if processed, err := objectCleaner.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("scheduled object cleanup processed=%d err=%v", processed, err)
	}

	observer := NewReceiptObserver(pool, now)
	for index := len(craftskyRecords) - 1; index >= 0; index-- {
		record := craftskyRecords[index]
		deleteAcceptanceProjection(t, pool, alice, record.URI)
		event := tap.Event{
			URI: record.URI, DID: alice, Collection: record.URI.Collection(), Action: "delete",
			ID: uint64(100 + index), Rev: "acceptance-rev-" + record.URI.RecordKey().String(),
		}
		if err := observer.ObserveHandled(ctx, event); err != nil {
			t.Fatalf("persist reordered receipt %s: %v", record.URI, err)
		}
		if index == len(craftskyRecords)-1 {
			if err := observer.ObserveHandled(ctx, event); err != nil {
				t.Fatalf("persist duplicate receipt %s: %v", record.URI, err)
			}
		}
	}

	current = current.Add(time.Second)
	processAcceptancePass(t, worker, ctx, "converged waiting phase")
	processAcceptancePass(t, worker, ctx, "terminal full-rescan phase")

	assertPrivateCleanupCount(t, pool, "account_deletion_operations", "id", jobID, 0)
	assertPrivateCleanupCount(t, pool, "oauth_sessions", "account_did", alice, 0)
	assertPrivateCleanupCount(t, pool, "account_deletion_audits", "job_id", jobID, 1)
	for _, operational := range []string{
		"account_deletion_status_credentials", "account_deletion_recovery_credentials",
		"account_deletion_expected_records", "account_deletion_index_receipts",
		"account_deletion_cleanup_steps", "account_deletion_cleanup_artifacts",
	} {
		assertPrivateCleanupCount(t, pool, operational, "job_id", jobID, 0)
	}
	assertPrivateCleanupCount(t, pool, "craftsky_profiles", "did", bob, 1)
	assertPrivateCleanupCount(t, pool, "atproto_identity_cache", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "bluesky_profiles", "did", alice, 1)
	assertPrivateCleanupCount(t, pool, "instagram_account_links", "owner_did", bob, 1)
	assertPrivateCleanupCount(t, pool, "scheduled_post_media", "owner_did", bob, 1)
	if pds.workerOAuthCalls < 2 {
		t.Fatalf("worker bound-OAuth calls=%d, want deletion plus final rescan", pds.workerOAuthCalls)
	}
}

func processAcceptancePass(t *testing.T, worker *Worker, ctx context.Context, label string) {
	t.Helper()
	processed, err := worker.ProcessOne(ctx)
	if err != nil || !processed {
		t.Fatalf("%s processed=%t err=%v", label, processed, err)
	}
}

func deleteAcceptanceProjection(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, uri syntax.ATURI) {
	t.Helper()
	var err error
	switch uri.Collection().String() {
	case "social.craftsky.feed.post":
		_, err = pool.Exec(context.Background(), `DELETE FROM craftsky_posts WHERE uri=$1`, uri)
	case "social.craftsky.feed.like":
		_, err = pool.Exec(context.Background(), `DELETE FROM craftsky_likes WHERE uri=$1`, uri)
	case "social.craftsky.feed.repost":
		_, err = pool.Exec(context.Background(), `DELETE FROM craftsky_reposts WHERE uri=$1`, uri)
	case "social.craftsky.actor.profile":
		_, err = pool.Exec(context.Background(), `DELETE FROM craftsky_profiles WHERE did=$1`, owner)
	}
	if err != nil {
		t.Fatalf("delete indexed projection %s: %v", uri, err)
	}
}

type acceptanceDeletionPDS struct {
	owner            syntax.DID
	records          []auth.PDSRecord
	preserved        []syntax.ATURI
	listCalls        int
	workerOAuthCalls int
}

func (pds *acceptanceDeletionPDS) ListRecords(
	_ context.Context,
	owner syntax.DID,
	collection string,
	cursor string,
	limit int,
) ([]auth.PDSRecord, string, error) {
	if owner != pds.owner {
		return nil, "", errors.New("wrong PDS owner")
	}
	pds.listCalls++
	matches := make([]auth.PDSRecord, 0)
	for _, record := range pds.records {
		if record.URI.Collection().String() == collection && record.URI.RecordKey().String() > cursor {
			matches = append(matches, record)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].URI.String() < matches[j].URI.String() })
	if len(matches) == 0 {
		return nil, "", nil
	}
	if limit > len(matches) {
		limit = len(matches)
	}
	page := append([]auth.PDSRecord(nil), matches[:limit]...)
	nextCursor := ""
	if len(matches) > limit {
		nextCursor = page[len(page)-1].URI.RecordKey().String()
	}
	return page, nextCursor, nil
}

func (pds *acceptanceDeletionPDS) DeleteRecord(
	_ context.Context,
	owner syntax.DID,
	collection string,
	rkey string,
) error {
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
