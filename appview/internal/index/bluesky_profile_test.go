// appview/internal/index/bluesky_profile_test.go
package index_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
)

// Reuses craftskyProfilesDDL from craftsky_profile_test.go.

func seedMember(t *testing.T, pool *pgxpool.Pool, did string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
		did, "seed"); err != nil {
		t.Fatal(err)
	}
}

func TestBlueskyProfile_CreateForMember(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:m")
	idx := index.NewBlueskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:m/app.bsky.actor.profile/self",
		CID:        "bafbluesky",
		DID:        "did:plc:m",
		Rkey:       "self",
		Collection: "app.bsky.actor.profile",
		Action:     "create",
		Record: json.RawMessage(`{
			"displayName": "Mallory",
			"description": "sews things",
			"avatar":   {"$type":"blob","ref":{"$link":"bafkavatar"},"mimeType":"image/jpeg","size":1},
			"banner":   {"$type":"blob","ref":{"$link":"bafkbanner"},"mimeType":"image/png","size":1}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var displayName, description, avatarCID, avatarMime, bannerCID, bannerMime, recordCID string
	err := pool.QueryRow(context.Background(), `
		SELECT display_name, description, avatar_cid, avatar_mime,
		       banner_cid, banner_mime, record_cid
		FROM bluesky_profiles WHERE did = $1`, ev.DID).
		Scan(&displayName, &description, &avatarCID, &avatarMime,
			&bannerCID, &bannerMime, &recordCID)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if displayName != "Mallory" {
		t.Errorf("display_name = %q", displayName)
	}
	if description != "sews things" {
		t.Errorf("description = %q", description)
	}
	if avatarCID != "bafkavatar" || avatarMime != "image/jpeg" {
		t.Errorf("avatar = (%q, %q)", avatarCID, avatarMime)
	}
	if bannerCID != "bafkbanner" || bannerMime != "image/png" {
		t.Errorf("banner = (%q, %q)", bannerCID, bannerMime)
	}
	if recordCID != "bafbluesky" {
		t.Errorf("record_cid = %q", recordCID)
	}
}

func TestBlueskyProfile_DropsForNonMember(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewBlueskyProfile(pool)

	ev := tap.Event{
		URI:        "at://did:plc:nm/app.bsky.actor.profile/self",
		CID:        "c",
		DID:        "did:plc:nm",
		Rkey:       "self",
		Collection: "app.bsky.actor.profile",
		Action:     "create",
		Record:     json.RawMessage(`{"displayName":"bob"}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Errorf("Handle should drop non-members without error; got %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&count)
	if count != 0 {
		t.Errorf("count = %d, want 0 (non-member must not be indexed)", count)
	}
}

func TestBlueskyProfile_UpdateReplacesFields(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:u")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	create := tap.Event{
		URI: "at://did:plc:u/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:u", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"old"}`),
	}
	update := create
	update.CID = "c2"
	update.Action = "update"
	update.Record = json.RawMessage(`{"displayName":"new"}`)

	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(ctx, update); err != nil {
		t.Fatal(err)
	}
	var dn, cid string
	if err := pool.QueryRow(ctx,
		`SELECT display_name, record_cid FROM bluesky_profiles WHERE did = $1`, create.DID).
		Scan(&dn, &cid); err != nil {
		t.Fatalf("select: %v", err)
	}
	if dn != "new" || cid != "c2" {
		t.Errorf("after update: display_name=%q record_cid=%q; want new, c2", dn, cid)
	}
}

func TestBlueskyProfile_ReplayedEventPreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:r")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	ev := tap.Event{
		URI: "at://did:plc:r/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:r", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"alice"}`),
	}
	if err := idx.Handle(ctx, ev); err != nil {
		t.Fatal(err)
	}

	var first string
	if err := pool.QueryRow(ctx,
		`SELECT indexed_at::text FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&first); err != nil {
		t.Fatalf("first select: %v", err)
	}

	if err := idx.Handle(ctx, ev); err != nil {
		t.Fatal(err)
	}

	var second string
	if err := pool.QueryRow(ctx,
		`SELECT indexed_at::text FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&second); err != nil {
		t.Fatalf("second select: %v", err)
	}

	if first != second {
		t.Errorf("indexed_at changed on replay: %q -> %q", first, second)
	}
}

func TestBlueskyProfile_DeleteRemovesRow(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:d")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	create := tap.Event{
		URI: "at://did:plc:d/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:d", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"x"}`),
	}
	if err := idx.Handle(ctx, create); err != nil {
		t.Fatal(err)
	}
	del := tap.Event{
		URI: create.URI, DID: create.DID, Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "delete",
	}
	if err := idx.Handle(ctx, del); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM bluesky_profiles WHERE did = $1`, del.DID).
		Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestBlueskyProfile_DeleteNonMemberIsNoop(t *testing.T) {
	t.Parallel()
	pool := withSchema(t, craftskyProfilesDDL)
	idx := index.NewBlueskyProfile(pool)
	del := tap.Event{
		URI: "at://did:plc:gone/app.bsky.actor.profile/self",
		DID: "did:plc:gone", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "delete",
	}
	if err := idx.Handle(context.Background(), del); err != nil {
		t.Errorf("delete on non-member should be silent; got %v", err)
	}
}
