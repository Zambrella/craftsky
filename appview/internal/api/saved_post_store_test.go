package api_test

import (
	"context"
	"errors"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const savedPostStorePreStateDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT NOT NULL PRIMARY KEY,
    record_cid  TEXT NOT NULL
);
CREATE TABLE craftsky_posts (
    uri         TEXT NOT NULL PRIMARY KEY,
    did         TEXT NOT NULL,
    rkey        TEXT NOT NULL,
    cid         TEXT NOT NULL
);
`

const savedPostContextDDL = `
CREATE TABLE craftsky_profiles (
    did TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE craftsky_posts (
    uri TEXT NOT NULL PRIMARY KEY,
    did TEXT NOT NULL,
    rkey TEXT NOT NULL,
    cid TEXT NOT NULL,
    reply_root_uri TEXT,
    reply_parent_uri TEXT
);
CREATE TABLE actor_mutes (
    owner_did TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    PRIMARY KEY (owner_did, subject_did)
);
CREATE TABLE atproto_blocks (
    uri TEXT NOT NULL PRIMARY KEY,
    blocker_did TEXT NOT NULL,
    subject_did TEXT NOT NULL
);
CREATE TABLE moderation_outputs (
    id UUID NOT NULL PRIMARY KEY,
    source_did TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_uri TEXT,
    value TEXT NOT NULL,
    action TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

func TestSavedPostStorePersistsTriStateOwnerScopedSave(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostStorePreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		VALUES ('at://did:plc:bob/social.craftsky.feed.post/one', 'did:plc:bob', 'one', 'post-cid');
		INSERT INTO saved_post_folders (id, owner_did, name, created_at, updated_at)
		VALUES
			('00000000-0000-4000-8000-000000000001', 'did:plc:alice', 'A', '2026-07-20T09:00:00Z', '2026-07-20T09:00:00Z'),
			('00000000-0000-4000-8000-000000000002', 'did:plc:alice', 'B', '2026-07-20T09:00:00Z', '2026-07-20T09:00:00Z'),
			('00000000-0000-4000-8000-000000000003', 'did:plc:bob', 'Bob', '2026-07-20T09:00:00Z', '2026-07-20T09:00:00Z');
	`); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	postURI := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/one")
	folderA := "00000000-0000-4000-8000-000000000001"
	folderB := "00000000-0000-4000-8000-000000000002"
	bobFolder := "00000000-0000-4000-8000-000000000003"
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	store := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time { return now }})

	created, err := store.Save(ctx, alice, postURI, api.FolderAssignment{})
	if err != nil {
		t.Fatalf("create unfiled save: %v", err)
	}
	if !created.Created || created.Changed || !created.State.SavedAt.Equal(now) || created.State.FolderID != nil {
		t.Fatalf("created result = %+v", created)
	}

	repeated, err := store.Save(ctx, alice, postURI, api.FolderAssignment{})
	if err != nil {
		t.Fatalf("repeat omitted save: %v", err)
	}
	if repeated.Created || repeated.Changed || !repeated.State.SavedAt.Equal(now) || repeated.State.FolderID != nil {
		t.Fatalf("repeated result = %+v", repeated)
	}

	movedA, err := store.Save(ctx, alice, postURI, api.FolderAssignment{Present: true, ID: &folderA})
	if err != nil {
		t.Fatalf("move to folder A: %v", err)
	}
	if movedA.Created || !movedA.Changed || !movedA.State.SavedAt.Equal(now) || movedA.State.FolderID == nil || *movedA.State.FolderID != folderA {
		t.Fatalf("move A result = %+v", movedA)
	}

	omitted, err := store.Save(ctx, alice, postURI, api.FolderAssignment{})
	if err != nil {
		t.Fatalf("repeat foldered save with omission: %v", err)
	}
	if omitted.Changed || omitted.State.FolderID == nil || *omitted.State.FolderID != folderA {
		t.Fatalf("omitted result = %+v", omitted)
	}

	movedB, err := store.Save(ctx, alice, postURI, api.FolderAssignment{Present: true, ID: &folderB})
	if err != nil {
		t.Fatalf("move to folder B: %v", err)
	}
	if !movedB.Changed || movedB.State.FolderID == nil || *movedB.State.FolderID != folderB || !movedB.State.SavedAt.Equal(now) {
		t.Fatalf("move B result = %+v", movedB)
	}

	for name, invalidFolder := range map[string]string{
		"missing":   "00000000-0000-4000-8000-000000000099",
		"foreign":   bobFolder,
		"malformed": "not-storage-shaped",
	} {
		t.Run(name+" folder", func(t *testing.T) {
			_, err := store.Save(ctx, alice, postURI, api.FolderAssignment{Present: true, ID: &invalidFolder})
			if !errors.Is(err, api.ErrSavedPostFolderNotFound) {
				t.Fatalf("Save error = %v, want folder not found", err)
			}
			state, err := store.ReadState(ctx, alice, postURI)
			if err != nil {
				t.Fatalf("read unchanged state: %v", err)
			}
			if state.FolderID == nil || *state.FolderID != folderB || !state.SavedAt.Equal(now) {
				t.Fatalf("invalid assignment changed state: %+v", state)
			}
		})
	}

	unfiled, err := store.Save(ctx, alice, postURI, api.FolderAssignment{Present: true})
	if err != nil {
		t.Fatalf("explicitly unfile: %v", err)
	}
	if !unfiled.Changed || unfiled.State.FolderID != nil || !unfiled.State.SavedAt.Equal(now) {
		t.Fatalf("unfile result = %+v", unfiled)
	}

	bobResult, err := store.Save(ctx, bob, postURI, api.FolderAssignment{Present: true, ID: &bobFolder})
	if err != nil {
		t.Fatalf("Bob save same post: %v", err)
	}
	if !bobResult.Created || bobResult.State.FolderID == nil || *bobResult.State.FolderID != bobFolder {
		t.Fatalf("Bob result = %+v", bobResult)
	}
	var saveCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM saved_posts WHERE post_uri = $1`, postURI).Scan(&saveCount); err != nil {
		t.Fatalf("count owner saves: %v", err)
	}
	if saveCount != 2 {
		t.Fatalf("save count = %d, want 2 owner-isolated rows", saveCount)
	}

	if err := store.Unsave(ctx, alice, postURI); err != nil {
		t.Fatalf("unsave: %v", err)
	}
	if err := store.Unsave(ctx, alice, postURI); err != nil {
		t.Fatalf("repeat unsave: %v", err)
	}
	if _, err := store.ReadState(ctx, alice, postURI); !errors.Is(err, api.ErrSavedPostNotFound) {
		t.Fatalf("read after unsave = %v, want saved post not found", err)
	}

	now = now.Add(time.Hour)
	resaved, err := store.Save(ctx, alice, postURI, api.FolderAssignment{})
	if err != nil {
		t.Fatalf("resave: %v", err)
	}
	if !resaved.Created || !resaved.State.SavedAt.Equal(now) || !resaved.State.SavedAt.After(created.State.SavedAt) {
		t.Fatalf("resaved result = %+v, original = %+v", resaved, created)
	}
}

func TestSavedPostStoreCreatesRenamesAndListsDuplicateFolders(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostStorePreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		VALUES ('at://did:plc:bob/social.craftsky.feed.post/one', 'did:plc:bob', 'one', 'post-cid');
	`); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	store := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time { return now }})

	ideasOne, err := store.CreateFolder(ctx, alice, "Ideas")
	if err != nil {
		t.Fatalf("create Ideas: %v", err)
	}
	ideasTwo, err := store.CreateFolder(ctx, alice, "Ideas")
	if err != nil {
		t.Fatalf("create duplicate Ideas: %v", err)
	}
	upper, err := store.CreateFolder(ctx, alice, "IDEAS")
	if err != nil {
		t.Fatalf("create IDEAS: %v", err)
	}
	bobFolder, err := store.CreateFolder(ctx, bob, "Ideas")
	if err != nil {
		t.Fatalf("create Bob folder: %v", err)
	}
	if ideasOne.ID == "" || ideasTwo.ID == "" || upper.ID == "" || ideasOne.ID == ideasTwo.ID || ideasOne.ID == upper.ID || ideasTwo.ID == upper.ID {
		t.Fatalf("folder IDs are not distinct opaque values: %q %q %q", ideasOne.ID, ideasTwo.ID, upper.ID)
	}
	for _, folder := range []api.SavedPostFolder{ideasOne, ideasTwo, upper, bobFolder} {
		if !folder.CreatedAt.Equal(now) || !folder.UpdatedAt.Equal(now) {
			t.Fatalf("folder timestamps = %+v, want %s", folder, now)
		}
	}

	pageOne, cursor, err := store.ListFolders(ctx, alice, 2, "")
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(pageOne) != 2 || cursor == "" {
		t.Fatalf("first page len/cursor = %d/%q", len(pageOne), cursor)
	}
	pageTwo, terminalCursor, err := store.ListFolders(ctx, alice, 2, cursor)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(pageTwo) != 1 || terminalCursor != "" {
		t.Fatalf("second page len/cursor = %d/%q", len(pageTwo), terminalCursor)
	}
	listed := append(append([]api.SavedPostFolder{}, pageOne...), pageTwo...)
	if slices.ContainsFunc(listed, func(folder api.SavedPostFolder) bool { return folder.ID == bobFolder.ID }) {
		t.Fatalf("Alice list exposed Bob folder: %+v", listed)
	}
	if !slices.IsSortedFunc(listed, func(a, b api.SavedPostFolder) int {
		if byName := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); byName != 0 {
			return byName
		}
		return strings.Compare(a.ID, b.ID)
	}) {
		t.Fatalf("folders not in lower(name)/ID order: %+v", listed)
	}

	now = now.Add(time.Hour)
	renamed, err := store.RenameFolder(ctx, alice, ideasOne.ID, "IDEAS")
	if err != nil {
		t.Fatalf("rename to duplicate name: %v", err)
	}
	if renamed.ID != ideasOne.ID || renamed.Name != "IDEAS" || !renamed.CreatedAt.Equal(ideasOne.CreatedAt) || !renamed.UpdatedAt.Equal(now) {
		t.Fatalf("renamed folder = %+v, original = %+v", renamed, ideasOne)
	}

	for name, id := range map[string]string{
		"missing":   "00000000-0000-4000-8000-000000000099",
		"foreign":   bobFolder.ID,
		"malformed": "not-storage-shaped",
	} {
		t.Run(name+" rename", func(t *testing.T) {
			_, err := store.RenameFolder(ctx, alice, id, "Still valid")
			if !errors.Is(err, api.ErrSavedPostFolderNotFound) {
				t.Fatalf("RenameFolder error = %v, want folder not found", err)
			}
		})
	}

	postURI := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/one")
	if _, err := store.Save(ctx, alice, postURI, api.FolderAssignment{Present: true, ID: &renamed.ID}); err != nil {
		t.Fatalf("save into renamed folder: %v", err)
	}
	page, _, err := store.ListFolders(ctx, alice, 100, "")
	if err != nil {
		t.Fatalf("list after save: %v", err)
	}
	for _, folder := range page {
		if folder.ID == renamed.ID && !folder.UpdatedAt.Equal(renamed.UpdatedAt) {
			t.Fatalf("save changed folder updatedAt: got %s want %s", folder.UpdatedAt, renamed.UpdatedAt)
		}
	}
}

func TestSavedPostStoreListsMoreThanOneHundredDuplicateFoldersExactlyOnce(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostStorePreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid')
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	store := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time {
		return time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	}})
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	want := make([]api.SavedPostFolder, 0, 103)
	for i := 0; i < 103; i++ {
		name := []string{"Ideas", "IDEAS", "ideas", "Projects", "projects"}[i%5]
		folder, err := store.CreateFolder(ctx, alice, name)
		if err != nil {
			t.Fatalf("create Alice folder %d: %v", i, err)
		}
		want = append(want, folder)
	}
	if _, err := store.CreateFolder(ctx, bob, "Ideas"); err != nil {
		t.Fatalf("create Bob folder: %v", err)
	}
	slices.SortFunc(want, func(a, b api.SavedPostFolder) int {
		if byName := strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)); byName != 0 {
			return byName
		}
		return strings.Compare(a.ID, b.ID)
	})

	got := make([]api.SavedPostFolder, 0, len(want))
	cursor := ""
	for {
		page, next, err := store.ListFolders(ctx, alice, 17, cursor)
		if err != nil {
			t.Fatalf("list dense folders after %q: %v", cursor, err)
		}
		got = append(got, page...)
		if next == "" {
			break
		}
		if next == cursor {
			t.Fatalf("dense folder cursor did not advance: %q", next)
		}
		cursor = next
	}
	if len(got) != len(want) {
		t.Fatalf("listed %d dense folders, want %d", len(got), len(want))
	}
	seen := make(map[string]struct{}, len(got))
	for i := range want {
		if got[i].ID != want[i].ID || got[i].Name != want[i].Name {
			t.Fatalf("folder[%d] = %+v, want %+v", i, got[i], want[i])
		}
		if _, duplicate := seen[got[i].ID]; duplicate {
			t.Fatalf("folder %q appeared more than once", got[i].ID)
		}
		seen[got[i].ID] = struct{}{}
	}
}

func TestSavedPostStoreDeleteFolderUnfilesSavesIdempotently(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostStorePreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		VALUES
			('at://did:plc:bob/social.craftsky.feed.post/one', 'did:plc:bob', 'one', 'post-one'),
			('at://did:plc:bob/social.craftsky.feed.post/two', 'did:plc:bob', 'two', 'post-two'),
			('at://did:plc:bob/social.craftsky.feed.post/three', 'did:plc:bob', 'three', 'post-three');
	`); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}

	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	store := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time { return now }})
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	aliceFolder, err := store.CreateFolder(ctx, alice, "Alice")
	if err != nil {
		t.Fatalf("create Alice folder: %v", err)
	}
	bobFolder, err := store.CreateFolder(ctx, bob, "Bob")
	if err != nil {
		t.Fatalf("create Bob folder: %v", err)
	}
	postOne := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/one")
	postTwo := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/two")
	postThree := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/three")
	first, err := store.Save(ctx, alice, postOne, api.FolderAssignment{Present: true, ID: &aliceFolder.ID})
	if err != nil {
		t.Fatalf("save first Alice post: %v", err)
	}
	now = now.Add(time.Minute)
	second, err := store.Save(ctx, alice, postTwo, api.FolderAssignment{Present: true, ID: &aliceFolder.ID})
	if err != nil {
		t.Fatalf("save second Alice post: %v", err)
	}
	if _, err := store.Save(ctx, bob, postThree, api.FolderAssignment{Present: true, ID: &bobFolder.ID}); err != nil {
		t.Fatalf("save Bob post: %v", err)
	}

	if err := store.DeleteFolder(ctx, alice, aliceFolder.ID); err != nil {
		t.Fatalf("delete non-empty Alice folder: %v", err)
	}
	for postURI, wantSavedAt := range map[syntax.ATURI]time.Time{postOne: first.State.SavedAt, postTwo: second.State.SavedAt} {
		state, err := store.ReadState(ctx, alice, postURI)
		if err != nil {
			t.Fatalf("read unfiled save: %v", err)
		}
		if state.FolderID != nil || !state.SavedAt.Equal(wantSavedAt) {
			t.Fatalf("state after folder delete = %+v, want unfiled at %s", state, wantSavedAt)
		}
	}
	aliceFolders, _, err := store.ListFolders(ctx, alice, 100, "")
	if err != nil {
		t.Fatalf("list Alice folders: %v", err)
	}
	if len(aliceFolders) != 0 {
		t.Fatalf("Alice folder remained: %+v", aliceFolders)
	}

	for name, id := range map[string]string{
		"repeat":    aliceFolder.ID,
		"missing":   "00000000-0000-4000-8000-000000000099",
		"foreign":   bobFolder.ID,
		"malformed": "not-storage-shaped",
	} {
		t.Run(name+" delete", func(t *testing.T) {
			if err := store.DeleteFolder(ctx, alice, id); err != nil {
				t.Fatalf("DeleteFolder: %v", err)
			}
		})
	}
	bobState, err := store.ReadState(ctx, bob, postThree)
	if err != nil {
		t.Fatalf("read Bob save: %v", err)
	}
	if bobState.FolderID == nil || *bobState.FolderID != bobFolder.ID {
		t.Fatalf("Alice delete changed Bob save: %+v", bobState)
	}
	bobFolders, _, err := store.ListFolders(ctx, bob, 100, "")
	if err != nil {
		t.Fatalf("list Bob folders: %v", err)
	}
	if len(bobFolders) != 1 || bobFolders[0].ID != bobFolder.ID {
		t.Fatalf("Alice delete changed Bob folder: %+v", bobFolders)
	}
}

func TestSavedPostStoreListsAllFolderAndUnfiledInBothDirections(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostStorePreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
	`); err != nil {
		t.Fatalf("insert profiles: %v", err)
	}

	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	store := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time { return now }})
	aliceFolder, err := store.CreateFolder(ctx, alice, "Alice")
	if err != nil {
		t.Fatalf("create Alice folder: %v", err)
	}
	bobFolder, err := store.CreateFolder(ctx, bob, "Bob")
	if err != nil {
		t.Fatalf("create Bob folder: %v", err)
	}

	all := make([]api.SavedPostRef, 0, 103)
	foldered := make([]api.SavedPostRef, 0, 52)
	unfiled := make([]api.SavedPostRef, 0, 51)
	for i := 0; i < 103; i++ {
		rkey := "post-" + strconv.Itoa(i)
		uri := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/" + rkey)
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_posts (uri, did, rkey, cid)
			VALUES ($1, 'did:plc:bob', $2, $3)
		`, uri, rkey, "cid-"+strconv.Itoa(i)); err != nil {
			t.Fatalf("insert post %d: %v", i, err)
		}
		now = time.Date(2026, 7, 20, 11, i/2, 0, 0, time.UTC)
		assignment := api.FolderAssignment{}
		if i%2 == 0 {
			assignment = api.FolderAssignment{Present: true, ID: &aliceFolder.ID}
		}
		result, err := store.Save(ctx, alice, uri, assignment)
		if err != nil {
			t.Fatalf("save post %d: %v", i, err)
		}
		ref := api.SavedPostRef{PostURI: uri, SavedAt: result.State.SavedAt, FolderID: result.State.FolderID}
		all = append(all, ref)
		if ref.FolderID == nil {
			unfiled = append(unfiled, ref)
		} else {
			foldered = append(foldered, ref)
		}
	}
	firstURI := all[0].PostURI
	if _, err := store.Save(ctx, bob, firstURI, api.FolderAssignment{Present: true, ID: &bobFolder.ID}); err != nil {
		t.Fatalf("save Bob row: %v", err)
	}

	for _, test := range []struct {
		name     string
		filter   api.SavedPostListFilter
		expected []api.SavedPostRef
	}{
		{name: "all newest", filter: api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 17}, expected: all},
		{name: "all oldest", filter: api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortOldest, Limit: 19}, expected: all},
		{name: "folder newest", filter: api.SavedPostListFilter{Scope: api.SavedPostScopeFolder, FolderID: aliceFolder.ID, Sort: api.SavedPostSortNewest, Limit: 11}, expected: foldered},
		{name: "unfiled oldest", filter: api.SavedPostListFilter{Scope: api.SavedPostScopeUnfiled, Sort: api.SavedPostSortOldest, Limit: 13}, expected: unfiled},
	} {
		t.Run(test.name, func(t *testing.T) {
			want := append([]api.SavedPostRef{}, test.expected...)
			slices.SortFunc(want, func(a, b api.SavedPostRef) int {
				byTime := a.SavedAt.Compare(b.SavedAt)
				if byTime == 0 {
					byTime = strings.Compare(a.PostURI.String(), b.PostURI.String())
				}
				if test.filter.Sort == api.SavedPostSortNewest {
					return -byTime
				}
				return byTime
			})

			got := make([]api.SavedPostRef, 0, len(want))
			cursor := ""
			for {
				filter := test.filter
				filter.Cursor = cursor
				page, next, err := store.ListSavedRefs(ctx, alice, filter)
				if err != nil {
					t.Fatalf("ListSavedRefs: %v", err)
				}
				got = append(got, page...)
				if next == "" {
					break
				}
				if next == cursor {
					t.Fatalf("cursor did not advance: %q", next)
				}
				cursor = next
			}
			if len(got) != len(want) {
				t.Fatalf("listed %d refs, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i].PostURI != want[i].PostURI || !got[i].SavedAt.Equal(want[i].SavedAt) || !sameTestStringPointer(got[i].FolderID, want[i].FolderID) {
					t.Fatalf("ref[%d] = %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}

	firstPage, cursor, err := store.ListSavedRefs(ctx, alice, api.SavedPostListFilter{Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 2})
	if err != nil || len(firstPage) != 2 || cursor == "" {
		t.Fatalf("first cursor page = %d/%q/%v", len(firstPage), cursor, err)
	}
	for name, filter := range map[string]api.SavedPostListFilter{
		"malformed cursor": {Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortNewest, Limit: 2, Cursor: "not-base64"},
		"cross sort":       {Scope: api.SavedPostScopeAll, Sort: api.SavedPostSortOldest, Limit: 2, Cursor: cursor},
		"cross scope":      {Scope: api.SavedPostScopeUnfiled, Sort: api.SavedPostSortNewest, Limit: 2, Cursor: cursor},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := store.ListSavedRefs(ctx, alice, filter); !errors.Is(err, api.ErrInvalidSavedPostCursor) {
				t.Fatalf("ListSavedRefs error = %v, want invalid cursor", err)
			}
		})
	}
	for name, folderID := range map[string]string{
		"missing":   "00000000-0000-4000-8000-000000000099",
		"foreign":   bobFolder.ID,
		"malformed": "not-storage-shaped",
	} {
		t.Run(name+" folder scope", func(t *testing.T) {
			_, _, err := store.ListSavedRefs(ctx, alice, api.SavedPostListFilter{Scope: api.SavedPostScopeFolder, FolderID: folderID, Sort: api.SavedPostSortNewest, Limit: 10})
			if !errors.Is(err, api.ErrSavedPostFolderNotFound) {
				t.Fatalf("ListSavedRefs error = %v, want folder not found", err)
			}
		})
	}

	t.Run("newer save between newest pages stays before the cursor", func(t *testing.T) {
		want := append([]api.SavedPostRef{}, all...)
		slices.SortFunc(want, func(a, b api.SavedPostRef) int {
			byTime := a.SavedAt.Compare(b.SavedAt)
			if byTime == 0 {
				byTime = strings.Compare(a.PostURI.String(), b.PostURI.String())
			}
			return -byTime
		})
		first, next, err := store.ListSavedRefs(ctx, alice, api.SavedPostListFilter{
			Scope: api.SavedPostScopeAll,
			Sort:  api.SavedPostSortNewest,
			Limit: 17,
		})
		if err != nil || len(first) != 17 || next == "" {
			t.Fatalf("newest first page = %d/%q/%v", len(first), next, err)
		}
		newURI := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/arrived-newest")
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_posts (uri, did, rkey, cid)
			VALUES ($1, 'did:plc:bob', 'arrived-newest', 'arrived-newest-cid')
		`, newURI); err != nil {
			t.Fatalf("insert newer post: %v", err)
		}
		now = time.Date(2026, 7, 20, 14, 0, 0, 0, time.UTC)
		if _, err := store.Save(ctx, alice, newURI, api.FolderAssignment{}); err != nil {
			t.Fatalf("insert newer save between pages: %v", err)
		}

		got := append([]api.SavedPostRef{}, first...)
		for next != "" {
			page, following, err := store.ListSavedRefs(ctx, alice, api.SavedPostListFilter{
				Scope:  api.SavedPostScopeAll,
				Sort:   api.SavedPostSortNewest,
				Limit:  17,
				Cursor: next,
			})
			if err != nil {
				t.Fatalf("continue newest traversal: %v", err)
			}
			got = append(got, page...)
			next = following
		}
		if len(got) != len(want) {
			t.Fatalf("newest traversal returned %d original refs, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i].PostURI != want[i].PostURI {
				t.Fatalf("newest ref[%d] = %s, want %s", i, got[i].PostURI, want[i].PostURI)
			}
			if got[i].PostURI == newURI {
				t.Fatalf("new save before the newest cursor leaked into the active traversal: %+v", got[i])
			}
		}
	})

	t.Run("newer save between oldest pages is appended once", func(t *testing.T) {
		first, next, err := store.ListSavedRefs(ctx, alice, api.SavedPostListFilter{
			Scope: api.SavedPostScopeAll,
			Sort:  api.SavedPostSortOldest,
			Limit: 19,
		})
		if err != nil || len(first) != 19 || next == "" {
			t.Fatalf("oldest first page = %d/%q/%v", len(first), next, err)
		}
		newURI := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/arrived-oldest")
		if _, err := pool.Exec(ctx, `
			INSERT INTO craftsky_posts (uri, did, rkey, cid)
			VALUES ($1, 'did:plc:bob', 'arrived-oldest', 'arrived-oldest-cid')
		`, newURI); err != nil {
			t.Fatalf("insert second newer post: %v", err)
		}
		now = time.Date(2026, 7, 20, 15, 0, 0, 0, time.UTC)
		if _, err := store.Save(ctx, alice, newURI, api.FolderAssignment{}); err != nil {
			t.Fatalf("insert newer save during oldest traversal: %v", err)
		}

		got := append([]api.SavedPostRef{}, first...)
		for next != "" {
			page, following, err := store.ListSavedRefs(ctx, alice, api.SavedPostListFilter{
				Scope:  api.SavedPostScopeAll,
				Sort:   api.SavedPostSortOldest,
				Limit:  19,
				Cursor: next,
			})
			if err != nil {
				t.Fatalf("continue oldest traversal: %v", err)
			}
			got = append(got, page...)
			next = following
		}
		seen := make(map[syntax.ATURI]int, len(got))
		for _, ref := range got {
			seen[ref.PostURI]++
		}
		if len(got) != 105 || seen[newURI] != 1 {
			t.Fatalf("oldest traversal count/new row = %d/%d, want 105/1", len(got), seen[newURI])
		}
		for uri, count := range seen {
			if count != 1 {
				t.Fatalf("oldest traversal returned %s %d times", uri, count)
			}
		}
	})
}

func sameTestStringPointer(first, second *string) bool {
	if first == nil || second == nil {
		return first == nil && second == nil
	}
	return *first == *second
}

func TestSavedPostStoreConcurrentMutationsRemainSerialValid(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostStorePreStateDDL)
	ctx := context.Background()
	up, err := os.ReadFile("../../migrations/000024_saved_posts.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
		INSERT INTO craftsky_posts (uri, did, rkey, cid)
		VALUES
			('at://did:plc:bob/social.craftsky.feed.post/one', 'did:plc:bob', 'one', 'post-one'),
			('at://did:plc:bob/social.craftsky.feed.post/two', 'did:plc:bob', 'two', 'post-two');
	`); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}

	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	store := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time { return now }})
	owner := syntax.DID("did:plc:alice")
	postOne := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/one")
	postTwo := syntax.ATURI("at://did:plc:bob/social.craftsky.feed.post/two")
	folderA, err := store.CreateFolder(ctx, owner, "A")
	if err != nil {
		t.Fatalf("create folder A: %v", err)
	}
	folderB, err := store.CreateFolder(ctx, owner, "B")
	if err != nil {
		t.Fatalf("create folder B: %v", err)
	}

	start := make(chan struct{})
	results := make(chan api.SaveMutationResult, 12)
	errorsCh := make(chan error, 12)
	var group sync.WaitGroup
	for range 12 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			result, err := store.Save(ctx, owner, postOne, api.FolderAssignment{})
			results <- result
			errorsCh <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	close(errorsCh)
	created := 0
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent duplicate save: %v", err)
		}
	}
	for result := range results {
		if result.Created {
			created++
		}
		if !result.State.SavedAt.Equal(now) {
			t.Fatalf("duplicate save timestamp = %s, want %s", result.State.SavedAt, now)
		}
	}
	if created != 1 {
		t.Fatalf("created outcomes = %d, want exactly 1", created)
	}

	moveErrors := make(chan error, 2)
	for _, folderID := range []string{folderA.ID, folderB.ID} {
		folderID := folderID
		go func() {
			_, err := store.Save(ctx, owner, postOne, api.FolderAssignment{Present: true, ID: &folderID})
			moveErrors <- err
		}()
	}
	for range 2 {
		if err := <-moveErrors; err != nil {
			t.Fatalf("concurrent folder move: %v", err)
		}
	}
	moved, err := store.ReadState(ctx, owner, postOne)
	if err != nil || moved.FolderID == nil || (*moved.FolderID != folderA.ID && *moved.FolderID != folderB.ID) || !moved.SavedAt.Equal(now) {
		t.Fatalf("state after concurrent moves = %+v, err %v", moved, err)
	}

	if _, err := store.Save(ctx, owner, postTwo, api.FolderAssignment{Present: true, ID: &folderA.ID}); err != nil {
		t.Fatalf("seed post two save: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE FUNCTION slow_saved_post_delete() RETURNS trigger AS $$
		BEGIN
			PERFORM pg_sleep(0.25);
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER slow_saved_post_delete
		BEFORE DELETE ON saved_posts
		FOR EACH ROW EXECUTE FUNCTION slow_saved_post_delete();
	`); err != nil {
		t.Fatalf("install controlled delete trigger: %v", err)
	}
	unsaveDone := make(chan error, 1)
	go func() { unsaveDone <- store.Unsave(ctx, owner, postTwo) }()
	time.Sleep(50 * time.Millisecond)
	resaveDone := make(chan error, 1)
	go func() {
		_, err := store.Save(ctx, owner, postTwo, api.FolderAssignment{Present: true, ID: &folderB.ID})
		resaveDone <- err
	}()
	if err := <-unsaveDone; err != nil {
		t.Fatalf("controlled concurrent unsave: %v", err)
	}
	if err := <-resaveDone; err != nil {
		t.Fatalf("controlled concurrent resave: %v", err)
	}
	resaved, err := store.ReadState(ctx, owner, postTwo)
	if err != nil || resaved.FolderID == nil || *resaved.FolderID != folderB.ID {
		t.Fatalf("state after unsave/resave = %+v, err %v", resaved, err)
	}

	folderC, err := store.CreateFolder(ctx, owner, "C")
	if err != nil {
		t.Fatalf("create folder C: %v", err)
	}
	if _, err := store.Save(ctx, owner, postTwo, api.FolderAssignment{Present: true, ID: &folderC.ID}); err != nil {
		t.Fatalf("move post two to C: %v", err)
	}
	folderDeleteDone := make(chan error, 1)
	unfileDone := make(chan error, 1)
	go func() { folderDeleteDone <- store.DeleteFolder(ctx, owner, folderC.ID) }()
	go func() {
		_, err := store.Save(ctx, owner, postTwo, api.FolderAssignment{Present: true})
		unfileDone <- err
	}()
	if err := <-folderDeleteDone; err != nil {
		t.Fatalf("concurrent folder delete: %v", err)
	}
	if err := <-unfileDone; err != nil {
		t.Fatalf("concurrent explicit unfile: %v", err)
	}
	unfiled, err := store.ReadState(ctx, owner, postTwo)
	if err != nil || unfiled.FolderID != nil || !unfiled.SavedAt.Equal(now) {
		t.Fatalf("state after delete/unfile = %+v, err %v", unfiled, err)
	}

	folderD, err := store.CreateFolder(ctx, owner, "D")
	if err != nil {
		t.Fatalf("create folder D: %v", err)
	}
	moveDeleteDone := make(chan error, 2)
	go func() { moveDeleteDone <- store.DeleteFolder(ctx, owner, folderD.ID) }()
	go func() {
		_, err := store.Save(ctx, owner, postOne, api.FolderAssignment{Present: true, ID: &folderD.ID})
		moveDeleteDone <- err
	}()
	for range 2 {
		err := <-moveDeleteDone
		if err != nil && !errors.Is(err, api.ErrSavedPostFolderNotFound) {
			t.Fatalf("concurrent move/delete outcome: %v", err)
		}
	}
	if folders, _, err := store.ListFolders(ctx, owner, 100, ""); err != nil {
		t.Fatalf("list folders after move/delete: %v", err)
	} else if slices.ContainsFunc(folders, func(folder api.SavedPostFolder) bool { return folder.ID == folderD.ID }) {
		t.Fatalf("folder D survived delete: %+v", folders)
	}
}

func TestSavedPostStoreRequiredContextStates(t *testing.T) {
	pool := testdb.WithSchema(t, savedPostContextDDL)
	ctx := context.Background()
	rootURI := syntax.ATURI("at://did:plc:root/social.craftsky.feed.post/root")
	parentURI := syntax.ATURI("at://did:plc:parent/social.craftsky.feed.post/parent")
	targetURI := syntax.ATURI("at://did:plc:target/social.craftsky.feed.post/target")
	missingURI := syntax.ATURI("at://did:plc:target/social.craftsky.feed.post/missing")
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles (did, record_cid)
		VALUES
			('did:plc:viewer', 'viewer-cid'),
			('did:plc:root', 'root-cid'),
			('did:plc:parent', 'parent-cid'),
			('did:plc:target', 'target-cid')
	`); err != nil {
		t.Fatalf("insert context profiles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_posts (uri, did, rkey, cid, reply_root_uri, reply_parent_uri)
		VALUES
			($1, 'did:plc:root', 'root', 'root-cid', NULL, NULL),
			($2, 'did:plc:parent', 'parent', 'parent-cid', $1, $1),
			($3, 'did:plc:target', 'target', 'target-cid', $1, $2),
			($4, 'did:plc:target', 'missing', 'missing-cid', $1, 'at://did:plc:gone/social.craftsky.feed.post/gone');
	`, rootURI, parentURI, targetURI, missingURI); err != nil {
		t.Fatalf("insert context posts: %v", err)
	}
	store := api.NewPostStore(pool)
	viewer := syntax.DID("did:plc:viewer")
	assertContext := func(uri syntax.ATURI, want bool) {
		t.Helper()
		states, err := store.RequiredContextStates(ctx, viewer, []syntax.ATURI{uri})
		if err != nil {
			t.Fatalf("RequiredContextStates: %v", err)
		}
		if got := states[uri]; got != want {
			t.Fatalf("context state for %s = %v, want %v", uri, got, want)
		}
	}
	assertContext(targetURI, true)
	assertContext(missingURI, false)

	if _, err := pool.Exec(ctx, `INSERT INTO actor_mutes (owner_did, subject_did) VALUES ($1, 'did:plc:parent')`, viewer); err != nil {
		t.Fatalf("insert mute: %v", err)
	}
	assertContext(targetURI, true)

	if _, err := pool.Exec(ctx, `INSERT INTO atproto_blocks (uri, blocker_did, subject_did) VALUES ('at://did:plc:viewer/app.bsky.graph.block/one', $1, 'did:plc:parent')`, viewer); err != nil {
		t.Fatalf("insert block: %v", err)
	}
	assertContext(targetURI, false)
	if _, err := pool.Exec(ctx, `DELETE FROM atproto_blocks`); err != nil {
		t.Fatalf("delete block: %v", err)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO moderation_outputs (id, source_did, subject_type, subject_did, subject_uri, value, action)
		VALUES ('00000000-0000-4000-8000-000000000001', 'did:plc:mod', 'post', 'did:plc:parent', $1, 'hide', 'apply')
	`, parentURI); err != nil {
		t.Fatalf("insert moderation hide: %v", err)
	}
	assertContext(targetURI, false)
	if _, err := pool.Exec(ctx, `DELETE FROM moderation_outputs`); err != nil {
		t.Fatalf("delete moderation hide: %v", err)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = 'did:plc:root'`); err != nil {
		t.Fatalf("remove root membership: %v", err)
	}
	assertContext(targetURI, false)
}
