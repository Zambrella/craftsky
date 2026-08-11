package accountdeletion

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
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
	profile := syntax.NSID("social.craftsky.actor.profile")
	fake := &fakeDeletionPDS{
		owner: owner,
		records: map[syntax.NSID][]auth.PDSRecord{
			post: {
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/one")},
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/two")},
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/three")},
			},
			like:    {{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/already-missing")}},
			profile: {{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.actor.profile/self")}},
		},
		missingRKey: "already-missing",
	}
	deleter := NewPDSDeleter(fake, 2)

	result, err := deleter.DeleteAll(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if result.Listed != 5 || result.DeleteCalls != 5 {
		t.Fatalf("deletion result = %+v, want five listed/delete attempts", result)
	}
	if len(fake.records[post]) != 0 || len(fake.records[like]) != 0 || len(fake.records[profile]) != 0 {
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
	if got := fake.deletedCollections[len(fake.deletedCollections)-1]; got != profile {
		t.Fatalf("last deleted collection = %q, want membership profile; order=%v", got, fake.deletedCollections)
	}
}

func TestPDSDeleterReplaysWithoutPerRecordPersistenceAndPreservesOtherData(t *testing.T) {
	t.Parallel()

	owner := syntax.DID("did:plc:alice")
	pds := &fakeDeletionPDS{
		owner: owner,
		records: map[syntax.NSID][]auth.PDSRecord{
			syntax.NSID("social.craftsky.feed.post"): {
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/one")},
				{URI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/two")},
			},
		},
		nonCraftsky: []syntax.ATURI{
			syntax.ATURI("at://did:plc:alice/app.bsky.feed.post/preserve"),
		},
		sharedBlobReferences: 2,
		failAfterDeletes:     1,
	}
	deleter := NewPDSDeleter(pds, 1)
	if _, err := deleter.DeleteAll(context.Background(), owner); err == nil {
		t.Fatal("first deletion unexpectedly survived injected interruption")
	}
	if got := len(pds.records[syntax.NSID("social.craftsky.feed.post")]); got != 1 {
		t.Fatalf("remaining after interruption = %d, want 1", got)
	}

	pds.failAfterDeletes = 0
	result, err := deleter.DeleteAll(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if result.Listed != 1 || len(pds.records[syntax.NSID("social.craftsky.feed.post")]) != 0 {
		t.Fatalf("replayed deletion result = %+v remaining = %#v", result, pds.records)
	}
	if len(pds.nonCraftsky) != 1 || pds.sharedBlobReferences != 2 {
		t.Fatalf("preservation controls changed: nonCraftsky=%v blobRefs=%d", pds.nonCraftsky, pds.sharedBlobReferences)
	}
}

type fakeDeletionPDS struct {
	owner                syntax.DID
	records              map[syntax.NSID][]auth.PDSRecord
	missingRKey          string
	listedCollections    []syntax.NSID
	deletedCollections   []syntax.NSID
	emptyCursorCalls     map[syntax.NSID]int
	nonCraftsky          []syntax.ATURI
	sharedBlobReferences int
	failAfterDeletes     int
	deleteCalls          int
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
	if repo != fake.owner || !containsCraftskyCollection(syntax.NSID(collection)) {
		return errors.New("out-of-bound delete")
	}
	uri := syntax.ATURI(fmt.Sprintf("at://%s/%s/%s", repo, collection, rkey))
	nsid := syntax.NSID(collection)
	fake.deletedCollections = append(fake.deletedCollections, nsid)
	records := fake.records[nsid]
	for index, record := range records {
		if record.URI == uri {
			fake.records[nsid] = append(records[:index], records[index+1:]...)
			break
		}
	}
	fake.deleteCalls++
	if fake.failAfterDeletes > 0 && fake.deleteCalls == fake.failAfterDeletes {
		return errors.New("injected interruption")
	}
	if rkey == fake.missingRKey {
		return auth.ErrRecordNotFound
	}
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
