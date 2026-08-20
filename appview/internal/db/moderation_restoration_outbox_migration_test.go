package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const moderationRestorationOutboxPreStateDDL = `
CREATE TABLE moderation_outputs (
    id TEXT PRIMARY KEY,
    source_did TEXT NOT NULL,
    subject_did TEXT NOT NULL
);
CREATE TABLE instagram_reconciliation_jobs (
    id UUID PRIMARY KEY
);
INSERT INTO moderation_outputs(id, source_did, subject_did)
VALUES ('legacy-output', 'did:plc:source', 'did:plc:target');
`

func TestModerationRestorationOutboxMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000044_moderation_restoration_outbox.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000044_moderation_restoration_outbox.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, moderationRestorationOutboxPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertModerationRestorationOutboxSchema(t, pool)
	assertTableCount(t, pool, "moderation_outputs", 0)
	assertModerationRestorationOutboxConstraints(t, pool)

	apply("down", down)
	for _, table := range []string{
		"moderation_restoration_outbox",
		"moderation_restoration_history",
		"moderation_idempotency_receipts",
	} {
		if tableExists(t, pool, table) {
			t.Errorf("table %s remained after down migration", table)
		}
	}

	apply("second up", up)
	assertModerationRestorationOutboxSchema(t, pool)
}

func assertModerationRestorationOutboxSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, table := range []string{
		"moderation_restoration_outbox",
		"moderation_restoration_history",
		"moderation_idempotency_receipts",
	} {
		if !tableExists(t, pool, table) {
			t.Errorf("table %s missing", table)
		}
	}
	for _, indexName := range []string{
		"moderation_restoration_outbox_pending_idx",
		"moderation_restoration_outbox_retention_idx",
		"moderation_restoration_outbox_reconciliation_job_id_idx",
		"moderation_restoration_history_retention_idx",
		"moderation_idempotency_receipts_expires_at_idx",
		"moderation_idempotency_receipts_output_id_idx",
	} {
		if !indexExists(t, pool, indexName) {
			t.Errorf("index %s missing", indexName)
		}
	}
	var (
		isUnique  bool
		column    string
		predicate *string
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT index.indisunique,
		       attribute.attname,
		       pg_get_expr(index.indpred, index.indrelid)
		FROM pg_index AS index
		JOIN pg_class AS index_class ON index_class.oid = index.indexrelid
		JOIN pg_attribute AS attribute
		  ON attribute.attrelid = index.indrelid
		 AND attribute.attnum = index.indkey[0]
		WHERE index_class.oid = to_regclass(
			'moderation_restoration_outbox_reconciliation_job_id_idx'
		)
	`).Scan(&isUnique, &column, &predicate); err != nil {
		t.Fatalf("inspect reconciliation-job index predicate: %v", err)
	}
	if isUnique {
		t.Error("reconciliation-job support index unexpectedly unique")
	}
	if column != "reconciliation_job_id" {
		t.Errorf("reconciliation-job support index leading column = %q", column)
	}
	if predicate == nil || *predicate != "(reconciliation_job_id IS NOT NULL)" {
		t.Errorf("reconciliation-job index predicate = %v", predicate)
	}
}

func assertModerationRestorationOutboxConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs(id, source_did, subject_did)
		VALUES ('output-one', 'did:plc:source', 'did:plc:target');
		INSERT INTO instagram_reconciliation_jobs(id)
		VALUES ('10000000-0000-4000-8000-000000000001');
		INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,reconciliation_job_id,
			created_at,processed_at
		) VALUES (
			'output-one','did:plc:target','queued',
			'10000000-0000-4000-8000-000000000001',now(),now()
		);
	`); err != nil {
		t.Fatalf("seed queued restoration intent: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM moderation_outputs WHERE id='output-one'`); err == nil {
		t.Fatal("moderation parent deletion succeeded while live outbox child exists")
	}
	if _, err := pool.Exec(ctx, `
		DELETE FROM instagram_reconciliation_jobs
		WHERE id='10000000-0000-4000-8000-000000000001'
	`); err != nil {
		t.Fatalf("delete reconciliation job: %v", err)
	}
	var linkedJob *string
	if err := pool.QueryRow(ctx, `
		SELECT reconciliation_job_id::text
		FROM moderation_restoration_outbox
		WHERE moderation_output_id='output-one'
	`).Scan(&linkedJob); err != nil {
		t.Fatalf("read outbox job after ON DELETE SET NULL: %v", err)
	}
	if linkedJob != nil {
		t.Fatalf("reconciliation job id = %q, want NULL", *linkedJob)
	}

	invalidStatements := []string{
		`INSERT INTO moderation_restoration_outbox(
			moderation_output_id,target_did,status,created_at
		 ) VALUES ('output-one','did:plc:target','unknown',now())`,
		`INSERT INTO moderation_idempotency_receipts(
			request_key_hash,request_fingerprint,output_id,output_status,
			created_at,expires_at
		 ) VALUES (
			decode('00','hex'),decode(repeat('11',32),'hex'),'output-one','indexed',
			now(),now()+interval '1 hour'
		 )`,
		`INSERT INTO moderation_idempotency_receipts(
			request_key_hash,request_fingerprint,output_id,output_status,
			created_at,expires_at
		 ) VALUES (
			decode(repeat('22',32),'hex'),decode(repeat('33',32),'hex'),
			'output-one','indexed',now(),now()
		 )`,
	}
	for _, statement := range invalidStatements {
		if _, err := pool.Exec(ctx, statement); err == nil {
			t.Errorf("invalid constrained insert succeeded: %s", statement)
		}
	}
}
