package db_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

// IT-001 / RULE-004, NFR-004 / AC-017: legacy profiles are never assumed public.
func TestGuestVisibilityMigrationLeavesLegacyProfilesUnknown(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, `CREATE TABLE bluesky_profiles (
		did TEXT PRIMARY KEY, display_name TEXT, record_cid TEXT NOT NULL
	); INSERT INTO bluesky_profiles VALUES ('did:plc:legacy','Original name','original-cid');`)
	apply := func(name string) {
		t.Helper()
		if err := testdb.ApplyMigrations(ctx, pool, name); err != nil {
			t.Fatal(err)
		}
	}
	assertUnknown := func() {
		t.Helper()
		var name, cid string
		var hidden *bool
		var revision, visibilityCID, evidence, hold *string
		if err := pool.QueryRow(ctx, `SELECT display_name,record_cid,
			guest_visibility_hidden,guest_visibility_revision,guest_visibility_record_cid,
			guest_visibility_evidence,guest_visibility_hold_key::text
			FROM bluesky_profiles WHERE did='did:plc:legacy'`).Scan(
			&name, &cid, &hidden, &revision, &visibilityCID, &evidence, &hold,
		); err != nil {
			t.Fatal(err)
		}
		if name != "Original name" || cid != "original-cid" || hidden != nil || revision != nil || visibilityCID != nil || evidence != nil || hold != nil {
			t.Fatalf("legacy profile changed or visibility inferred: name=%q cid=%q hidden=%v revision=%v evidence=%v", name, cid, hidden, revision, evidence)
		}
		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM bluesky_profiles`).Scan(&count); err != nil || count != 1 {
			t.Fatalf("migration populated profiles: count=%d err=%v", count, err)
		}
	}
	apply("000118_profile_guest_visibility.up.sql")
	assertUnknown()
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles(did,record_cid) VALUES ('did:plc:new','new-cid')`); err != nil {
		t.Fatal(err)
	}
	var isUnknown bool
	if err := pool.QueryRow(ctx, `SELECT guest_visibility_hidden IS NULL FROM bluesky_profiles WHERE did='did:plc:new'`).Scan(&isUnknown); err != nil || !isUnknown {
		t.Fatalf("new profile default is not unknown: %v %v", isUnknown, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE bluesky_profiles SET guest_visibility_hidden=false WHERE did='did:plc:legacy'`); err == nil {
		t.Fatal("known visibility without authoritative evidence accepted")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM bluesky_profiles WHERE did='did:plc:new'`); err != nil {
		t.Fatal(err)
	}
	apply("000118_profile_guest_visibility.down.sql")
	apply("000118_profile_guest_visibility.up.sql")
	assertUnknown()
}
