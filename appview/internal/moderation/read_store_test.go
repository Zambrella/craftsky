package moderation

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestModerationReadStoreSeparatesAdminEvidenceFromOwnerHistory(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL+`
		ALTER TABLE moderation_reports ADD COLUMN reporter_did TEXT NOT NULL DEFAULT 'did:plc:reporter-sensitive';
		ALTER TABLE moderation_reports ADD COLUMN reason_type TEXT NOT NULL DEFAULT 'spam';
		ALTER TABLE moderation_reports ADD COLUMN details TEXT;
	`)
	ctx := context.Background()
	store := NewStore(pool)
	owner := syntax.DID("did:plc:owner")
	caseRow := seedAdjudicationCase(t, pool, store, "read-report", owner)
	caseReference, err := FormatCaseReference(caseRow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE moderation_reports SET details='private-report-sentinel' WHERE id='read-report'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE moderation_cases SET safe_snapshot=safe_snapshot || '{"reportText":"raw-snapshot-private-sentinel"}'::jsonb WHERE id=$1`, caseRow.ID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 10, 13, 0, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	if _, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "admin-api", ReplayID: "read-store-replay",
		ActorID: "private-moderator-sentinel", SourceDID: syntax.DID("did:plc:moderation"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "private-evidence-sentinel", UserSafeDetail: "owner-safe-detail", Consequences: []EffectType{EffectStrike}},
	}); err != nil {
		t.Fatal(err)
	}

	queue, err := store.Queue(ctx, QueueFilter{State: "resolved", SubjectType: "account", Limit: 10})
	if err != nil || len(queue.Items) != 1 || queue.Items[0].ReportCount != 1 {
		t.Fatalf("queue = %+v, err=%v", queue, err)
	}
	detail, err := store.AdminDetail(ctx, caseReference)
	if err != nil || len(detail.Reports) != 1 || detail.Reports[0].Details != "private-report-sentinel" || len(detail.Events) != 1 {
		t.Fatalf("admin detail = %+v, err=%v", detail, err)
	}
	if detail.SubjectType != SubjectAccount || detail.SubjectDID != owner || detail.SafeSnapshot.DID != owner {
		t.Fatalf("canonical subject detail = %+v", detail)
	}
	if len(detail.Decisions) != 1 || detail.Decisions[0].Reason != ReasonSpam || detail.Decisions[0].EvidenceNotes != "private-evidence-sentinel" || detail.Decisions[0].UserSafeDetail != "owner-safe-detail" {
		t.Fatalf("separate admin decision fields = %+v", detail.Decisions)
	}
	adminWire, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(adminWire), "raw-snapshot-private-sentinel") {
		t.Fatalf("admin snapshot leaked uncurated persistence JSON: %s", adminWire)
	}
	if detail.StrikeContext == nil || detail.StrikeContext.LogicalEffectID == "" || !detail.StrikeContext.Active || !detail.StrikeContext.IssuedAt.Equal(now) || !detail.StrikeContext.DueAt.After(now) {
		t.Fatalf("strike context = %+v", detail.StrikeContext)
	}
	history, err := store.OwnerHistory(ctx, owner, "", 10)
	if err != nil || len(history.Items) != 1 || history.Items[0].Reason != ReasonSpam || history.Items[0].UserSafeDetail != "owner-safe-detail" {
		t.Fatalf("owner history = %+v, err=%v", history, err)
	}
	wire, err := json.Marshal(history)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-report-sentinel", "private-evidence-sentinel", "private-moderator-sentinel", "did:plc:reporter-sensitive", "raw-snapshot-private-sentinel"} {
		if strings.Contains(string(wire), secret) {
			t.Fatalf("owner history leaked %q: %s", secret, wire)
		}
	}
	standing, err := store.Standing(ctx, owner)
	if err != nil || standing.ActiveStrikeCount != 1 || standing.Suspended {
		t.Fatalf("standing = %+v, err=%v", standing, err)
	}
	if _, err := store.OwnerEntry(ctx, syntax.DID("did:plc:other"), caseReference); err != ErrCaseNotFound {
		t.Fatalf("cross-owner entry error = %v", err)
	}
}

func TestModerationQueueFiltersAndPaginatesWithoutGaps(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	base := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	type seededCase struct {
		id          uuid.UUID
		state       string
		subjectType SubjectType
		createdAt   time.Time
	}
	cases := []seededCase{
		{id: uuid.New(), state: "resolved", subjectType: SubjectAccount, createdAt: base.Add(time.Minute)},
		{id: uuid.New(), state: "open", subjectType: SubjectEvent, createdAt: base.Add(2 * time.Minute)},
		{id: uuid.New(), state: "resolved", subjectType: SubjectPost, createdAt: base.Add(3 * time.Minute)},
		{id: uuid.New(), state: "open", subjectType: SubjectAccount, createdAt: base.Add(4 * time.Minute)},
	}
	for index, item := range cases {
		resolvedAt := any(nil)
		if item.state == "resolved" {
			resolvedAt = item.createdAt
		}
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot,state,created_at,resolved_at,updated_at) VALUES($1,$2,$3,$4,$4,'{}',$5,$6,$7,$6)`, item.id, "subject-"+item.id.String(), item.subjectType, "did:plc:owner"+string(rune('a'+index)), item.state, item.createdAt, resolvedAt); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 2; index++ {
		reportID := "queue-report-" + string(rune('a'+index))
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_reports(id,subject_type,subject_did,created_at) VALUES($1,'account','did:plc:owner', $2)`, reportID, base); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO moderation_case_reports(case_id,report_id,attached_at) VALUES($1,$2,$3)`, cases[3].id, reportID, base); err != nil {
			t.Fatal(err)
		}
	}

	first, err := store.Queue(ctx, QueueFilter{Limit: 2})
	if err != nil || len(first.Items) != 2 || first.Cursor == "" {
		t.Fatalf("first page = %+v, err=%v", first, err)
	}
	if first.Items[0].SubjectType != SubjectAccount || first.Items[0].ReportCount != 2 || first.Items[1].SubjectType != SubjectPost {
		t.Fatalf("first page items = %+v", first.Items)
	}
	second, err := store.Queue(ctx, QueueFilter{Limit: 2, Cursor: first.Cursor})
	if err != nil || len(second.Items) != 2 || second.Cursor != "" {
		t.Fatalf("second page = %+v, err=%v", second, err)
	}
	seen := map[string]bool{}
	for _, item := range append(first.Items, second.Items...) {
		if seen[item.CaseReference] {
			t.Fatalf("duplicate case across pages: %s", item.CaseReference)
		}
		seen[item.CaseReference] = true
	}
	if len(seen) != len(cases) {
		t.Fatalf("paginated cases = %d, want %d", len(seen), len(cases))
	}

	open, err := store.Queue(ctx, QueueFilter{State: "open", Limit: 10})
	if err != nil || len(open.Items) != 2 {
		t.Fatalf("open filter = %+v, err=%v", open, err)
	}
	events, err := store.Queue(ctx, QueueFilter{SubjectType: "event", AppealStatus: "none", Limit: 10})
	if err != nil || len(events.Items) != 1 || events.Items[0].SubjectType != SubjectEvent {
		t.Fatalf("event/appeal filter = %+v, err=%v", events, err)
	}
}

func TestModerationQueueRejectsCursorWithUnknownFields(t *testing.T) {
	cursor := base64.RawURLEncoding.EncodeToString([]byte(`{"createdAt":"2026-09-10T12:00:00Z","id":"` + uuid.NewString() + `","caseId":"private"}`))
	_, err := (*Store)(nil).Queue(context.Background(), QueueFilter{Cursor: cursor, Limit: 10})
	if !errors.Is(err, ErrInvalidQueueCursor) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidQueueCursor)
	}
}

func TestOwnerHistoryRejectsMalformedOpaqueCursors(t *testing.T) {
	validID := uuid.NewString()
	for _, raw := range []string{
		`{"createdAt":"2026-09-10T12:00:00Z","id":"` + validID + `","private":"leak"}`,
		`{"createdAt":"2026-09-10T12:00:00Z","id":"` + validID + `"} {}`,
		`{"createdAt":"0001-01-01T00:00:00Z","id":"` + validID + `"}`,
	} {
		cursor := base64.RawURLEncoding.EncodeToString([]byte(raw))
		_, err := (*Store)(nil).OwnerHistory(context.Background(), syntax.DID("did:plc:owner"), cursor, 10)
		if !errors.Is(err, ErrInvalidQueueCursor) {
			t.Fatalf("cursor %q error = %v, want %v", raw, err, ErrInvalidQueueCursor)
		}
	}
}

func TestOwnerHistorySerializesImmutableEffectEventsAndOriginalStrikeDeadlines(t *testing.T) {
	pool := testdb.WithSchema(t, adjudicationTestDDL)
	ctx := context.Background()
	store := NewStore(pool)
	owner := syntax.DID("did:plc:owner")
	caseRow := seedAdjudicationCase(t, pool, store, "effect-history-report", owner)
	now := time.Date(2024, 2, 29, 8, 30, 0, 0, time.UTC)
	service := NewService(store, testVisibilityWriter{}, func() time.Time { return now })
	decision, err := service.ResolveCase(ctx, TrustedCommand{
		CaseID: caseRow.ID, ExpectedRevision: 0, SourceSystem: "admin-api", ReplayID: "effect-history-decision",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:moderation"),
		Decision: Decision{Disposition: DispositionViolation, Reason: ReasonSpam, Evidence: "private evidence", Consequences: []EffectType{EffectStrike}},
	})
	if err != nil {
		t.Fatal(err)
	}
	firstDue := StrikeDeadline(now)
	now = time.Date(2024, 3, 2, 9, 0, 0, 0, time.UTC)
	reversed, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: decision.Revision, SourceSystem: "admin-api", ReplayID: "effect-history-negate",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:moderation"), Negate: []EffectType{EffectStrike}, Rationale: "appeal changed outcome",
	})
	if err != nil {
		t.Fatal(err)
	}
	now = time.Date(2024, 4, 3, 10, 0, 0, 0, time.UTC)
	if _, err := service.ChangeEffects(ctx, EffectChangeCommand{
		CaseID: caseRow.ID, ExpectedRevision: reversed.Revision, SourceSystem: "admin-api", ReplayID: "effect-history-reapply",
		ActorID: "moderator", SourceDID: syntax.DID("did:plc:moderation"), Apply: []EffectType{EffectStrike}, Rationale: "new evidence",
	}); err != nil {
		t.Fatal(err)
	}

	history, err := store.OwnerHistory(ctx, owner, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Items) != 3 {
		t.Fatalf("history = %+v, want three immutable events", history.Items)
	}
	wantActions := []string{"apply", "negate", "apply"}
	wantTimes := []time.Time{now, time.Date(2024, 3, 2, 9, 0, 0, 0, time.UTC), time.Date(2024, 2, 29, 8, 30, 0, 0, time.UTC)}
	for index, item := range history.Items {
		if !item.OccurredAt.Equal(wantTimes[index]) || len(item.Effects) != 1 || item.Effects[0].Action != wantActions[index] {
			t.Fatalf("history[%d] = %+v", index, item)
		}
	}
	if history.Items[0].Effects[0].DueAt == nil || !history.Items[0].Effects[0].DueAt.Equal(StrikeDeadline(now)) {
		t.Fatalf("reapplication dueAt = %v", history.Items[0].Effects[0].DueAt)
	}
	if history.Items[1].Effects[0].DueAt != nil {
		t.Fatalf("negation dueAt = %v, want omitted", history.Items[1].Effects[0].DueAt)
	}
	if history.Items[2].Effects[0].DueAt == nil || !history.Items[2].Effects[0].DueAt.Equal(firstDue) {
		t.Fatalf("original issuance dueAt = %v, want %v", history.Items[2].Effects[0].DueAt, firstDue)
	}
	wire, err := json.Marshal(history)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(wire), `"active"`) {
		t.Fatalf("immutable history contains current projection state: %s", wire)
	}
}
