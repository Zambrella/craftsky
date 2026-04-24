// appview/internal/index/craftsky_profile_test.go
package index_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

const craftskyProfilesDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- bluesky_profiles is needed for the delete-cascade test; owned by Chunk 4 indexer.
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
`

// noopBackfiller is the default backfiller for existing tests that
// predate the backfill path. Satisfies index.BlueskyBackfiller.
type noopBackfiller struct{}

func (noopBackfiller) Backfill(context.Context, syntax.DID) error { return nil }

// testLogger returns a logger that discards output. Equivalent patterns
// live elsewhere in the repo — inline here to avoid a new exported helper.
// Both `noopBackfiller` and `testLogger` are unexported; because this file
// and `bluesky_backfiller_test.go` share `package index_test`, the helpers
// are visible to both test files (no sharing-helper-file needed).
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCraftskyProfile_Create(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:abc/social.craftsky.actor.profile/self",
		CID:        "bafy1",
		DID:        "did:plc:abc",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["knitting","sewing"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var crafts []string
	var cid string
	err := pool.QueryRow(context.Background(),
		`SELECT crafts, record_cid FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&crafts, &cid)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if cid != "bafy1" {
		t.Errorf("record_cid = %q, want bafy1", cid)
	}
	if len(crafts) != 2 || crafts[0] != "knitting" || crafts[1] != "sewing" {
		t.Errorf("crafts = %v, want [knitting sewing]", crafts)
	}
}

func TestCraftskyProfile_UpdateReplacesCrafts(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())

	create := tap.Event{
		URI:        "at://did:plc:x/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:x",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["knitting"]}`),
	}
	update := create
	update.CID = "c2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"crafts":["knitting","quilting"]}`)

	if err := idx.Handle(context.Background(), create); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(context.Background(), update); err != nil {
		t.Fatal(err)
	}

	var crafts []string
	var cid string
	if err := pool.QueryRow(context.Background(),
		`SELECT crafts, record_cid FROM craftsky_profiles WHERE did = $1`, create.DID).
		Scan(&crafts, &cid); err != nil {
		t.Fatalf("select: %v", err)
	}
	if cid != "c2" {
		t.Errorf("record_cid = %q, want c2", cid)
	}
	if len(crafts) != 2 || crafts[1] != "quilting" {
		t.Errorf("crafts = %v, want [knitting quilting]", crafts)
	}
}

func TestCraftskyProfile_ReplayedEventPreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:y/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:y",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["crochet"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var firstIndexedAt string
	if err := pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&firstIndexedAt); err != nil {
		t.Fatalf("select first indexed_at: %v", err)
	}

	// Re-deliver identical event.
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	var secondIndexedAt string
	if err := pool.QueryRow(context.Background(),
		`SELECT indexed_at::text FROM craftsky_profiles WHERE did = $1`, ev.DID).
		Scan(&secondIndexedAt); err != nil {
		t.Fatalf("select second indexed_at: %v", err)
	}

	if firstIndexedAt != secondIndexedAt {
		t.Errorf("indexed_at changed on replay: %q -> %q", firstIndexedAt, secondIndexedAt)
	}
}

func TestCraftskyProfile_DeleteRemovesBothRows(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())
	ctx := context.Background()

	create := tap.Event{
		URI:        "at://did:plc:z/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:z",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}
	// Seed a bluesky_profiles row for the same DID.
	if _, err := pool.Exec(ctx,
		`INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES ($1, $2, $3)`,
		create.DID, "alice", "bskyCID"); err != nil {
		t.Fatal(err)
	}

	del := tap.Event{
		URI: create.URI, DID: create.DID, Rkey: "self",
		Collection: "social.craftsky.actor.profile", Action: "delete",
	}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete Handle: %v", err)
	}

	var crCount, bsCount int
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM craftsky_profiles WHERE did = $1`, del.DID).Scan(&crCount)
	_ = pool.QueryRow(ctx,
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`, del.DID).Scan(&bsCount)
	if crCount != 0 || bsCount != 0 {
		t.Errorf("post-delete counts = (craftsky:%d, bluesky:%d), want (0,0)", crCount, bsCount)
	}
}

func TestCraftskyProfile_UnknownAction(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())
	ev := tap.Event{
		URI:        "at://did:plc:a/social.craftsky.actor.profile/self",
		CID:        "c",
		DID:        "did:plc:a",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "weird",
		Record:     json.RawMessage(`{"crafts":[]}`),
	}
	if err := idx.Handle(context.Background(), ev); err == nil {
		t.Error("want error for unknown action; got nil")
	}
}

func TestCraftskyProfile_OtherCollectionIgnored(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())
	ev := tap.Event{
		URI: "at://did:plc:b/app.bsky.feed.post/k", CID: "c", DID: "did:plc:b", Rkey: "k",
		Collection: "app.bsky.feed.post", Action: "create",
		Record: json.RawMessage(`{}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("want nil for other collection; got %v", err)
	}
}

// spyBackfiller records every call so tests can assert arity and DID.
type spyBackfiller struct {
	calls []string
	err   error
}

func (s *spyBackfiller) Backfill(_ context.Context, did syntax.DID) error {
	s.calls = append(s.calls, did.String())
	return s.err
}

func TestCraftskyProfile_Handle_NewRow_CallsBackfill(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	spy := &spyBackfiller{}
	idx := index.NewCraftskyProfile(pool, spy, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:new/social.craftsky.actor.profile/self",
		CID:        "c1",
		DID:        "did:plc:new",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(spy.calls) != 1 || spy.calls[0] != "did:plc:new" {
		t.Errorf("backfill calls = %v; want [did:plc:new]", spy.calls)
	}
}
