// appview/internal/api/profile_store_test.go
package api_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const profileStoreDDL = `
CREATE TABLE craftsky_profiles (
    did         TEXT        NOT NULL PRIMARY KEY,
    crafts      TEXT[]      NOT NULL DEFAULT '{}',
    record_cid  TEXT        NOT NULL,
    indexed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
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

func TestProfileStore_ReadByDID_MemberWithBothRows(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, $2, $3)`,
		"did:plc:a", []string{"sewing"}, "cid1")
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO bluesky_profiles (did, display_name, avatar_cid, avatar_mime, record_cid)
		VALUES ($1, $2, $3, $4, $5)`,
		"did:plc:a", "Alice", "bafav", "image/jpeg", "cid2")
	if err != nil {
		t.Fatal(err)
	}

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:a")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.DID != "did:plc:a" {
		t.Errorf("DID = %q", got.DID)
	}
	if got.DisplayName == nil || *got.DisplayName != "Alice" {
		t.Errorf("DisplayName = %v", got.DisplayName)
	}
	if got.AvatarCID == nil || *got.AvatarCID != "bafav" {
		t.Errorf("AvatarCID = %v", got.AvatarCID)
	}
	if len(got.Crafts) != 1 || got.Crafts[0] != "sewing" {
		t.Errorf("Crafts = %v", got.Crafts)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("CreatedAt is zero")
	}
}

func TestProfileStore_ReadByDID_NonMember(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	store := api.NewProfileStore(pool)
	_, err := store.Read(context.Background(), "did:plc:nobody")
	if err == nil {
		t.Fatal("want error; got nil")
	}
	if err != api.ErrProfileNotFound {
		t.Errorf("want ErrProfileNotFound; got %v", err)
	}
}

func TestProfileStore_ReadByDID_MemberWithoutBlueskyRow(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, profileStoreDDL)
	ctx := context.Background()
	_, _ = pool.Exec(ctx,
		`INSERT INTO craftsky_profiles (did, crafts, record_cid) VALUES ($1, $2, $3)`,
		"did:plc:b", []string{}, "cid1")

	store := api.NewProfileStore(pool)
	got, err := store.Read(ctx, "did:plc:b")
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != nil {
		t.Errorf("DisplayName should be nil; got %v", *got.DisplayName)
	}
	if len(got.Crafts) != 0 {
		t.Errorf("Crafts = %v, want empty", got.Crafts)
	}
}
