package instagram

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

	"social.craftsky/appview/internal/testdb"
)

func TestAutomaticFollowLedgerSuppressesReconciliationUntilVerificationRevocation(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 11, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-suppression-importer")
	target := syntax.DID("did:plc:synthetic-suppression-target")
	firstImport := uuid.MustParse("00000000-0000-0000-0000-000000001001")
	secondImport := uuid.MustParse("00000000-0000-0000-0000-000000001002")
	firstOperation := uuid.MustParse("00000000-0000-0000-0000-000000001003")
	newGenerationOperation := uuid.MustParse("00000000-0000-0000-0000-000000001004")
	seedSuggestionImport(t, pool, firstImport, importer, "synthetic.suppression", now)
	seedSuggestionImport(t, pool, secondImport, importer, "synthetic.suppression", now)
	seedSuggestionLink(t, pool, target, "synthetic.suppression", now)
	applyAutomaticFollowMigration(t, pool)

	store := NewAutomaticFollowStore(pool)
	first, err := store.ReconcileCandidate(ctx, ReconcileAutomaticFollowParams{
		ID: firstOperation, ImporterDID: importer, TargetDID: target,
		ImportID: firstImport, Username: "synthetic.suppression",
		Rkey: "3lautomaticsuppress", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Queued || first.Suppressed {
		t.Fatalf("first reconciliation = %+v, want queued", first)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE instagram_follow_suggestions
		SET state='followed', terminal_at=$2, updated_at=$2
		WHERE id=$1
	`, first.Operation.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("complete automatic follow: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status='followed', completed_at=$2, updated_at=$2
		WHERE suggestion_id=$1
	`, first.Operation.ID, now.Add(time.Minute)); err != nil {
		t.Fatalf("complete PDS operation: %v", err)
	}

	for name, importID := range map[string]uuid.UUID{
		"same import after manual unfollow":       firstImport,
		"different supporting import":             secondImport,
		"relationship restoration reconciliation": secondImport,
	} {
		t.Run(name, func(t *testing.T) {
			reconciled, err := store.ReconcileCandidate(ctx, ReconcileAutomaticFollowParams{
				ID: uuid.New(), ImporterDID: importer, TargetDID: target,
				ImportID: importID, Username: "synthetic.suppression",
				Rkey: "3lshouldnotreplace", Now: now.Add(2 * time.Minute),
			})
			if err != nil {
				t.Fatal(err)
			}
			if reconciled.Queued || !reconciled.Suppressed || reconciled.Operation.ID != first.Operation.ID {
				t.Fatalf("reconciliation = %+v, want original terminal suppression", reconciled)
			}
		})
	}

	var operations int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM pds_follow_operations
		WHERE owner_did=$1 AND target_did=$2
	`, importer, target).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if operations != 1 {
		t.Fatalf("operations=%d, want one terminal operation after repeated triggers", operations)
	}

	if err := store.DeleteVerificationLedger(ctx, importer); err != nil {
		t.Fatalf("delete verification ledger: %v", err)
	}
	fresh, err := store.ReconcileCandidate(ctx, ReconcileAutomaticFollowParams{
		ID: newGenerationOperation, ImporterDID: importer, TargetDID: target,
		ImportID: secondImport, Username: "synthetic.suppression",
		Rkey: "3lautomaticfresh", Now: now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.Queued || fresh.Suppressed || fresh.Operation.ID != newGenerationOperation {
		t.Fatalf("fresh reconciliation = %+v, want new authorization", fresh)
	}
}

func TestReconciliationLoadsImportAndTargetScopedCandidates(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 22, 20, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-scoped-importer")
	target := syntax.DID("did:plc:synthetic-scoped-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000909")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.scoped", now)
	seedSuggestionLink(t, pool, target, "synthetic.scoped", now)
	worker := newReconciliationWorkerForTest(
		t,
		pool,
		newReconciliationPolicy(),
		func() time.Time { return now },
	)

	for name, job := range map[string]reconciliationJob{
		"import and target": {
			OwnerDID:  importer,
			ImportID:  &importID,
			TargetDID: &target,
		},
		"target": {
			OwnerDID:  importer,
			TargetDID: &target,
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidates, err := worker.loadCandidates(ctx, job)
			if err != nil {
				t.Fatal(err)
			}
			if len(candidates) != 1 ||
				candidates[0].ImportID != importID ||
				candidates[0].TargetDID != target {
				t.Fatalf("candidates = %+v", candidates)
			}
		})
	}
}

func TestInitialImportMatchingNeverCreatesInstagramMatchNotification(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 22, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-initial-importer")
	target := syntax.DID("did:plc:synthetic-initial-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000911")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.initial", now)
	seedSuggestionLink(t, pool, target, "synthetic.initial", now)
	applyAutomaticFollowMigration(t, pool)
	store := NewAutomaticFollowStore(pool)
	matcher := NewAutomaticFollowMatcher(pool, store, newReconciliationPolicy(), func() time.Time { return now })
	if created, err := matcher.MatchImport(ctx, importer, importID); err != nil || created != 1 {
		t.Fatalf("initial match created=%d err=%v", created, err)
	}
	var suggestions, events int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_follow_suggestions`).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || events != 0 {
		t.Fatalf("initial suggestions=%d notification events=%d", suggestions, events)
	}
}

func TestReconciliationPolicyErrorReleasesLeaseForBoundedRetry(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 23, 45, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-policy-error-importer")
	target := syntax.DID("did:plc:synthetic-policy-error-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000941")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.policy.error", now)
	seedSuggestionLink(t, pool, target, "synthetic.policy.error", now)
	queueLinkReconciliation(t, pool, target, now)

	policy := newReconciliationPolicy()
	policy.errorStage = EligibilityAtPersist
	applyAutomaticFollowMigration(t, pool)
	worker := newReconciliationWorkerForTest(t, pool, policy, func() time.Time { return now })
	if claimed, err := worker.ProcessBatch(ctx, 1); err == nil || claimed != 1 {
		t.Fatalf("batch claimed=%d err=%v, want persisted retry error", claimed, err)
	}
	var status string
	var attempts int
	var leaseToken uuid.NullUUID
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT status, attempts, lease_token, next_attempt_at
		FROM instagram_reconciliation_jobs
	`).Scan(&status, &attempts, &leaseToken, &nextAttempt); err != nil {
		t.Fatal(err)
	}
	if status != "retryable" || attempts != 1 || leaseToken.Valid || !nextAttempt.Equal(now.Add(time.Second)) {
		t.Fatalf("status=%s attempts=%d lease=%v next=%s", status, attempts, leaseToken, nextAttempt)
	}
}

func TestReconciliationWorkerInactivatesDepartedTargetWithoutOperation(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:current-reconciliation-importer")
	target := syntax.DID("did:plc:departed-reconciliation-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000951")
	if _, err := pool.Exec(ctx, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did) VALUES ($1)
	`, importer); err != nil {
		t.Fatal(err)
	}
	seedSuggestionImport(t, pool, importID, importer, "synthetic.departed", now)
	seedSuggestionLink(t, pool, target, "synthetic.departed", now)
	queueLinkReconciliation(t, pool, target, now)
	applyAutomaticFollowMigration(t, pool)

	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool:             pool,
		AutomaticFollows: NewAutomaticFollowStore(pool),
		Policy:           newReconciliationPolicy(),
		Membership:       NewMembershipStore(pool),
		MembershipInactivator: NewPrivateDataService(
			pool,
			nil,
			func() time.Time { return now },
		),
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := worker.ProcessBatch(ctx, 1); err != nil || claimed != 1 {
		t.Fatalf("ProcessBatch claimed=%d err=%v", claimed, err)
	}

	var linkState, jobState string
	if err := pool.QueryRow(ctx, `
		SELECT state FROM instagram_account_links WHERE owner_did=$1
	`, target).Scan(&linkState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status FROM instagram_reconciliation_jobs
	`).Scan(&jobState); err != nil {
		t.Fatal(err)
	}
	var operations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_follow_operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if linkState != string(LinkMembershipInactive) || jobState != "ignored" || operations != 0 {
		t.Fatalf(
			"link=%s job=%s operations=%d, want membershipInactive/ignored/0",
			linkState,
			jobState,
			operations,
		)
	}
}

func TestReconciliationWorkerInactivatesDepartedOwnerWithoutCandidates(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 27, 16, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:departed-reconciliation-owner")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000952")
	if _, err := pool.Exec(ctx, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY)
	`); err != nil {
		t.Fatal(err)
	}
	seedSuggestionImport(t, pool, importID, owner, "synthetic.no.candidate", now)
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_reconciliation_jobs (
			id, owner_did, import_id, reason, status, next_attempt_at,
			created_at, updated_at
		) VALUES ($1,$2,$3,'syntheticDepartedOwnerNoCandidates','queued',$4,$4,$4)
	`, uuid.New(), owner, importID, now); err != nil {
		t.Fatal(err)
	}
	applyAutomaticFollowMigration(t, pool)

	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool:             pool,
		AutomaticFollows: NewAutomaticFollowStore(pool),
		Policy:           newReconciliationPolicy(),
		Membership:       NewMembershipStore(pool),
		MembershipInactivator: NewPrivateDataService(
			pool,
			nil,
			func() time.Time { return now },
		),
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed, err := worker.ProcessBatch(ctx, 1); err != nil || claimed != 1 {
		t.Fatalf("ProcessBatch claimed=%d err=%v", claimed, err)
	}

	var importState, jobState string
	if err := pool.QueryRow(ctx, `
		SELECT state FROM instagram_graph_imports WHERE id=$1
	`, importID).Scan(&importState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT status FROM instagram_reconciliation_jobs
	`).Scan(&jobState); err != nil {
		t.Fatal(err)
	}
	var operations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_follow_operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if importState != string(ImportMembershipInactive) || jobState != "ignored" || operations != 0 {
		t.Fatalf(
			"import=%s job=%s operations=%d, want membershipInactive/ignored/0",
			importState,
			jobState,
			operations,
		)
	}
}

func TestReconciliationWorkerRetriesJobOwnerMembershipFailures(t *testing.T) {
	for _, test := range []struct {
		name            string
		membershipErr   error
		inactivationErr error
	}{
		{
			name:          "membership lookup",
			membershipErr: errors.New("synthetic membership lookup failure"),
		},
		{
			name:            "membership inactivation",
			inactivationErr: errors.New("synthetic membership inactivation failure"),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := newReconciliationTestPool(t)
			ctx := context.Background()
			now := time.Date(2026, 7, 27, 16, 45, 0, 0, time.UTC)
			owner := syntax.DID("did:plc:reconciliation-membership-failure")
			importID := uuid.New()
			seedSuggestionImport(t, pool, importID, owner, "synthetic.membership.failure", now)
			if _, err := pool.Exec(ctx, `
				INSERT INTO instagram_reconciliation_jobs (
					id, owner_did, import_id, reason, status, next_attempt_at,
					created_at, updated_at
				) VALUES ($1,$2,$3,'syntheticMembershipFailure','queued',$4,$4,$4)
			`, uuid.New(), owner, importID, now); err != nil {
				t.Fatal(err)
			}
			applyAutomaticFollowMigration(t, pool)
			membershipEvents := []string{}
			inactivator := &reconciliationInactivatorStub{err: test.inactivationErr}
			worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
				Pool:             pool,
				AutomaticFollows: NewAutomaticFollowStore(pool),
				Policy:           newReconciliationPolicy(),
				Membership: &fakeWebhookMembership{
					current: false,
					err:     test.membershipErr,
					events:  &membershipEvents,
				},
				MembershipInactivator: inactivator,
				Now:                   func() time.Time { return now },
			})
			if err != nil {
				t.Fatal(err)
			}
			if claimed, err := worker.ProcessBatch(ctx, 1); err == nil || claimed != 1 {
				t.Fatalf("ProcessBatch claimed=%d err=%v, want retryable membership error", claimed, err)
			}

			var status string
			var attempts int
			var leaseToken uuid.NullUUID
			var nextAttempt time.Time
			if err := pool.QueryRow(ctx, `
				SELECT status, attempts, lease_token, next_attempt_at
				FROM instagram_reconciliation_jobs
			`).Scan(&status, &attempts, &leaseToken, &nextAttempt); err != nil {
				t.Fatal(err)
			}
			if status != "retryable" || attempts != 1 || leaseToken.Valid ||
				!nextAttempt.Equal(now.Add(time.Second)) {
				t.Fatalf(
					"status=%s attempts=%d lease=%v next=%s, want retryable/1/no lease/%s",
					status,
					attempts,
					leaseToken,
					nextAttempt,
					now.Add(time.Second),
				)
			}
			if len(membershipEvents) != 1 || membershipEvents[0] != "membership" {
				t.Fatalf("membership events=%v, want one owner lookup", membershipEvents)
			}
			wantInactivations := 1
			if test.membershipErr != nil {
				wantInactivations = 0
			}
			if len(inactivator.owners) != wantInactivations {
				t.Fatalf("inactivated owners=%v, want %d call(s)", inactivator.owners, wantInactivations)
			}
		})
	}
}

type reconciliationInactivatorStub struct {
	owners []syntax.DID
	err    error
}

func (s *reconciliationInactivatorStub) InactivateMembership(
	_ context.Context,
	did syntax.DID,
) error {
	s.owners = append(s.owners, did)
	return s.err
}

type reconciliationPolicy struct {
	mu          sync.Mutex
	deniedStage EligibilityStage
	errorStage  EligibilityStage
	calls       []EligibilityStage
}

func newReconciliationPolicy() *reconciliationPolicy { return &reconciliationPolicy{} }

func (p *reconciliationPolicy) Evaluate(_ context.Context, stage EligibilityStage, _ SuggestionEligibilityRequest) (EligibilityDecision, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, stage)
	if stage == p.errorStage {
		return EligibilityDecision{}, errors.New("synthetic policy unavailable")
	}
	if stage == p.deniedStage {
		return EligibilityDecision{Reason: EligibilityRelationshipSafety}, nil
	}
	return EligibilityDecision{Eligible: true, Reason: EligibilityAllowed}, nil
}

func (p *reconciliationPolicy) stages() []EligibilityStage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]EligibilityStage(nil), p.calls...)
}

func newReconciliationWorkerForTest(t *testing.T, pool *pgxpool.Pool, policy InstagramSuggestionEligibilityPolicy, now func() time.Time) *ReconciliationWorker {
	t.Helper()
	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool: pool, AutomaticFollows: NewAutomaticFollowStore(pool),
		Policy: policy, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return worker
}

func newReconciliationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	coreMigration, err := os.ReadFile("../../migrations/000025_instagram_migration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(coreMigration))
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000026_system_notifications.up.sql",
		"../../migrations/000029_notification_client_owned_destination.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
	return pool
}

func applyAutomaticFollowMigration(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000030_instagram_automatic_follows.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply automatic-follow migration: %v", err)
	}
}

func queueLinkReconciliation(t *testing.T, pool *pgxpool.Pool, target syntax.DID, now time.Time) {
	t.Helper()
	var linkID uuid.UUID
	if err := pool.QueryRow(context.Background(), `
		SELECT id FROM instagram_account_links
		WHERE owner_did=$1 AND state='active'
	`, target).Scan(&linkID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_reconciliation_jobs (
			id, owner_did, link_id, reason, status, next_attempt_at,
			created_at, updated_at
		) VALUES ($1,$2,$3,'syntheticTargetedReconciliation','queued',$4,$4,$4)
	`, uuid.New(), target, linkID, now); err != nil {
		t.Fatal(err)
	}
}

func seedReconciliationSubscription(t *testing.T, pool *pgxpool.Pool, recipient syntax.DID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_installations (id, device_id, platform, fcm_token)
		VALUES ($1, $2, 'ios', $3)
	`, uuid.New(), "synthetic-reconciliation-device", "synthetic-reconciliation-token"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO push_account_subscriptions (id, installation_id, account_did, routing_id)
		SELECT $1, id, $2, $3 FROM push_installations
		WHERE device_id='synthetic-reconciliation-device'
	`, uuid.New(), recipient, uuid.New()); err != nil {
		t.Fatal(err)
	}
}

var _ InstagramSuggestionEligibilityPolicy = (*reconciliationPolicy)(nil)
