package safetyalerts

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/moderation"
	"social.craftsky/appview/internal/testdb"
)

type fakeSender struct {
	ids  []uuid.UUID
	fail bool
}

func (s *fakeSender) Send(_ context.Context, id uuid.UUID) error {
	s.ids = append(s.ids, id)
	if s.fail {
		return errors.New("PRIVATE-PROVIDER-CANARY moderation@craftsky.social credential-canary")
	}
	return nil
}

func TestReportAlertDurableRetriesGuestCoverageAndDeduplication(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	for _, did := range []string{"did:plc:reporter", "did:plc:subject"} {
		if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, did); err != nil {
			t.Fatal(err)
		}
	}
	store := api.NewReportStore(pool, moderation.NewStore(pool))
	add := func(reason string) {
		t.Helper()
		text := "PRIVATE-REPORT-CANARY"
		_, err := store.CreateReport(ctx, api.CreateReportInput{SubjectType: api.ReportSubjectAccount, SubjectDID: "did:plc:subject", ReasonType: reason, Details: &text, ForwardingStatus: "prepared_not_submitted", ForwardingPreparedAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now().UTC()
	sender := &fakeSender{fail: true}
	var logs bytes.Buffer
	observer := observability.New(observability.Config{Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
	w := NewWorker(pool, sender, observer)
	w.now = func() time.Time { return now }
	add("spam")
	if n, err := w.ProcessBatch(ctx); err != nil || n != 0 || len(sender.ids) != 0 {
		t.Fatalf("routine report alerted: %d %v", n, err)
	}
	// An urgent guest report on the existing routine case must alert before review.
	add("childAbuseMaterial")
	if n, err := w.ProcessBatch(ctx); err != nil || n != 1 {
		t.Fatalf("first attempt: %d %v", n, err)
	}
	if len(sender.ids) != 1 {
		t.Fatal("missing urgent guest alert")
	}
	var sent int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_email_deliveries WHERE sent_at IS NOT NULL`).Scan(&sent); err != nil || sent != 0 {
		t.Fatal("failure marked delivered")
	}
	diagnostic := logs.String()
	for _, positive := range []string{"safety_email.send", "provider", "retry"} {
		if !strings.Contains(diagnostic, positive) {
			t.Fatalf("missing diagnostic %s: %s", positive, diagnostic)
		}
	}
	for _, canary := range []string{"PRIVATE-PROVIDER-CANARY", "PRIVATE-REPORT-CANARY", "credential-canary", "moderation@craftsky.social", "did:plc:subject"} {
		if strings.Contains(diagnostic, canary) {
			t.Fatalf("private value in diagnostics: %s", canary)
		}
	}
	// Restart the worker and retain its durable retry schedule and delivery ID.
	sender.fail = false
	restarted := NewWorker(pool, sender, nil)
	restarted.now = func() time.Time { return now }
	if n, err := restarted.ProcessBatch(ctx); err != nil || n != 0 {
		t.Fatalf("ignored retry delay: %d %v", n, err)
	}
	now = now.Add(time.Minute)
	if n, err := restarted.ProcessBatch(ctx); err != nil || n != 1 {
		t.Fatalf("retry: %d %v", n, err)
	}
	if len(sender.ids) != 2 || sender.ids[0] != sender.ids[1] {
		t.Fatal("retry changed idempotency key")
	}
	// Repeated allegations on the same case do not create an email storm.
	add("immediateDanger")
	add("childAbuseMaterial")
	now = now.Add(2 * time.Minute)
	if n, err := restarted.ProcessBatch(ctx); err != nil || n != 0 || len(sender.ids) != 2 {
		t.Fatalf("duplicate case alert: %d %v", n, err)
	}
	// Exact production migration round-trip; no unrelated schema fixtures.
	if err := testdb.ApplyMigrations(ctx, pool, "000123_safety_email_alerts.down.sql", "000123_safety_email_alerts.up.sql"); err != nil {
		t.Fatal(err)
	}
}

func TestClaimLeaseExcludesOtherWorkerAndRecoversAfterCrash(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Now().UTC()
	_, err := pool.Exec(t.Context(), `INSERT INTO external_safety_intakes(id,reference,provider_message_ref,intake_kind,urgency,canonical_subject,owner_actor_id,attachment_status,received_at,created_at) VALUES($1,'reference','private-provider-ref','allegation','urgent','private-subject','operator','none',$2,$2)`, uuid.New(), now)
	if err != nil {
		t.Fatal(err)
	}
	sender := &fakeSender{}
	w := NewWorker(pool, sender, nil)
	w.now = func() time.Time { return now }
	id, _, _, err := w.claim(t.Context())
	if err != nil || id == uuid.Nil {
		t.Fatalf("claim: %v", err)
	}
	other := NewWorker(pool, sender, nil)
	other.now = func() time.Time { return now }
	if n, err := other.ProcessBatch(t.Context()); err != nil || n != 0 {
		t.Fatalf("stole active lease: %d %v", n, err)
	}
	now = now.Add(2 * time.Minute)
	if n, err := other.ProcessBatch(t.Context()); err != nil || n != 1 || sender.ids[0] != id {
		t.Fatalf("crash recovery: %d %v", n, err)
	}
}

func TestIncidentAndWorkflowDiscoveryAndDigestRateLimit(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	now := time.Now().UTC()
	resultID, incidentID := uuid.New(), uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state,created_at,updated_at) VALUES($1,'bafy-fixture','fixture','policy-1','corpus-1','pending',$2,$2)`, resultID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO safety_incidents(id,scan_result_id,provider_reference,integrity_metadata_reference,detected_at,updated_at) VALUES($1,$2,'PRIVATE-PROVIDER-REF','PRIVATE-EVIDENCE-REF',$3,$3)`, incidentID, resultID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO safety_workflows(id,workflow_class,state,urgency,responsible_actor,received_at,created_at,updated_at) VALUES($1,'credibleThreat','open','immediate','operator',$2,$2,$2)`, uuid.New(), now); err != nil {
		t.Fatal(err)
	}
	sender := &fakeSender{}
	w := NewWorker(pool, sender, nil)
	w.now = func() time.Time { return now }
	if n, err := w.ProcessBatch(ctx); err != nil || n != 1 {
		t.Fatalf("digest: %d %v", n, err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_email_sources`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("sources: %d %v", count, err)
	}
	// Newly received work is batched into the next digest, rather than dropped.
	intakeID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO external_safety_intakes(id,reference,provider_message_ref,intake_kind,urgency,canonical_subject,owner_actor_id,attachment_status,received_at,created_at) VALUES($1,'next','private-ref','allegation','urgent','private-subject','operator','none',$2,$2)`, intakeID, now); err != nil {
		t.Fatal(err)
	}
	if n, err := w.ProcessBatch(ctx); err != nil || n != 0 {
		t.Fatalf("rate limit: %d %v", n, err)
	}
	now = now.Add(time.Minute)
	if n, err := w.ProcessBatch(ctx); err != nil || n != 1 || len(sender.ids) != 2 {
		t.Fatalf("next digest: %d %v", n, err)
	}
	// Private-source removal also clears its deduplication reference on the next poll.
	if _, err := pool.Exec(ctx, `DELETE FROM external_safety_intakes WHERE id=$1`, intakeID); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if _, err := w.ProcessBatch(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_email_sources WHERE source_kind='intake'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphaned intake reference: %d %v", count, err)
	}
}

func TestAllUrgentReportReasonsPersistAndEnterDigest(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	now := time.Now().UTC()
	reasons := []string{"childSexualExploitation", "grooming", "childAbuseMaterial", "childSexualAbuse", "childGrooming", "childEndangerment", "intimateImageAbuse", "nonConsensualIntimateImage", "sexualExploitation", "immediateDanger", "credibleThreat", "terrorism", "violence", "selfHarm", "stalking", "doxxing"}
	store := api.NewReportStore(pool, moderation.NewStore(pool))
	for i, reason := range reasons {
		did := fmt.Sprintf("did:plc:urgent%d", i)
		if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, did); err != nil {
			t.Fatal(err)
		}
		_, err := store.CreateReport(ctx, api.CreateReportInput{SubjectType: api.ReportSubjectAccount, SubjectDID: did, ReasonType: reason, ForwardingStatus: "prepared_not_submitted", ForwardingPreparedAt: now})
		if err != nil {
			t.Fatalf("reason %s: %v", reason, err)
		}
	}
	sender := &fakeSender{}
	w := NewWorker(pool, sender, nil)
	if n, err := w.ProcessBatch(ctx); err != nil || n != 1 || len(sender.ids) != 1 {
		t.Fatalf("digest: %d %v", n, err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM safety_email_sources WHERE source_kind='case'`).Scan(&count); err != nil || count != len(reasons) {
		t.Fatalf("urgent coverage: %d/%d %v", count, len(reasons), err)
	}
}

func TestDeliveryEscalatesPersistentProviderFailuresAndKeepsRetrying(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Now().UTC()
	if _, err := pool.Exec(t.Context(), `INSERT INTO external_safety_intakes(id,reference,provider_message_ref,intake_kind,urgency,canonical_subject,owner_actor_id,attachment_status,received_at,created_at) VALUES($1,'reference','PRIVATE_PROVIDER_REF','allegation','urgent','PRIVATE_SUBJECT','operator','none',$2,$2)`, uuid.New(), now); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	observer := observability.New(observability.Config{Logger: slog.New(slog.NewJSONHandler(&logs, nil))})
	sender := &fakeSender{fail: true}
	worker := NewWorker(pool, sender, observer)
	worker.now = func() time.Time { return now }
	for attempt := 1; attempt <= 6; attempt++ {
		if count, err := worker.ProcessBatch(t.Context()); err != nil || count != 1 {
			t.Fatalf("attempt %d: count=%d err=%v", attempt, count, err)
		}
		now = now.Add(retryDelay(attempt))
	}
	if len(sender.ids) != 6 {
		t.Fatalf("provider attempts=%d", len(sender.ids))
	}
	for _, want := range []string{`"result":"retry"`, `"result":"exhausted"`, `"attempt":5`, `"attempt":6`, "*errors.errorString", sender.ids[0].String()} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("missing %s: %s", want, logs.String())
		}
	}
	for _, forbidden := range []string{"PRIVATE_PROVIDER_REF", "PRIVATE_SUBJECT", "PRIVATE-PROVIDER-CANARY", "moderation@craftsky.social", "credential-canary"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("private value leaked: %s", logs.String())
		}
	}
	for _, id := range sender.ids {
		if id != sender.ids[0] {
			t.Fatal("retry changed delivery identity")
		}
	}
}
