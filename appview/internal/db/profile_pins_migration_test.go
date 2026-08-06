package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const profilePinsMigrationPreStateDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE craftsky_posts (
    uri             TEXT        NOT NULL PRIMARY KEY,
    did             TEXT        NOT NULL,
    rkey            TEXT        NOT NULL,
    cid              TEXT        NOT NULL,
    profile_sort_at  TIMESTAMPTZ NOT NULL,
    UNIQUE (did, rkey)
);
CREATE TABLE saved_posts (
    owner_did TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    post_uri  TEXT        NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    saved_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (owner_did, post_uri)
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
INSERT INTO craftsky_posts (uri, did, rkey, cid, profile_sort_at)
VALUES
    ('at://did:plc:alice/social.craftsky.feed.post/standard', 'did:plc:alice', 'standard', 'standard-cid', '2026-08-01T10:00:00Z'),
    ('at://did:plc:alice/social.craftsky.feed.post/project', 'did:plc:alice', 'project', 'project-cid', '2026-08-01T11:00:00Z'),
    ('at://did:plc:bob/social.craftsky.feed.post/sentinel', 'did:plc:bob', 'sentinel', 'sentinel-cid', '2026-08-01T12:00:00Z');
INSERT INTO saved_posts (owner_did, post_uri, saved_at)
VALUES ('did:plc:bob', 'at://did:plc:bob/social.craftsky.feed.post/sentinel', '2026-08-02T10:00:00Z');
`

func TestProfilePinsMigration(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000035_profile_pins.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}

	pool := testdb.WithSchema(t, profilePinsMigrationPreStateDDL)
	ctx := context.Background()
	apply := func(label string, sql []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply %s migration: %v", label, err)
		}
	}

	apply("up", up)
	assertProfilePinsSchema(t, pool)
	assertProfilePinsUnrelatedState(t, pool)

	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_pins (
			owner_did, slot, post_uri, state_token, created_at, updated_at
		) VALUES
			('did:plc:alice', 'standard', 'at://did:plc:alice/social.craftsky.feed.post/standard', '00000000-0000-4000-8000-000000000001', '2026-08-03T10:00:00Z', '2026-08-03T10:00:00Z'),
			('did:plc:alice', 'project', 'at://did:plc:alice/social.craftsky.feed.post/project', '00000000-0000-4000-8000-000000000002', '2026-08-03T10:01:00Z', '2026-08-03T10:01:00Z')
	`); err != nil {
		t.Fatalf("insert independent pin slots: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_pins (
			owner_did, slot, post_uri, state_token, created_at, updated_at
		) VALUES (
			'did:plc:alice', 'standard', 'at://did:plc:alice/social.craftsky.feed.post/project',
			'00000000-0000-4000-8000-000000000003', now(), now()
		)
	`); err == nil {
		t.Fatal("duplicate owner/slot pin succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_pins (
			owner_did, slot, post_uri, state_token, created_at, updated_at
		) VALUES (
			'did:plc:alice', 'other', 'at://did:plc:alice/social.craftsky.feed.post/project',
			'00000000-0000-4000-8000-000000000004', now(), now()
		)
	`); err == nil {
		t.Fatal("invalid pin slot succeeded")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_pins (
			owner_did, slot, post_uri, state_token, created_at, updated_at
		) VALUES (
			'did:plc:missing', 'standard', 'at://did:plc:alice/social.craftsky.feed.post/standard',
			'00000000-0000-4000-8000-000000000005', now(), now()
		)
	`); err == nil {
		t.Fatal("pin for missing owner succeeded")
	}

	if _, err := pool.Exec(ctx, `
		DELETE FROM craftsky_posts
		WHERE uri = 'at://did:plc:alice/social.craftsky.feed.post/standard'
	`); err != nil {
		t.Fatalf("delete pinned target: %v", err)
	}
	assertProfilePinCount(t, pool, "did:plc:alice", 1)

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:alice'`); err != nil {
		t.Fatalf("delete pin owner membership: %v", err)
	}
	assertProfilePinCount(t, pool, "did:plc:alice", 0)
	assertProfilePinsUnrelatedState(t, pool)

	apply("down", down)
	if tableExists(t, pool, "profile_pins") {
		t.Fatal("profile_pins remained after down migration")
	}
	assertProfilePinsUnrelatedState(t, pool)

	apply("second up", up)
	assertProfilePinsSchema(t, pool)
	assertProfilePinsUnrelatedState(t, pool)
}

func assertProfilePinsSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if !tableExists(t, pool, "profile_pins") {
		t.Fatal("profile_pins table missing")
	}
	for _, constraint := range []string{
		"profile_pins_pkey",
		"profile_pins_owner_did_fkey",
		"profile_pins_post_uri_fkey",
		"profile_pins_slot_check",
	} {
		if !constraintExists(t, pool, constraint) {
			t.Errorf("constraint %s missing", constraint)
		}
	}
	if !indexExists(t, pool, "profile_pins_post_uri_idx") {
		t.Error("index profile_pins_post_uri_idx missing")
	}
}

func assertProfilePinsUnrelatedState(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var (
		profileSortAt string
		savedCount    int
	)
	if err := pool.QueryRow(context.Background(), `
		SELECT profile_sort_at::text
		FROM craftsky_posts
		WHERE uri = 'at://did:plc:bob/social.craftsky.feed.post/sentinel'
	`).Scan(&profileSortAt); err != nil {
		t.Fatalf("read chronology sentinel: %v", err)
	}
	if profileSortAt != "2026-08-01 12:00:00+00" {
		t.Fatalf("chronology sentinel changed: %q", profileSortAt)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM saved_posts`).Scan(&savedCount); err != nil {
		t.Fatalf("count saved-post sentinel: %v", err)
	}
	if savedCount != 1 {
		t.Fatalf("saved-post sentinel count = %d, want 1", savedCount)
	}
}

func assertProfilePinCount(t *testing.T, pool *pgxpool.Pool, ownerDID string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM profile_pins WHERE owner_did = $1
	`, ownerDID).Scan(&got); err != nil {
		t.Fatalf("count profile pins: %v", err)
	}
	if got != want {
		t.Fatalf("profile pin count for %s = %d, want %d", ownerDID, got, want)
	}
}
