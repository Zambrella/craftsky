// appview/internal/api/profile_store_test.go
package api_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

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
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL REFERENCES craftsky_profiles(did) ON DELETE CASCADE,
    rkey             TEXT        NOT NULL,
    cid              TEXT        NOT NULL,
    text             TEXT        NOT NULL,
    facets           JSONB,
    images           JSONB,
    reply_root_uri   TEXT,
    reply_root_cid   TEXT,
    reply_parent_uri TEXT,
    reply_parent_cid TEXT,
    quote_uri        TEXT,
    quote_cid        TEXT,
    tags             TEXT[]      NOT NULL DEFAULT '{}',
    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
CREATE INDEX atproto_follows_subject_created_uri_desc_idx
    ON atproto_follows (subject_did, created_at DESC, uri DESC);
CREATE INDEX atproto_follows_did_created_uri_desc_idx
    ON atproto_follows (did, created_at DESC, uri DESC);
CREATE INDEX craftsky_posts_root_did_created_idx
    ON craftsky_posts (did, created_at DESC)
    WHERE reply_root_uri IS NULL AND reply_parent_uri IS NULL;
`

func TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
		"did:plc:alice",
	); err != nil {
		t.Fatalf("seed craftsky profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, reply_root_uri, reply_root_cid,
			reply_parent_uri, reply_parent_cid, record, created_at
		)
		VALUES
			($1, 'did:plc:alice', 'root-recent', 'cid1', 'recent root', NULL, NULL, NULL, NULL, '{}', $2),
			($3, 'did:plc:alice', 'root-old', 'cid2', 'old root', NULL, NULL, NULL, NULL, '{}', $4),
			($5, 'did:plc:alice', 'reply-recent', 'cid3', 'reply', $1, 'cid1', $1, 'cid1', '{}', $2),
			($6, 'did:plc:alice', 'quote-recent', 'cid4', 'quote root', NULL, NULL, NULL, NULL, '{}', $2)
	`,
		"at://did:plc:alice/social.craftsky.feed.post/root-recent", now.Add(-24*time.Hour),
		"at://did:plc:alice/social.craftsky.feed.post/root-old", now.Add(-8*24*time.Hour),
		"at://did:plc:alice/social.craftsky.feed.post/reply-recent",
		"at://did:plc:alice/social.craftsky.feed.post/quote-recent",
	); err != nil {
		t.Fatalf("seed posts: %v", err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:alice", "did:plc:viewer")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.PostCount == nil || *got.PostCount != 3 {
		t.Fatalf("postCount = %v, want 3 root posts", got.PostCount)
	}
	if got.PostsLast7Days == nil || *got.PostsLast7Days != 2 {
		t.Fatalf("postsLast7Days = %v, want 2 recent root posts", got.PostsLast7Days)
	}
	if got.ProjectCount == nil || *got.ProjectCount != 0 {
		t.Fatalf("projectCount = %v, want data-driven zero", got.ProjectCount)
	}
}

func TestProfileStore_ReadByDID_MutualFollowerCountUsesViewerGraph(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	for _, did := range []string{
		"did:plc:viewer",
		"did:plc:profile",
		"did:plc:mutual",
		"did:plc:viewer-only",
		"did:plc:profile-only",
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
			did,
		); err != nil {
			t.Fatalf("seed craftsky profile %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES
			('at://did:plc:viewer/app.bsky.graph.follow/f1', 'did:plc:viewer', 'f1', 'c1', 'did:plc:mutual', '{"subject":"did:plc:mutual"}', now()),
			('at://did:plc:mutual/app.bsky.graph.follow/f2', 'did:plc:mutual', 'f2', 'c2', 'did:plc:profile', '{"subject":"did:plc:profile"}', now()),
			('at://did:plc:viewer/app.bsky.graph.follow/f3', 'did:plc:viewer', 'f3', 'c3', 'did:plc:viewer-only', '{"subject":"did:plc:viewer-only"}', now()),
			('at://did:plc:profile-only/app.bsky.graph.follow/f4', 'did:plc:profile-only', 'f4', 'c4', 'did:plc:profile', '{"subject":"did:plc:profile"}', now())
	`); err != nil {
		t.Fatalf("seed follows: %v", err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:profile", "did:plc:viewer")
	if err != nil {
		t.Fatalf("Read visitor profile: %v", err)
	}
	if got.MutualFollowerCount == nil || *got.MutualFollowerCount != 1 {
		t.Fatalf("mutualFollowerCount = %v, want 1", got.MutualFollowerCount)
	}

	self, err := store.Read(ctx, "did:plc:viewer", "did:plc:viewer")
	if err != nil {
		t.Fatalf("Read self profile: %v", err)
	}
	if self.MutualFollowerCount != nil {
		t.Fatalf("self mutualFollowerCount = %v, want nil", self.MutualFollowerCount)
	}
}

func TestProfileStore_ListMutualFollowers_PaginatesDisplayRows(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)

	for _, did := range []string{
		"did:plc:viewer",
		"did:plc:profile",
		"did:plc:newest",
		"did:plc:middle",
		"did:plc:oldest",
		"did:plc:not-mutual",
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
			did,
		); err != nil {
			t.Fatalf("seed craftsky profile %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, description, avatar_cid, avatar_mime, record_cid)
		VALUES
			('did:plc:newest', 'Newest', 'new desc', 'bafnew', 'image/jpeg', 'cid-bp-1'),
			('did:plc:middle', 'Middle', NULL, NULL, NULL, 'cid-bp-2'),
			('did:plc:oldest', 'Oldest', NULL, NULL, NULL, 'cid-bp-3')
	`); err != nil {
		t.Fatalf("seed bluesky profiles: %v", err)
	}

	followRows := []struct {
		uri     string
		did     string
		rkey    string
		subject string
		created time.Time
	}{
		{"at://did:plc:viewer/app.bsky.graph.follow/v1", "did:plc:viewer", "v1", "did:plc:newest", base.Add(-1 * time.Hour)},
		{"at://did:plc:viewer/app.bsky.graph.follow/v2", "did:plc:viewer", "v2", "did:plc:middle", base.Add(-2 * time.Hour)},
		{"at://did:plc:viewer/app.bsky.graph.follow/v3", "did:plc:viewer", "v3", "did:plc:oldest", base.Add(-3 * time.Hour)},
		{"at://did:plc:newest/app.bsky.graph.follow/m1", "did:plc:newest", "m1", "did:plc:profile", base.Add(-10 * time.Minute)},
		{"at://did:plc:middle/app.bsky.graph.follow/m2", "did:plc:middle", "m2", "did:plc:profile", base.Add(-20 * time.Minute)},
		{"at://did:plc:oldest/app.bsky.graph.follow/m3", "did:plc:oldest", "m3", "did:plc:profile", base.Add(-30 * time.Minute)},
		{"at://did:plc:viewer/app.bsky.graph.follow/v4", "did:plc:viewer", "v4", "did:plc:not-mutual", base.Add(-4 * time.Hour)},
	}
	for _, row := range followRows {
		if _, err := pool.Exec(ctx, `
			INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
			VALUES ($1, $2, $3, 'cid', $4, jsonb_build_object('subject', $4::text), $5)
		`, row.uri, row.did, row.rkey, row.subject, row.created); err != nil {
			t.Fatalf("seed follow %s: %v", row.uri, err)
		}
	}

	store := api.NewProfileStore(pool)
	first, cursor, total, err := store.ListMutualFollowers(ctx, "did:plc:viewer", "did:plc:profile", 2, "")
	if err != nil {
		t.Fatalf("ListMutualFollowers first page: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}
	if cursor == "" {
		t.Fatal("cursor = empty, want next page cursor")
	}
	if got := []string{first[0].DID, first[1].DID}; got[0] != "did:plc:newest" || got[1] != "did:plc:middle" {
		t.Fatalf("first page DIDs = %v, want newest,middle", got)
	}
	if first[0].DisplayName == nil || *first[0].DisplayName != "Newest" {
		t.Fatalf("displayName = %v, want Newest", first[0].DisplayName)
	}
	if !first[0].IsCraftskyProfile {
		t.Fatalf("isCraftskyProfile = false, want true")
	}

	second, next, total, err := store.ListMutualFollowers(ctx, "did:plc:viewer", "did:plc:profile", 2, cursor)
	if err != nil {
		t.Fatalf("ListMutualFollowers second page: %v", err)
	}
	if total != 3 || next != "" {
		t.Fatalf("second total,next = %d,%q; want 3,empty", total, next)
	}
	if len(second) != 1 || second[0].DID != "did:plc:oldest" {
		t.Fatalf("second page = %+v, want oldest only", second)
	}
}

func TestProfileStore_ListFollowersAndFollowing_OrderNewestFirst(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC)

	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol", "did:plc:dana"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
			did,
		); err != nil {
			t.Fatalf("seed craftsky profile %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_follows (uri, did, rkey, cid, subject_did, record, created_at)
		VALUES
			('at://did:plc:bob/app.bsky.graph.follow/f1', 'did:plc:bob', 'f1', 'c1', 'did:plc:alice', '{"subject":"did:plc:alice"}', $1),
			('at://did:plc:carol/app.bsky.graph.follow/f2', 'did:plc:carol', 'f2', 'c2', 'did:plc:alice', '{"subject":"did:plc:alice"}', $2),
			('at://did:plc:dana/app.bsky.graph.follow/f3', 'did:plc:dana', 'f3', 'c3', 'did:plc:alice', '{"subject":"did:plc:alice"}', $3),
			('at://did:plc:alice/app.bsky.graph.follow/f4', 'did:plc:alice', 'f4', 'c4', 'did:plc:bob', '{"subject":"did:plc:bob"}', $1),
			('at://did:plc:alice/app.bsky.graph.follow/f5', 'did:plc:alice', 'f5', 'c5', 'did:plc:carol', '{"subject":"did:plc:carol"}', $2),
			('at://did:plc:alice/app.bsky.graph.follow/f6', 'did:plc:alice', 'f6', 'c6', 'did:plc:dana', '{"subject":"did:plc:dana"}', $3)
	`, base.Add(-3*time.Hour), base.Add(-2*time.Hour), base.Add(-1*time.Hour)); err != nil {
		t.Fatalf("seed follows: %v", err)
	}

	store := api.NewProfileStore(pool)
	followers, _, followerTotal, err := store.ListFollowers(ctx, "did:plc:alice", 10, "")
	if err != nil {
		t.Fatalf("ListFollowers: %v", err)
	}
	if followerTotal != 3 {
		t.Fatalf("follower total = %d, want 3", followerTotal)
	}
	if got := []string{followers[0].DID, followers[1].DID, followers[2].DID}; got[0] != "did:plc:dana" || got[1] != "did:plc:carol" || got[2] != "did:plc:bob" {
		t.Fatalf("followers order = %v, want dana,carol,bob", got)
	}

	following, _, followingTotal, err := store.ListFollowing(ctx, "did:plc:alice", 10, "")
	if err != nil {
		t.Fatalf("ListFollowing: %v", err)
	}
	if followingTotal != 3 {
		t.Fatalf("following total = %d, want 3", followingTotal)
	}
	if got := []string{following[0].DID, following[1].DID, following[2].DID}; got[0] != "did:plc:dana" || got[1] != "did:plc:carol" || got[2] != "did:plc:bob" {
		t.Fatalf("following order = %v, want dana,carol,bob", got)
	}
}

func TestProfileStore_SocialSummaryIndexesCoverOrderedQueries(t *testing.T) {
	wantFragments := []string{
		"CREATE INDEX atproto_follows_subject_created_uri_desc_idx",
		"ON atproto_follows (subject_did, created_at DESC, uri DESC)",
		"CREATE INDEX atproto_follows_did_created_uri_desc_idx",
		"ON atproto_follows (did, created_at DESC, uri DESC)",
		"CREATE INDEX craftsky_posts_root_did_created_idx",
		"ON craftsky_posts (did, created_at DESC)",
		"WHERE reply_root_uri IS NULL AND reply_parent_uri IS NULL",
	}
	for _, fragment := range wantFragments {
		if !strings.Contains(profileStoreDDL, fragment) {
			t.Fatalf("profileStoreDDL missing index fragment %q", fragment)
		}
	}
}

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
