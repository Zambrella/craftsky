package main

import (
	"context"
	"testing"
	"time"

	"social.craftsky/appview/internal/testdb"
)

const demoSeedDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT NOT NULL PRIMARY KEY,
    crafts TEXT[] NOT NULL DEFAULT '{}',
    record_cid TEXT NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did TEXT NOT NULL PRIMARY KEY,
    display_name TEXT,
    description TEXT,
    avatar_cid TEXT,
    avatar_mime TEXT,
    banner_cid TEXT,
    banner_mime TEXT,
    record_cid TEXT NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_identity_cache (
    did TEXT NOT NULL PRIMARY KEY,
    handle TEXT NOT NULL,
    handle_lower TEXT NOT NULL UNIQUE,
    resolved_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_follows (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);
CREATE TABLE craftsky_posts (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    text TEXT NOT NULL,
    facets JSONB,
    images JSONB,
    reply_root_uri TEXT,
    reply_root_cid TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,
    quote_uri TEXT,
    quote_cid TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    is_project BOOLEAN NOT NULL DEFAULT false,
    project_craft_type TEXT,
    record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_project_posts (
    uri TEXT PRIMARY KEY REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    raw_project JSONB NOT NULL,
    common_craft_type TEXT NOT NULL,
    common_status TEXT,
    common_title TEXT,
    common_duration TEXT,
    pattern_url TEXT,
    pattern_name TEXT,
    pattern_name_facets JSONB,
    pattern_difficulty TEXT,
    pattern_designer TEXT,
    pattern_designer_facets JSONB,
    pattern_publisher TEXT,
    pattern_publisher_facets JSONB,
    materials TEXT[] NOT NULL DEFAULT '{}',
    colors TEXT[] NOT NULL DEFAULT '{}',
    design_tags TEXT[] NOT NULL DEFAULT '{}',
    project_tags TEXT[] NOT NULL DEFAULT '{}',
    details_type TEXT,
    raw_details JSONB,
    knitting_project_type TEXT,
    knitting_project_subtype TEXT,
    knitting_yarn_weight TEXT,
    knitting_needle_size_mm TEXT,
    knitting_gauge JSONB,
    knitting_finished_size TEXT,
    crochet_project_type TEXT,
    crochet_project_subtype TEXT,
    crochet_yarn_weight TEXT,
    crochet_hook_size_mm TEXT,
    crochet_gauge JSONB,
    crochet_finished_size TEXT,
    quilting_project_type TEXT,
    quilting_project_subtype TEXT,
    quilting_piecing_technique TEXT,
    quilting_quilting_method TEXT,
    quilting_size TEXT,
    sewing_project_type TEXT,
    sewing_project_subtype TEXT,
    sewing_size_made TEXT,
    sewing_fit_notes TEXT,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE craftsky_likes (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT NOT NULL,
    record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_reposts (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    subject_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    subject_cid TEXT NOT NULL,
    record JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_sessions (
    account_did TEXT NOT NULL,
    revoked_at TIMESTAMPTZ
);
`

func TestRunDemoSeedCreatesScreenshotDatasetAndIsIdempotent(t *testing.T) {
	pool := testdb.WithSchema(t, demoSeedDDL)
	ctx := context.Background()
	args := demoSeedArgs{UserDID: "did:plc:viewer", Seed: "shot", Now: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)}

	stats, err := runDemoSeed(ctx, pool, args)
	if err != nil {
		t.Fatalf("runDemoSeed: %v", err)
	}
	if stats.Profiles < 30 || stats.Posts < 80 || stats.Projects < 12 || stats.Comments < 30 || stats.Likes < 100 || stats.Reposts < 20 || stats.Follows < 60 {
		t.Fatalf("stats = %+v", stats)
	}
	stats2, err := runDemoSeed(ctx, pool, args)
	if err != nil {
		t.Fatalf("runDemoSeed second pass: %v", err)
	}
	if stats2.Deleted != 0 {
		t.Fatalf("second pass deleted = %d, want 0", stats2.Deleted)
	}

	var posts, projects, mediaPosts, follows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_posts`).Scan(&posts); err != nil {
		t.Fatalf("count posts: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_project_posts`).Scan(&projects); err != nil {
		t.Fatalf("count projects: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_posts WHERE images::text LIKE '%devmedia:%'`).Scan(&mediaPosts); err != nil {
		t.Fatalf("count media posts: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_follows WHERE did = 'did:plc:viewer'`).Scan(&follows); err != nil {
		t.Fatalf("count viewer follows: %v", err)
	}
	if posts < 80 || projects < 12 || mediaPosts < 10 || follows < 30 {
		t.Fatalf("posts=%d projects=%d mediaPosts=%d follows=%d", posts, projects, mediaPosts, follows)
	}
}

func TestRunDemoSeedResetDeletesPreviousSeedRows(t *testing.T) {
	pool := testdb.WithSchema(t, demoSeedDDL)
	ctx := context.Background()
	args := demoSeedArgs{UserDID: "did:plc:viewer", Seed: "reset", Now: time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)}
	if _, err := runDemoSeed(ctx, pool, args); err != nil {
		t.Fatalf("runDemoSeed: %v", err)
	}
	args.Reset = true
	stats, err := runDemoSeed(ctx, pool, args)
	if err != nil {
		t.Fatalf("runDemoSeed reset: %v", err)
	}
	if stats.Deleted == 0 {
		t.Fatalf("reset deleted = 0, want previous seed rows removed")
	}
}

func TestCollectDemoViewerDIDsIncludesConfigDevDIDAndActiveSessions(t *testing.T) {
	pool := testdb.WithSchema(t, demoSeedDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions (account_did, revoked_at)
		VALUES
			('did:plc:viewer-session', NULL),
			('did:plc:viewer-session', NULL),
			('did:plc:revoked-session', now())
	`); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	got, err := collectDemoViewerDIDs(ctx, pool, "did:plc:configured-dev")
	if err != nil {
		t.Fatalf("collectDemoViewerDIDs: %v", err)
	}
	want := []string{"did:plc:configured-dev", "did:plc:viewer-session"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("viewer DIDs = %v, want %v", got, want)
	}
}
