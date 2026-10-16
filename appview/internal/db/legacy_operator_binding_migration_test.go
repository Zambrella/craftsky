package db_test

import (
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/operatorconsole"
	"social.craftsky/appview/internal/testdb"
)

func TestRetiredOperatorBindingMigrationPreservesIdentityAuditAndSessions(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	var exists bool
	if err := pool.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='safety_operators' AND column_name='retool_user_id')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	// Restore the historical pre-state when the full migration chain already
	// includes the cleanup, then exercise its upgrade against populated data.
	if !exists {
		if err := testdb.ApplyMigrations(t.Context(), pool, "000101_remove_legacy_operator_binding.down.sql"); err != nil {
			t.Fatal(err)
		}
	}
	store := operatorconsole.NewStore(pool, time.Now)
	op, err := store.Allow(t.Context(), syntax.DID("did:example:binding-cleanup"), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE safety_operators SET retool_user_id=42 WHERE id=$1`, op.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO operator_browser_sessions(token_digest,operator_id,created_at,expires_at,last_used_at) VALUES(decode(repeat('01',32),'hex'),$1,now(),now()+interval '1 hour',now())`, op.ID); err != nil {
		t.Fatal(err)
	}
	if err := testdb.ApplyMigrations(t.Context(), pool, "000101_remove_legacy_operator_binding.up.sql"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(t.Context(), `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='safety_operators' AND column_name='retool_user_id')`).Scan(&exists); err != nil || exists {
		t.Fatalf("retired binding remains: exists=%t err=%v", exists, err)
	}
	current, err := store.Lookup(t.Context(), op.DID)
	if err != nil || current.ID != op.ID || current.ActorID != op.ActorID || current.PermissionSet != "admin" {
		t.Fatalf("operator identity changed: %v", err)
	}
	events, err := store.Audit(t.Context(), op.ID)
	if err != nil || len(events) != 1 || events[0].Action != "allowlisted" {
		t.Fatalf("audit changed: %v", err)
	}
	var sessions int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM operator_browser_sessions WHERE operator_id=$1`, op.ID).Scan(&sessions); err != nil || sessions != 1 {
		t.Fatalf("session changed: count=%d err=%v", sessions, err)
	}
	if err := testdb.ApplyMigrations(t.Context(), pool, "000101_remove_legacy_operator_binding.down.sql"); err != nil {
		t.Fatal(err)
	}
	var empty bool
	if err := pool.QueryRow(t.Context(), `SELECT retool_user_id IS NULL FROM safety_operators WHERE id=$1`, op.ID).Scan(&empty); err != nil || !empty {
		t.Fatalf("rollback must restore an empty optional column: %v", err)
	}
}
