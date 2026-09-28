package moderation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
)

const adjudicationTestDDL = caseIntakeTestDDL + `
CREATE TABLE moderation_outputs (
    id TEXT PRIMARY KEY, source_did TEXT NOT NULL, subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL, subject_collection TEXT, subject_rkey TEXT,
    subject_uri TEXT, value TEXT NOT NULL, action TEXT NOT NULL,
    internal_reason TEXT, expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL, indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE moderation_account_standings (
    owner_did TEXT PRIMARY KEY, active_strike_count INTEGER NOT NULL DEFAULT 0,
    threshold_suspended BOOLEAN NOT NULL DEFAULT false,
    severe_suspended BOOLEAN NOT NULL DEFAULT false,
    effective_suspended BOOLEAN GENERATED ALWAYS AS (threshold_suspended OR severe_suspended) STORED,
    revision BIGINT NOT NULL DEFAULT 0, updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE moderation_case_events (
    id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id),
    event_type TEXT NOT NULL, actor_id TEXT NOT NULL, source_system TEXT NOT NULL,
    replay_id TEXT NOT NULL, request_fingerprint BYTEA NOT NULL,
    expected_revision BIGINT NOT NULL, result_revision BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL, UNIQUE(source_system,replay_id),
    UNIQUE(case_id,result_revision)
);
CREATE TABLE moderation_decisions (
    id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id),
    case_event_id UUID NOT NULL UNIQUE REFERENCES moderation_case_events(id),
    disposition TEXT NOT NULL, reason TEXT, internal_evidence_notes TEXT,
    user_safe_detail TEXT, severity_rationale TEXT, created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_effect_events (
    id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id),
    case_event_id UUID NOT NULL REFERENCES moderation_case_events(id),
    logical_effect_id UUID NOT NULL, effect_type TEXT NOT NULL, action TEXT NOT NULL,
    moderation_output_id TEXT REFERENCES moderation_outputs(id), rationale TEXT,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_active_case_effects (
    case_id UUID NOT NULL REFERENCES moderation_cases(id), effect_type TEXT NOT NULL,
    logical_effect_id UUID NOT NULL UNIQUE,
    applied_event_id UUID NOT NULL REFERENCES moderation_case_events(id),
    moderation_output_id TEXT REFERENCES moderation_outputs(id), applied_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(case_id,effect_type)
);
CREATE TABLE moderation_case_strikes (
    case_id UUID PRIMARY KEY REFERENCES moderation_cases(id), logical_effect_id UUID NOT NULL UNIQUE,
    owner_did TEXT NOT NULL, issued_at TIMESTAMPTZ NOT NULL, due_at TIMESTAMPTZ NOT NULL,
    expired_at TIMESTAMPTZ, overturned_at TIMESTAMPTZ, updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE moderation_appeal_correspondence (
    id UUID PRIMARY KEY, case_id UUID NOT NULL REFERENCES moderation_cases(id),
    source_system TEXT NOT NULL, replay_id TEXT NOT NULL, sender_reference_hash BYTEA,
    received_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL,
    UNIQUE(source_system,replay_id)
);
CREATE TABLE moderation_appeals (
    case_id UUID PRIMARY KEY REFERENCES moderation_cases(id),
    confirmed_event_id UUID NOT NULL UNIQUE REFERENCES moderation_case_events(id),
    resolved_event_id UUID UNIQUE REFERENCES moderation_case_events(id), status TEXT NOT NULL,
    confirmed_at TIMESTAMPTZ NOT NULL, resolved_at TIMESTAMPTZ
);
CREATE TABLE moderation_notification_intents (
    event_id UUID PRIMARY KEY, case_id UUID NOT NULL, recipient_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
`

type testVisibilityWriter struct{}

func (testVisibilityWriter) InsertOutputTx(ctx context.Context, tx pgx.Tx, input VisibilityOutputWrite) (string, error) {
	id := uuid.NewString()
	_, err := tx.Exec(ctx, `INSERT INTO moderation_outputs(
		id,source_did,subject_type,subject_did,subject_collection,subject_rkey,
		subject_uri,value,action,internal_reason,created_at
	) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, input.SourceDID,
		input.SubjectType, input.SubjectDID, nullIfEmpty(input.SubjectCollection),
		nullIfEmpty(input.SubjectRkey), nullIfEmpty(input.SubjectURI.String()), input.Value,
		input.Action, nullIfEmpty(input.InternalReason), input.CreatedAt)
	return id, err
}

type testNotificationWriter struct{}

func (testNotificationWriter) InsertModerationIntentTx(ctx context.Context, tx pgx.Tx, input notifications.ModerationIntent) error {
	_, err := tx.Exec(ctx, `INSERT INTO moderation_notification_intents(event_id,case_id,recipient_did,created_at) VALUES($1,$2,$3,$4)`, input.EventID, input.CaseID, input.RecipientDID, input.CreatedAt)
	return err
}

type moderationOperationCall struct {
	operation, result, actorType, requestID, caseReference string
	occurredAt                                             time.Time
}

type moderationOperationObserver struct{ calls []moderationOperationCall }

func (o *moderationOperationObserver) ObserveModerationCommand(_ context.Context, operation, result, actorType, requestID, caseReference string, occurredAt time.Time) {
	o.calls = append(o.calls, moderationOperationCall{operation, result, actorType, requestID, caseReference, occurredAt})
}

type serviceModerationLogEvent struct {
	message string
	attrs   observability.EventContext
}

type serviceModerationLogSink struct{ events []serviceModerationLogEvent }

func (s *serviceModerationLogSink) Emit(_ context.Context, _ slog.Level, message string, attrs observability.EventContext) {
	s.events = append(s.events, serviceModerationLogEvent{message: message, attrs: attrs})
}

func TestModerationProductionOperationsEmitBoundedSafeTelemetry(t *testing.T) {
	const (
		ownerSentinel     = "did:plc:SENTINEL_OWNER"
		evidenceSentinel  = "SENTINEL_PRIVATE_EVIDENCE"
		actorSentinel     = "SENTINEL_MODERATOR_ID"
		sourceSentinel    = "SENTINEL_SOURCE_SYSTEM"
		replaySentinel    = "SENTINEL_REPLAY_ID"
		rationaleSentinel = "SENTINEL_INTERNAL_RATIONALE"
	)
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2025, 9, 10, 12, 0, 0, 0, time.UTC)
	recorder := observability.NewInMemoryMetricRecorder()
	sink := &serviceModerationLogSink{}
	var logOutput bytes.Buffer
	observer := observability.New(observability.Config{
		SentryDSN: "https://public@example.invalid/1", LogsEnabled: true, MetricsEnabled: true,
		MetricRecorder: recorder, LogSink: sink, Logger: slog.New(slog.NewJSONHandler(&logOutput, nil)),
	})
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now }, WithOperationObserver(observer))
	owner := syntax.DID(ownerSentinel)

	resolveStrikeCase := func(report, replay string) (Case, CommandResult) {
		t.Helper()
		caseRow := seedAdjudicationCase(t, pool, store, report, owner)
		result, err := service.ResolveCase(ctx, TrustedCommand{
			CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: sourceSentinel,
			ReplayID: replay, ActorID: actorSentinel, SourceDID: syntax.DID("did:plc:SENTINEL_RAW_SOURCE"),
			Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam,
				Evidence: evidenceSentinel, Consequences: []EffectType{EffectStrike}},
		})
		if err != nil {
			t.Fatalf("resolve strike case: %v", err)
		}
		return caseRow, result
	}

	appealCase, appealDecision := resolveStrikeCase("telemetry-appeal", replaySentinel+"-decision-appeal")
	confirmed, err := service.ConfirmAppeal(ctx, AppealCommand{
		CaseID: appealCase.ID, ExpectedRevision: appealDecision.Revision, SourceSystem: sourceSentinel,
		ReplayID: replaySentinel + "-confirm", ActorID: actorSentinel,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ResolveAppeal(ctx, AppealResolutionCommand{
		CaseID: appealCase.ID, ExpectedRevision: confirmed.Revision, SourceSystem: sourceSentinel,
		ReplayID: replaySentinel + "-resolve", ActorID: actorSentinel,
		Outcome: AppealStatusUpheld, Rationale: rationaleSentinel,
	}); err != nil {
		t.Fatal(err)
	}

	reversalCase, reversalDecision := resolveStrikeCase("telemetry-reversal", replaySentinel+"-decision-reversal")
	if _, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: reversalCase.ID, ExpectedRevision: reversalDecision.Revision, SourceSystem: sourceSentinel,
		ReplayID: replaySentinel + "-reverse", ActorID: actorSentinel,
		SourceDID: syntax.DID("did:plc:SENTINEL_RAW_SOURCE"), Negate: []EffectType{EffectStrike},
		Rationale: rationaleSentinel,
	}); err != nil {
		t.Fatal(err)
	}

	_, _ = resolveStrikeCase("telemetry-expiry", replaySentinel+"-decision-expiry")
	now = StrikeDeadline(now).Add(time.Hour + time.Microsecond)
	processor := NewExpiryProcessor(store, nil, func() time.Time { return now }, observer)
	if processed, err := processor.ProcessDue(ctx, 10); err != nil || processed != 2 {
		t.Fatalf("expiry processed = %d, err = %v", processed, err)
	}

	wantOperations := map[string]int{
		"decision": 3, "appeal_confirmation": 1, "appeal_resolution": 1,
		"effect_change": 1, "strike_expiry": 2,
	}
	gotOperations := map[string]int{}
	expiryAlerts := 0
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_moderation_operations_total" {
			if len(call.Attributes) != 2 || call.Attributes["result"] != "success" {
				t.Fatalf("operation metric attributes = %#v", call.Attributes)
			}
			gotOperations[call.Attributes["operation"]]++
		}
		if strings.HasPrefix(call.Name, "craftsky_appview_moderation_work_") || call.Name == "craftsky_appview_moderation_alert_active" {
			if len(call.Attributes) != 1 || call.Attributes["kind"] != "strike_expiry" {
				t.Fatalf("expiry metric attributes = %#v", call.Attributes)
			}
			if call.Name == "craftsky_appview_moderation_alert_active" && call.Value == 1 {
				expiryAlerts++
			}
		}
	}
	if fmt.Sprint(gotOperations) != fmt.Sprint(wantOperations) {
		t.Fatalf("operation metric labels = %#v, want %#v", gotOperations, wantOperations)
	}
	if expiryAlerts != 2 {
		t.Fatalf("active expiry alerts = %d, want 2", expiryAlerts)
	}
	operationLogs := 0
	for _, event := range sink.events {
		if event.message != "moderation operation completed" {
			continue
		}
		operationLogs++
		if len(event.attrs) != 7 || event.attrs["component"] != "moderation" ||
			event.attrs["result"] != "success" || event.attrs["occurred_at"] == "" ||
			event.attrs["actor_type"] == "" || event.attrs["request_id"] == "" || event.attrs["case_reference"] == "" {
			t.Fatalf("operation log attrs = %#v", event.attrs)
		}
	}
	if operationLogs != 8 {
		t.Fatalf("operation logs = %d, want 8", operationLogs)
	}
	observable := logOutput.String()
	for _, call := range recorder.Calls() {
		observable += fmt.Sprintf("%s%v", call.Name, call.Attributes)
	}
	for _, event := range sink.events {
		observable += event.message + fmt.Sprint(event.attrs)
	}
	for _, sensitive := range []string{ownerSentinel, evidenceSentinel, actorSentinel, sourceSentinel, replaySentinel, rationaleSentinel, "SENTINEL_RAW_SOURCE", "did:plc:"} {
		if strings.Contains(observable, sensitive) {
			t.Fatalf("production moderation telemetry leaked %q: %s", sensitive, observable)
		}
	}
}

func TestAdjudicationAuditAndObservationExcludeSensitiveValues(t *testing.T) {
	const (
		privateEvidence = "SENTINEL_PRIVATE_EVIDENCE"
		rawSourceDID    = "did:plc:SENTINEL_RAW_SOURCE"
	)
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	caseRow := seedAdjudicationCase(t, pool, store, "safe-observation", syntax.DID("did:plc:owner"))
	observer := &moderationOperationObserver{}
	service := NewService(store, testVisibilityWriter{}, time.Now, WithOperationObserver(observer))
	_, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "admin-api",
		ReplayID: "safe-request-0001", ActorID: "trusted-moderator",
		SourceDID: syntax.DID(rawSourceDID),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam,
			Evidence: privateEvidence, Consequences: []EffectType{EffectVisibilityHide}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(observer.calls) != 1 {
		t.Fatalf("operation observations = %d, want 1", len(observer.calls))
	}
	call := observer.calls[0]
	if call.operation != "decision" || call.result != "success" || call.actorType != "moderator" ||
		len(call.requestID) != 64 || call.caseReference != "MOD-"+caseRow.ID.String() || call.occurredAt.IsZero() {
		t.Fatalf("unsafe or incomplete operation observation: %+v", call)
	}
	_, err = service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 99, SourceSystem: "admin-api",
		ReplayID: "failed-request-0002", ActorID: "trusted-moderator",
		SourceDID: syntax.DID(rawSourceDID),
		Decision:  Decision{Disposition: DispositionNoAction},
	})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("failed command error = %v, want revision conflict", err)
	}
	if len(observer.calls) != 2 || observer.calls[1].result != "failure" || len(observer.calls[1].requestID) != 64 {
		t.Fatalf("failed operation observation = %+v", observer.calls)
	}

	var eventType, actorID, sourceSystem, replayID string
	var expectedRevision, resultRevision int64
	var createdAt time.Time
	if err := pool.QueryRow(ctx, `SELECT event_type,actor_id,source_system,replay_id,expected_revision,result_revision,created_at FROM moderation_case_events WHERE case_id=$1`, caseRow.ID).Scan(
		&eventType, &actorID, &sourceSystem, &replayID, &expectedRevision, &resultRevision, &createdAt,
	); err != nil {
		t.Fatal(err)
	}
	audit, err := json.Marshal(map[string]any{
		"caseReference": call.caseReference, "eventType": eventType,
		"actorType": call.actorType, "requestId": call.requestID,
		"expectedRevision": expectedRevision, "resultRevision": resultRevision,
		"createdAt": createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(audit)
	for _, sensitive := range []string{privateEvidence, rawSourceDID, actorID, sourceSystem, replayID, "did:plc:"} {
		if strings.Contains(serialized, sensitive) {
			t.Fatalf("safe audit representation leaked %q: %s", sensitive, serialized)
		}
	}
}

func TestAdjudicationServiceAtomicDecisionReplayAndVisibility(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	caseRow := seedAdjudicationCase(t, pool, store, "case-resolution", syntax.DID("did:plc:owner"))
	now := time.Date(2026, 9, 10, 13, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	command := TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool",
		ReplayID: "decision-replay-0001", ActorID: "moderator-one",
		SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam,
			Evidence: "private evidence", Consequences: []EffectType{EffectVisibilityHide, EffectStrike}},
	}

	first, err := service.ResolveCase(ctx, command)
	if err != nil {
		t.Fatalf("resolve case: %v", err)
	}
	replay, err := service.ResolveCase(ctx, command)
	if err != nil {
		t.Fatalf("replay case: %v", err)
	}
	if !replay.Replayed || replay.EventID != first.EventID || replay.Revision != 1 {
		t.Fatalf("replay = %+v, first = %+v", replay, first)
	}

	var events, decisions, effects, active, strikes, outputs, activeCount int
	var state string
	if err := pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM moderation_case_events),
		(SELECT count(*) FROM moderation_decisions),
		(SELECT count(*) FROM moderation_effect_events),
		(SELECT count(*) FROM moderation_active_case_effects),
		(SELECT count(*) FROM moderation_case_strikes),
		(SELECT count(*) FROM moderation_outputs),
		(SELECT active_strike_count FROM moderation_account_standings WHERE owner_did='did:plc:owner'),
		(SELECT state FROM moderation_cases WHERE id=$1)
	`, caseRow.ID).Scan(&events, &decisions, &effects, &active, &strikes, &outputs, &activeCount, &state); err != nil {
		t.Fatal(err)
	}
	if events != 1 || decisions != 1 || effects != 2 || active != 2 || strikes != 1 || outputs != 1 || activeCount != 1 || state != "resolved" {
		t.Fatalf("rows=%d/%d/%d/%d/%d/%d count=%d state=%s", events, decisions, effects, active, strikes, outputs, activeCount, state)
	}

	conflict := command
	conflict.Decision.UserSafeDetail = "different"
	if _, err := service.ResolveCase(ctx, conflict); !errors.Is(err, ErrReplayConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}
	stale := command
	stale.ReplayID = "decision-replay-0002"
	if _, err := service.ResolveCase(ctx, stale); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale revision error = %v", err)
	}
}

func TestAdjudicationServiceDoesNotInferStrikeAndRollsBack(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	firstCase := seedAdjudicationCase(t, pool, store, "no-implicit-strike", syntax.DID("did:plc:owner"))
	service := NewService(store, testVisibilityWriter{}, time.Now)
	_, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: firstCase.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "no-strike-0000001",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam,
			Evidence: "evidence", Consequences: []EffectType{EffectVisibilityWarn}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var strikes int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM moderation_case_strikes`).Scan(&strikes); err != nil || strikes != 0 {
		t.Fatalf("implicit strikes = %d, err=%v", strikes, err)
	}

	rollbackCase := seedAdjudicationCase(t, pool, store, "rollback", syntax.DID("did:plc:owner"))
	if _, err := pool.Exec(ctx, `CREATE FUNCTION reject_active_effect() RETURNS trigger LANGUAGE plpgsql AS $$
	BEGIN RAISE EXCEPTION 'injected active effect failure'; END $$;
	CREATE CONSTRAINT TRIGGER reject_active_effect AFTER INSERT ON moderation_active_case_effects
	DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION reject_active_effect();`); err != nil {
		t.Fatal(err)
	}
	_, err = service.ResolveCase(ctx, TrustedCommand{
		CaseID: rollbackCase.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "rollback-00000001",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam,
			Evidence: "evidence", Consequences: []EffectType{EffectVisibilityHide}},
	})
	if err == nil {
		t.Fatal("resolve succeeded with deferred failure")
	}
	var caseState string
	var revision int
	if err := pool.QueryRow(ctx, `SELECT state,revision FROM moderation_cases WHERE id=$1`, rollbackCase.ID).Scan(&caseState, &revision); err != nil {
		t.Fatal(err)
	}
	if caseState != "open" || revision != 0 {
		t.Fatalf("case after rollback = %s/%d", caseState, revision)
	}
	for _, table := range []string{"moderation_case_events", "moderation_decisions", "moderation_effect_events", "moderation_outputs"} {
		var count int
		if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if table == "moderation_case_events" && count != 1 || table != "moderation_case_events" && table != "moderation_outputs" && count != 1 || table == "moderation_outputs" && count != 1 {
			t.Fatalf("%s count after rollback = %d, want prior decision rows only", table, count)
		}
	}
}

func TestAdjudicationServiceRollsBackEveryCoupledWriteFailure(t *testing.T) {
	faults := []struct {
		name string
		sql  string
	}{
		{name: "case event", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_case_events FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "decision", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_decisions FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "visibility output", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_outputs FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "effect event", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_effect_events FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "active effect", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_active_case_effects FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "strike", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_case_strikes FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "standing projection", sql: `CREATE TRIGGER reject_write BEFORE UPDATE ON moderation_account_standings FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "case projection", sql: `CREATE TRIGGER reject_write BEFORE UPDATE ON moderation_cases FOR EACH ROW EXECUTE FUNCTION reject_write()`},
		{name: "notification intent", sql: `CREATE TRIGGER reject_write BEFORE INSERT ON moderation_notification_intents FOR EACH ROW EXECUTE FUNCTION reject_write()`},
	}
	for _, fault := range faults {
		t.Run(fault.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL)
			ctx := context.Background()
			store := NewStore(pool)
			caseRow := seedAdjudicationCase(t, pool, store, "fault-report", syntax.DID("did:plc:fault"))
			if _, err := pool.Exec(ctx, `CREATE FUNCTION reject_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected write failure'; END $$;`+fault.sql); err != nil {
				t.Fatal(err)
			}
			service := NewService(store, testVisibilityWriter{}, time.Now, WithNotificationWriter(testNotificationWriter{}))
			_, err := service.ResolveCase(ctx, TrustedCommand{
				CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool", ReplayID: "fault-replay-0001",
				ActorID: "moderator", SourceDID: syntax.DID("did:plc:labeler"),
				Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "evidence", Consequences: []EffectType{EffectVisibilityHide, EffectStrike}},
			})
			if err == nil {
				t.Fatal("decision succeeded with injected write failure")
			}
			var state string
			var revision int
			if err := pool.QueryRow(ctx, `SELECT state,revision FROM moderation_cases WHERE id=$1`, caseRow.ID).Scan(&state, &revision); err != nil {
				t.Fatal(err)
			}
			if state != "open" || revision != 0 {
				t.Fatalf("case projection after rollback = %s/%d", state, revision)
			}
			for _, table := range []string{"moderation_case_events", "moderation_decisions", "moderation_effect_events", "moderation_active_case_effects", "moderation_case_strikes", "moderation_outputs", "moderation_account_standings", "moderation_notification_intents"} {
				var count int
				if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != 0 {
					t.Fatalf("%s count after rollback = %d", table, count)
				}
			}
		})
	}
}

func TestAdjudicationServiceAppliesAndNegatesExistingVisibilityOutputs(t *testing.T) {
	for _, effect := range []EffectType{EffectVisibilityWarn, EffectVisibilityHide, EffectVisibilityTakedown} {
		t.Run(string(effect), func(t *testing.T) {
			pool := testdb.WithSchema(t, adjudicationTestDDL)
			ctx := context.Background()
			store := NewStore(pool)
			caseRow := seedAdjudicationCase(t, pool, store, "visibility-"+string(effect), syntax.DID("did:plc:visibility"))
			now := time.Date(2026, 9, 10, 13, 0, 0, 0, time.UTC)
			service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })

			resolved, err := service.ResolveCase(ctx, TrustedCommand{
				CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "retool",
				ReplayID: "visibility-apply-" + string(effect), ActorID: "moderator-one",
				SourceDID: syntax.DID("did:plc:labeler"),
				Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam,
					Evidence: "private evidence", Consequences: []EffectType{EffectFormalWarning, effect}},
			})
			if err != nil {
				t.Fatalf("resolve case: %v", err)
			}
			now = now.Add(time.Hour)
			if _, err := service.ChangeEffects(ctx, EffectChangeCommand{
				CaseID: caseRow.ID, ExpectedRevision: resolved.Revision, SourceSystem: "retool",
				ReplayID: "visibility-negate-" + string(effect), ActorID: "moderator-one",
				SourceDID: syntax.DID("did:plc:labeler"), Negate: []EffectType{effect},
				Rationale: "visibility consequence reversed",
			}); err != nil {
				t.Fatalf("negate visibility effect: %v", err)
			}

			rows, err := pool.Query(ctx, `SELECT value,action FROM moderation_outputs ORDER BY created_at,id`)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			var outputs [][2]string
			for rows.Next() {
				var output [2]string
				if err := rows.Scan(&output[0], &output[1]); err != nil {
					t.Fatal(err)
				}
				outputs = append(outputs, output)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			wantValue := visibilityValue(effect)
			if len(outputs) != 2 || outputs[0] != [2]string{wantValue, string(EffectApply)} || outputs[1] != [2]string{wantValue, string(EffectNegate)} {
				t.Fatalf("visibility outputs = %#v", outputs)
			}
			var formalWarnings, visibilityEffects int
			if err := pool.QueryRow(ctx, `SELECT
				count(*) FILTER (WHERE effect_type='formalWarning'),
				count(*) FILTER (WHERE effect_type=$2)
				FROM moderation_active_case_effects WHERE case_id=$1`, caseRow.ID, effect).Scan(&formalWarnings, &visibilityEffects); err != nil {
				t.Fatal(err)
			}
			if formalWarnings != 1 || visibilityEffects != 0 {
				t.Fatalf("active formal/visibility effects = %d/%d, want 1/0", formalWarnings, visibilityEffects)
			}
		})
	}
}

func seedAdjudicationCase(t *testing.T, pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, store *Store, reportID string, owner syntax.DID) Case {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	if _, err := tx.Exec(ctx, `INSERT INTO moderation_reports(id,subject_type,subject_did,created_at) VALUES($1,'account',$2,$3)`, reportID, owner, now); err != nil {
		t.Fatal(err)
	}
	caseRow, err := store.AttachAcceptedReportTx(ctx, tx, AcceptedReport{ID: reportID, SubjectType: SubjectAccount, SubjectDID: owner, CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return caseRow
}
