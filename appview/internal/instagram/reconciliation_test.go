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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

func TestReconciliationCreatesOneFixedFiveMinuteDigestForFutureMatches(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	base := time.Date(2026, 7, 19, 22, 0, 0, 0, time.UTC)
	now := base
	importer := syntax.DID("did:plc:synthetic-reconciliation-importer")
	firstTarget := syntax.DID("did:plc:synthetic-reconciliation-first")
	secondTarget := syntax.DID("did:plc:synthetic-reconciliation-second")
	firstImport := uuid.MustParse("00000000-0000-0000-0000-000000000901")
	secondImport := uuid.MustParse("00000000-0000-0000-0000-000000000902")
	seedSuggestionImport(t, pool, firstImport, importer, "synthetic.first", base)
	seedSuggestionImport(t, pool, secondImport, importer, "synthetic.second", base)
	seedSuggestionLink(t, pool, firstTarget, "synthetic.first", base)
	seedSuggestionLink(t, pool, secondTarget, "synthetic.second", base)
	queueLinkReconciliation(t, pool, firstTarget, base)

	service := notifications.NewService()
	policy := newReconciliationPolicy()
	worker := newReconciliationWorkerForTest(t, pool, service, policy, func() time.Time { return now })
	if claimed, err := worker.ProcessBatch(ctx, 1); err != nil || claimed != 1 {
		t.Fatalf("first batch claimed=%d err=%v", claimed, err)
	}

	now = base.Add(4 * time.Minute)
	queueLinkReconciliation(t, pool, secondTarget, now)
	if claimed, err := worker.ProcessBatch(ctx, 1); err != nil || claimed != 1 {
		t.Fatalf("second batch claimed=%d err=%v", claimed, err)
	}

	var (
		events        int
		count         int
		firstActivity time.Time
		activity      time.Time
		coalesceUntil time.Time
		supports      int
	)
	if err := pool.QueryRow(ctx, `
		SELECT count(*), min(system_count), min(first_activity_at),
		       min(activity_at), min(coalesce_until)
		FROM notification_events
		WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
	`, importer).Scan(&events, &count, &firstActivity, &activity, &coalesceUntil); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_notification_suggestions`).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if events != 1 || count != 2 || supports != 2 || !firstActivity.Equal(base) || !activity.Equal(now) || !coalesceUntil.Equal(base.Add(5*time.Minute)) {
		t.Fatalf("digest events=%d count=%d supports=%d first=%s activity=%s close=%s", events, count, supports, firstActivity, activity, coalesceUntil)
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
	service := notifications.NewService()
	store := NewSuggestionStore(pool, service)
	matcher := NewSuggestionMatcher(pool, store, newReconciliationPolicy(), func() time.Time { return now })
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

func TestReconciliationDuplicateJobsAndConcurrentWorkersAreIdempotent(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 23, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-concurrent-importer")
	target := syntax.DID("did:plc:synthetic-concurrent-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000921")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.concurrent", now)
	seedSuggestionLink(t, pool, target, "synthetic.concurrent", now)
	queueLinkReconciliation(t, pool, target, now)
	queueLinkReconciliation(t, pool, target, now)

	service := notifications.NewService()
	worker := newReconciliationWorkerForTest(t, pool, service, newReconciliationPolicy(), func() time.Time { return now })
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			claimed, err := worker.ProcessBatch(ctx, 1)
			if err == nil && claimed != 1 {
				err = errors.New("worker did not claim exactly one job")
			}
			results <- err
		}()
	}
	close(start)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("concurrent reconciliation: %v", err)
		}
	}

	var suggestions, events, supports, completed int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_follow_suggestions`).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events WHERE kind='system'`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_notification_suggestions`).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_reconciliation_jobs WHERE status='completed'`).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if suggestions != 1 || events != 1 || supports != 1 || completed != 2 {
		t.Fatalf("suggestions=%d events=%d supports=%d completed=%d", suggestions, events, supports, completed)
	}
}

func TestReconciliationFailsClosedAtNotificationPolicyBoundary(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 23, 30, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-policy-importer")
	target := syntax.DID("did:plc:synthetic-policy-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000931")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.policy", now)
	seedSuggestionLink(t, pool, target, "synthetic.policy", now)
	queueLinkReconciliation(t, pool, target, now)

	policy := newReconciliationPolicy()
	policy.deniedStage = EligibilityAtNotificationCreate
	worker := newReconciliationWorkerForTest(t, pool, notifications.NewService(), policy, func() time.Time { return now })
	if claimed, err := worker.ProcessBatch(ctx, 1); err != nil || claimed != 1 {
		t.Fatalf("batch claimed=%d err=%v", claimed, err)
	}
	var suggestions, events, ignored int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_follow_suggestions`).Scan(&suggestions); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM notification_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_reconciliation_jobs WHERE status='ignored'`).Scan(&ignored); err != nil {
		t.Fatal(err)
	}
	if suggestions != 0 || events != 0 || ignored != 1 {
		t.Fatalf("suggestions=%d events=%d ignored=%d", suggestions, events, ignored)
	}
	if got := policy.stages(); len(got) != 3 || got[0] != EligibilityAtMatch || got[1] != EligibilityAtPersist || got[2] != EligibilityAtNotificationCreate {
		t.Fatalf("policy stages=%v", got)
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
	worker := newReconciliationWorkerForTest(t, pool, notifications.NewService(), policy, func() time.Time { return now })
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

func TestSuggestionDismissalRetractsReconciliationDigestAndCancelsPush(t *testing.T) {
	pool := newReconciliationTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-retraction-importer")
	target := syntax.DID("did:plc:synthetic-retraction-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000951")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.retraction", now)
	seedSuggestionLink(t, pool, target, "synthetic.retraction", now)
	seedReconciliationSubscription(t, pool, importer)
	queueLinkReconciliation(t, pool, target, now)

	service := notifications.NewService()
	store := NewSuggestionStore(pool, service)
	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool: pool, Store: store, Policy: newReconciliationPolicy(),
		Notifications: service, Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := worker.ProcessBatch(ctx, 1); err != nil {
		t.Fatal(err)
	}
	var suggestionID uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT id FROM instagram_follow_suggestions`).Scan(&suggestionID); err != nil {
		t.Fatal(err)
	}
	if err := store.DismissSuggestion(ctx, importer, suggestionID, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	var suggestionState InstagramSuggestionState
	var eventState, deliveryState string
	var supports int
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id=$1`, suggestionID).Scan(&suggestionState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM notification_events WHERE recipient_did=$1`, importer).Scan(&eventState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&deliveryState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_notification_suggestions`).Scan(&supports); err != nil {
		t.Fatal(err)
	}
	if suggestionState != SuggestionDismissed || eventState != "retracted" || deliveryState != "cancelled" || supports != 0 {
		t.Fatalf("suggestion=%s event=%s delivery=%s supports=%d", suggestionState, eventState, deliveryState, supports)
	}
}

func TestSuggestionStoreRetractsEveryIndividualTerminalTransition(t *testing.T) {
	for _, terminalState := range []InstagramSuggestionState{
		SuggestionDismissed,
		SuggestionInvalidated,
		SuggestionAccepted,
		SuggestionAlreadyFollowing,
	} {
		t.Run(string(terminalState), func(t *testing.T) {
			_, pool := newSuggestionTestStore(t)
			ctx := context.Background()
			now := time.Date(2026, 7, 20, 0, 30, 0, 0, time.UTC)
			importer := syntax.DID("did:plc:synthetic-terminal-importer")
			target := syntax.DID("did:plc:synthetic-terminal-target")
			importID := uuid.MustParse("00000000-0000-0000-0000-000000000961")
			suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000962")
			seedSuggestionImport(t, pool, importID, importer, "synthetic.terminal", now)
			seedSuggestionLink(t, pool, target, "synthetic.terminal", now)
			notifier := &recordingInstagramMatchNotifications{}
			store := NewSuggestionStore(pool, notifier)
			if _, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
				ID: suggestionID, ImporterDID: importer, TargetDID: target,
				ImportID: importID, Username: "synthetic.terminal", Now: now,
			}); err != nil {
				t.Fatal(err)
			}

			switch terminalState {
			case SuggestionDismissed:
				if err := store.DismissSuggestion(ctx, importer, suggestionID, now.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
			case SuggestionInvalidated:
				if err := store.InvalidateSuggestion(ctx, importer, suggestionID, now.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
			case SuggestionAccepted, SuggestionAlreadyFollowing:
				if _, err := store.ClaimSuggestionAcceptance(ctx, importer, suggestionID, "3kterminal22z", now.Add(time.Minute)); err != nil {
					t.Fatal(err)
				}
				if _, err := store.CompleteSuggestionAcceptance(ctx, importer, suggestionID, terminalState, now.Add(2*time.Minute)); err != nil {
					t.Fatal(err)
				}
			default:
				t.Fatalf("unsupported test state %s", terminalState)
			}
			if got := notifier.retractedIDs(); len(got) != 1 || got[0] != suggestionID {
				t.Fatalf("retracted IDs=%v", got)
			}
			var persisted InstagramSuggestionState
			if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id=$1`, suggestionID).Scan(&persisted); err != nil {
				t.Fatal(err)
			}
			if persisted != terminalState {
				t.Fatalf("persisted state=%s want=%s", persisted, terminalState)
			}
		})
	}
}

func TestSuggestionTerminalTransitionRollsBackWhenNotificationRetractionFails(t *testing.T) {
	_, pool := newSuggestionTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 20, 1, 0, 0, 0, time.UTC)
	importer := syntax.DID("did:plc:synthetic-rollback-importer")
	target := syntax.DID("did:plc:synthetic-rollback-target")
	importID := uuid.MustParse("00000000-0000-0000-0000-000000000971")
	suggestionID := uuid.MustParse("00000000-0000-0000-0000-000000000972")
	seedSuggestionImport(t, pool, importID, importer, "synthetic.rollback", now)
	seedSuggestionLink(t, pool, target, "synthetic.rollback", now)
	notifier := &recordingInstagramMatchNotifications{retractErr: errors.New("synthetic notification failure")}
	store := NewSuggestionStore(pool, notifier)
	if _, err := store.UpsertPendingSuggestion(ctx, UpsertSuggestionParams{
		ID: suggestionID, ImporterDID: importer, TargetDID: target,
		ImportID: importID, Username: "synthetic.rollback", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.DismissSuggestion(ctx, importer, suggestionID, now.Add(time.Minute)); err == nil {
		t.Fatal("dismiss succeeded despite notification failure")
	}
	var state InstagramSuggestionState
	if err := pool.QueryRow(ctx, `SELECT state FROM instagram_follow_suggestions WHERE id=$1`, suggestionID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != SuggestionPending {
		t.Fatalf("state=%s, want transaction rollback to pending", state)
	}
}

type reconciliationPolicy struct {
	mu          sync.Mutex
	deniedStage EligibilityStage
	errorStage  EligibilityStage
	calls       []EligibilityStage
}

type recordingInstagramMatchNotifications struct {
	mu         sync.Mutex
	retracted  []uuid.UUID
	retractErr error
}

func (*recordingInstagramMatchNotifications) ActivateInstagramMatch(context.Context, pgx.Tx, notifications.InstagramMatchActivation) error {
	return nil
}

func (n *recordingInstagramMatchNotifications) RetractInstagramMatch(_ context.Context, _ pgx.Tx, retraction notifications.InstagramMatchRetraction) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.retracted = append(n.retracted, retraction.SuggestionID)
	return n.retractErr
}

func (n *recordingInstagramMatchNotifications) retractedIDs() []uuid.UUID {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]uuid.UUID(nil), n.retracted...)
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

func newReconciliationWorkerForTest(t *testing.T, pool *pgxpool.Pool, service *notifications.Service, policy InstagramSuggestionEligibilityPolicy, now func() time.Time) *ReconciliationWorker {
	t.Helper()
	worker, err := NewReconciliationWorker(ReconciliationWorkerOptions{
		Pool: pool, Store: NewSuggestionStore(pool, service), Policy: policy,
		Notifications: service, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return worker
}

func newReconciliationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	coreMigration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(coreMigration))
	for _, path := range []string{
		"../../migrations/000021_appview_notifications.up.sql",
		"../../migrations/000022_notification_newness.up.sql",
		"../../migrations/000024_system_notifications.up.sql",
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
