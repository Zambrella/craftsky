package index_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

// Reuses craftskyProfilesDDL from craftsky_profile_test.go (same package index_test,
// same rationale as bluesky_profile_test.go:16).

// fakeBackfiller satisfies the exported BlueskyBackfiller interface.
// Used by TestBlueskyBackfiller_InterfaceShape only.
type fakeBackfiller struct {
	calls []syntax.DID
	err   error
}

type fakeRepoTracker struct {
	dids []syntax.DID
	err  error
}

func (f *fakeRepoTracker) AddRepo(_ context.Context, did syntax.DID) error {
	f.dids = append(f.dids, did)
	return f.err
}

func (f *fakeBackfiller) Backfill(_ context.Context, did syntax.DID) error {
	f.calls = append(f.calls, did)
	return f.err
}

func TestBlueskyBackfiller_InterfaceShape(t *testing.T) {
	var _ index.BlueskyBackfiller = (*fakeBackfiller)(nil)
}

// fakeAnonPDS implements auth.PDSClient for backfiller tests. GetRecord
// returns the configured value+cid; PutRecord is never used.
type fakeAnonPDS struct {
	cid   string
	value map[string]any
	err   error
	calls int
}

func (f *fakeAnonPDS) GetRecord(_ context.Context, _ syntax.DID, _, _ string, out any) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if m, ok := out.(*map[string]any); ok {
		*m = f.value
	}
	return f.cid, nil
}

func (f *fakeAnonPDS) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
	panic("fakeAnonPDS.PutRecord must not be called in backfiller tests")
}

func (f *fakeAnonPDS) CreateRecord(_ context.Context, _ syntax.DID, _ string, _ any) (syntax.ATURI, syntax.CID, error) {
	panic("fakeAnonPDS.CreateRecord must not be called in backfiller tests")
}

func (f *fakeAnonPDS) DeleteRecord(_ context.Context, _ syntax.DID, _, _ string) error {
	panic("fakeAnonPDS.DeleteRecord must not be called in backfiller tests")
}

func (f *fakeAnonPDS) UploadBlob(_ context.Context, _ string, _ []byte) (*auth.UploadedBlob, error) {
	panic("fakeAnonPDS.UploadBlob must not be called in backfiller tests")
}

func TestBlueskyBackfiller_Backfill_RecordPresent_WritesBlueskyRow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)

	// Seed membership so BlueskyProfile.Handle's gate passes.
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		"did:plc:abc", "cidcsky"); err != nil {
		t.Fatal(err)
	}

	pds := &fakeAnonPDS{
		cid:   "bafbluesky",
		value: map[string]any{"displayName": "alice"},
	}
	bsky := index.NewBlueskyProfile(pool)
	bf := index.NewBlueskyBackfiller(pds, bsky)

	if err := bf.Backfill(context.Background(), syntax.DID("did:plc:abc")); err != nil {
		t.Fatalf("Backfill: %v", err)
	}
	if pds.calls != 1 {
		t.Fatalf("PDS GetRecord called %d times; want 1", pds.calls)
	}

	var displayName, recordCID string
	if err := pool.QueryRow(context.Background(),
		`SELECT display_name, record_cid FROM bluesky_profiles WHERE did = $1`,
		"did:plc:abc").Scan(&displayName, &recordCID); err != nil {
		t.Fatalf("select: %v", err)
	}
	if displayName != "alice" {
		t.Errorf("display_name = %q", displayName)
	}
	if recordCID != "bafbluesky" {
		t.Errorf("record_cid = %q", recordCID)
	}
}

func TestBlueskyBackfiller_Backfill_RecordNotFound_IsNoOp(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		"did:plc:none", "cidcsky"); err != nil {
		t.Fatal(err)
	}
	pds := &fakeAnonPDS{err: auth.ErrRecordNotFound}
	bf := index.NewBlueskyBackfiller(pds, index.NewBlueskyProfile(pool))

	if err := bf.Backfill(context.Background(), syntax.DID("did:plc:none")); err != nil {
		t.Errorf("want nil for RecordNotFound; got %v", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`,
		"did:plc:none").Scan(&count); err != nil {
		t.Fatalf("count select: %v", err)
	}
	if count != 0 {
		t.Errorf("bluesky_profiles count = %d; want 0", count)
	}
}

func TestBlueskyBackfiller_Backfill_PDSError_Propagates(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		"did:plc:err", "cidcsky"); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("pds on fire")
	pds := &fakeAnonPDS{err: boom}
	bf := index.NewBlueskyBackfiller(pds, index.NewBlueskyProfile(pool))

	err := bf.Backfill(context.Background(), syntax.DID("did:plc:err"))
	if !errors.Is(err, boom) {
		t.Errorf("want wrapped %v; got %v", boom, err)
	}
}

func TestBlueskyBackfillerRequestsTapTrackingAndStillBackfillsProfileOnTrackingFailure(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	did := syntax.DID("did:plc:joining")
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		did, "cidcsky"); err != nil {
		t.Fatal(err)
	}
	pds := &fakeAnonPDS{cid: "bafbluesky", value: map[string]any{"displayName": "Joining"}}
	trackingErr := errors.New("Tap temporarily unavailable")
	tracker := &fakeRepoTracker{err: trackingErr}
	bf := index.NewBlueskyBackfiller(pds, index.NewBlueskyProfile(pool), tracker)

	err := bf.Backfill(context.Background(), did)
	if !errors.Is(err, trackingErr) {
		t.Fatalf("Backfill error = %v, want tracking error", err)
	}
	if len(tracker.dids) != 1 || tracker.dids[0] != did {
		t.Fatalf("tracking requests = %v, want %s", tracker.dids, did)
	}
	var displayName string
	if err := pool.QueryRow(context.Background(),
		`SELECT display_name FROM bluesky_profiles WHERE did = $1`, did).Scan(&displayName); err != nil {
		t.Fatalf("profile backfill after tracking failure: %v", err)
	}
	if displayName != "Joining" {
		t.Fatalf("display name = %q", displayName)
	}
}

func TestCraftskyProfile_Handle_NewRow_BackfillsBluesky(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	pds := &fakeAnonPDS{
		cid:   "bafbluesky",
		value: map[string]any{"displayName": "alice"},
	}
	bsky := index.NewBlueskyProfile(pool)
	bf := index.NewBlueskyBackfiller(pds, bsky)
	idx := index.NewCraftskyProfile(pool, bf, testLogger())

	ev := tap.Event{
		URI:        "at://did:plc:e2e/social.craftsky.actor.profile/self",
		CID:        "ccsky",
		DID:        "did:plc:e2e",
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Both tables populated after a single handle call.
	var craftskyCount int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM craftsky_profiles WHERE did = $1`, ev.DID).Scan(&craftskyCount)
	if craftskyCount != 1 {
		t.Errorf("craftsky_profiles count = %d; want 1", craftskyCount)
	}

	var displayName string
	if err := pool.QueryRow(context.Background(),
		`SELECT display_name FROM bluesky_profiles WHERE did = $1`, ev.DID).
		Scan(&displayName); err != nil {
		t.Fatalf("select bluesky: %v", err)
	}
	if displayName != "alice" {
		t.Errorf("display_name = %q; want alice", displayName)
	}

	var recordCID string
	if err := pool.QueryRow(context.Background(),
		`SELECT record_cid FROM bluesky_profiles WHERE did = $1`, ev.DID).
		Scan(&recordCID); err != nil {
		t.Fatalf("select record_cid: %v", err)
	}
	if recordCID != "bafbluesky" {
		t.Errorf("record_cid = %q; want bafbluesky", recordCID)
	}
}

func TestMembershipAndBlockBackfillConvergeAcrossRestartWithoutReadinessState(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyProfilesDDL+`
		CREATE TABLE actor_mutes (
			owner_did TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (owner_did, subject_did)
		);
	`+atprotoBlocksDDL)
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	joining := syntax.DID("did:plc:joining")
	if _, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`, alice, "alice-cid"); err != nil {
		t.Fatal(err)
	}

	blockIndexer := index.NewBlueskyBlock(pool)
	inbound := tap.Event{
		URI:        "at://did:plc:alice/app.bsky.graph.block/inbound",
		CID:        "bafy-inbound",
		DID:        alice,
		Rkey:       "inbound",
		Collection: "app.bsky.graph.block",
		Action:     "create",
		Record:     json.RawMessage(`{"subject":"did:plc:joining","createdAt":"2026-07-19T12:00:00Z"}`),
	}
	if err := blockIndexer.Handle(ctx, inbound); err != nil {
		t.Fatalf("retain inbound block before membership: %v", err)
	}
	store := relationships.NewStore(pool)
	if err := relationships.RequireCurrentMember(ctx, store, joining); !errors.Is(err, relationships.ErrProfileNotFound) {
		t.Fatalf("joining membership before profile = %v", err)
	}

	tracker := &fakeRepoTracker{}
	missingProfilePDS := &fakeAnonPDS{err: auth.ErrRecordNotFound}
	firstBackfiller := index.NewBlueskyBackfiller(missingProfilePDS, index.NewBlueskyProfile(pool), tracker)
	profileIndexer := index.NewCraftskyProfile(pool, firstBackfiller, testLogger())
	profile := tap.Event{
		URI:        "at://did:plc:joining/social.craftsky.actor.profile/self",
		CID:        "joining-cid",
		DID:        joining,
		Rkey:       "self",
		Collection: "social.craftsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if err := profileIndexer.Handle(ctx, profile); err != nil {
		t.Fatalf("index joining membership: %v", err)
	}
	if err := relationships.RequireCurrentMember(ctx, store, joining); err != nil {
		t.Fatalf("membership after profile row: %v", err)
	}
	state, err := store.State(ctx, joining, alice)
	if err != nil {
		t.Fatalf("state after joining: %v", err)
	}
	if state.Blocking || !state.BlockedBy {
		t.Fatalf("retained inbound state after joining = %+v", state)
	}
	if len(tracker.dids) != 1 || tracker.dids[0] != joining {
		t.Fatalf("initial tracking requests = %v", tracker.dids)
	}

	// Recreate the backfill service and request tracking again, representing
	// an OAuth/restart retry before Tap resumes the held repository events.
	restartedBackfiller := index.NewBlueskyBackfiller(missingProfilePDS, index.NewBlueskyProfile(pool), tracker)
	if err := restartedBackfiller.Backfill(ctx, joining); err != nil {
		t.Fatalf("restart tracking retry: %v", err)
	}
	if len(tracker.dids) != 2 || tracker.dids[1] != joining {
		t.Fatalf("restart tracking requests = %v", tracker.dids)
	}
	state, err = store.State(ctx, joining, alice)
	if err != nil {
		t.Fatalf("state before outbound Tap event: %v", err)
	}
	if state.Blocking {
		t.Fatalf("joining-owned outbound block appeared before Tap: %+v", state)
	}

	outbound := tap.Event{
		URI:        "at://did:plc:joining/app.bsky.graph.block/outbound",
		CID:        "bafy-outbound",
		DID:        joining,
		Rkey:       "outbound",
		Collection: "app.bsky.graph.block",
		Action:     "create",
		Record:     json.RawMessage(`{"subject":"did:plc:alice","createdAt":"2026-07-19T12:01:00Z"}`),
	}
	if err := blockIndexer.Handle(ctx, outbound); err != nil {
		t.Fatalf("resume joining-owned block event: %v", err)
	}
	state, err = store.State(ctx, joining, alice)
	if err != nil {
		t.Fatalf("state after outbound Tap event: %v", err)
	}
	if !state.Blocking || !state.BlockedBy {
		t.Fatalf("converged mutual state = %+v", state)
	}

	var readinessTables int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = current_schema()
		  AND table_name IN ('relationship_activation', 'block_readiness', 'membership_readiness')
	`).Scan(&readinessTables); err != nil {
		t.Fatalf("inspect readiness tables: %v", err)
	}
	if readinessTables != 0 {
		t.Fatalf("found %d forbidden readiness/activation tables", readinessTables)
	}
}
