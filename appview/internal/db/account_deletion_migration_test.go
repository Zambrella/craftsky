package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const accountDeletionMigrationPreStateDDL = `
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_sessions (
    token_hash BYTEA PRIMARY KEY,
    account_did TEXT NOT NULL,
    oauth_session_id TEXT NOT NULL,
    device_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    FOREIGN KEY (account_did, oauth_session_id)
        REFERENCES oauth_sessions(account_did, session_id) ON DELETE CASCADE
);
CREATE TABLE migration_sentinel (id INTEGER PRIMARY KEY, value TEXT NOT NULL);
INSERT INTO migration_sentinel(id, value) VALUES (1, 'preserve-me');
`

func TestAccountDeletionMigrationUpDown(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../../migrations/000037_account_deletion.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000037_account_deletion.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	pool := testdb.WithSchema(t, accountDeletionMigrationPreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply account deletion migration: %v", err)
	}

	if !tableExists(t, pool, "account_deletion_operations") {
		t.Fatal("table account_deletion_operations missing")
	}
	for _, table := range []string{
		"account_deletion_status_credentials",
		"account_deletion_recovery_credentials",
		"account_deletion_expected_records",
		"account_deletion_index_receipts",
		"account_deletion_cleanup_steps",
		"account_deletion_cleanup_artifacts",
		"account_deletion_audits",
	} {
		if tableExists(t, pool, table) {
			t.Errorf("superseded table %s exists", table)
		}
	}
	for _, column := range []string{"purpose", "account_deletion_owner_did", "account_deletion_job_id"} {
		if !columnExists(t, pool, "oauth_auth_requests", column) {
			t.Errorf("oauth_auth_requests.%s missing", column)
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES('did:plc:alice','oauth-bound','{}'),('did:plc:alice','oauth-other','{}');
		INSERT INTO account_deletion_operations(
			id,owner_did,state,accepted_at,deletion_oauth_session_id
		) VALUES (
			'10000000-0000-0000-0000-000000000001','did:plc:alice','active',now(),'oauth-bound'
		);
	`); err != nil {
		t.Fatalf("seed bound deletion operation: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM oauth_sessions WHERE account_did='did:plc:alice' AND session_id='oauth-bound'`); err == nil {
		t.Fatal("bound OAuth session deletion succeeded while operation references it")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(id,owner_did,state)
		VALUES('10000000-0000-0000-0000-000000000002','did:plc:alice','intent')
	`); err == nil {
		t.Fatal("second active operation for the same owner succeeded")
	}

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply account deletion down migration: %v", err)
	}
	for _, table := range []string{
		"account_deletion_operations", "account_deletion_status_credentials", "account_deletion_recovery_credentials",
		"account_deletion_expected_records", "account_deletion_index_receipts", "account_deletion_cleanup_steps",
		"account_deletion_cleanup_artifacts", "account_deletion_audits",
	} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}
	for _, column := range []string{"purpose", "account_deletion_owner_did", "account_deletion_job_id"} {
		if columnExists(t, pool, "oauth_auth_requests", column) {
			t.Errorf("oauth_auth_requests.%s remained after down migration", column)
		}
	}
	assertTableCount(t, pool, "migration_sentinel", 1)
}

func tableColumns(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema=current_schema() AND table_name=$1
	`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, column)
	}
	return columns
}

func assertTableCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM `+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
}
