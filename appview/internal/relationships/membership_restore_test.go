package relationships

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/testdb"
)

func TestSubjectMembershipLossAndRejoinHideRetainAndRestoreRelationships(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	pool := testdb.WithSchema(t, relationshipStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO actor_mutes (owner_did, subject_did, created_at)
		VALUES ('did:plc:alice', 'did:plc:bob', '2026-07-19T12:00:00Z');
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES
		  ('at://did:plc:alice/app.bsky.graph.block/alice', 'did:plc:alice', 'alice', 'bafy-alice', 'did:plc:bob', '{}', '2026-07-19T12:00:01Z'),
		  ('at://did:plc:bob/app.bsky.graph.block/bob', 'did:plc:bob', 'bob', 'bafy-bob', 'did:plc:alice', '{}', '2026-07-19T12:00:02Z');
	`); err != nil {
		t.Fatalf("seed relationships: %v", err)
	}
	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = $1`, bob); err != nil {
		t.Fatalf("remove Bob membership: %v", err)
	}
	if err := RequireCurrentMember(ctx, store, bob); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("absent target membership = %v", err)
	}
	mutes, muteMore, err := store.ListMutes(ctx, alice, 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("list hidden mutes: %v", err)
	}
	blocks, blockMore, err := store.ListBlocks(ctx, alice, 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("list hidden blocks: %v", err)
	}
	if len(mutes) != 0 || muteMore || len(blocks) != 0 || blockMore {
		t.Fatalf("absent Bob surfaced: mutes=%+v more=%v blocks=%+v more=%v", mutes, muteMore, blocks, blockMore)
	}
	var muteRows, blockRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM actor_mutes`).Scan(&muteRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_blocks`).Scan(&blockRows); err != nil {
		t.Fatal(err)
	}
	if muteRows != 1 || blockRows != 2 {
		t.Fatalf("retained rows = mutes %d, blocks %d; want 1/2", muteRows, blockRows)
	}

	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`, bob, "bob-rejoined-cid"); err != nil {
		t.Fatalf("re-add Bob membership: %v", err)
	}
	mutes, muteMore, err = store.ListMutes(ctx, alice, 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("list restored mutes: %v", err)
	}
	blocks, blockMore, err = store.ListBlocks(ctx, alice, 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("list restored blocks: %v", err)
	}
	if len(mutes) != 1 || mutes[0].SubjectDID != bob || muteMore || len(blocks) != 1 || blocks[0].SubjectDID != bob || blockMore {
		t.Fatalf("restored lists = mutes %+v more=%v blocks %+v more=%v", mutes, muteMore, blocks, blockMore)
	}
	state, err := store.State(ctx, alice, bob)
	if err != nil {
		t.Fatalf("restored state: %v", err)
	}
	if !state.Muted || !state.Blocking || !state.BlockedBy {
		t.Fatalf("restored state = %+v", state)
	}
}
