// appview/internal/api/profile_store_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const profileStoreDDL = `
CREATE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$ SELECT false $$;
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
    pronouns     TEXT,
    avatar_cid   TEXT,
    avatar_mime  TEXT,
    banner_cid   TEXT,
    banner_mime  TEXT,
    record_cid   TEXT        NOT NULL,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE tap_source_records (
    uri        TEXT PRIMARY KEY,
    rkey       TEXT        NOT NULL,
    cid        TEXT,
    updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE pds_set_sources (
    source_uri  TEXT PRIMARY KEY,
    kind        TEXT        NOT NULL,
    actor_did   TEXT        NOT NULL,
    scope_key   TEXT        NOT NULL,
    subject_did TEXT,
    activity_at TIMESTAMPTZ NOT NULL,
    eligible BOOLEAN NOT NULL
);
CREATE TABLE pds_set_aggregates (
    kind                      TEXT        NOT NULL,
    actor_did                 TEXT        NOT NULL,
    scope_key                 TEXT        NOT NULL,
	subject_did               TEXT,
	eligible_source_count     INTEGER     NOT NULL DEFAULT 1,
    representative_source_uri TEXT        NOT NULL,
    activated_at              TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (kind, actor_did, scope_key)
);
CREATE INDEX pds_set_aggregates_block_actor_pagination_idx
    ON pds_set_aggregates (actor_did, activated_at DESC, subject_did DESC)
    WHERE kind = 'block';
CREATE INDEX pds_set_aggregates_actor_purge_idx
    ON pds_set_aggregates (actor_did, kind, scope_key);
CREATE INDEX pds_set_aggregates_subject_did_purge_idx
    ON pds_set_aggregates (subject_did, kind, actor_did, scope_key)
    WHERE subject_did IS NOT NULL;
CREATE VIEW craftsky_profile_follower_counts AS
SELECT
    profile.did AS profile_did,
    COUNT(follower.did)::BIGINT AS follower_count
FROM craftsky_profiles profile
LEFT JOIN pds_set_aggregates follow
    ON follow.kind='follow' AND follow.subject_did = profile.did
    AND NOT appview_owner_is_terminal(follow.actor_did)
    AND NOT appview_owner_is_terminal(follow.subject_did)
LEFT JOIN craftsky_profiles follower
    ON follower.did = follow.actor_did
    AND NOT appview_owner_is_terminal(follower.did)
WHERE NOT appview_owner_is_terminal(profile.did)
GROUP BY profile.did;
CREATE TABLE craftsky_posts (
    uri              TEXT        NOT NULL PRIMARY KEY,
    did              TEXT        NOT NULL,
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
    is_project       BOOLEAN     NOT NULL DEFAULT false,
    project_craft_type TEXT,
    record           JSONB       NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL,
    indexed_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (did, rkey)
);
CREATE INDEX pds_set_aggregates_follow_subject_pagination_idx
    ON pds_set_aggregates(subject_did, activated_at DESC, representative_source_uri DESC)
    WHERE kind='follow';
CREATE INDEX pds_set_aggregates_follow_actor_pagination_idx
    ON pds_set_aggregates(actor_did, activated_at DESC, representative_source_uri DESC)
    WHERE kind='follow';
CREATE INDEX craftsky_posts_root_did_created_idx
    ON craftsky_posts (did, created_at DESC)
    WHERE reply_root_uri IS NULL AND reply_parent_uri IS NULL;
CREATE TABLE moderation_outputs (
    id                  TEXT        NOT NULL PRIMARY KEY,
    source_did          TEXT        NOT NULL,
    subject_type        TEXT        NOT NULL CHECK (subject_type IN ('post', 'account')),
    subject_did         TEXT        NOT NULL,
    subject_collection  TEXT,
    subject_rkey        TEXT,
    subject_uri         TEXT,
    value               TEXT        NOT NULL CHECK (value IN ('hide', 'takedown', 'warn')),
    action              TEXT        NOT NULL CHECK (action IN ('apply', 'negate')),
    internal_reason     TEXT,
    expires_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    indexed_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
` + profileRelationshipDDL

func TestProfileStore_ReadByDID_ProfileSummaryCountsRootPosts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
		"did:plc:alice",
	); err != nil {
		t.Fatalf("seed craftsky profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (
			uri, did, rkey, cid, text, reply_root_uri, reply_root_cid,
			reply_parent_uri, reply_parent_cid, is_project, project_craft_type, record, created_at
		)
		VALUES
			($1, 'did:plc:alice', 'root-recent', 'cid1', 'recent root', NULL, NULL, NULL, NULL, true, 'social.craftsky.feed.defs#knitting', '{}', $2),
			($3, 'did:plc:alice', 'root-old', 'cid2', 'old root', NULL, NULL, NULL, NULL, false, NULL, '{}', $4),
			($5, 'did:plc:alice', 'reply-recent', 'cid3', 'reply', $1, 'cid1', $1, 'cid1', true, 'social.craftsky.feed.defs#knitting', '{}', $2),
			($6, 'did:plc:alice', 'quote-recent', 'cid4', 'quote root', NULL, NULL, NULL, NULL, false, NULL, '{}', $2),
			($7, 'did:plc:alice', 'hidden-project', 'cid5', 'hidden project', NULL, NULL, NULL, NULL, true, 'social.craftsky.feed.defs#knitting', '{}', $2),
			($8, 'did:plc:alice', 'hidden-general', 'cid6', 'hidden general', NULL, NULL, NULL, NULL, false, NULL, '{}', $2)
	`,
		"at://did:plc:alice/social.craftsky.feed.post/root-recent", now.Add(-24*time.Hour),
		"at://did:plc:alice/social.craftsky.feed.post/root-old", now.Add(-8*24*time.Hour),
		"at://did:plc:alice/social.craftsky.feed.post/reply-recent",
		"at://did:plc:alice/social.craftsky.feed.post/quote-recent",
		"at://did:plc:alice/social.craftsky.feed.post/hidden-project",
		"at://did:plc:alice/social.craftsky.feed.post/hidden-general",
	); err != nil {
		t.Fatalf("seed posts: %v", err)
	}
	seedModerationOutput(t, pool, "post", "did:plc:alice", "at://did:plc:alice/social.craftsky.feed.post/hidden-project", "hide", now)
	seedModerationOutput(t, pool, "post", "did:plc:alice", "at://did:plc:alice/social.craftsky.feed.post/hidden-general", "hide", now)

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:alice", "did:plc:viewer")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.PostCount == nil || *got.PostCount != 2 {
		t.Fatalf("postCount = %v, want 2 non-project root posts", got.PostCount)
	}
	if got.PostsLast7Days == nil || *got.PostsLast7Days != 1 {
		t.Fatalf("postsLast7Days = %v, want 1 recent non-project root post", got.PostsLast7Days)
	}
	if got.ProjectCount == nil || *got.ProjectCount != 1 {
		t.Fatalf("projectCount = %v, want one visible top-level project", got.ProjectCount)
	}
}

func TestProfileStore_ReadByDID_HiddenAccountReturnsNotFound(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ('did:plc:bob', '{}', 'cid')`); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	seedModerationOutput(t, pool, "account", "did:plc:bob", "", "hide", time.Now())

	store := api.NewProfileStore(pool)
	_, err := store.Read(ctx, "did:plc:bob", "did:plc:viewer")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("want ErrProfileNotFound, got %v", err)
	}
}

func TestProfileStore_ReadByDID_HiddenCraftskyAccountWithBlueskyCacheReturnsNotFound(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ('did:plc:bob', '{}', 'cid')`); err != nil {
		t.Fatalf("seed craftsky profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES ('did:plc:bob', 'Bob', 'cid-bsky')`); err != nil {
		t.Fatalf("seed bluesky profile: %v", err)
	}
	seedModerationOutput(t, pool, "account", "did:plc:bob", "", "hide", time.Now())

	store := api.NewProfileStore(pool)
	_, err := store.Read(ctx, "did:plc:bob", "did:plc:viewer")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("want ErrProfileNotFound without non-Craftsky fallback, got %v", err)
	}
}

func TestProfileStore_ReadByDID_HiddenNonCraftskyAccountReturnsNotFound(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES ('did:plc:carol', 'Carol', 'cid-bsky')`); err != nil {
		t.Fatalf("seed bluesky profile: %v", err)
	}
	seedModerationOutput(t, pool, "account", "did:plc:carol", "", "takedown", time.Now())

	store := api.NewProfileStore(pool)
	_, err := store.Read(ctx, "did:plc:carol", "did:plc:viewer")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("want ErrProfileNotFound for hidden non-Craftsky profile, got %v", err)
	}
}

func TestProfileStore_ReadByDID_WarnedNonCraftskyAccountRemainsNotFound(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES ('did:plc:carol', 'Carol', 'cid-bsky')`); err != nil {
		t.Fatalf("seed bluesky profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs (id, source_did, subject_type, subject_did, value, action, internal_reason, created_at)
		VALUES ('warn-carol', 'did:plc:labeler', 'account', 'did:plc:carol', 'warn', 'apply', 'raw unsafe reason fixture', now())
	`); err != nil {
		t.Fatalf("seed moderation output: %v", err)
	}

	store := api.NewProfileStore(pool)
	_, err := store.Read(ctx, "did:plc:carol", "did:plc:viewer")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("warned non-member error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileStore_Read_AttachesWarningMetadataWithoutRawReason(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)

	if _, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`,
		"did:plc:bob",
	); err != nil {
		t.Fatalf("seed profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs (
			id, source_did, subject_type, subject_did, value, action, internal_reason, created_at, indexed_at
		)
		VALUES (
			'warn-profile', 'did:plc:labeler', 'account', 'did:plc:bob', 'warn', 'apply', 'raw unsafe reason fixture', $1, $1
		)`, now); err != nil {
		t.Fatalf("seed warning: %v", err)
	}

	row, err := api.NewProfileStore(pool).Read(ctx, "did:plc:bob", "did:plc:viewer")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if row.ModerationWarningKind == nil || *row.ModerationWarningKind != "profile" {
		t.Fatalf("ModerationWarningKind = %v, want profile", row.ModerationWarningKind)
	}

	out := api.BuildProfileResponse(row, "bob.example", true)
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if !strings.Contains(string(data), `"warningKind":"profile"`) {
		t.Fatalf("response missing profile warning: %s", data)
	}
	if strings.Contains(string(data), "raw unsafe reason fixture") || strings.Contains(string(data), "internalReason") {
		t.Fatalf("response leaked raw moderation reason: %s", data)
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
	for _, row := range []struct{ actor, subject, rkey string }{
		{"did:plc:viewer", "did:plc:mutual", "f1"},
		{"did:plc:mutual", "did:plc:profile", "f2"},
		{"did:plc:viewer", "did:plc:viewer-only", "f3"},
		{"did:plc:profile-only", "did:plc:profile", "f4"},
	} {
		seedFollowReadModel(t, pool, api.FollowRow{
			URI: "at://" + row.actor + "/app.bsky.graph.follow/" + row.rkey,
			DID: row.actor, Rkey: row.rkey, CID: "cid-" + row.rkey,
			SubjectDID: row.subject, CreatedAt: time.Now(),
		}, true)
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
		seedFollowReadModel(t, pool, api.FollowRow{
			URI: row.uri, DID: row.did, Rkey: row.rkey, CID: "cid",
			SubjectDID: row.subject, CreatedAt: row.created,
		}, true)
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
	for _, row := range []struct {
		actor, subject, rkey string
		created              time.Time
	}{
		{"did:plc:bob", "did:plc:alice", "f1", base.Add(-3 * time.Hour)},
		{"did:plc:carol", "did:plc:alice", "f2", base.Add(-2 * time.Hour)},
		{"did:plc:dana", "did:plc:alice", "f3", base.Add(-time.Hour)},
		{"did:plc:alice", "did:plc:bob", "f4", base.Add(-3 * time.Hour)},
		{"did:plc:alice", "did:plc:carol", "f5", base.Add(-2 * time.Hour)},
		{"did:plc:alice", "did:plc:dana", "f6", base.Add(-time.Hour)},
		{"did:plc:alice", "did:plc:erin", "f7", base.Add(-30 * time.Minute)},
	} {
		seedFollowReadModel(t, pool, api.FollowRow{
			URI: "at://" + row.actor + "/app.bsky.graph.follow/" + row.rkey,
			DID: row.actor, Rkey: row.rkey, CID: "cid-" + row.rkey,
			SubjectDID: row.subject, CreatedAt: row.created,
		}, true)
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
		t.Fatalf("following total = %d, want 3 Craftsky profiles only", followingTotal)
	}
	if got := []string{following[0].DID, following[1].DID, following[2].DID}; got[0] != "did:plc:dana" || got[1] != "did:plc:carol" || got[2] != "did:plc:bob" {
		t.Fatalf("following order = %v, want dana,carol,bob with non-Craftsky Erin excluded", got)
	}
}

func TestProfileStore_ListFollowingUsesAggregateActivationCursor(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	base := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)

	for _, did := range []string{
		"did:plc:viewer", "did:plc:newest", "did:plc:middle", "did:plc:oldest",
	} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, '{}', 'cid')`, did); err != nil {
			t.Fatalf("seed profile %s: %v", did, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO pds_set_aggregates(
			kind,actor_did,scope_key,subject_did,representative_source_uri,activated_at
		) VALUES
			('follow','did:plc:viewer','did:plc:newest','did:plc:newest','at://did:plc:viewer/app.bsky.graph.follow/z-representative',$1),
			('follow','did:plc:viewer','did:plc:middle','did:plc:middle','at://did:plc:viewer/app.bsky.graph.follow/m-representative',$2),
			('follow','did:plc:viewer','did:plc:oldest','did:plc:oldest','at://did:plc:viewer/app.bsky.graph.follow/a-representative',$3)
	`, base.Add(3*time.Minute), base.Add(2*time.Minute), base.Add(time.Minute)); err != nil {
		t.Fatalf("seed follow aggregates: %v", err)
	}

	store := api.NewProfileStore(pool)
	first, cursor, total, err := store.ListFollowing(ctx, "did:plc:viewer", 2, "")
	if err != nil {
		t.Fatalf("ListFollowing first page: %v", err)
	}
	if total != 3 || cursor == "" {
		t.Fatalf("first page total,cursor = %d,%q; want 3,non-empty", total, cursor)
	}
	if got := []string{first[0].DID, first[1].DID}; !slices.Equal(got, []string{"did:plc:newest", "did:plc:middle"}) {
		t.Fatalf("first page DIDs = %v", got)
	}
	if !first[0].FollowCreatedAt.Equal(base.Add(3*time.Minute)) || first[0].FollowURI != "at://did:plc:viewer/app.bsky.graph.follow/z-representative" {
		t.Fatalf("first row cursor fields = %s,%q", first[0].FollowCreatedAt, first[0].FollowURI)
	}

	second, next, total, err := store.ListFollowing(ctx, "did:plc:viewer", 2, cursor)
	if err != nil {
		t.Fatalf("ListFollowing second page: %v", err)
	}
	if total != 3 || next != "" || len(second) != 1 || second[0].DID != "did:plc:oldest" {
		t.Fatalf("second page = %+v total=%d next=%q", second, total, next)
	}
}

func TestProfileStore_SocialSummaryIndexesCoverOrderedQueries(t *testing.T) {
	wantFragments := []string{
		"CREATE INDEX pds_set_aggregates_actor_purge_idx",
		"ON pds_set_aggregates (actor_did, kind, scope_key)",
		"CREATE INDEX pds_set_aggregates_subject_did_purge_idx",
		"ON pds_set_aggregates (subject_did, kind, actor_did, scope_key)",
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
		INSERT INTO bluesky_profiles (did, display_name, pronouns, avatar_cid, avatar_mime, record_cid)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		"did:plc:a", "Alice", "she/her", "bafav", "image/jpeg", "cid2")
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
	if got.Pronouns == nil || *got.Pronouns != "she/her" {
		t.Errorf("Pronouns = %v", got.Pronouns)
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
	for _, row := range []struct{ actor, subject, rkey string }{
		{"did:plc:alice", "did:plc:bob", "f1"},
		{"did:plc:alice", "did:plc:dana", "f2"},
		{"did:plc:dana", "did:plc:bob", "f3"},
	} {
		seedFollowReadModel(t, pool, api.FollowRow{
			URI: "at://" + row.actor + "/app.bsky.graph.follow/" + row.rkey,
			DID: row.actor, Rkey: row.rkey, CID: "cid-" + row.rkey,
			SubjectDID: row.subject, CreatedAt: time.Now(),
		}, true)
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

func TestProfileStore_ReadByDID_NonCraftskyBlueskyCacheIsNotMembership(t *testing.T) {
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
	_, err := store.Read(ctx, "did:plc:carol", "did:plc:alice")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("non-member cache error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileStore_ReadByDID_NonCraftskyFollowDoesNotMakeMember(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, record_cid)
		VALUES ($1, $2, $3)
	`, "did:plc:carol", "Carol", "cid-bsky"); err != nil {
		t.Fatalf("seed bluesky profile: %v", err)
	}
	seedFollowReadModel(t, pool, api.FollowRow{
		URI: "at://did:plc:alice/app.bsky.graph.follow/f1", DID: "did:plc:alice",
		Rkey: "f1", CID: "cid-follow", SubjectDID: "did:plc:carol", CreatedAt: time.Now(),
	}, true)

	store := api.NewProfileStore(pool)
	_, err := store.Read(ctx, "did:plc:carol", "did:plc:alice")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("non-member follow error = %v, want ErrProfileNotFound", err)
	}
}

func TestProfileStore_ReadByDID_DoesNotHydrateNonCraftskyWhenCacheMisses(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	store := api.NewProfileStore(pool)
	_, err := store.Read(ctx, "did:plc:carol", "did:plc:alice")
	if !errors.Is(err, api.ErrProfileNotFound) {
		t.Fatalf("cache miss error = %v, want ErrProfileNotFound", err)
	}
}
