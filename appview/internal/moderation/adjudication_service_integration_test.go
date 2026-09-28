package moderation

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/testdb"
)

const caseIntakeTestDDL = `
CREATE TABLE moderation_reports (
    id TEXT PRIMARY KEY,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_collection TEXT,
    subject_rkey TEXT,
    subject_uri TEXT,
    subject_cid_snapshot TEXT,
    submitted_handle_snapshot TEXT,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_cases (
    id UUID PRIMARY KEY,
    subject_key TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_collection TEXT,
    subject_rkey TEXT,
    subject_uri TEXT,
    subject_cid_snapshot TEXT,
    owner_did TEXT NOT NULL,
    safe_snapshot JSONB NOT NULL,
    state TEXT NOT NULL DEFAULT 'open',
    revision BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX moderation_cases_one_open_subject_idx
    ON moderation_cases(subject_key) WHERE state='open';
CREATE TABLE moderation_case_reports (
    case_id UUID NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
    report_id TEXT NOT NULL UNIQUE REFERENCES moderation_reports(id) ON DELETE RESTRICT,
    attached_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(case_id,report_id)
);
`

const moderationNotificationTestDDL = `
CREATE TABLE notification_events (
 id UUID PRIMARY KEY, recipient_did TEXT NOT NULL, actor_did TEXT,
 category TEXT NOT NULL, subject_key TEXT NOT NULL, source_uri TEXT, source_cid TEXT,
 source_rkey TEXT, eligibility_scope TEXT NOT NULL, recipient_followed_actor BOOLEAN NOT NULL,
 push_enabled_snapshot BOOLEAN NOT NULL, state TEXT NOT NULL, first_activity_at TIMESTAMPTZ NOT NULL,
 activity_at TIMESTAMPTZ NOT NULL, indexed_at TIMESTAMPTZ NOT NULL,
 initial_push_evaluated_at TIMESTAMPTZ NOT NULL, moderation_case_reference TEXT,
 moderation_event_id UUID
);
CREATE UNIQUE INDEX notification_events_moderation_event_unique
 ON notification_events(moderation_event_id) WHERE category='moderation';
CREATE TABLE notification_preferences (
 account_did TEXT NOT NULL, category TEXT NOT NULL, scope TEXT NOT NULL,
 push_enabled BOOLEAN NOT NULL, PRIMARY KEY(account_did,category)
);
CREATE TABLE push_installations (
 id UUID PRIMARY KEY, device_id TEXT NOT NULL, platform TEXT NOT NULL,
 fcm_token TEXT NOT NULL, active BOOLEAN NOT NULL
);
CREATE TABLE push_account_subscriptions (
 id UUID PRIMARY KEY, installation_id UUID NOT NULL REFERENCES push_installations(id),
 account_did TEXT NOT NULL, routing_id UUID NOT NULL, active BOOLEAN NOT NULL
);
CREATE TABLE push_deliveries (
 id UUID PRIMARY KEY, notification_id UUID NOT NULL REFERENCES notification_events(id),
 account_subscription_id UUID NOT NULL REFERENCES push_account_subscriptions(id), status TEXT NOT NULL,
 next_attempt_at TIMESTAMPTZ NOT NULL, deadline_at TIMESTAMPTZ NOT NULL,
 UNIQUE(notification_id,account_subscription_id)
);`

func TestAttachAcceptedReportsGroupsConcurrentSubject(t *testing.T) {
	pool := testdb.WithSchema(t, caseIntakeTestDDL)
	store := NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:owner")
	uri := syntax.ATURI("at://did:plc:owner/social.craftsky.feed.post/3abc")
	now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)

	start := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	for _, reportID := range []string{"report-one", "report-two"} {
		reportID := reportID
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- insertAndAttachReport(ctx, pool, store, AcceptedReport{
				ID: reportID, SubjectType: SubjectPost, SubjectDID: owner,
				SubjectCollection: "social.craftsky.feed.post", SubjectRkey: "3abc",
				SubjectURI: uri, SubjectCIDSnapshot: "bafycase", CreatedAt: now,
			})
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent report intake: %v", err)
		}
	}

	var caseCount, reportCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM moderation_cases WHERE state='open'`).Scan(&caseCount); err != nil {
		t.Fatalf("count open cases: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM moderation_case_reports`).Scan(&reportCount); err != nil {
		t.Fatalf("count attached reports: %v", err)
	}
	if caseCount != 1 || reportCount != 2 {
		t.Fatalf("open cases/reports = %d/%d, want 1/2", caseCount, reportCount)
	}

	var caseID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM moderation_cases`).Scan(&caseID); err != nil {
		t.Fatalf("read case id: %v", err)
	}
	if _, err := ParseCaseReference("MOD-" + caseID); err != nil {
		t.Fatalf("created case id is not a public UUIDv4 reference: %v", err)
	}
}

func TestNoActionResolutionRemainsPrivateAndCreatesNoOwnerState(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	owner := syntax.DID("did:plc:owner")
	caseRow := seedAdjudicationCase(t, pool, store, "no-action-private-report", owner)
	now := time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(testNotificationWriter{}))

	if _, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "admin-api", ReplayID: "no-action-private",
		ActorID: "private-moderator", SourceDID: syntax.DID("did:plc:moderation"),
		Decision: Decision{Disposition: DispositionNoAction, Evidence: "private moderator notes"},
	}); err != nil {
		t.Fatal(err)
	}

	history, err := store.OwnerHistory(ctx, owner, "", 10)
	if err != nil || len(history.Items) != 0 {
		t.Fatalf("owner history = %+v, err=%v", history, err)
	}
	standing, err := store.Standing(ctx, owner)
	if err != nil || standing.ActiveStrikeCount != 0 || standing.Suspended {
		t.Fatalf("standing = %+v, err=%v", standing, err)
	}
	var effects, intents int
	if err := pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM moderation_effect_events),(SELECT count(*) FROM moderation_notification_intents)`).Scan(&effects, &intents); err != nil {
		t.Fatal(err)
	}
	if effects != 0 || intents != 0 {
		t.Fatalf("noAction effects/intents = %d/%d, want 0/0", effects, intents)
	}
}

func TestAttachAcceptedReportCreatesNewCaseAfterResolution(t *testing.T) {
	pool := testdb.WithSchema(t, caseIntakeTestDDL)
	store := NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:owner")
	uri := syntax.ATURI("at://did:plc:owner/social.craftsky.feed.post/3resolved")
	now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
	first := AcceptedReport{
		ID: "report-before-resolution", SubjectType: SubjectPost, SubjectDID: owner,
		SubjectCollection: "social.craftsky.feed.post", SubjectRkey: "3resolved",
		SubjectURI: uri, SubjectCIDSnapshot: "bafyfirst", CreatedAt: now,
	}
	if err := insertAndAttachReport(ctx, pool, store, first); err != nil {
		t.Fatalf("attach first report: %v", err)
	}

	var firstID string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM moderation_cases WHERE state='open'`).Scan(&firstID); err != nil {
		t.Fatalf("read first case: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE moderation_cases SET state='resolved',resolved_at=$2,updated_at=$2
		WHERE id=$1
	`, firstID, now.Add(time.Hour)); err != nil {
		t.Fatalf("resolve first case: %v", err)
	}

	second := first
	second.ID = "report-after-resolution"
	second.SubjectCIDSnapshot = "bafysecond"
	second.CreatedAt = now.Add(2 * time.Hour)
	if err := insertAndAttachReport(ctx, pool, store, second); err != nil {
		t.Fatalf("attach later report: %v", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT id::text,state FROM moderation_cases ORDER BY created_at,id
	`)
	if err != nil {
		t.Fatalf("list subject cases: %v", err)
	}
	defer rows.Close()
	type caseState struct{ id, state string }
	var cases []caseState
	for rows.Next() {
		var item caseState
		if err := rows.Scan(&item.id, &item.state); err != nil {
			t.Fatalf("scan subject case: %v", err)
		}
		cases = append(cases, item)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate subject cases: %v", err)
	}
	if len(cases) != 2 || cases[0].id != firstID || cases[0].state != "resolved" || cases[1].state != "open" || cases[1].id == firstID {
		t.Fatalf("case history = %+v, want retained resolved case then distinct open case", cases)
	}
}

func TestModerationNotificationIntentPolicyMatrix(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, *pgxpool.Pool, *Store, *Service, *time.Time, syntax.DID)
		want int
	}{
		{
			name: "accepted report",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, _ *Service, _ *time.Time, owner syntax.DID) {
				seedAdjudicationCase(t, pool, store, "notification-report", owner)
			},
			want: 0,
		},
		{
			name: "no-action decision and replay",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, _ *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-no-action", owner)
				command := notificationDecision(caseRow, "notification-no-action-replay", Decision{Disposition: DispositionNoAction})
				assertCommandReplay(t, func() (CommandResult, error) { return service.ResolveCase(context.Background(), command) })
			},
			want: 0,
		},
		{
			name: "consequential decision and replay",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, _ *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-decision", owner)
				command := notificationDecision(caseRow, "notification-decision-replay", Decision{
					Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning},
				})
				assertCommandReplay(t, func() (CommandResult, error) { return service.ResolveCase(context.Background(), command) })
			},
			want: 1,
		},
		{
			name: "effect reversal and reapplication replay",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, now *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-effects", owner)
				decision, err := service.ResolveCase(context.Background(), notificationDecision(caseRow, "notification-effects-decision", Decision{
					Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning},
				}))
				if err != nil {
					t.Fatal(err)
				}
				reverse := EffectChangeCommand{CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "retool", ReplayID: "notification-effect-reverse", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Negate: []EffectType{EffectFormalWarning}, Rationale: "reversed"}
				reversed := assertCommandReplay(t, func() (CommandResult, error) { return service.ChangeEffects(context.Background(), reverse) })
				*now = now.Add(time.Minute)
				reapply := EffectChangeCommand{CaseID: caseRow.ID, ExpectedRevision: reversed.Revision, SourceSystem: "retool", ReplayID: "notification-effect-reapply", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Apply: []EffectType{EffectFormalWarning}, Rationale: "reapplied"}
				assertCommandReplay(t, func() (CommandResult, error) { return service.ChangeEffects(context.Background(), reapply) })
			},
			want: 3,
		},
		{
			name: "appeal confirmation and upheld outcome",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, _ *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-appeal-upheld", owner)
				decision, err := service.ResolveCase(context.Background(), notificationDecision(caseRow, "notification-appeal-upheld-decision", Decision{
					Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning},
				}))
				if err != nil {
					t.Fatal(err)
				}
				confirm := AppealCommand{CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "retool", ReplayID: "notification-appeal-confirm", ActorID: "moderator"}
				confirmed := assertCommandReplay(t, func() (CommandResult, error) { return service.ConfirmAppeal(context.Background(), confirm) })
				uphold := AppealResolutionCommand{CaseID: caseRow.ID, ExpectedRevision: confirmed.Revision, SourceSystem: "retool", ReplayID: "notification-appeal-uphold", ActorID: "moderator", Outcome: AppealStatusUpheld, Rationale: "upheld"}
				assertCommandReplay(t, func() (CommandResult, error) { return service.ResolveAppeal(context.Background(), uphold) })
			},
			want: 1,
		},
		{
			name: "appeal consequence change",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, _ *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-appeal-changed", owner)
				decision, err := service.ResolveCase(context.Background(), notificationDecision(caseRow, "notification-appeal-changed-decision", Decision{
					Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectFormalWarning},
				}))
				if err != nil {
					t.Fatal(err)
				}
				confirmed, err := service.ConfirmAppeal(context.Background(), AppealCommand{CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "retool", ReplayID: "notification-changed-confirm", ActorID: "moderator"})
				if err != nil {
					t.Fatal(err)
				}
				changed := AppealResolutionCommand{CaseID: caseRow.ID, ExpectedRevision: confirmed.Revision, SourceSystem: "retool", ReplayID: "notification-appeal-change", ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Outcome: AppealStatusChanged, Negate: []EffectType{EffectFormalWarning}, Rationale: "changed"}
				assertCommandReplay(t, func() (CommandResult, error) { return service.ResolveAppeal(context.Background(), changed) })
			},
			want: 2,
		},
		{
			name: "explicit severe restoration",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, _ *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-severe-restoration", owner)
				decision, err := service.ResolveCase(context.Background(), notificationDecision(caseRow, "notification-severe-decision", Decision{
					Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence", SeverityRationale: "severe", Consequences: []EffectType{EffectSevereSuspension},
				}))
				if err != nil {
					t.Fatal(err)
				}
				var effectID uuid.UUID
				if err := pool.QueryRow(context.Background(), `SELECT logical_effect_id FROM moderation_active_case_effects WHERE case_id=$1`, caseRow.ID).Scan(&effectID); err != nil {
					t.Fatal(err)
				}
				restore := SevereRestorationCommand{CaseID: caseRow.ID, EffectID: effectID, ExpectedRevision: decision.Revision, SourceSystem: "retool", ReplayID: "notification-severe-restore", ActorID: "moderator", Rationale: "restored"}
				assertCommandReplay(t, func() (CommandResult, error) { return service.RestoreSevereSuspension(context.Background(), restore) })
			},
			want: 2,
		},
		{
			name: "quiet expiry under severe suspension",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, now *time.Time, owner syntax.DID) {
				caseRow := seedAdjudicationCase(t, pool, store, "notification-quiet-expiry", owner)
				if _, err := service.ResolveCase(context.Background(), notificationDecision(caseRow, "notification-quiet-decision", Decision{
					Disposition: DispositionViolation, Reason: ReasonHate, Evidence: "evidence", SeverityRationale: "severe", Consequences: []EffectType{EffectStrike, EffectSevereSuspension},
				})); err != nil {
					t.Fatal(err)
				}
				*now = StrikeDeadline(*now)
				processor := NewExpiryProcessor(store, notifications.ModerationWriter{}, func() time.Time { return *now })
				if processed, err := processor.ProcessDue(context.Background(), 100); err != nil || processed != 1 {
					t.Fatalf("quiet expiry = %d, %v", processed, err)
				}
			},
			want: 1,
		},
		{
			name: "expiry restoring threshold enforcement and retry",
			run: func(t *testing.T, pool *pgxpool.Pool, store *Store, service *Service, now *time.Time, owner syntax.DID) {
				issued := *now
				for index := 1; index <= 3; index++ {
					caseRow := seedAdjudicationCase(t, pool, store, fmt.Sprintf("notification-expiry-%d", index), owner)
					if _, err := service.ResolveCase(context.Background(), notificationDecision(caseRow, fmt.Sprintf("notification-expiry-decision-%d", index), Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectStrike}})); err != nil {
						t.Fatal(err)
					}
					*now = now.Add(time.Minute)
				}
				*now = StrikeDeadline(issued)
				processor := NewExpiryProcessor(store, notifications.ModerationWriter{}, func() time.Time { return *now })
				if processed, err := processor.ProcessDue(context.Background(), 100); err != nil || processed != 1 {
					t.Fatalf("restoring expiry = %d, %v", processed, err)
				}
				if processed, err := processor.ProcessDue(context.Background(), 100); err != nil || processed != 0 {
					t.Fatalf("expiry retry = %d, %v", processed, err)
				}
			},
			want: 4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL+moderationNotificationTestDDL)
			store := NewStore(pool)
			now := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
			owner := syntax.DID("did:plc:notification-matrix")
			service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithNotificationWriter(notifications.ModerationWriter{}))
			test.run(t, pool, store, service, &now, owner)
			var got int
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_events WHERE recipient_did=$1 AND category='moderation'`, owner).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("moderation notification intents = %d, want %d", got, test.want)
			}
		})
	}
}

func TestModerationNotificationTransactionFailuresRollBackAllCoupledState(t *testing.T) {
	faults := []struct {
		name      string
		table     string
		timing    string
		operation string
	}{
		{name: "case event", table: "moderation_case_events", timing: "BEFORE", operation: "INSERT"},
		{name: "effect", table: "moderation_effect_events", timing: "BEFORE", operation: "INSERT"},
		{name: "standing", table: "moderation_account_standings", timing: "BEFORE", operation: "UPDATE"},
		{name: "notification", table: "notification_events", timing: "BEFORE", operation: "INSERT"},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL+moderationNotificationTestDDL)
			ctx := context.Background()
			store := NewStore(pool)
			owner := syntax.DID("did:plc:notification-rollback")
			caseRow := seedAdjudicationCase(t, pool, store, "notification-rollback-report", owner)
			trigger := fmt.Sprintf(`CREATE FUNCTION reject_notification_matrix_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected write failure'; END $$;
				CREATE TRIGGER reject_notification_matrix_write %s %s ON %s FOR EACH ROW EXECUTE FUNCTION reject_notification_matrix_write()`, fault.timing, fault.operation, fault.table)
			if _, err := pool.Exec(ctx, trigger); err != nil {
				t.Fatal(err)
			}
			service := NewService(store, testVisibilityWriter{}, time.Now, WithNotificationWriter(notifications.ModerationWriter{}))
			if _, err := service.ResolveCase(ctx, notificationDecision(caseRow, "notification-rollback-replay", Decision{
				Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectVisibilityHide, EffectStrike},
			})); err == nil {
				t.Fatal("decision succeeded with injected transaction failure")
			}
			var state string
			var revision int
			if err := pool.QueryRow(ctx, `SELECT state,revision FROM moderation_cases WHERE id=$1`, caseRow.ID).Scan(&state, &revision); err != nil {
				t.Fatal(err)
			}
			if state != "open" || revision != 0 {
				t.Fatalf("case after rollback = %s/%d, want open/0", state, revision)
			}
			for _, table := range []string{"moderation_case_events", "moderation_decisions", "moderation_effect_events", "moderation_active_case_effects", "moderation_case_strikes", "moderation_outputs", "moderation_account_standings", "notification_events"} {
				var count int
				if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatalf("%s count after rollback = %d, want 0", table, count)
				}
			}
		})
	}
}

func notificationDecision(caseRow Case, replayID string, decision Decision) TrustedCommand {
	return TrustedCommand{CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: replayID, ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"), Decision: decision}
}

func assertCommandReplay(t *testing.T, run func() (CommandResult, error)) CommandResult {
	t.Helper()
	first, err := run()
	if err != nil {
		t.Fatal(err)
	}
	replay, err := run()
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.EventID != first.EventID || replay.Revision != first.Revision {
		t.Fatalf("replay = %+v, first = %+v", replay, first)
	}
	return first
}

func insertAndAttachReport(
	ctx context.Context,
	pool *pgxpool.Pool,
	store *Store,
	report AcceptedReport,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO moderation_reports(
			id,subject_type,subject_did,subject_collection,subject_rkey,
			subject_uri,subject_cid_snapshot,created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, report.ID, report.SubjectType, report.SubjectDID, report.SubjectCollection,
		report.SubjectRkey, report.SubjectURI, report.SubjectCIDSnapshot, report.CreatedAt); err != nil {
		return fmt.Errorf("insert report: %w", err)
	}
	if _, err := store.AttachAcceptedReportTx(ctx, tx, report); err != nil {
		return fmt.Errorf("attach report: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
