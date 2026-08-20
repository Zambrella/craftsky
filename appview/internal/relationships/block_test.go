package relationships

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
)

type recordingBlockPDS struct {
	createURI    syntax.ATURI
	createCID    syntax.CID
	createErr    error
	createCalls  int
	createRepo   syntax.DID
	createNSID   string
	createValue  any
	deleteRkeys  []string
	deleteCIDs   []syntax.CID
	deleteErrs   map[string]error
	resolveCalls int
	resolveErr   error
	expected     []ownerlifecycle.ExpectedOwner
	readRecord   *bsky.GraphBlock
	readCID      syntax.CID
	readErr      error
}

type recordingSafetyRestoration struct {
	pairs [][2]syntax.DID
	err   error
}

func (r *recordingSafetyRestoration) EnqueueRelationshipSafetyRestoration(
	_ context.Context,
	owner, subject syntax.DID,
) error {
	r.pairs = append(r.pairs, [2]syntax.DID{owner, subject})
	return r.err
}

func (p *recordingBlockPDS) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	p.resolveCalls++
	if p.resolveErr != nil {
		return nil, p.resolveErr
	}
	p.expected = []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:alice", Generation: ownerGeneration}}
	for _, target := range targets {
		p.expected = append(p.expected, ownerlifecycle.ExpectedOwner{Owner: target, Generation: 1})
	}
	return append([]ownerlifecycle.ExpectedOwner(nil), p.expected...), nil
}

func (p *recordingBlockPDS) ReadRecord(_ context.Context, _ pdseffects.ReadRecordRequest, out any) (syntax.CID, error) {
	if p.readErr != nil {
		return "", p.readErr
	}
	if p.readRecord == nil {
		return "", auth.ErrRecordNotFound
	}
	*(out.(*bsky.GraphBlock)) = *p.readRecord
	return p.readCID, nil
}

func (p *recordingBlockPDS) PutRecord(_ context.Context, request pdseffects.PutRecordRequest) (pdseffects.RecordResult, error) {
	p.createCalls++
	p.createRepo, p.createNSID, p.createValue = request.Owner, request.Collection.String(), request.Record
	if p.createErr != nil {
		return pdseffects.RecordResult{}, p.createErr
	}
	uri := syntax.ATURI("at://" + request.Owner.String() + "/" + request.Collection.String() + "/" + request.Rkey.String())
	return pdseffects.RecordResult{URI: uri, CID: p.createCID}, nil
}

func (p *recordingBlockPDS) DeleteRecord(_ context.Context, request pdseffects.DeleteRecordRequest) (pdseffects.RecordResult, error) {
	p.deleteRkeys = append(p.deleteRkeys, request.Rkey.String())
	p.deleteCIDs = append(p.deleteCIDs, request.ExpectedCID)
	return pdseffects.RecordResult{}, p.deleteErrs[request.Rkey.String()]
}

func (*recordingBlockPDS) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	return nil, errors.New("not implemented")
}

func blockEffectsFactory(executor *recordingBlockPDS) pdseffects.ExecutorFactory {
	return func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return executor, nil
	}
}

func TestMutationServiceBlockWaitsForPDSAndDoesNotProjectLocally(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	pool := testdb.WithSchema(t, relationshipStorePreStateDDL)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	pds := &recordingBlockPDS{
		createURI: "at://did:plc:alice/app.bsky.graph.block/block-1",
		createCID: "bafyblock1",
	}
	observer := &mutationRelationshipObserver{}
	factoryCalls := 0
	service := NewMutationService(
		NewStore(pool),
		func(_ context.Context, did syntax.DID, sid string) (pdseffects.EffectExecutor, error) {
			factoryCalls++
			if did != alice || sid != "session-alice" {
				t.Fatalf("factory did/sid = %s/%s", did, sid)
			}
			return pds, nil
		},
		func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC) },
		observer,
	)

	result, err := service.Block(ctx, alice, bob, "session-alice")
	if err != nil {
		t.Fatalf("Block: %v", err)
	}
	if factoryCalls != 1 || pds.resolveCalls != 1 || pds.createCalls != 1 {
		t.Fatalf("factory/resolve/put calls = %d/%d/%d, want 1/1/1", factoryCalls, pds.resolveCalls, pds.createCalls)
	}
	if pds.createRepo != alice || pds.createNSID != "app.bsky.graph.block" {
		t.Fatalf("create repo/collection = %s/%s", pds.createRepo, pds.createNSID)
	}
	record, ok := pds.createValue.(*bsky.GraphBlock)
	if !ok {
		t.Fatalf("create record type = %T, want *bsky.GraphBlock", pds.createValue)
	}
	if record.Subject != bob.String() || record.CreatedAt != "2026-07-19T12:00:00Z" {
		t.Fatalf("create record = %+v", record)
	}
	if !result.State.Blocking || result.State.BlockedBy || result.CID != pds.createCID ||
		result.Rkey == "" || result.URI.RecordKey() != result.Rkey {
		t.Fatalf("result = %+v", result)
	}
	var projected int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_blocks`).Scan(&projected); err != nil {
		t.Fatalf("count local blocks: %v", err)
	}
	if projected != 0 {
		t.Fatalf("API block synchronously projected %d rows, want 0", projected)
	}

	pds.createErr = errors.New("PDS write failed")
	pds.createCalls = 0
	if _, err := service.Block(ctx, alice, bob, "session-alice"); err == nil {
		t.Fatal("Block succeeded after PDS failure")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_blocks`).Scan(&projected); err != nil {
		t.Fatalf("count local blocks after failure: %v", err)
	}
	if projected != 0 {
		t.Fatalf("failed API block projected %d rows, want 0", projected)
	}
	if !observer.has("block", "pds", "error", "pds") {
		t.Fatalf("missing bounded PDS failure observation: %+v", observer.calls)
	}
}

func TestMutationServiceBlockAndRapidUnblockUseTheSameDeterministicRecordKey(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	pool := testdb.WithSchema(t, relationshipStorePreStateDDL)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid'), ('did:plc:carol', 'carol-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	pds := &recordingBlockPDS{createCID: "bafy-block"}
	service := NewMutationService(
		NewStore(pool),
		blockEffectsFactory(pds),
		func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC) },
	)

	blocked, err := service.Block(ctx, alice, bob, "session-alice")
	if err != nil {
		t.Fatalf("Block: %v", err)
	}
	if pds.createCalls != 1 || blocked.Rkey == "" {
		t.Fatalf("put calls/result = %d/%+v", pds.createCalls, blocked)
	}
	if !blocked.State.Blocking {
		t.Fatalf("Block result = %+v", blocked)
	}

	unblocked, err := service.Unblock(ctx, alice, bob, "session-alice")
	if err != nil {
		t.Fatalf("rapid Unblock: %v", err)
	}
	if unblocked.State.Blocking || unblocked.State.BlockedBy {
		t.Fatalf("rapid Unblock state = %+v", unblocked.State)
	}
	if got, want := pds.deleteRkeys, []string{blocked.Rkey.String()}; !slices.Equal(got, want) {
		t.Fatalf("delete rkeys = %v, want %v", got, want)
	}

	pds.deleteRkeys = nil
	if _, err := service.Unblock(ctx, alice, bob, "session-alice"); err != nil {
		t.Fatalf("retry Unblock: %v", err)
	}
	if got, want := pds.deleteRkeys, []string{blocked.Rkey.String()}; !slices.Equal(got, want) {
		t.Fatalf("retry delete rkeys = %v, want %v", got, want)
	}
	var projected int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_blocks`).Scan(&projected); err != nil {
		t.Fatalf("count local blocks: %v", err)
	}
	if projected != 0 {
		t.Fatalf("rapid block/unblock synchronously projected %d rows, want 0", projected)
	}
}

func TestMutationServiceUnmuteAndUnblockEnqueueSafetyRestoration(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, relationshipStorePreStateDDL)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ($1, 'alice-cid'), ($2, 'bob-cid')
	`, alice, bob); err != nil {
		t.Fatal(err)
	}
	store := NewStore(pool)
	if _, err := pool.Exec(ctx, `
		INSERT INTO actor_mutes (owner_did, subject_did) VALUES ($1, $2)
	`, alice, bob); err != nil {
		t.Fatal(err)
	}
	restoration := &recordingSafetyRestoration{}
	service := NewMutationServiceWithRestoration(
		store,
		blockEffectsFactory(&recordingBlockPDS{}),
		nil,
		restoration,
	)

	if _, err := service.Unmute(ctx, alice, bob); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Unblock(ctx, alice, bob, "session-alice"); err != nil {
		t.Fatal(err)
	}
	want := [][2]syntax.DID{{alice, bob}, {alice, bob}}
	if !slices.Equal(restoration.pairs, want) {
		t.Fatalf("restoration pairs=%v, want %v", restoration.pairs, want)
	}
}

func TestMutationServiceUnblockDeletesIndexedCIDAndDeterministicCleanupKey(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000023_mutes_blocks.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	pool := testdb.WithSchema(t, relationshipStorePreStateDDL)
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	if _, err := pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES (
			'at://did:plc:alice/app.bsky.graph.block/indexed-one',
			'did:plc:alice', 'indexed-one', 'bafy-indexed', 'did:plc:bob',
			'{"subject":"did:plc:bob"}', '2026-07-19T12:00:00Z'
		)
	`); err != nil {
		t.Fatalf("seed indexed block: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	pds := &recordingBlockPDS{}
	service := NewMutationService(
		NewStore(pool),
		blockEffectsFactory(pds),
		nil,
	)

	if _, err := service.Unblock(ctx, alice, bob, "session-alice"); err != nil {
		t.Fatalf("Unblock: %v", err)
	}
	if got := pds.deleteRkeys; len(got) != 2 || got[0] == "indexed-one" || got[1] != "indexed-one" {
		t.Fatalf("delete rkeys = %v, want deterministic key then indexed-one", got)
	}
	if got, want := pds.deleteCIDs, []syntax.CID{"", "bafy-indexed"}; !slices.Equal(got, want) {
		t.Fatalf("delete expected CIDs = %v, want %v", got, want)
	}
}
