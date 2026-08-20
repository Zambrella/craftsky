// appview/internal/api/moderation_store_test.go
package api_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

const moderationOutboxTestPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE oauth_sessions (
    account_did TEXT NOT NULL,
    session_id TEXT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_did, session_id)
);
CREATE TABLE oauth_auth_requests (
    state TEXT PRIMARY KEY,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    handoff_mode TEXT NOT NULL DEFAULT 'deep_link',
    loopback_redirect_uri TEXT,
    purpose TEXT NOT NULL DEFAULT 'login',
    device_id TEXT,
    account_deletion_owner_did TEXT,
    account_deletion_job_id UUID
);
CREATE TABLE craftsky_sessions (
    token_hash BYTEA PRIMARY KEY,
    account_did TEXT NOT NULL,
    oauth_session_id TEXT NOT NULL,
    device_label TEXT,
    last_device_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    FOREIGN KEY (account_did, oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id) ON DELETE CASCADE
);
CREATE TABLE account_deletion_operations (
    id UUID PRIMARY KEY,
    owner_did TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL,
    accepted_at TIMESTAMPTZ,
    reauth_oauth_session_id TEXT,
    deletion_oauth_session_id TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ,
    error_category TEXT,
    intent_proof_hash BYTEA,
    confirmation_handle_hash BYTEA,
    intent_expires_at TIMESTAMPTZ,
    lease_owner TEXT,
    lease_token UUID,
    lease_expires_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (owner_did, deletion_oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id),
    FOREIGN KEY (owner_did, reauth_oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id)
);
CREATE TABLE instagram_reconciliation_jobs (id UUID PRIMARY KEY);
`

func moderationStoreDDL(t *testing.T) string {
	t.Helper()
	ddl := moderationFlowMigrationDDL(t) + moderationOutboxTestPreStateDDL
	for _, path := range []string{
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000044_moderation_restoration_outbox.up.sql",
	} {
		up, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read moderation dependency migration %s: %v", path, err)
		}
		ddl += string(up)
	}
	return ddl
}

func newModerationStore(
	t *testing.T,
	pool *pgxpool.Pool,
	now func() time.Time,
) (*api.ModerationStore, *ownerlifecycle.Store) {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, now)
	if err != nil {
		t.Fatal(err)
	}
	store, err := api.NewModerationStoreWithClock(pool, lifecycles, now)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []syntax.DID{"did:plc:labeler", "did:plc:ozone"} {
		if _, err := lifecycles.EnsureOnboardingOwner(context.Background(), source); err != nil {
			t.Fatal(err)
		}
	}
	return store, lifecycles
}

func TestModerationStore_TerminalSourceOrSubjectCannotRecreateOutput(t *testing.T) {
	for _, terminalRole := range []string{"source", "subject"} {
		t.Run(terminalRole, func(t *testing.T) {
			pool := testdb.WithSchema(t, moderationStoreDDL(t))
			store, lifecycles := newModerationStore(t, pool, time.Now)
			owner := syntax.DID("did:plc:target")
			if terminalRole == "source" {
				owner = syntax.DID("did:plc:labeler")
			}
			if _, err := lifecycles.Terminalize(context.Background(), ownerlifecycle.TerminalizeRequest{
				Owner: owner, Reason: "identityDeleted",
			}); err != nil {
				t.Fatal(err)
			}

			_, err := store.InsertOutput(context.Background(), "moderation-terminal-0001", api.ModerationOutputInput{
				SourceDID:       "did:plc:labeler",
				SourceAuthority: api.ModerationSourceTrustedExternal,
				SubjectType:     api.ModerationSubjectAccount,
				SubjectDID:      "did:plc:target",
				Value:           api.ModerationValueHide,
				Action:          api.ModerationActionNegate,
			})
			if !errors.Is(err, ownerlifecycle.ErrTerminalOwner) {
				t.Fatalf("InsertOutput error = %v, want terminal owner", err)
			}
			var outputs, intents, receipts int
			if err := pool.QueryRow(context.Background(), `
				SELECT
					(SELECT count(*)::int FROM moderation_outputs),
					(SELECT count(*)::int FROM moderation_restoration_outbox),
					(SELECT count(*)::int FROM moderation_idempotency_receipts)
			`).Scan(&outputs, &intents, &receipts); err != nil {
				t.Fatal(err)
			}
			if outputs != 0 || intents != 0 || receipts != 0 {
				t.Fatalf("terminal write left rows %d/%d/%d", outputs, intents, receipts)
			}
		})
	}
}

func TestModerationStore_MissingSourceRequiresExplicitTrustedAuthority(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	store, _ := newModerationStore(t, pool, time.Now)
	input := api.ModerationOutputInput{
		SourceDID:   "did:plc:external-labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:target",
		Value:       api.ModerationValueWarn,
		Action:      api.ModerationActionApply,
	}

	if _, err := store.InsertOutput(context.Background(), "moderation-source-0001", input); !errors.Is(err, api.ErrModerationSourceLifecycleRequired) {
		t.Fatalf("missing source lifecycle error = %v", err)
	}
	input.SourceAuthority = api.ModerationSourceTrustedExternal
	if _, err := store.InsertOutput(context.Background(), "moderation-source-0002", input); err != nil {
		t.Fatalf("explicit trusted external source: %v", err)
	}
}

func TestModerationStore_QualifyingOutputCommitsWithOutboxAndReceipt(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	store, _ := newModerationStore(t, pool, time.Now)
	ctx := context.Background()
	createdAt := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	result, err := store.InsertOutput(ctx, "moderation-request-0001", api.ModerationOutputInput{
		SourceDID:   "did:plc:labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:target",
		Value:       api.ModerationValueHide,
		Action:      api.ModerationActionNegate,
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("InsertOutput: %v", err)
	}
	if result.OutputID == "" || result.Status != "indexed" || result.Replayed {
		t.Fatalf("result = %+v", result)
	}

	var outputs, intents, receipts int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::int FROM moderation_outputs),
			(SELECT count(*)::int FROM moderation_restoration_outbox),
			(SELECT count(*)::int FROM moderation_idempotency_receipts)
	`).Scan(&outputs, &intents, &receipts); err != nil {
		t.Fatalf("count transaction rows: %v", err)
	}
	if outputs != 1 || intents != 1 || receipts != 1 {
		t.Fatalf("row counts = outputs %d intents %d receipts %d, want 1/1/1", outputs, intents, receipts)
	}

	var target, status, receiptOutput, receiptStatus string
	if err := pool.QueryRow(ctx, `
		SELECT outbox.target_did,outbox.status,receipt.output_id,receipt.output_status
		FROM moderation_restoration_outbox AS outbox
		JOIN moderation_idempotency_receipts AS receipt
		  ON receipt.output_id=outbox.moderation_output_id
		WHERE outbox.moderation_output_id=$1
	`, result.OutputID).Scan(&target, &status, &receiptOutput, &receiptStatus); err != nil {
		t.Fatalf("read transaction rows: %v", err)
	}
	if target != "did:plc:target" || status != "pending" ||
		receiptOutput != result.OutputID || receiptStatus != "indexed" {
		t.Fatalf("outbox/receipt = target %q status %q output %q response %q", target, status, receiptOutput, receiptStatus)
	}
}

func TestModerationStore_IdempotencyReplayAndConflict(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	store, _ := newModerationStore(t, pool, time.Now)
	ctx := context.Background()
	key := "moderation-request-replay-0001"
	input := api.ModerationOutputInput{
		SourceDID:   "did:plc:labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:target",
		Value:       api.ModerationValueHide,
		Action:      api.ModerationActionNegate,
	}

	first, err := store.InsertOutput(ctx, key, input)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := store.InsertOutput(ctx, key, input)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.OutputID != first.OutputID || replay.Status != first.Status {
		t.Fatalf("replay = %+v, first = %+v", replay, first)
	}

	conflicting := input
	conflicting.Value = api.ModerationValueTakedown
	if _, err := store.InsertOutput(ctx, key, conflicting); !errors.Is(err, api.ErrModerationIdempotencyKeyConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}

	var outputs, intents, receipts int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::int FROM moderation_outputs),
			(SELECT count(*)::int FROM moderation_restoration_outbox),
			(SELECT count(*)::int FROM moderation_idempotency_receipts)
	`).Scan(&outputs, &intents, &receipts); err != nil {
		t.Fatal(err)
	}
	if outputs != 1 || intents != 1 || receipts != 1 {
		t.Fatalf("rows after replay/conflict = %d/%d/%d, want 1/1/1", outputs, intents, receipts)
	}
}

func TestModerationStore_WriteFaultRollsBackOutputIntentAndReceipt(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		triggerSQL string
	}{
		{
			name: "output insert",
			triggerSQL: `
				CREATE TRIGGER reject_moderation_write
				BEFORE INSERT ON moderation_outputs
				FOR EACH ROW EXECUTE FUNCTION reject_moderation_write();
			`,
		},
		{
			name: "outbox insert",
			triggerSQL: `
				CREATE TRIGGER reject_moderation_write
				BEFORE INSERT ON moderation_restoration_outbox
				FOR EACH ROW EXECUTE FUNCTION reject_moderation_write();
			`,
		},
		{
			name: "receipt insert",
			triggerSQL: `
				CREATE TRIGGER reject_moderation_write
				BEFORE INSERT ON moderation_idempotency_receipts
				FOR EACH ROW EXECUTE FUNCTION reject_moderation_write();
			`,
		},
		{
			name: "deferred commit",
			triggerSQL: `
				CREATE CONSTRAINT TRIGGER reject_moderation_write
				AFTER INSERT ON moderation_idempotency_receipts
				DEFERRABLE INITIALLY DEFERRED
				FOR EACH ROW EXECUTE FUNCTION reject_moderation_write();
			`,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			pool := testdb.WithSchema(t, moderationStoreDDL(t))
			ctx := context.Background()
			if _, err := pool.Exec(ctx, `
				CREATE FUNCTION reject_moderation_write()
				RETURNS trigger LANGUAGE plpgsql AS $$
				BEGIN
					RAISE EXCEPTION 'injected moderation write failure';
				END;
				$$;
			`+testCase.triggerSQL); err != nil {
				t.Fatalf("install fault: %v", err)
			}
			store, _ := newModerationStore(t, pool, time.Now)
			_, err := store.InsertOutput(ctx, "moderation-fault-0001", api.ModerationOutputInput{
				SourceDID:   "did:plc:labeler",
				SubjectType: api.ModerationSubjectAccount,
				SubjectDID:  "did:plc:target",
				Value:       api.ModerationValueHide,
				Action:      api.ModerationActionNegate,
			})
			if err == nil {
				t.Fatal("InsertOutput succeeded with injected fault")
			}

			var outputs, intents, receipts int
			if err := pool.QueryRow(ctx, `
				SELECT
					(SELECT count(*)::int FROM moderation_outputs),
					(SELECT count(*)::int FROM moderation_restoration_outbox),
					(SELECT count(*)::int FROM moderation_idempotency_receipts)
			`).Scan(&outputs, &intents, &receipts); err != nil {
				t.Fatal(err)
			}
			if outputs != 0 || intents != 0 || receipts != 0 {
				t.Fatalf("partial write after fault = %d/%d/%d", outputs, intents, receipts)
			}
		})
	}
}

func TestModerationStore_IdempotencyReceiptSurvivesArchivalUntilExpiry(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	store, _ := newModerationStore(t, pool, func() time.Time { return now })
	key := "moderation-retention-0001"
	input := api.ModerationOutputInput{
		SourceDID:         "did:plc:labeler",
		SubjectType:       api.ModerationSubjectPost,
		SubjectDID:        "did:plc:target",
		SubjectCollection: ptrString("social.craftsky.feed.post"),
		SubjectRkey:       ptrString("3lf2abc"),
		SubjectURI:        ptrString("at://did:plc:target/social.craftsky.feed.post/3lf2abc"),
		Value:             api.ModerationValueWarn,
		Action:            api.ModerationActionApply,
	}

	first, err := store.InsertOutput(ctx, key, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM moderation_outputs WHERE id=$1`, first.OutputID); err != nil {
		t.Fatalf("archive moderation output: %v", err)
	}

	now = now.Add(23 * time.Hour)
	replay, err := store.InsertOutput(ctx, key, input)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.OutputID != first.OutputID || replay.Row != nil {
		t.Fatalf("archived replay = %+v, first = %+v", replay, first)
	}

	now = now.Add(2 * time.Hour)
	afterExpiry, err := store.InsertOutput(ctx, key, input)
	if err != nil {
		t.Fatal(err)
	}
	if afterExpiry.Replayed || afterExpiry.OutputID == first.OutputID || afterExpiry.Row == nil {
		t.Fatalf("post-expiry result = %+v, first = %+v", afterExpiry, first)
	}
	var outputs, receipts int
	if err := pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*)::int FROM moderation_outputs),
			(SELECT count(*)::int FROM moderation_idempotency_receipts)
	`).Scan(&outputs, &receipts); err != nil {
		t.Fatal(err)
	}
	if outputs != 1 || receipts != 1 {
		t.Fatalf("post-expiry row counts = outputs %d receipts %d, want 1/1", outputs, receipts)
	}
}

func TestModerationStore_SweepsExpiredReceiptsInBoundedBatches(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	store, _ := newModerationStore(t, pool, func() time.Time { return now })
	for index, key := range []string{"moderation-sweep-0001", "moderation-sweep-0002"} {
		if _, err := store.InsertOutput(ctx, key, api.ModerationOutputInput{
			SourceDID:         "did:plc:labeler",
			SubjectType:       api.ModerationSubjectPost,
			SubjectDID:        "did:plc:target",
			SubjectCollection: ptrString("social.craftsky.feed.post"),
			SubjectRkey:       ptrString("3lf2abc"),
			SubjectURI:        ptrString("at://did:plc:target/social.craftsky.feed.post/3lf2abc"),
			Value:             api.ModerationValueWarn,
			Action:            api.ModerationActionApply,
			InternalReason:    ptrString(string(rune('a' + index))),
		}); err != nil {
			t.Fatal(err)
		}
	}
	now = now.Add(25 * time.Hour)

	for call, want := range []int{1, 1, 0} {
		removed, err := store.SweepExpiredIdempotencyReceipts(ctx, 1)
		if err != nil {
			t.Fatalf("sweep %d: %v", call, err)
		}
		if removed != want {
			t.Fatalf("sweep %d removed %d, want %d", call, removed, want)
		}
	}
	var receipts int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM moderation_idempotency_receipts`).Scan(&receipts); err != nil {
		t.Fatal(err)
	}
	if receipts != 0 {
		t.Fatalf("receipts after sweep = %d", receipts)
	}
}

func TestModerationStore_InsertOutput_PersistsPostAndAccountOutputs(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	store, _ := newModerationStore(t, pool, time.Now)
	ctx := context.Background()
	expiresAt := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	internalReason := "private moderator reason"

	postResult, err := store.InsertOutput(ctx, "moderation-request-0002", api.ModerationOutputInput{
		SourceDID:         "did:plc:labeler",
		SubjectType:       api.ModerationSubjectPost,
		SubjectDID:        "did:plc:bob",
		SubjectCollection: ptrString("social.craftsky.feed.post"),
		SubjectRkey:       ptrString("3lf2abc"),
		SubjectURI:        ptrString("at://did:plc:bob/social.craftsky.feed.post/3lf2abc"),
		Value:             api.ModerationValueHide,
		Action:            api.ModerationActionApply,
		InternalReason:    &internalReason,
		ExpiresAt:         &expiresAt,
		CreatedAt:         createdAt,
	})
	if err != nil {
		t.Fatalf("InsertOutput post: %v", err)
	}
	postRow := postResult.Row
	if postRow.ID == "" {
		t.Fatal("post output ID is empty")
	}
	assertModerationOutputRow(t, postRow, api.ModerationOutputRow{
		SourceDID:         "did:plc:labeler",
		SubjectType:       api.ModerationSubjectPost,
		SubjectDID:        "did:plc:bob",
		SubjectCollection: ptrString("social.craftsky.feed.post"),
		SubjectRkey:       ptrString("3lf2abc"),
		SubjectURI:        ptrString("at://did:plc:bob/social.craftsky.feed.post/3lf2abc"),
		Value:             api.ModerationValueHide,
		Action:            api.ModerationActionApply,
		InternalReason:    &internalReason,
		ExpiresAt:         &expiresAt,
		CreatedAt:         createdAt,
	})

	accountResult, err := store.InsertOutput(ctx, "moderation-request-0003", api.ModerationOutputInput{
		SourceDID:   "did:plc:labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:bob",
		Value:       api.ModerationValueWarn,
		Action:      api.ModerationActionNegate,
		CreatedAt:   createdAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("InsertOutput account: %v", err)
	}
	accountRow := accountResult.Row
	if accountRow.ID == "" || accountRow.ID == postRow.ID {
		t.Fatalf("account output ID = %q, post ID = %q", accountRow.ID, postRow.ID)
	}
	assertModerationOutputRow(t, accountRow, api.ModerationOutputRow{
		SourceDID:   "did:plc:labeler",
		SubjectType: api.ModerationSubjectAccount,
		SubjectDID:  "did:plc:bob",
		Value:       api.ModerationValueWarn,
		Action:      api.ModerationActionNegate,
		CreatedAt:   createdAt.Add(time.Minute),
	})

	var storedCount int
	if err := pool.QueryRow(ctx, `SELECT count(*)::int FROM moderation_outputs`).Scan(&storedCount); err != nil {
		t.Fatalf("count moderation outputs: %v", err)
	}
	if storedCount != 2 {
		t.Fatalf("stored outputs = %d, want 2", storedCount)
	}
}

func TestModerationStore_ActivePolicyForSubject_HandlesNegateAndExpiry(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	store, _ := newModerationStore(t, pool, time.Now)
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	postURI := "at://did:plc:bob/social.craftsky.feed.post/3lf2abc"

	_, _ = store.InsertOutput(ctx, "moderation-request-0004", api.ModerationOutputInput{SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueHide, Action: api.ModerationActionApply, CreatedAt: now.Add(-4 * time.Minute)})
	_, _ = store.InsertOutput(ctx, "moderation-request-0005", api.ModerationOutputInput{SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueHide, Action: api.ModerationActionNegate, CreatedAt: now.Add(-3 * time.Minute)})
	_, _ = store.InsertOutput(ctx, "moderation-request-0006", api.ModerationOutputInput{SourceDID: "did:plc:ozone", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueWarn, Action: api.ModerationActionApply, CreatedAt: now.Add(-2 * time.Minute)})
	_, _ = store.InsertOutput(ctx, "moderation-request-0007", api.ModerationOutputInput{SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectPost, SubjectDID: "did:plc:bob", SubjectCollection: ptrString("social.craftsky.feed.post"), SubjectRkey: ptrString("3lf2abc"), SubjectURI: &postURI, Value: api.ModerationValueTakedown, Action: api.ModerationActionApply, ExpiresAt: &past, CreatedAt: now.Add(-time.Minute)})

	policy, err := store.ActivePolicyForSubject(ctx, api.ModerationSubjectRef{Type: api.ModerationSubjectPost, DID: "did:plc:bob", URI: &postURI}, now)
	if err != nil {
		t.Fatalf("ActivePolicyForSubject: %v", err)
	}
	if policy.Hidden || !policy.Warning || policy.Value != api.ModerationValueWarn {
		t.Fatalf("policy = %+v, want visible warning", policy)
	}
}

func TestModerationStore_OnlyAccountSafetyNegateCreatesRestorationIntent(t *testing.T) {
	pool := testdb.WithSchema(t, moderationStoreDDL(t))
	store, _ := newModerationStore(t, pool, time.Now)
	ctx := context.Background()

	for index, input := range []api.ModerationOutputInput{
		{
			SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectAccount,
			SubjectDID: "did:plc:target", Value: api.ModerationValueHide,
			Action: api.ModerationActionApply,
		},
		{
			SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectAccount,
			SubjectDID: "did:plc:target", Value: api.ModerationValueHide,
			Action: api.ModerationActionNegate,
		},
		{
			SourceDID: "did:plc:labeler", SubjectType: api.ModerationSubjectAccount,
			SubjectDID: "did:plc:target", Value: api.ModerationValueWarn,
			Action: api.ModerationActionNegate,
		},
	} {
		if _, err := store.InsertOutput(ctx, "moderation-request-000"+string(rune('8'+index)), input); err != nil {
			t.Fatal(err)
		}
	}

	var intents int
	var target string
	if err := pool.QueryRow(ctx, `SELECT count(*)::int,max(target_did) FROM moderation_restoration_outbox`).Scan(&intents, &target); err != nil {
		t.Fatal(err)
	}
	if intents != 1 || target != syntax.DID("did:plc:target").String() {
		t.Fatalf("restoration intents=%d target=%q, want one account target", intents, target)
	}
}

func assertModerationOutputRow(t *testing.T, got *api.ModerationOutputRow, want api.ModerationOutputRow) {
	t.Helper()
	if got.SourceDID != want.SourceDID || got.SubjectType != want.SubjectType || got.SubjectDID != want.SubjectDID || got.Value != want.Value || got.Action != want.Action {
		t.Fatalf("row = %+v, want %+v", got, want)
	}
	assertStringPtr(t, "SubjectCollection", got.SubjectCollection, want.SubjectCollection)
	assertStringPtr(t, "SubjectRkey", got.SubjectRkey, want.SubjectRkey)
	assertStringPtr(t, "SubjectURI", got.SubjectURI, want.SubjectURI)
	assertStringPtr(t, "InternalReason", got.InternalReason, want.InternalReason)
	if got.ExpiresAt == nil || want.ExpiresAt == nil {
		if got.ExpiresAt != want.ExpiresAt {
			t.Fatalf("ExpiresAt = %v, want %v", got.ExpiresAt, want.ExpiresAt)
		}
	} else if !got.ExpiresAt.Equal(*want.ExpiresAt) {
		t.Fatalf("ExpiresAt = %s, want %s", *got.ExpiresAt, *want.ExpiresAt)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf("CreatedAt = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if got.IndexedAt.IsZero() {
		t.Fatal("IndexedAt is zero")
	}
}
