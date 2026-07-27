package instagram_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

const backgroundSessionSchema = `
	CREATE TABLE oauth_sessions (
		account_did TEXT NOT NULL,
		session_id TEXT NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		PRIMARY KEY (account_did, session_id)
	);
	CREATE TABLE craftsky_sessions (
		token_hash BYTEA NOT NULL PRIMARY KEY,
		account_did TEXT NOT NULL,
		oauth_session_id TEXT NOT NULL,
		last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		revoked_at TIMESTAMPTZ,
		FOREIGN KEY (account_did, oauth_session_id)
			REFERENCES oauth_sessions (account_did, session_id)
			ON DELETE CASCADE
	);
`

func TestAutomaticFollowWorkerOptionsEnforceProviderAttemptMaximum(t *testing.T) {
	options := instagram.AutomaticFollowWorkerOptions{
		Store:               &automaticFollowStoreStub{},
		Policy:              allowAutomaticFollowPolicy{},
		Sessions:            automaticFollowSessionStub{},
		Writer:              &automaticFollowWriterStub{},
		MaxProviderAttempts: instagram.AutomaticFollowProviderAttempts,
	}
	if _, err := instagram.NewAutomaticFollowWorker(options); err != nil {
		t.Fatalf("exact provider-attempt maximum rejected: %v", err)
	}
	options.MaxProviderAttempts++
	if _, err := instagram.NewAutomaticFollowWorker(options); err == nil {
		t.Fatal("provider attempts above maximum accepted")
	}
}

func TestAutomaticFollowWorker_InactivatesDepartedOwnerBeforePDSWrite(t *testing.T) {
	now := time.Date(2026, 7, 27, 15, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:departed-automatic-owner")
	target := syntax.DID("did:plc:current-automatic-target")
	store := &automaticFollowStoreStub{
		operations: []instagram.AutomaticFollowOperation{{
			ID:               uuid.New(),
			OwnerDID:         owner,
			TargetDID:        target,
			ImportedUsername: "current.target",
			Rkey:             "3kdepartedautomatic",
			CreatedAt:        now,
		}},
	}
	membership := &automaticFollowMembershipStub{
		current: map[syntax.DID]bool{owner: false, target: true},
	}
	inactivator := &automaticFollowInactivatorStub{}
	writer := &automaticFollowWriterStub{}
	worker, err := instagram.NewAutomaticFollowWorker(
		instagram.AutomaticFollowWorkerOptions{
			Store:                 store,
			Policy:                allowAutomaticFollowPolicy{},
			Sessions:              automaticFollowSessionStub{},
			Writer:                writer,
			Membership:            membership,
			MembershipInactivator: inactivator,
			Now:                   func() time.Time { return now },
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if processed, err := worker.ProcessBatch(context.Background(), 1); err != nil || processed != 1 {
		t.Fatalf("ProcessBatch processed=%d err=%v", processed, err)
	}
	if len(inactivator.owners) != 1 || inactivator.owners[0] != owner {
		t.Fatalf("inactivated owners=%v, want %s", inactivator.owners, owner)
	}
	if len(writer.sessions) != 0 || store.followed != 0 || store.alreadyFollowing != 0 {
		t.Fatalf(
			"writer sessions=%v followed=%d alreadyFollowing=%d",
			writer.sessions,
			store.followed,
			store.alreadyFollowing,
		)
	}
}

func TestAutomaticFollowWorker_FinalSafetyChangeInvalidatesBeforePDSWrite(t *testing.T) {
	for _, test := range []struct {
		name   string
		reason instagram.EligibilityReason
	}{
		{name: "importer_blocks_target", reason: instagram.EligibilityRelationshipSafety},
		{name: "target_blocks_importer", reason: instagram.EligibilityRelationshipSafety},
		{name: "importer_mutes_target", reason: instagram.EligibilityRelationshipSafety},
		{name: "target_hidden", reason: instagram.EligibilityModeration},
		{name: "target_taken_down", reason: instagram.EligibilityModeration},
	} {
		t.Run(test.name, func(t *testing.T) {
			now := time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC)
			store := &automaticFollowStoreStub{
				operations: []instagram.AutomaticFollowOperation{{
					ID:               uuid.New(),
					OwnerDID:         syntax.DID("did:plc:final-safety-owner"),
					TargetDID:        syntax.DID("did:plc:final-safety-target"),
					ImportedUsername: "final.safety.target",
					Rkey:             "3kfinalsafety",
					CreatedAt:        now,
				}},
			}
			writer := &automaticFollowWriterStub{}
			worker, err := instagram.NewAutomaticFollowWorker(
				instagram.AutomaticFollowWorkerOptions{
					Store:    store,
					Policy:   fixedAutomaticFollowPolicy{reason: test.reason},
					Sessions: automaticFollowSessionStub{},
					Writer:   writer,
					Now:      func() time.Time { return now },
				},
			)
			if err != nil {
				t.Fatal(err)
			}

			if processed, err := worker.ProcessBatch(context.Background(), 1); err != nil || processed != 1 {
				t.Fatalf("ProcessBatch processed=%d err=%v", processed, err)
			}
			if store.invalidated != 1 || store.lastInvalidationCode != string(test.reason) {
				t.Fatalf(
					"invalidated=%d code=%q, want 1/%q",
					store.invalidated,
					store.lastInvalidationCode,
					test.reason,
				)
			}
			if len(writer.sessions) != 0 || store.followed != 0 ||
				store.alreadyFollowing != 0 || store.retried != 0 {
				t.Fatalf(
					"writer=%v followed=%d already=%d retried=%d",
					writer.sessions,
					store.followed,
					store.alreadyFollowing,
					store.retried,
				)
			}
		})
	}
}

func TestAutomaticFollowWorker_RotatesOnlyAmongOwnerSessions(t *testing.T) {
	pool := testdb.WithSchema(t, backgroundSessionSchema)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	now := time.Now().UTC()

	seedBackgroundSession(t, pool, alice, "alice-valid", now.Add(-time.Minute))
	seedBackgroundSession(t, pool, alice, "alice-invalid", now)
	seedBackgroundSession(t, pool, bob, "bob-newer", now.Add(time.Hour))

	store := &automaticFollowStoreStub{
		operations: []instagram.AutomaticFollowOperation{{
			ID:               uuid.New(),
			OwnerDID:         alice,
			TargetDID:        syntax.DID("did:plc:target"),
			ImportedUsername: "target",
			Rkey:             syntax.RecordKey("3kautomaticfollow"),
			CreatedAt:        now,
		}},
	}
	writer := &automaticFollowWriterStub{
		errorsBySession: map[string]error{
			"alice-invalid": auth.ErrPDSSessionExpired,
		},
	}
	worker, err := instagram.NewAutomaticFollowWorker(instagram.AutomaticFollowWorkerOptions{
		Store:    store,
		Policy:   allowAutomaticFollowPolicy{},
		Sessions: auth.NewBackgroundSessionSelector(pool),
		Writer:   writer,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAutomaticFollowWorker: %v", err)
	}

	processed, err := worker.ProcessBatch(ctx, 1)
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if len(writer.sessions) != 2 ||
		writer.sessions[0] != "alice-invalid" ||
		writer.sessions[1] != "alice-valid" {
		t.Fatalf("writer sessions = %v, want Alice invalid then valid", writer.sessions)
	}
	for _, sessionID := range writer.sessions {
		if sessionID == "bob-newer" {
			t.Fatalf("worker used Bob session: %v", writer.sessions)
		}
	}
	if store.followed != 1 || store.retried != 0 {
		t.Fatalf("followed=%d retried=%d, want 1/0", store.followed, store.retried)
	}
	var invalidAlice, validAlice, validBob int
	if err := pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE account_did=$1 AND session_id='alice-invalid'),
			count(*) FILTER (WHERE account_did=$1 AND session_id='alice-valid'),
			count(*) FILTER (WHERE account_did=$2 AND session_id='bob-newer')
		FROM oauth_sessions
	`, alice, bob).Scan(&invalidAlice, &validAlice, &validBob); err != nil {
		t.Fatalf("inspect sessions: %v", err)
	}
	if invalidAlice != 0 || validAlice != 1 || validBob != 1 {
		t.Fatalf(
			"session counts invalidAlice=%d validAlice=%d validBob=%d",
			invalidAlice,
			validAlice,
			validBob,
		)
	}
}

func TestAutomaticFollowWorker_NoOwnerSessionRetriesWithoutPDSWrite(t *testing.T) {
	pool := testdb.WithSchema(t, backgroundSessionSchema)
	now := time.Now().UTC()
	store := &automaticFollowStoreStub{
		operations: []instagram.AutomaticFollowOperation{{
			ID:               uuid.New(),
			OwnerDID:         syntax.DID("did:plc:alice"),
			TargetDID:        syntax.DID("did:plc:target"),
			ImportedUsername: "target",
			Rkey:             syntax.RecordKey("3kautomaticfollow"),
			CreatedAt:        now,
		}},
	}
	writer := &automaticFollowWriterStub{}
	worker, err := instagram.NewAutomaticFollowWorker(instagram.AutomaticFollowWorkerOptions{
		Store:    store,
		Policy:   allowAutomaticFollowPolicy{},
		Sessions: auth.NewBackgroundSessionSelector(pool),
		Writer:   writer,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewAutomaticFollowWorker: %v", err)
	}

	processed, err := worker.ProcessBatch(context.Background(), 1)
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 1 || len(writer.sessions) != 0 {
		t.Fatalf("processed=%d writer sessions=%v, want 1 and no write", processed, writer.sessions)
	}
	if store.retried != 1 || store.lastRetryCode != "ownerSessionUnavailable" {
		t.Fatalf("retried=%d code=%q", store.retried, store.lastRetryCode)
	}
}

func TestAutomaticFollowWorker_TransientPDSFailureReleasesForRetry(t *testing.T) {
	now := time.Date(2026, 7, 27, 16, 15, 0, 0, time.UTC)
	store := &automaticFollowStoreStub{
		operations: []instagram.AutomaticFollowOperation{{
			ID:               uuid.New(),
			OwnerDID:         syntax.DID("did:plc:transient-owner"),
			TargetDID:        syntax.DID("did:plc:transient-target"),
			ImportedUsername: "transient.target",
			Rkey:             syntax.RecordKey("3ktransientfollow"),
			CreatedAt:        now,
		}},
	}
	writer := &automaticFollowWriterStub{
		errorsBySession: map[string]error{
			"owner-session": errors.New("synthetic transient PDS failure"),
		},
	}
	worker, err := instagram.NewAutomaticFollowWorker(
		instagram.AutomaticFollowWorkerOptions{
			Store:    store,
			Policy:   allowAutomaticFollowPolicy{},
			Sessions: automaticFollowSessionStub{},
			Writer:   writer,
			Now:      func() time.Time { return now },
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if processed, err := worker.ProcessBatch(context.Background(), 1); err != nil || processed != 1 {
		t.Fatalf("ProcessBatch processed=%d err=%v", processed, err)
	}
	if len(writer.sessions) != 1 || writer.sessions[0] != "owner-session" {
		t.Fatalf("writer sessions=%v", writer.sessions)
	}
	if store.retried != 1 || store.lastRetryCode != "followWriteUnavailable" {
		t.Fatalf("retried=%d code=%q", store.retried, store.lastRetryCode)
	}
	if store.followed != 0 || store.alreadyFollowing != 0 ||
		store.invalidated != 0 {
		t.Fatalf(
			"followed=%d already=%d invalidated=%d",
			store.followed,
			store.alreadyFollowing,
			store.invalidated,
		)
	}
}

func TestAutomaticFollowWorker_RecoversNotificationAfterPDSWriteCrash(t *testing.T) {
	pool := testdb.WithSchema(t, backgroundSessionSchema)
	ctx := context.Background()
	now := time.Now().UTC()
	owner := syntax.DID("did:plc:automatic-recovery-owner")
	seedBackgroundSession(t, pool, owner, "owner-session", now)
	store := &automaticFollowStoreStub{
		operations: []instagram.AutomaticFollowOperation{{
			ID:               uuid.New(),
			OwnerDID:         owner,
			TargetDID:        syntax.DID("did:plc:automatic-recovery-target"),
			ImportedUsername: "automatic.recovery.target",
			Rkey:             syntax.RecordKey("3kautomaticrecovery"),
			AttemptCount:     2,
			CreatedAt:        now.Add(-time.Minute),
		}},
	}
	writer := &automaticFollowWriterStub{deterministicFollowExists: true}
	worker, err := instagram.NewAutomaticFollowWorker(
		instagram.AutomaticFollowWorkerOptions{
			Store:    store,
			Policy:   alreadyFollowingPolicy{},
			Sessions: auth.NewBackgroundSessionSelector(pool),
			Writer:   writer,
			Now:      func() time.Time { return now },
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if processed, err := worker.ProcessBatch(ctx, 1); err != nil || processed != 1 {
		t.Fatalf("ProcessBatch processed=%d err=%v", processed, err)
	}
	if store.followed != 1 || store.alreadyFollowing != 0 {
		t.Fatalf(
			"completion followed=%d alreadyFollowing=%d, want crash recovery",
			store.followed,
			store.alreadyFollowing,
		)
	}
}

func TestAutomaticFollowStore_LeasesAndCompletesOneDeterministicOperation(t *testing.T) {
	pool := newAutomaticFollowTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:synthetic-leased-owner")
	target := syntax.DID("did:plc:synthetic-leased-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001101")
	operationID := uuid.MustParse("00000000-0000-0000-0000-000000001102")
	seedAutomaticFollowEvidence(t, pool, owner, target, importID, now)

	store := instagram.NewAutomaticFollowStore(pool)
	reconciled, err := store.ReconcileCandidate(ctx, instagram.ReconcileAutomaticFollowParams{
		ID: operationID, ImporterDID: owner, TargetDID: target,
		ImportID: importID, Username: "synthetic.leased",
		Rkey: "3kleasedautomatic", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Operation.Rkey != syntax.RecordKey("3kleasedautomatic") {
		t.Fatalf("operation rkey=%q", reconciled.Operation.Rkey)
	}

	first, err := store.ClaimBatch(ctx, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := instagram.NewAutomaticFollowStore(pool).ClaimBatch(ctx, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("first claims=%d second claims=%d, want 1/0", len(first), len(second))
	}
	if first[0].LeaseToken == uuid.Nil || first[0].ID != operationID {
		t.Fatalf("claim=%+v", first[0])
	}

	stale := first[0]
	stale.LeaseToken = uuid.New()
	if err := store.CompleteFollowed(ctx, stale, now.Add(time.Minute)); !errors.Is(err, instagram.ErrAutomaticFollowLeaseLost) {
		t.Fatalf("stale completion error=%v, want ErrAutomaticFollowLeaseLost", err)
	}
	if err := store.CompleteFollowed(ctx, first[0], now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	var operationState, ledgerState string
	if err := pool.QueryRow(ctx, `
		SELECT operation.status, ledger.state
		FROM pds_follow_operations operation
		JOIN instagram_follow_suggestions ledger
		  ON ledger.id=operation.suggestion_id
		WHERE operation.id=$1
	`, operationID).Scan(&operationState, &ledgerState); err != nil {
		t.Fatal(err)
	}
	if operationState != "followed" || ledgerState != "followed" {
		t.Fatalf("operation=%q ledger=%q", operationState, ledgerState)
	}
	replayed, err := store.ReconcileCandidate(ctx, instagram.ReconcileAutomaticFollowParams{
		ID: uuid.New(), ImporterDID: owner, TargetDID: target,
		ImportID: importID, Username: "synthetic.leased",
		Rkey: "3kshouldnotreplace", Now: now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Suppressed || replayed.Operation.ID != operationID ||
		replayed.Operation.Rkey != syntax.RecordKey("3kleasedautomatic") {
		t.Fatalf("replayed operation=%+v", replayed)
	}
}

func TestAutomaticFollowStore_ConcurrentWorkersClaimOperationOnce(t *testing.T) {
	pool := newAutomaticFollowTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 16, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:concurrent-claim-owner")
	target := syntax.DID("did:plc:concurrent-claim-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001131")
	operationID := uuid.MustParse("00000000-0000-0000-0000-000000001132")
	seedAutomaticFollowEvidence(t, pool, owner, target, importID, now)
	if _, err := instagram.NewAutomaticFollowStore(pool).ReconcileCandidate(
		ctx,
		instagram.ReconcileAutomaticFollowParams{
			ID: operationID, ImporterDID: owner, TargetDID: target,
			ImportID: importID, Username: "synthetic.leased",
			Rkey: "3kconcurrentclaim", Now: now,
		},
	); err != nil {
		t.Fatal(err)
	}

	const workerCount = 5
	start := make(chan struct{})
	type claimResult struct {
		operations []instagram.AutomaticFollowOperation
		err        error
	}
	results := make(chan claimResult, workerCount)
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			<-start
			operations, err := instagram.NewAutomaticFollowStore(pool).
				ClaimBatch(ctx, 1, now)
			results <- claimResult{operations: operations, err: err}
		}()
	}
	close(start)
	workers.Wait()
	close(results)

	claims := 0
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		claims += len(result.operations)
		for _, operation := range result.operations {
			if operation.ID != operationID {
				t.Fatalf("claimed operation=%s, want %s", operation.ID, operationID)
			}
		}
	}
	if claims != 1 {
		t.Fatalf("concurrent claims=%d, want 1", claims)
	}
}

func TestAutomaticFollowStore_RetryUsesClaimAttemptBackoff(t *testing.T) {
	pool := newAutomaticFollowTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 15, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:synthetic-retry-owner")
	target := syntax.DID("did:plc:synthetic-retry-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001121")
	operationID := uuid.MustParse("00000000-0000-0000-0000-000000001122")
	seedAutomaticFollowEvidence(t, pool, owner, target, importID, now)

	store := instagram.NewAutomaticFollowStore(pool)
	if _, err := store.ReconcileCandidate(ctx, instagram.ReconcileAutomaticFollowParams{
		ID: operationID, ImporterDID: owner, TargetDID: target,
		ImportID: importID, Username: "synthetic.leased",
		Rkey: "3kretryautomatic", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE pds_follow_operations SET attempt_count=2 WHERE id=$1
	`, operationID); err != nil {
		t.Fatal(err)
	}
	claimed, err := store.ClaimBatch(ctx, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].AttemptCount != 3 {
		t.Fatalf("claimed=%+v, want attempt 3", claimed)
	}
	if err := store.Retry(ctx, claimed[0], "followWriteUnavailable", now); err != nil {
		t.Fatal(err)
	}
	var next time.Time
	if err := pool.QueryRow(ctx, `
		SELECT next_attempt_at FROM pds_follow_operations WHERE id=$1
	`, operationID).Scan(&next); err != nil {
		t.Fatal(err)
	}
	if want := now.Add(4 * time.Second); !next.Equal(want) {
		t.Fatalf("next attempt=%s, want %s", next, want)
	}
}

func TestAutomaticFollowWorker_ProcessesRealLeasedStore(t *testing.T) {
	pool := newAutomaticFollowTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 12, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:synthetic-worker-owner")
	target := syntax.DID("did:plc:synthetic-worker-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001111")
	operationID := uuid.MustParse("00000000-0000-0000-0000-000000001112")
	seedAutomaticFollowEvidence(t, pool, owner, target, importID, now)
	store := instagram.NewAutomaticFollowStore(pool, notifications.NewService())
	if _, err := store.ReconcileCandidate(ctx, instagram.ReconcileAutomaticFollowParams{
		ID: operationID, ImporterDID: owner, TargetDID: target,
		ImportID: importID, Username: "synthetic.leased",
		Rkey: "3kworkerautomatic", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	writer := &automaticFollowWriterStub{}
	worker, err := instagram.NewAutomaticFollowWorker(instagram.AutomaticFollowWorkerOptions{
		Store:    store,
		Policy:   allowAutomaticFollowPolicy{},
		Sessions: automaticFollowSessionStub{},
		Writer:   writer,
		Now:      func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := worker.ProcessBatch(ctx, 1); err != nil || processed != 1 {
		t.Fatalf("processed=%d error=%v", processed, err)
	}
	if len(writer.sessions) != 1 || writer.sessions[0] != "owner-session" {
		t.Fatalf("writer sessions=%v", writer.sessions)
	}
	var status, notificationActor string
	if err := pool.QueryRow(ctx, `
		SELECT status FROM pds_follow_operations WHERE id=$1
	`, operationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT actor_did
		FROM notification_events
		WHERE recipient_did=$1
		  AND category='instagramMatch'
		  AND subject_key=$2
	`, owner, operationID.String()).Scan(&notificationActor); err != nil {
		t.Fatal(err)
	}
	if status != "followed" {
		t.Fatalf("operation status=%q", status)
	}
	if notificationActor != target.String() {
		t.Fatalf("notification actor=%q", notificationActor)
	}
}

func TestAutomaticFollowWorker_AlreadyFollowingCompletesWithoutWriteOrNotification(t *testing.T) {
	pool := newAutomaticFollowTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 16, 40, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:already-following-owner")
	target := syntax.DID("did:plc:already-following-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001151")
	operationID := uuid.MustParse("00000000-0000-0000-0000-000000001152")
	seedAutomaticFollowEvidence(t, pool, owner, target, importID, now)
	store := instagram.NewAutomaticFollowStore(pool, notifications.NewService())
	if _, err := store.ReconcileCandidate(ctx, instagram.ReconcileAutomaticFollowParams{
		ID: operationID, ImporterDID: owner, TargetDID: target,
		ImportID: importID, Username: "synthetic.leased",
		Rkey: "3kalreadyfollowing", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	writer := &automaticFollowWriterStub{}
	worker, err := instagram.NewAutomaticFollowWorker(
		instagram.AutomaticFollowWorkerOptions{
			Store:    store,
			Policy:   alreadyFollowingPolicy{},
			Sessions: automaticFollowSessionStub{},
			Writer:   writer,
			Now:      func() time.Time { return now },
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := worker.ProcessBatch(ctx, 1); err != nil || processed != 1 {
		t.Fatalf("processed=%d err=%v", processed, err)
	}

	var (
		status            string
		notificationCount int
	)
	if err := pool.QueryRow(ctx, `
		SELECT status FROM pds_follow_operations WHERE id=$1
	`, operationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM notification_events
		WHERE recipient_did=$1
		  AND category='instagramMatch'
		  AND subject_key=$2
	`, owner, operationID.String()).Scan(&notificationCount); err != nil {
		t.Fatal(err)
	}
	if status != "alreadyFollowing" || notificationCount != 0 ||
		len(writer.sessions) != 0 {
		t.Fatalf(
			"status=%q notifications=%d writer=%v",
			status,
			notificationCount,
			writer.sessions,
		)
	}
}

func TestAutomaticFollowWorker_RecoversOneNotificationAfterExpiredLeaseAndPDSWrite(t *testing.T) {
	pool := newAutomaticFollowTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 16, 45, 0, 0, time.UTC)
	recoveryNow := now.Add(2 * instagram.AutomaticFollowLeaseDuration)
	owner := syntax.DID("did:plc:real-recovery-owner")
	target := syntax.DID("did:plc:real-recovery-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000001141")
	operationID := uuid.MustParse("00000000-0000-0000-0000-000000001142")
	seedAutomaticFollowEvidence(t, pool, owner, target, importID, now)
	store := instagram.NewAutomaticFollowStore(pool, notifications.NewService())
	if _, err := store.ReconcileCandidate(ctx, instagram.ReconcileAutomaticFollowParams{
		ID: operationID, ImporterDID: owner, TargetDID: target,
		ImportID: importID, Username: "synthetic.leased",
		Rkey: "3krealrecovery", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	initialClaim, err := store.ClaimBatch(ctx, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(initialClaim) != 1 || initialClaim[0].AttemptCount != 1 {
		t.Fatalf("initial claim=%+v", initialClaim)
	}

	writer := &automaticFollowWriterStub{deterministicFollowExists: true}
	worker, err := instagram.NewAutomaticFollowWorker(
		instagram.AutomaticFollowWorkerOptions{
			Store:    store,
			Policy:   alreadyFollowingPolicy{},
			Sessions: automaticFollowSessionStub{},
			Writer:   writer,
			Now:      func() time.Time { return recoveryNow },
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := worker.ProcessBatch(ctx, 1); err != nil || processed != 1 {
		t.Fatalf("recovery processed=%d err=%v", processed, err)
	}
	if processed, err := worker.ProcessBatch(ctx, 1); err != nil || processed != 0 {
		t.Fatalf("replay processed=%d err=%v", processed, err)
	}

	var (
		status            string
		attemptCount      int
		notificationCount int
	)
	if err := pool.QueryRow(ctx, `
		SELECT status, attempt_count
		FROM pds_follow_operations
		WHERE id=$1
	`, operationID).Scan(&status, &attemptCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM notification_events
		WHERE recipient_did=$1
		  AND actor_did=$2
		  AND category='instagramMatch'
		  AND subject_key=$3
	`, owner, target, operationID.String()).Scan(&notificationCount); err != nil {
		t.Fatal(err)
	}
	if status != "followed" || attemptCount != 2 || notificationCount != 1 {
		t.Fatalf(
			"status=%q attempts=%d notifications=%d, want followed/2/1",
			status,
			attemptCount,
			notificationCount,
		)
	}
	if len(writer.sessions) != 1 || writer.sessions[0] != "owner-session" {
		t.Fatalf("recovery writer sessions=%v", writer.sessions)
	}
}

func seedBackgroundSession(
	t *testing.T,
	pool *pgxpool.Pool,
	did syntax.DID,
	sessionID string,
	lastSeen time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions (
			account_did, session_id, data, created_at, updated_at
		) VALUES ($1, $2, '{}', $3, $3)
	`, did, sessionID, lastSeen); err != nil {
		t.Fatalf("seed background OAuth session %s/%s: %v", did, sessionID, err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions (
			token_hash, account_did, oauth_session_id, last_seen_at
		) VALUES (convert_to($1 || '/' || $2, 'UTF8'), $1, $2, $3)
	`, did, sessionID, lastSeen); err != nil {
		t.Fatalf("seed background CraftSky session %s/%s: %v", did, sessionID, err)
	}
}

func newAutomaticFollowTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, "")
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000025_instagram_migration.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
		"../../migrations/000030_instagram_automatic_follows.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	return pool
}

func seedAutomaticFollowEvidence(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	target syntax.DID,
	importID uuid.UUID,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version, igsid_digest,
			username, username_normalized, discoverable, conflict_pending,
			verified_at, created_at, updated_at
		) VALUES (
			$1,$2,'active','synthetic-leased-igsid',1,
			decode(repeat('11',32),'hex'),
			'synthetic.leased','synthetic.leased',true,false,$3,$3,$3
		)
	`, uuid.New(), target, now); err != nil {
		t.Fatalf("seed automatic-follow link: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, following_count,
			created_at, updated_at
		) VALUES ($1,$2,'active','manual',1,$3,$3)
	`, importID, owner, now); err != nil {
		t.Fatalf("seed automatic-follow import: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, matched, created_at
		) VALUES ($1,'synthetic.leased',false,$2)
	`, importID, now); err != nil {
		t.Fatalf("seed automatic-follow handle: %v", err)
	}
}

type automaticFollowStoreStub struct {
	operations           []instagram.AutomaticFollowOperation
	followed             int
	alreadyFollowing     int
	invalidated          int
	lastInvalidationCode string
	retried              int
	lastRetryCode        string
}

func (s *automaticFollowStoreStub) ClaimBatch(
	context.Context,
	int,
	time.Time,
) ([]instagram.AutomaticFollowOperation, error) {
	return s.operations, nil
}

func (s *automaticFollowStoreStub) CompleteFollowed(
	context.Context,
	instagram.AutomaticFollowOperation,
	time.Time,
) error {
	s.followed++
	return nil
}

func (s *automaticFollowStoreStub) CompleteAlreadyFollowing(
	context.Context,
	instagram.AutomaticFollowOperation,
	time.Time,
) error {
	s.alreadyFollowing++
	return nil
}

func (s *automaticFollowStoreStub) Invalidate(
	_ context.Context,
	_ instagram.AutomaticFollowOperation,
	code string,
	_ time.Time,
) error {
	s.invalidated++
	s.lastInvalidationCode = code
	return nil
}

func (s *automaticFollowStoreStub) Retry(
	_ context.Context,
	_ instagram.AutomaticFollowOperation,
	code string,
	_ time.Time,
) error {
	s.retried++
	s.lastRetryCode = code
	return nil
}

type automaticFollowWriterStub struct {
	sessions                  []string
	errorsBySession           map[string]error
	deterministicFollowExists bool
}

func (w *automaticFollowWriterStub) Write(
	_ context.Context,
	_, _ syntax.DID,
	sessionID string,
	_ *syntax.RecordKey,
	_ time.Time,
) error {
	w.sessions = append(w.sessions, sessionID)
	return w.errorsBySession[sessionID]
}

func (w *automaticFollowWriterStub) HasDeterministicFollow(
	_ context.Context,
	_, _ syntax.DID,
	sessionID string,
	_ syntax.RecordKey,
) (bool, error) {
	w.sessions = append(w.sessions, sessionID)
	return w.deterministicFollowExists, w.errorsBySession[sessionID]
}

type automaticFollowSessionStub struct{}

func (automaticFollowSessionStub) Select(context.Context, syntax.DID) (string, error) {
	return "owner-session", nil
}

func (automaticFollowSessionStub) Invalidate(context.Context, syntax.DID, string) error {
	return nil
}

type automaticFollowMembershipStub struct {
	current map[syntax.DID]bool
	err     error
}

func (s *automaticFollowMembershipStub) IsCurrentMember(
	_ context.Context,
	did syntax.DID,
) (bool, error) {
	return s.current[did], s.err
}

type automaticFollowInactivatorStub struct {
	owners []syntax.DID
	err    error
}

func (s *automaticFollowInactivatorStub) InactivateMembership(
	_ context.Context,
	did syntax.DID,
) error {
	s.owners = append(s.owners, did)
	return s.err
}

type allowAutomaticFollowPolicy struct{}

func (allowAutomaticFollowPolicy) Evaluate(
	context.Context,
	instagram.EligibilityStage,
	instagram.SuggestionEligibilityRequest,
) (instagram.EligibilityDecision, error) {
	return instagram.EligibilityDecision{
		Eligible: true,
		Reason:   instagram.EligibilityAllowed,
	}, nil
}

type alreadyFollowingPolicy struct{}

func (alreadyFollowingPolicy) Evaluate(
	context.Context,
	instagram.EligibilityStage,
	instagram.SuggestionEligibilityRequest,
) (instagram.EligibilityDecision, error) {
	return instagram.EligibilityDecision{
		Reason: instagram.EligibilityAlreadyFollowing,
	}, nil
}

type fixedAutomaticFollowPolicy struct {
	reason instagram.EligibilityReason
}

func (p fixedAutomaticFollowPolicy) Evaluate(
	context.Context,
	instagram.EligibilityStage,
	instagram.SuggestionEligibilityRequest,
) (instagram.EligibilityDecision, error) {
	return instagram.EligibilityDecision{Reason: p.reason}, nil
}
