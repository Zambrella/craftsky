package api_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const profileRelationshipDDL = `
CREATE TABLE actor_mutes (
    owner_did TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_did, subject_did)
);
`

func TestProfileStoreReadShapesViewerRelationshipsWithoutLeakingPrivateMute(t *testing.T) {
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, crafts, record_cid)
		VALUES
		  ('did:plc:alice', '{knitting}', 'alice-cid'),
		  ('did:plc:bob', '{sewing}', 'bob-cid'),
		  ('did:plc:carol', '{quilting}', 'carol-cid');
		INSERT INTO bluesky_profiles (did, display_name, description, avatar_cid, avatar_mime, record_cid)
		VALUES
		  ('did:plc:alice', 'Alice', 'Alice bio', 'baf-alice', 'image/jpeg', 'alice-bsky'),
		  ('did:plc:bob', 'Bob', 'Bob bio', 'baf-bob', 'image/jpeg', 'bob-bsky'),
		  ('did:plc:carol', 'Carol', 'Carol bio', 'baf-carol', 'image/jpeg', 'carol-bsky');
		INSERT INTO actor_mutes (owner_did, subject_did)
		VALUES ('did:plc:alice', 'did:plc:bob');
		INSERT INTO pds_set_aggregates (kind, actor_did, scope_key, subject_did, representative_source_uri, activated_at)
		VALUES
		  ('block', 'did:plc:alice', 'did:plc:bob', 'did:plc:bob', 'at://did:plc:alice/app.bsky.graph.block/alice', now()),
		  ('block', 'did:plc:bob', 'did:plc:alice', 'did:plc:alice', 'at://did:plc:bob/app.bsky.graph.block/bob', now());
	`); err != nil {
		t.Fatalf("seed profiles and relationships: %v", err)
	}
	store := api.NewProfileStore(pool)

	aliceView, err := store.Read(ctx, "did:plc:bob", "did:plc:alice")
	if err != nil {
		t.Fatalf("Alice reads Bob: %v", err)
	}
	if !aliceView.Muted || !aliceView.Blocking || !aliceView.BlockedBy {
		t.Fatalf("Alice relationship state = %+v", aliceView)
	}
	aliceWire, err := json.Marshal(api.BuildProfileResponse(aliceView, "bob.example", true))
	if err != nil {
		t.Fatal(err)
	}
	var aliceBody map[string]any
	_ = json.Unmarshal(aliceWire, &aliceBody)
	if _, leaked := aliceBody["description"]; leaked {
		t.Fatalf("blocked response leaked bio: %s", aliceWire)
	}

	bobView, err := store.Read(ctx, "did:plc:alice", "did:plc:bob")
	if err != nil {
		t.Fatalf("Bob reads Alice: %v", err)
	}
	if bobView.Muted || !bobView.Blocking || !bobView.BlockedBy {
		t.Fatalf("Bob relationship state leaked Alice mute or lost block: %+v", bobView)
	}

	carolView, err := store.Read(ctx, "did:plc:bob", "did:plc:carol")
	if err != nil {
		t.Fatalf("Carol reads Bob: %v", err)
	}
	if carolView.Muted || carolView.Blocking || carolView.BlockedBy {
		t.Fatalf("Carol received Alice relationship state: %+v", carolView)
	}
	carolWire, err := json.Marshal(api.BuildProfileResponse(carolView, "bob.example", true))
	if err != nil {
		t.Fatal(err)
	}
	var carolBody map[string]any
	_ = json.Unmarshal(carolWire, &carolBody)
	if carolBody["description"] != "Bob bio" {
		t.Fatalf("unrelated viewer did not receive full eligible profile: %s", carolWire)
	}
}

func TestProfileGraphHidesBlockedPairWithoutDeletingFollowsAndRestoresOnUnblock(t *testing.T) {
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES
		  ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid'),
		  ('did:plc:carol', 'carol-cid'), ('did:plc:dana', 'dana-cid');
		INSERT INTO pds_set_aggregates (kind, actor_did, scope_key, subject_did, representative_source_uri, activated_at)
		VALUES ('block', 'did:plc:alice', 'did:plc:bob', 'did:plc:bob', 'at://did:plc:alice/app.bsky.graph.block/bob', now());
	`); err != nil {
		t.Fatalf("seed graph: %v", err)
	}
	for index, pair := range []struct{ actor, subject, rkey string }{
		{"did:plc:alice", "did:plc:bob", "bob"},
		{"did:plc:bob", "did:plc:alice", "alice"},
		{"did:plc:alice", "did:plc:carol", "carol"},
		{"did:plc:carol", "did:plc:alice", "alice"},
		{"did:plc:carol", "did:plc:bob", "bob"},
		{"did:plc:dana", "did:plc:alice", "alice"},
	} {
		seedFollowReadModel(t, pool, api.FollowRow{
			URI: "at://" + pair.actor + "/app.bsky.graph.follow/" + pair.rkey,
			DID: pair.actor, Rkey: pair.rkey, CID: "cid-follow",
			SubjectDID: pair.subject, CreatedAt: time.Date(2026, 7, 19, 12, 0, index, 0, time.UTC),
		}, true)
	}
	store := api.NewProfileStore(pool)

	bobForAlice, err := store.Read(ctx, "did:plc:bob", "did:plc:alice")
	if err != nil {
		t.Fatalf("Alice reads Bob: %v", err)
	}
	if bobForAlice.ViewerIsFollowing {
		t.Fatal("blocked pair exposed Alice's stored follow state")
	}
	bobForCarol, err := store.Read(ctx, "did:plc:bob", "did:plc:carol")
	if err != nil {
		t.Fatalf("Carol reads Bob: %v", err)
	}
	if !bobForCarol.ViewerIsFollowing || bobForCarol.FollowerCount == nil || *bobForCarol.FollowerCount != 2 {
		t.Fatalf("unrelated Carol view = %+v, want record-based follow/count", bobForCarol)
	}

	following, _, followingTotal, err := store.ListFollowing(ctx, "did:plc:alice", 100, "")
	if err != nil {
		t.Fatalf("Alice following: %v", err)
	}
	if followingTotal != 1 || len(following) != 1 || following[0].DID != "did:plc:carol" {
		t.Fatalf("Alice following across block = total %d rows %+v", followingTotal, following)
	}
	followers, _, followerTotal, err := store.ListFollowers(ctx, "did:plc:alice", 100, "")
	if err != nil {
		t.Fatalf("Alice followers: %v", err)
	}
	if followerTotal != 2 || len(followers) != 2 || followers[0].DID != "did:plc:dana" || followers[1].DID != "did:plc:carol" {
		t.Fatalf("Alice followers across block = total %d rows %+v", followerTotal, followers)
	}
	mutuals, _, mutualTotal, err := store.ListMutualFollowers(ctx, "did:plc:alice", "did:plc:bob", 100, "")
	if err != nil {
		t.Fatalf("Alice/Bob mutuals: %v", err)
	}
	if mutualTotal != 0 || len(mutuals) != 0 {
		t.Fatalf("protected mutual list = total %d rows %+v", mutualTotal, mutuals)
	}
	var stored int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_set_sources WHERE kind='follow'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 6 {
		t.Fatalf("block rewrote follow storage: %d rows", stored)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM pds_set_aggregates WHERE kind = 'block'`); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	following, _, followingTotal, err = store.ListFollowing(ctx, "did:plc:alice", 100, "")
	if err != nil {
		t.Fatalf("Alice following after unblock: %v", err)
	}
	if followingTotal != 2 || len(following) != 2 || following[1].DID != "did:plc:bob" {
		t.Fatalf("restored following = total %d rows %+v", followingTotal, following)
	}
}
