// appview/internal/api/profile_store_test.go
package api_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

type fakeProfileHydrator struct {
	record map[string]any
	cid    string
	err    error
}

func (f fakeProfileHydrator) GetRecord(_ context.Context, _ syntax.DID, _ string, _ string, out any) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	*(out.(*map[string]any)) = f.record
	if f.cid == "" {
		return "cid-hydrated", nil
	}
	return f.cid, nil
}

func (f fakeProfileHydrator) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return errors.New("not implemented")
}

func (f fakeProfileHydrator) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("not implemented")
}

func (f fakeProfileHydrator) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("not implemented")
}

func (f fakeProfileHydrator) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("not implemented")
}

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
	got, err := store.Read(ctx, "did:plc:a", "did:plc:viewer")
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
	_, err := store.Read(context.Background(), "did:plc:nobody", "did:plc:viewer")
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
	got, err := store.Read(ctx, "did:plc:b", "did:plc:viewer")
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
	bob, err := store.Read(ctx, "did:plc:bob", "did:plc:alice")
	if err != nil {
		t.Fatalf("Read bob: %v", err)
	}
	alice, err := store.Read(ctx, "did:plc:alice", "did:plc:alice")
	if err != nil {
		t.Fatalf("Read alice: %v", err)
	}

	if bob.FollowerCount == nil || *bob.FollowerCount != 1 {
		t.Fatalf("bob followerCount = %v, want 1", bob.FollowerCount)
	}
	if !bob.ViewerIsFollowing {
		t.Fatalf("bob viewerIsFollowing = false, want true for alice viewer")
	}
	if alice.FollowingCount == nil || *alice.FollowingCount != 1 {
		t.Fatalf("alice followingCount = %v, want 1", alice.FollowingCount)
	}
	if alice.ViewerIsFollowing {
		t.Fatalf("alice viewerIsFollowing = true, want false for self profile")
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
	got, err := store.Read(ctx, "did:plc:carol", "did:plc:alice")
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

func TestProfileStore_ReadByDID_NonCraftskyViewerStateFromGraph(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, record_cid)
		VALUES ($1, $2, $3)
	`, "did:plc:carol", "Carol", "cid-bsky"); err != nil {
		t.Fatalf("seed bluesky profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
	`, "at://did:plc:alice/app.bsky.graph.follow/f1", "did:plc:alice", "f1", "cid-follow", "did:plc:carol", `{"subject":"did:plc:carol"}`); err != nil {
		t.Fatalf("seed follow: %v", err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:carol", "did:plc:alice")
	if err != nil {
		t.Fatalf("Read non-craftsky: %v", err)
	}
	if !got.ViewerIsFollowing {
		t.Fatalf("viewerIsFollowing = false, want true")
	}
}

func TestProfileStore_ReadByDID_HydratesNonCraftskyWhenCacheMisses(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	store := api.NewProfileStore(pool, fakeProfileHydrator{
		cid: "cid-carol",
		record: map[string]any{
			"displayName": "Carol",
			"description": "external account",
			"avatar": map[string]any{
				"ref":      map[string]any{"$link": "baf-avatar"},
				"mimeType": "image/jpeg",
			},
		},
	})
	got, err := store.Read(ctx, "did:plc:carol", "did:plc:alice")
	if err != nil {
		t.Fatalf("Read hydrated non-craftsky: %v", err)
	}

	if got.IsCraftskyProfile {
		t.Fatalf("isCraftskyProfile = true, want false")
	}
	if got.DisplayName == nil || *got.DisplayName != "Carol" {
		t.Fatalf("displayName = %v, want Carol", got.DisplayName)
	}
	if got.AvatarCID == nil || *got.AvatarCID != "baf-avatar" {
		t.Fatalf("avatarCID = %v, want baf-avatar", got.AvatarCID)
	}
}
