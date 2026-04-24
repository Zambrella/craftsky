package index_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/testdb"
)

// Reuses craftskyProfilesDDL from craftsky_profile_test.go (same package index_test,
// same rationale as bluesky_profile_test.go:16).

// fakeBackfiller is used by CraftskyProfile tests in Chunk 4 but we also
// verify here that it satisfies the exported interface.
type fakeBackfiller struct {
	calls []syntax.DID
	err   error
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
	return errors.New("not used")
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
		t.Errorf("PDS GetRecord called %d times; want 1", pds.calls)
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
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`,
		"did:plc:none").Scan(&count)
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
