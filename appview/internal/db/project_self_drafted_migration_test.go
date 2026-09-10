package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestProjectSelfDraftedMigrationBackfillsAndReverses(t *testing.T) {
	projectMigration, err := os.ReadFile("../../migrations/000016_project_posts.up.sql")
	if err != nil {
		t.Fatalf("read project migration: %v", err)
	}
	up, err := os.ReadFile("../../migrations/000071_project_self_drafted.up.sql")
	if err != nil {
		t.Fatalf("read self-drafted up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000071_project_self_drafted.down.sql")
	if err != nil {
		t.Fatalf("read self-drafted down migration: %v", err)
	}

	pool := testdb.WithSchema(t, projectPostsMigrationPreStateDDL+string(projectMigration))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid) VALUES ('did:plc:self-drafted', 'c');
		INSERT INTO craftsky_posts (uri, did, rkey, cid, text, record, created_at) VALUES
			('at://did:plc:self-drafted/social.craftsky.feed.post/true', 'did:plc:self-drafted', 'true', 'c1', 't', '{}', now()),
			('at://did:plc:self-drafted/social.craftsky.feed.post/false', 'did:plc:self-drafted', 'false', 'c2', 'f', '{}', now()),
			('at://did:plc:self-drafted/social.craftsky.feed.post/missing', 'did:plc:self-drafted', 'missing', 'c3', 'm', '{}', now()),
			('at://did:plc:self-drafted/social.craftsky.feed.post/invalid', 'did:plc:self-drafted', 'invalid', 'c4', 'i', '{}', now());
		INSERT INTO craftsky_project_posts (uri, raw_project, common_craft_type) VALUES
			('at://did:plc:self-drafted/social.craftsky.feed.post/true', '{"common":{"craftType":"knitting","pattern":{"selfDrafted":true}}}', 'knitting'),
			('at://did:plc:self-drafted/social.craftsky.feed.post/false', '{"common":{"craftType":"knitting","pattern":{"selfDrafted":false}}}', 'knitting'),
			('at://did:plc:self-drafted/social.craftsky.feed.post/missing', '{"common":{"craftType":"knitting","pattern":{}}}', 'knitting'),
			('at://did:plc:self-drafted/social.craftsky.feed.post/invalid', '{"common":{"craftType":"knitting","pattern":{"selfDrafted":"yes"}}}', 'knitting');
	`); err != nil {
		t.Fatalf("seed project rows: %v", err)
	}

	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply self-drafted up migration: %v", err)
	}
	assertColumn(t, pool, "craftsky_project_posts", "pattern_self_drafted", "boolean")
	if !indexExists(t, pool, "craftsky_project_posts_self_drafted_true_idx") {
		t.Fatal("self-drafted partial index missing")
	}

	rows, err := pool.Query(ctx, `SELECT uri, pattern_self_drafted FROM craftsky_project_posts ORDER BY uri`)
	if err != nil {
		t.Fatalf("read backfilled values: %v", err)
	}
	defer rows.Close()
	got := map[string]*bool{}
	for rows.Next() {
		var uri string
		var value *bool
		if err := rows.Scan(&uri, &value); err != nil {
			t.Fatalf("scan backfilled value: %v", err)
		}
		got[uri] = value
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read backfilled values: %v", err)
	}
	trueURI := "at://did:plc:self-drafted/social.craftsky.feed.post/true"
	falseURI := "at://did:plc:self-drafted/social.craftsky.feed.post/false"
	if got[trueURI] == nil || !*got[trueURI] {
		t.Fatalf("true backfill = %v, want true", got[trueURI])
	}
	if got[falseURI] == nil || *got[falseURI] {
		t.Fatalf("false backfill = %v, want false", got[falseURI])
	}
	for _, suffix := range []string{"missing", "invalid"} {
		uri := "at://did:plc:self-drafted/social.craftsky.feed.post/" + suffix
		if got[uri] != nil {
			t.Fatalf("%s backfill = %v, want null", suffix, *got[uri])
		}
	}

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply self-drafted down migration: %v", err)
	}
	if columnExists(t, pool, "craftsky_project_posts", "pattern_self_drafted") {
		t.Fatal("pattern_self_drafted remained after down migration")
	}
}
