// appview/internal/api/profile_store_test.go
package api_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const profileStoreDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE bluesky_profiles (
    did          TEXT        NOT NULL PRIMARY KEY,
    display_name TEXT,
    description  TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_follows (
    uri         TEXT        NOT NULL PRIMARY KEY,
    did         TEXT        NOT NULL,
    rkey        TEXT        NOT NULL,
    cid         TEXT        NOT NULL,
    subject_did TEXT        NOT NULL,
    record      JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey),
    UNIQUE (did, subject_did)
);
`

func TestProfileStore_ReadByDID_MemberWithBothRows(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, $2, $3)`,
		"did:plc:a", []string{"sewing"}, "cid1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, avatar_cid, avatar_mime, record_cid)
		VALUES ($1, $2, $3, $4, $5)`,
		"did:plc:a", "Alice", "bafav", "image/jpeg", "cid2")
	if err != nil {
		t.Fatal(err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:a")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.DID != "did:plc:a" {
		t.Errorf("DID = %q", got.DID)
	}
	if got.DisplayName == nil || *got.DisplayName != "Alice" {
		t.Errorf("DisplayName = %v", got.DisplayName)
	}
	if got.AvatarCID == nil || *got.AvatarCID != "bafav" {
		t.Errorf("AvatarCID = %v", got.AvatarCID)
	}
	if len(got.Crafts) != 1 || got.Crafts[0] != "sewing" {
		t.Errorf("Crafts = %v", got.Crafts)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("CreatedAt is zero")
	}
}

func TestProfileStore_ReadByDID_NonMember(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	store := api.NewProfileStore(pool)
	_, err := store.Read(context.Background(), "did:plc:nobody")
	if err == nil {
		t.Fatal("want error; got nil")
	}
	if err != api.ErrProfileNotFound {
		t.Errorf("want ErrProfileNotFound; got %v", err)
	}
}

func TestProfileStore_ReadByDID_MemberWithoutBlueskyRow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	_, _ = pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, $2, $3)`,
		"did:plc:b", []string{}, "cid1")

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:b")
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != nil {
		t.Errorf("DisplayName should be nil; got %v", *got.DisplayName)
	}
	if len(got.Crafts) != 0 {
		t.Errorf("Crafts = %v, want empty", got.Crafts)
	}
}

func TestProfileStore_ReadByDID_CraftskyOnlyCounts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	// Craftsky members.
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
			did,
		); err != nil {
			t.Fatalf("insert craftsky profile %s: %v", did, err)
		}
	}

	// Active follows:
	// - alice -> bob (counts)
	// - alice -> dana (non-craftsky target, excluded from followingCount)
	// - dana -> bob (non-craftsky follower, excluded from followerCount)
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES
			('at://did:plc:alice/app.bsky.graph.follow/f1', 'did:plc:alice', 'f1', 'c1', 'did:plc:bob', '{"subject":"did:plc:bob"}', now()),
			('at://did:plc:alice/app.bsky.graph.follow/f2', 'did:plc:alice', 'f2', 'c2', 'did:plc:dana', '{"subject":"did:plc:dana"}', now()),
			('at://did:plc:dana/app.bsky.graph.follow/f3', 'did:plc:dana', 'f3', 'c3', 'did:plc:bob', '{"subject":"did:plc:bob"}', now())
	`); err != nil {
		t.Fatalf("seed follows: %v", err)
	}

	store := api.NewProfileStore(pool)
	bob, err := store.Read(ctx, "did:plc:bob")
	if err != nil {
		t.Fatalf("Read bob: %v", err)
	}
	alice, err := store.Read(ctx, "did:plc:alice")
	if err != nil {
		t.Fatalf("Read alice: %v", err)
	}

	if bob.FollowerCount == nil || *bob.FollowerCount != 1 {
		t.Fatalf("bob followerCount = %v, want 1", bob.FollowerCount)
	}
	if alice.FollowingCount == nil || *alice.FollowingCount != 1 {
		t.Fatalf("alice followingCount = %v, want 1", alice.FollowingCount)
	}
}

func TestProfileStore_ReadByDID_NonCraftskyFromBlueskyCache(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, description, record_cid)
		VALUES ($1, $2, $3, $4)
	`, "did:plc:carol", "Carol", "external account", "cid-bsky"); err != nil {
		t.Fatalf("seed bluesky profile: %v", err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:carol")
	if err != nil {
		t.Fatalf("Read non-craftsky: %v", err)
	}

	if got.IsCraftskyProfile {
		t.Fatalf("isCraftskyProfile = true, want false")
	}
	if got.FollowerCount != nil || got.FollowingCount != nil {
		t.Fatalf("counts = (%v,%v), want nil,nil", got.FollowerCount, got.FollowingCount)
	}
	if got.DisplayName == nil || *got.DisplayName != "Carol" {
		t.Fatalf("displayName = %v, want Carol", got.DisplayName)
	}
	if got.Crafts == nil || len(got.Crafts) != 0 {
		t.Fatalf("crafts = %v, want empty []", got.Crafts)
	}
}
