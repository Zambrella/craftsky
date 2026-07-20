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
	"social.craftsky/appview/internal/testdb"
)

type recordingBlockPDS struct {
	listRecords []auth.PDSRecord
	listCursor  string
	listPages   map[string]recordingBlockPDSPage
	listCursors []string
	listErr     error
	listCalls   int
	createURI   syntax.ATURI
	createCID   syntax.CID
	createErr   error
	createCalls int
	createRepo  syntax.DID
	createNSID  string
	createValue any
	deleteRkeys []string
	deleteErrs  map[string]error
}

type recordingBlockPDSPage struct {
	records []auth.PDSRecord
	cursor  string
}

func (*recordingBlockPDS) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", errors.New("not implemented")
}
func (*recordingBlockPDS) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return errors.New("not implemented")
}
func (p *recordingBlockPDS) CreateRecord(_ context.Context, repo syntax.DID, collection string, record any) (syntax.ATURI, syntax.CID, error) {
	p.createCalls++
	p.createRepo, p.createNSID, p.createValue = repo, collection, record
	return p.createURI, p.createCID, p.createErr
}
func (p *recordingBlockPDS) DeleteRecord(_ context.Context, _ syntax.DID, _ string, rkey string) error {
	p.deleteRkeys = append(p.deleteRkeys, rkey)
	return p.deleteErrs[rkey]
}
func (*recordingBlockPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("not implemented")
}
func (p *recordingBlockPDS) ListRecords(_ context.Context, _ syntax.DID, _ string, cursor string, _ int) ([]auth.PDSRecord, string, error) {
	p.listCalls++
	p.listCursors = append(p.listCursors, cursor)
	if page, ok := p.listPages[cursor]; ok {
		return page.records, page.cursor, p.listErr
	}
	return p.listRecords, p.listCursor, p.listErr
}

func TestMutationServiceBlockWaitsForPDSAndDoesNotProjectLocally(t *testing.T) {
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
		func(_ context.Context, did syntax.DID, sid string) (auth.PDSClient, error) {
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
	if factoryCalls != 1 || pds.listCalls != 1 || pds.createCalls != 1 {
		t.Fatalf("factory/list/create calls = %d/%d/%d, want 1/1/1", factoryCalls, pds.listCalls, pds.createCalls)
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
	if !result.State.Blocking || result.State.BlockedBy || result.URI != pds.createURI || result.CID != pds.createCID || result.Rkey != syntax.RecordKey("block-1") {
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

func TestMutationServiceBlockRetryAndRapidUnblockReconcilePDSRecords(t *testing.T) {
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
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid'), ('did:plc:carol', 'carol-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	pds := &recordingBlockPDS{
		listPages: map[string]recordingBlockPDSPage{
			"": {
				records: []auth.PDSRecord{
					{URI: "at://did:plc:alice/app.bsky.graph.block/z-last", CID: "bafy-z", Value: &bsky.GraphBlock{Subject: bob.String()}},
					{URI: "at://did:plc:alice/app.bsky.graph.block/carol", CID: "bafy-carol", Value: &bsky.GraphBlock{Subject: "did:plc:carol"}},
				},
				cursor: "page-2",
			},
			"page-2": {
				records: []auth.PDSRecord{
					{URI: "at://did:plc:alice/app.bsky.graph.block/a-first", CID: "bafy-a", Value: &bsky.GraphBlock{Subject: bob.String()}},
				},
			},
		},
		deleteErrs: map[string]error{"z-last": auth.ErrRecordNotFound},
	}
	service := NewMutationService(
		NewStore(pool),
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		func() time.Time { return time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC) },
	)

	blocked, err := service.Block(ctx, alice, bob, "session-alice")
	if err != nil {
		t.Fatalf("retry Block: %v", err)
	}
	if pds.createCalls != 0 {
		t.Fatalf("retry Block created %d records, want 0", pds.createCalls)
	}
	if !blocked.State.Blocking || blocked.Rkey != syntax.RecordKey("a-first") {
		t.Fatalf("retry Block result = %+v, want deterministic existing record", blocked)
	}

	pds.listCursors = nil
	unblocked, err := service.Unblock(ctx, alice, bob, "session-alice")
	if err != nil {
		t.Fatalf("rapid Unblock: %v", err)
	}
	if unblocked.State.Blocking || unblocked.State.BlockedBy {
		t.Fatalf("rapid Unblock state = %+v", unblocked.State)
	}
	if got, want := pds.deleteRkeys, []string{"a-first", "z-last"}; !slices.Equal(got, want) {
		t.Fatalf("delete rkeys = %v, want %v", got, want)
	}
	if got, want := pds.listCursors, []string{"", "page-2"}; !slices.Equal(got, want) {
		t.Fatalf("list cursors = %v, want %v", got, want)
	}

	pds.deleteRkeys = nil
	if _, err := service.Unblock(ctx, alice, bob, "session-alice"); err != nil {
		t.Fatalf("retry Unblock: %v", err)
	}
	if got, want := pds.deleteRkeys, []string{"a-first", "z-last"}; !slices.Equal(got, want) {
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

func TestMutationServiceUnblockReconcilesIndexedAndPDSOnlyDuplicates(t *testing.T) {
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
	pds := &recordingBlockPDS{listRecords: []auth.PDSRecord{
		{URI: "at://did:plc:alice/app.bsky.graph.block/indexed-one", CID: "bafy-indexed", Value: &bsky.GraphBlock{Subject: bob.String()}},
		{URI: "at://did:plc:alice/app.bsky.graph.block/pds-only", CID: "bafy-pds-only", Value: &bsky.GraphBlock{Subject: bob.String()}},
	}}
	service := NewMutationService(
		NewStore(pool),
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil },
		nil,
	)

	if _, err := service.Unblock(ctx, alice, bob, "session-alice"); err != nil {
		t.Fatalf("Unblock: %v", err)
	}
	if pds.listCalls != 1 {
		t.Fatalf("PDS list calls = %d, want 1", pds.listCalls)
	}
	if got, want := pds.deleteRkeys, []string{"indexed-one", "pds-only"}; !slices.Equal(got, want) {
		t.Fatalf("delete rkeys = %v, want %v", got, want)
	}
}
