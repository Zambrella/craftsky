package relationships

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

const relationshipStorePreStateDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestStoreMuteIsOwnerScopedImmediateAndIdempotent(t *testing.T) {
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
		VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid'),
			('did:plc:carol', 'carol-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	carol := syntax.DID("did:plc:carol")

	if err := store.Mute(ctx, alice, bob); err != nil {
		t.Fatalf("mute Alice -> Bob: %v", err)
	}
	if err := store.Mute(ctx, alice, bob); err != nil {
		t.Fatalf("repeat mute Alice -> Bob: %v", err)
	}
	assertMuted(t, store, alice, bob, true)
	assertMuted(t, store, carol, bob, false)
	assertMuted(t, store, bob, alice, false)
	assertTableCount(t, pool, "actor_mutes", 1)

	if err := store.Mute(ctx, carol, bob); err != nil {
		t.Fatalf("mute Carol -> Bob: %v", err)
	}
	if err := store.Unmute(ctx, alice, bob); err != nil {
		t.Fatalf("unmute Alice -> Bob: %v", err)
	}
	if err := store.Unmute(ctx, alice, bob); err != nil {
		t.Fatalf("repeat unmute Alice -> Bob: %v", err)
	}
	assertMuted(t, store, alice, bob, false)
	assertMuted(t, store, carol, bob, true)
	assertTableCount(t, pool, "actor_mutes", 1)
}

func TestMutationServiceMuteRollsBackWhenDeliveryCancellationFails(t *testing.T) {
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
		CREATE TABLE notification_events (
			id UUID PRIMARY KEY,
			recipient_did TEXT NOT NULL,
			actor_did TEXT NOT NULL
		);
		CREATE TABLE push_deliveries (
			id UUID PRIMARY KEY,
			notification_id UUID NOT NULL REFERENCES notification_events(id),
			status TEXT NOT NULL,
			lease_owner TEXT,
			lease_expires_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		INSERT INTO notification_events (id, recipient_did, actor_did)
		VALUES ('00000000-0000-0000-0000-000000000001', 'did:plc:alice', 'did:plc:bob');
		INSERT INTO push_deliveries (id, notification_id, status)
		VALUES ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'pending');
		CREATE FUNCTION fail_relationship_delivery_cancel() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			RAISE EXCEPTION 'forced cancellation failure';
		END
		$$;
		CREATE TRIGGER fail_relationship_delivery_cancel
		BEFORE UPDATE ON push_deliveries
		FOR EACH ROW EXECUTE FUNCTION fail_relationship_delivery_cancel();
	`); err != nil {
		t.Fatalf("seed delivery failure fixture: %v", err)
	}

	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	observer := &mutationRelationshipObserver{}
	service := NewMutationService(store, nil, nil, observer)
	if _, err := service.Mute(ctx, alice, bob); err == nil {
		t.Fatal("Mute succeeded despite forced delivery cancellation failure")
	}
	muted, err := store.IsMuted(ctx, alice, bob)
	if err != nil {
		t.Fatalf("read mute after failure: %v", err)
	}
	if muted {
		t.Fatal("mute row committed despite delivery cancellation failure")
	}
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM push_deliveries`).Scan(&status); err != nil {
		t.Fatalf("read delivery after failure: %v", err)
	}
	if status != "pending" {
		t.Fatalf("delivery status = %q, want pending rollback", status)
	}
	if !observer.has("mute", "store", "error", "store") {
		t.Fatalf("missing bounded mute failure observation: %+v", observer.calls)
	}
}

func TestMutationServiceMuteObservesPushCancellation(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, relationshipStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid)
		VALUES('did:plc:alice','alice-cid'),('did:plc:bob','bob-cid');
		CREATE TABLE notification_events(id UUID PRIMARY KEY,recipient_did TEXT NOT NULL,actor_did TEXT NOT NULL);
		CREATE TABLE push_deliveries(
			id UUID PRIMARY KEY,notification_id UUID NOT NULL REFERENCES notification_events(id),
			status TEXT NOT NULL,lease_owner TEXT,lease_expires_at TIMESTAMPTZ,updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		INSERT INTO notification_events(id,recipient_did,actor_did)
		VALUES('00000000-0000-0000-0000-000000000001','did:plc:alice','did:plc:bob');
		INSERT INTO push_deliveries(id,notification_id,status)
		VALUES('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','pending');
	`); err != nil {
		t.Fatal(err)
	}
	observer := &mutationRelationshipObserver{}
	service := NewMutationService(NewStore(pool), nil, nil, observer)
	if _, err := service.Mute(ctx, "did:plc:alice", "did:plc:bob"); err != nil {
		t.Fatal(err)
	}
	if !observer.has("push_cancellation", "delivery", "some", "none") {
		t.Fatalf("missing mute push cancellation observation: %+v", observer.calls)
	}
}

type mutationRelationshipObservation struct {
	operation, stage, result, errorClass string
}

type mutationRelationshipObserver struct {
	calls []mutationRelationshipObservation
}

func (o *mutationRelationshipObserver) ObserveRelationship(operation, result string, _ time.Duration) {
	o.calls = append(o.calls, mutationRelationshipObservation{operation: operation, stage: "complete", result: result})
}

func (o *mutationRelationshipObserver) ObserveRelationshipOutcome(operation, stage, result, errorClass string, _ time.Duration) {
	o.calls = append(o.calls, mutationRelationshipObservation{operation: operation, stage: stage, result: result, errorClass: errorClass})
}

func (o *mutationRelationshipObserver) has(operation, stage, result, errorClass string) bool {
	for _, call := range o.calls {
		if call == (mutationRelationshipObservation{operation: operation, stage: stage, result: result, errorClass: errorClass}) {
			return true
		}
	}
	return false
}

func TestStoreStateDoesNotExposeAnotherOwnersMute(t *testing.T) {
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
		VALUES
			('did:plc:alice', 'alice-cid'),
			('did:plc:bob', 'bob-cid'),
			('did:plc:carol', 'carol-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	carol := syntax.DID("did:plc:carol")
	if err := store.Mute(ctx, alice, bob); err != nil {
		t.Fatalf("mute Alice -> Bob: %v", err)
	}

	aliceState, err := store.State(ctx, alice, bob)
	if err != nil {
		t.Fatalf("read Alice state: %v", err)
	}
	if !aliceState.Muted {
		t.Fatalf("Alice state = %+v, want muted", aliceState)
	}
	for name, pair := range map[string][2]syntax.DID{
		"Bob viewing Alice":   {bob, alice},
		"Carol viewing Bob":   {carol, bob},
		"Carol viewing Alice": {carol, alice},
	} {
		state, err := store.State(ctx, pair[0], pair[1])
		if err != nil {
			t.Fatalf("%s state: %v", name, err)
		}
		if state.Muted {
			t.Fatalf("%s leaked Alice's mute: %+v", name, state)
		}
	}
}

func TestStoreOwnedBlockRecordsReturnsOnlyCallerOwnedRecords(t *testing.T) {
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
		INSERT INTO atproto_blocks (
			uri, blocker_did, rkey, cid, subject_did, record, created_at
		) VALUES
			('at://did:plc:alice/app.bsky.graph.block/alice-block', 'did:plc:alice', 'alice-block', 'cid-alice', 'did:plc:bob', '{}', now()),
			('at://did:plc:bob/app.bsky.graph.block/bob-block', 'did:plc:bob', 'bob-block', 'cid-bob', 'did:plc:alice', '{}', now())
	`); err != nil {
		t.Fatalf("insert blocks: %v", err)
	}

	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	carol := syntax.DID("did:plc:carol")

	aliceRows, err := store.OwnedBlockRecords(ctx, alice, bob)
	if err != nil {
		t.Fatalf("Alice owned blocks: %v", err)
	}
	if len(aliceRows) != 1 || aliceRows[0].Rkey != syntax.RecordKey("alice-block") {
		t.Fatalf("Alice owned blocks = %+v, want only alice-block", aliceRows)
	}
	bobRows, err := store.OwnedBlockRecords(ctx, bob, alice)
	if err != nil {
		t.Fatalf("Bob owned blocks: %v", err)
	}
	if len(bobRows) != 1 || bobRows[0].Rkey != syntax.RecordKey("bob-block") {
		t.Fatalf("Bob owned blocks = %+v, want only bob-block", bobRows)
	}
	carolRows, err := store.OwnedBlockRecords(ctx, carol, bob)
	if err != nil {
		t.Fatalf("Carol owned blocks: %v", err)
	}
	if len(carolRows) != 0 {
		t.Fatalf("Carol enumerated foreign blocks: %+v", carolRows)
	}
}

func TestStoreRelationshipListsAreOwnerScopedEligibleStableAndDeduplicated(t *testing.T) {
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
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:carol', 'carol-cid')
	`); err != nil {
		t.Fatalf("insert owners: %v", err)
	}

	fixed := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	for i := range 112 {
		did := fmt.Sprintf("did:plc:user%03d", i)
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)
		`, did, fmt.Sprintf("cid-%03d", i)); err != nil {
			t.Fatalf("insert subject %d: %v", i, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO actor_mutes (owner_did, subject_did, created_at, updated_at)
			VALUES ('did:plc:alice', $1, $2, $2)
		`, did, fixed); err != nil {
			t.Fatalf("insert mute %d: %v", i, err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:user050'`); err != nil {
		t.Fatalf("remove former member: %v", err)
	}

	store := NewStore(pool)
	alice := syntax.DID("did:plc:alice")
	first, more, err := store.ListMutes(ctx, alice, 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("first mute page: %v", err)
	}
	if len(first) != 100 || !more {
		t.Fatalf("first mute page len/more = %d/%v, want 100/true", len(first), more)
	}
	last := first[len(first)-1]
	second, more, err := store.ListMutes(ctx, alice, 100, last.CreatedAt, last.SubjectDID)
	if err != nil {
		t.Fatalf("second mute page: %v", err)
	}
	if len(second) != 11 || more {
		t.Fatalf("second mute page len/more = %d/%v, want 11/false", len(second), more)
	}
	seen := make(map[syntax.DID]bool, 111)
	for _, item := range append(first, second...) {
		if item.SubjectDID == syntax.DID("did:plc:user050") {
			t.Fatal("former member appeared in mute list")
		}
		if seen[item.SubjectDID] {
			t.Fatalf("duplicate mute list subject %s", item.SubjectDID)
		}
		seen[item.SubjectDID] = true
	}
	if len(seen) != 111 {
		t.Fatalf("eligible mute union = %d, want 111", len(seen))
	}
	carolMutes, more, err := store.ListMutes(ctx, syntax.DID("did:plc:carol"), 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("Carol mute page: %v", err)
	}
	if len(carolMutes) != 0 || more {
		t.Fatalf("Carol enumerated Alice mutes: len/more=%d/%v", len(carolMutes), more)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES
			('at://did:plc:alice/app.bsky.graph.block/one', 'did:plc:alice', 'one', 'cid-one', 'did:plc:user001', '{}', $1),
			('at://did:plc:alice/app.bsky.graph.block/two', 'did:plc:alice', 'two', 'cid-two', 'did:plc:user001', '{}', $1),
			('at://did:plc:alice/app.bsky.graph.block/three', 'did:plc:alice', 'three', 'cid-three', 'did:plc:user002', '{}', $1),
			('at://did:plc:alice/app.bsky.graph.block/former', 'did:plc:alice', 'former', 'cid-former', 'did:plc:user050', '{}', $1)
	`, fixed); err != nil {
		t.Fatalf("insert block list fixtures: %v", err)
	}
	blocks, more, err := store.ListBlocks(ctx, alice, 100, time.Time{}, "")
	if err != nil {
		t.Fatalf("block page: %v", err)
	}
	if len(blocks) != 2 || more {
		t.Fatalf("block page len/more = %d/%v, want 2/false: %+v", len(blocks), more, blocks)
	}
	if blocks[0].SubjectDID == blocks[1].SubjectDID {
		t.Fatalf("duplicate external block pair was not collapsed: %+v", blocks)
	}
}

func assertMuted(t *testing.T, store *Store, owner, subject syntax.DID, want bool) {
	t.Helper()
	got, err := store.IsMuted(context.Background(), owner, subject)
	if err != nil {
		t.Fatalf("is muted %s -> %s: %v", owner, subject, err)
	}
	if got != want {
		t.Fatalf("is muted %s -> %s = %v, want %v", owner, subject, got, want)
	}
}

func assertTableCount(t *testing.T, pool *pgxpool.Pool, table string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s row count = %d, want %d", table, got, want)
	}
}
