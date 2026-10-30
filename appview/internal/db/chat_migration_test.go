package db_test

import (
	"testing"

	"social.craftsky/appview/internal/testdb"
)

// IT-005 / FR-016, NFR-001 / AC-014, AC-019.
func TestChatAccountsMigrationEnforcesOwnerAndGeneration(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	var table *string
	if err := pool.QueryRow(t.Context(), `SELECT to_regclass('chat_accounts')::text`).Scan(&table); err != nil {
		t.Fatal(err)
	}
	if table == nil {
		t.Fatal("messaging has no durable account enablement/generation state")
	}
	const owner = "did:plc:chat-schema-owner"
	if _, err := pool.Exec(t.Context(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at)
		VALUES($1,'active',1,1,'test',now())`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO chat_accounts(owner_did) VALUES($1)`, owner); err != nil {
		t.Fatal(err)
	}
	var enabled, push bool
	var generation int64
	if err := pool.QueryRow(t.Context(), `SELECT enabled,push_enabled,generation FROM chat_accounts WHERE owner_did=$1`, owner).Scan(&enabled, &push, &generation); err != nil {
		t.Fatal(err)
	}
	if enabled || !push || generation != 1 {
		t.Fatal("new account did not start disabled with default-on push and generation one")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE chat_accounts SET generation=0 WHERE owner_did=$1`, owner); !isCheckViolation(err) {
		t.Fatal("invalid messaging generation was accepted")
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO chat_accounts(owner_did) VALUES('did:plc:chat-orphan')`); err == nil {
		t.Fatal("orphan messaging account was accepted")
	}
}

// IT-005 / FR-016: local rollback restores the prior auth contract.
func TestChatMigrationRollbackRestoresAuthPurposeContract(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	down, err := testdb.ReadMigration("000111_direct_messaging.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), string(down)); err != nil {
		t.Fatal(err)
	}
	var allowsChat bool
	if err := pool.QueryRow(t.Context(), `SELECT position('chat' in pg_get_constraintdef(oid))>0 FROM pg_constraint WHERE conrelid='oauth_auth_requests'::regclass AND conname='oauth_auth_requests_purpose_check'`).Scan(&allowsChat); err != nil {
		t.Fatal(err)
	}
	if allowsChat {
		t.Fatal("rollback leaves optional consent admitted without its private state")
	}
	var columns int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='oauth_auth_requests' AND column_name LIKE 'chat_%'`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if columns != 0 {
		t.Fatal("rollback retains chat consent metadata")
	}
	up, err := testdb.ReadMigration("000111_direct_messaging.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), string(up)); err != nil {
		t.Fatalf("reapply: %v", err)
	}
}
