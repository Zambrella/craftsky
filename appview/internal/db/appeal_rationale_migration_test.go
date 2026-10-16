package db_test

import (
	"social.craftsky/appview/internal/testdb"
	"strings"
	"testing"
)

func TestAppealRationaleMigrationRetainsKnownHistoryAndGuardsRollback(t *testing.T) {
	pool := testdb.WithSchema(t, `
 CREATE TABLE moderation_appeals(case_id UUID PRIMARY KEY,status TEXT,resolved_event_id UUID);
 CREATE TABLE moderation_effect_events(id UUID,case_event_id UUID,rationale TEXT,created_at TIMESTAMPTZ);
 INSERT INTO moderation_appeals VALUES('63000000-0000-4000-8000-000000000001','changed','63000000-0000-4000-8000-000000000002'),('63000000-0000-4000-8000-000000000003','upheld',NULL);
 INSERT INTO moderation_effect_events VALUES('63000000-0000-4000-8000-000000000004','63000000-0000-4000-8000-000000000002','retained earlier rationale',now());
 `)
	up, err := testdb.ReadMigration("000099_appeal_resolution_rationale.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := testdb.ReadMigration("000099_appeal_resolution_rationale.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), string(up)); err != nil {
		t.Fatal(err)
	}
	var known string
	if err := pool.QueryRow(t.Context(), `SELECT resolution_rationale FROM moderation_appeals WHERE status='changed'`).Scan(&known); err != nil || known != "retained earlier rationale" {
		t.Fatalf("known history=%q, %v", known, err)
	}
	var unknown bool
	if err := pool.QueryRow(t.Context(), `SELECT resolution_rationale IS NULL FROM moderation_appeals WHERE status='upheld'`).Scan(&unknown); err != nil || !unknown {
		t.Fatalf("missing historical rationale must not be invented: %v", err)
	}
	if _, err := pool.Exec(t.Context(), string(down)); err == nil || !strings.Contains(err.Error(), "refusing to discard") {
		t.Fatalf("rollback did not protect rationale: %v", err)
	}
}
