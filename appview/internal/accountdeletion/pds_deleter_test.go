package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/testdb"
)

func TestPDSDeleterIsOwnerScopedAndNamespaceClosed(t *testing.T) {
	t.Parallel()

	capability := reflect.TypeOf((*auth.DeletionPDSClient)(nil)).Elem()
	if capability.NumMethod() != 2 {
		t.Fatalf("deletion PDS capability has %d methods, want exactly ListRecords and DeleteRecord", capability.NumMethod())
	}
	for _, name := range []string{"ListRecords", "DeleteRecord"} {
		if _, ok := capability.MethodByName(name); !ok {
			t.Fatalf("deletion PDS capability is missing %s", name)
		}
	}
	for _, forbidden := range []string{"DeleteAccount", "UploadBlob", "GetRecord", "PutRecord", "CreateRecord"} {
		if _, ok := capability.MethodByName(forbidden); ok {
			t.Fatalf("deletion PDS capability exposes forbidden method %s", forbidden)
		}
	}

	owner := syntax.DID("did:plc:alice")
	post := syntax.NSID("social.craftsky.feed.post")
	like := syntax.NSID("social.craftsky.feed.like")
	fake := &fakeDeletionPDS{
		owner: owner,
		records: map[syntax.NSID][]auth.PDSRecord{
			post: {
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/one")},
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/two")},
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/three")},
			},
			like: {{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/already-missing")}},
		},
		missingRKey: "already-missing",
	}
	registrar := &recordingExpectedRegistrar{registered: make(map[syntax.ATURI]bool)}
	fake.registrar = registrar
	deleter := NewPDSDeleter(fake, registrar, 2)

	result, err := deleter.DeleteAll(context.Background(), "job-1", owner)
	if err != nil {
		t.Fatal(err)
	}
	if result.Listed != 4 || result.DeleteCalls != 4 {
		t.Fatalf("deletion result = %+v, want four listed/delete attempts", result)
	}
	if len(fake.records[post]) != 0 || len(fake.records[like]) != 0 {
		t.Fatalf("records remain after convergent rescan: %#v", fake.records)
	}
	if fake.emptyCursorCalls[post] < 2 {
		t.Fatal("pagination mutation was not recovered by a collection rescan")
	}
	for _, collection := range fake.listedCollections {
		if !containsCraftskyCollection(collection) {
			t.Fatalf("listed non-CraftSky or unregistered collection %q", collection)
		}
	}
	if got := fake.listedCollections[len(fake.listedCollections)-1]; got != syntax.NSID("social.craftsky.actor.profile") {
		t.Fatalf("last listed collection = %q, want membership profile", got)
	}
}

type fakeDeletionPDS struct {
	owner             syntax.DID
	records           map[syntax.NSID][]auth.PDSRecord
	missingRKey       string
	registrar         *recordingExpectedRegistrar
	listedCollections []syntax.NSID
	emptyCursorCalls  map[syntax.NSID]int
}

func (fake *fakeDeletionPDS) ListRecords(_ context.Context, repo syntax.DID, collection string, cursor string, limit int) ([]auth.PDSRecord, string, error) {
	if repo != fake.owner {
		return nil, "", fmt.Errorf("wrong repo %s", repo)
	}
	nsid := syntax.NSID(collection)
	if !containsCraftskyCollection(nsid) {
		return nil, "", fmt.Errorf("unregistered collection %s", collection)
	}
	fake.listedCollections = append(fake.listedCollections, nsid)
	if fake.emptyCursorCalls == nil {
		fake.emptyCursorCalls = make(map[syntax.NSID]int)
	}
	if cursor == "" {
		fake.emptyCursorCalls[nsid]++
	}
	offset := 0
	if cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil {
			return nil, "", err
		}
		offset = parsed
	}
	records := fake.records[nsid]
	if offset >= len(records) {
		return nil, "", nil
	}
	end := offset + limit
	if end > len(records) {
		end = len(records)
	}
	page := append([]auth.PDSRecord(nil), records[offset:end]...)
	next := ""
	if end < len(records) {
		next = strconv.Itoa(end)
	}
	return page, next, nil
}

func (fake *fakeDeletionPDS) DeleteRecord(_ context.Context, repo syntax.DID, collection string, rkey string) error {
	if repo != fake.owner {
		return fmt.Errorf("wrong repo %s", repo)
	}
	uri := syntax.ATURI(fmt.Sprintf("at://%s/%s/%s", repo, collection, rkey))
	if !fake.registrar.registered[uri] {
		return fmt.Errorf("record %s was deleted before expected registration", uri)
	}
	nsid := syntax.NSID(collection)
	records := fake.records[nsid]
	for index, record := range records {
		if record.URI == uri {
			fake.records[nsid] = append(records[:index], records[index+1:]...)
			break
		}
	}
	if rkey == fake.missingRKey {
		return auth.ErrRecordNotFound
	}
	return nil
}

type recordingExpectedRegistrar struct {
	registered map[syntax.ATURI]bool
}

func (registrar *recordingExpectedRegistrar) RegisterExpected(_ context.Context, _ string, _ syntax.DID, uri syntax.ATURI, _ syntax.NSID) error {
	registrar.registered[uri] = true
	return nil
}

func containsCraftskyCollection(collection syntax.NSID) bool {
	for _, candidate := range CraftskyRecordCollections() {
		if collection == candidate {
			return true
		}
	}
	return false
}

var _ auth.DeletionPDSClient = (*fakeDeletionPDS)(nil)

func TestPDSDeleterPersistsExpectedRecordsAndPreservesOtherData(t *testing.T) {
	t.Parallel()

	up, err := os.ReadFile("../../migrations/000037_account_deletion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, accountDeletionStorePreStateDDL)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	owner := syntax.DID("did:plc:alice")
	jobID := uuid.MustParse("10000000-0000-0000-0000-000000000031")
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_deletion_operations(id,owner_did,state,phase,accepted_at)
		VALUES($1,$2,'active','removingCraftskyRecords',$3)
	`, jobID, owner, time.Date(2026, 8, 10, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}

	pds := &durableBoundaryPDS{
		pool:  pool,
		jobID: jobID,
		owner: owner,
		craftsky: []auth.PDSRecord{
			{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/one"), Value: map[string]any{"image": "blob-shared"}},
			{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/two")},
		},
		nonCraftsky: []syntax.ATURI{
			syntax.ATURI("at://did:plc:alice/app.bsky.feed.post/preserve"),
		},
		sharedBlobReferences: 2,
	}
	store := NewStore(pool, func() time.Time { return time.Date(2026, 8, 10, 15, 1, 0, 0, time.UTC) })
	deleter := NewPDSDeleter(pds, store, 1)
	result, err := deleter.DeleteAll(ctx, jobID.String(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if result.Listed != 2 || len(pds.craftsky) != 0 {
		t.Fatalf("first deletion result = %+v remaining = %#v", result, pds.craftsky)
	}
	result, err = deleter.DeleteAll(ctx, jobID.String(), owner)
	if err != nil || result.Listed != 0 {
		t.Fatalf("repeat deletion = (%+v, %v)", result, err)
	}
	if len(pds.nonCraftsky) != 1 || pds.sharedBlobReferences != 2 {
		t.Fatalf("preservation controls changed: nonCraftsky=%v blobRefs=%d", pds.nonCraftsky, pds.sharedBlobReferences)
	}
	assertJobRowCount(t, pool, "account_deletion_expected_records", jobID, 2)
}

type durableBoundaryPDS struct {
	pool                 *pgxpool.Pool
	jobID                uuid.UUID
	owner                syntax.DID
	craftsky             []auth.PDSRecord
	nonCraftsky          []syntax.ATURI
	sharedBlobReferences int
}

func (pds *durableBoundaryPDS) ListRecords(_ context.Context, repo syntax.DID, collection string, _ string, limit int) ([]auth.PDSRecord, string, error) {
	if repo != pds.owner || !containsCraftskyCollection(syntax.NSID(collection)) {
		return nil, "", errors.New("out-of-bound list")
	}
	var records []auth.PDSRecord
	for _, record := range pds.craftsky {
		if record.URI.Collection() == syntax.NSID(collection) {
			records = append(records, record)
			if len(records) == limit {
				break
			}
		}
	}
	return records, "", nil
}

func (pds *durableBoundaryPDS) DeleteRecord(ctx context.Context, repo syntax.DID, collection string, rkey string) error {
	uri := syntax.ATURI(fmt.Sprintf("at://%s/%s/%s", repo, collection, rkey))
	var exists bool
	if err := pds.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM account_deletion_expected_records WHERE job_id=$1 AND uri=$2)
	`, pds.jobID, uri).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New("delete called before durable expected record")
	}
	for index, record := range pds.craftsky {
		if record.URI == uri {
			pds.craftsky = append(pds.craftsky[:index], pds.craftsky[index+1:]...)
			return nil
		}
	}
	return auth.ErrRecordNotFound
}

var _ auth.DeletionPDSClient = (*durableBoundaryPDS)(nil)
