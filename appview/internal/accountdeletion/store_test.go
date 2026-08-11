package accountdeletion

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const accountDeletionStorePreStateDDL = `
CREATE TABLE oauth_sessions (
    account_did TEXT NOT NULL,
    session_id TEXT NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(account_did,session_id)
);
CREATE TABLE oauth_auth_requests (
    state TEXT PRIMARY KEY,
    data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_sessions (
    token_hash BYTEA PRIMARY KEY,
    account_did TEXT NOT NULL,
    oauth_session_id TEXT NOT NULL,
    last_device_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    FOREIGN KEY(account_did,oauth_session_id)
        REFERENCES oauth_sessions(account_did,session_id) ON DELETE CASCADE
);
CREATE TABLE push_installations (
    id UUID PRIMARY KEY,
    device_id TEXT NOT NULL UNIQUE
);
CREATE TABLE push_account_subscriptions (
    id UUID PRIMARY KEY,
    installation_id UUID NOT NULL REFERENCES push_installations(id) ON DELETE CASCADE,
    account_did TEXT NOT NULL,
    UNIQUE(installation_id,account_did)
);
`

func TestStoreDurableAcceptanceAndTerminalMinimization(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../../migrations/000037_account_deletion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, accountDeletionStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply account deletion migration: %v", err)
	}
	owner := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	jobID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	now := time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data) VALUES
		($1,'alice-fresh','{}'),($1,'alice-old','{}'),($2,'bob-session','{}')
	`, owner, bob); err != nil {
		t.Fatalf("seed OAuth sessions: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(token_hash,account_did,oauth_session_id,last_device_id) VALUES
		(decode('a1','hex'),$1,'alice-old','alice-phone'),
		(decode('a2','hex'),$1,'alice-old','alice-tablet'),
		(decode('b1','hex'),$2,'bob-session','bob-phone')
	`, owner, bob); err != nil {
		t.Fatalf("seed CraftSky sessions: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_installations(id,device_id) VALUES
		('20000000-0000-0000-0000-000000000001','alice-phone'),
		('20000000-0000-0000-0000-000000000002','bob-phone')
	`); err != nil {
		t.Fatalf("seed push installations: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO push_account_subscriptions(id,installation_id,account_did) VALUES
		('30000000-0000-0000-0000-000000000001','20000000-0000-0000-0000-000000000001',$1),
		('30000000-0000-0000-0000-000000000002','20000000-0000-0000-0000-000000000002',$2)
	`, owner, bob); err != nil {
		t.Fatalf("seed push subscriptions: %v", err)
	}

	store := NewStore(pool, func() time.Time { return now })
	metrics := &recordingDeletionMetrics{}
	store.SetTelemetry(NewDeletionTelemetry(nil, metrics))
	if err := store.CreateIntent(ctx, IntentRecord{
		JobID:                  jobID,
		Owner:                  owner,
		DeviceID:               "alice-phone",
		StatusCapabilityHash:   HashSecret("status-token"),
		ConfirmationHandleHash: HashSecret("alice.test"),
		ExpiresAt:              now.Add(10 * time.Minute),
	}); err != nil {
		t.Fatalf("create intent: %v", err)
	}
	if err := store.CompleteReauthentication(ctx, jobID, owner, "alice-fresh", HashSecret("proof-secret")); err != nil {
		t.Fatalf("complete reauthentication: %v", err)
	}

	accepted, err := store.Accept(ctx, AcceptanceRequest{
		JobID:              jobID,
		Owner:              owner,
		StatusCapability:   "status-token",
		ReauthProof:        "proof-secret",
		ConfirmationHandle: "alice.test",
	})
	if err != nil {
		t.Fatalf("accept deletion: %v", err)
	}
	if accepted.Status != StatusActive || accepted.Phase != PhaseQueued || accepted.DeletionOAuthSessionID != "alice-fresh" {
		t.Fatalf("accepted operation = %+v", accepted)
	}

	restartedStore := NewStore(pool, func() time.Time { return now.Add(time.Minute) })
	restartedStore.SetTelemetry(NewDeletionTelemetry(nil, metrics))
	loaded, err := restartedStore.GetOperation(ctx, jobID, owner)
	if err != nil || loaded.DeletionOAuthSessionID != "alice-fresh" {
		t.Fatalf("restart load = (%+v, %v)", loaded, err)
	}
	duplicate, err := restartedStore.Accept(ctx, AcceptanceRequest{
		JobID: jobID, Owner: owner, StatusCapability: "status-token",
		ReauthProof: "proof-secret", ConfirmationHandle: "alice.test",
	})
	if err != nil || duplicate != loaded {
		t.Fatalf("duplicate acceptance = (%+v, %v), want same operation", duplicate, err)
	}

	assertOwnerRowCount(t, pool, "craftsky_sessions", "account_did", owner, 0)
	assertOwnerRowCount(t, pool, "craftsky_sessions", "account_did", bob, 1)
	assertOwnerRowCount(t, pool, "push_account_subscriptions", "account_did", owner, 0)
	assertOwnerRowCount(t, pool, "push_account_subscriptions", "account_did", bob, 1)
	assertOwnerRowCount(t, pool, "push_installations", "device_id", "alice-phone", 0)
	assertOwnerRowCount(t, pool, "push_installations", "device_id", "bob-phone", 1)
	assertOwnerRowCount(t, pool, "oauth_sessions", "account_did", owner, 1)
	assertOwnerRowCount(t, pool, "oauth_sessions", "account_did", bob, 1)
	assertOwnerRowCount(t, pool, "account_deletion_recovery_credentials", "owner_did", owner, 2)

	terminalAt := now.Add(2 * time.Hour)
	if err := restartedStore.FinalizeSuccess(ctx, jobID, owner, terminalAt); err != nil {
		t.Fatalf("finalize success: %v", err)
	}
	assertOwnerRowCount(t, pool, "account_deletion_operations", "owner_did", owner, 0)
	assertOwnerRowCount(t, pool, "oauth_sessions", "account_did", owner, 0)
	for _, table := range []string{
		"account_deletion_status_credentials", "account_deletion_recovery_credentials",
		"account_deletion_expected_records", "account_deletion_index_receipts",
		"account_deletion_cleanup_steps", "account_deletion_cleanup_artifacts",
	} {
		assertJobRowCount(t, pool, table, jobID, 0)
	}
	events := deletionEventNames(metrics.events)
	if countDeletionEvent(events, "accepted") != 1 || countDeletionEvent(events, "terminalSuccess") != 1 {
		t.Fatalf("production acceptance/terminal telemetry=%v", events)
	}

	var audit DeletionAudit
	if err := pool.QueryRow(ctx, `
		SELECT job_id,did,accepted_at,terminal_at,outcome,expires_at
		FROM account_deletion_audits WHERE job_id=$1
	`, jobID).Scan(&audit.JobID, &audit.DID, &audit.AcceptedAt, &audit.TerminalAt, &audit.Outcome, &audit.ExpiresAt); err != nil {
		t.Fatalf("read terminal audit: %v", err)
	}
	if audit.DID != owner || audit.Outcome != AuditOutcomeDeleted || !audit.ExpiresAt.Equal(terminalAt.Add(30*24*time.Hour)) {
		t.Fatalf("terminal audit = %+v", audit)
	}
}

func assertOwnerRowCount(t *testing.T, pool *pgxpool.Pool, table, column string, owner syntax.DID, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table+` WHERE `+column+`=$1`, owner).Scan(&got); err != nil {
		t.Fatalf("count %s for owner: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count for %s = %d, want %d", table, owner, got, want)
	}
}

func assertJobRowCount(t *testing.T, pool *pgxpool.Pool, table string, jobID uuid.UUID, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table+` WHERE job_id=$1`, jobID).Scan(&got); err != nil {
		t.Fatalf("count %s for job: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count for job = %d, want %d", table, got, want)
	}
}
