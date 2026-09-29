// appview/internal/index/bluesky_profile_test.go
package index_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/imagesafety"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
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
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
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
			"pronouns": "she/her",
			"avatar":   {"$type":"blob","ref":{"$link":"bafkavatar"},"mimeType":"image/jpeg","size":1},
			"banner":   {"$type":"blob","ref":{"$link":"bafkbanner"},"mimeType":"image/png","size":1}
		}`),
	}
	if err := idx.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var displayName, description, pronouns, avatarCID, avatarMime, bannerCID, bannerMime, recordCID string
	err := pool.QueryRow(context.Background(), `
		SELECT display_name, description, pronouns, avatar_cid, avatar_mime,
		       banner_cid, banner_mime, record_cid
		FROM bluesky_profiles WHERE did = $1`, ev.DID).
		Scan(&displayName, &description, &pronouns, &avatarCID, &avatarMime,
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
	if pronouns != "she/her" {
		t.Errorf("pronouns = %q", pronouns)
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

func TestBlueskyProfile_CreatesForNonMember(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
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
		t.Errorf("Handle create for non-member: %v", err)
	}
	var count int
	_ = pool.QueryRow(context.Background(),
		`SELECT count(*) FROM bluesky_profiles WHERE did = $1`, ev.DID).Scan(&count)
	if count != 1 {
		t.Errorf("count = %d, want 1 (non-member should be indexed)", count)
	}
}

func TestBlueskyProfile_UpdateReplacesFields(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
	seedMember(t, pool, "did:plc:u")
	idx := index.NewBlueskyProfile(pool)
	ctx := context.Background()

	create := tap.Event{
		URI: "at://did:plc:u/app.bsky.actor.profile/self", CID: "c1",
		DID: "did:plc:u", Rkey: "self",
		Collection: "app.bsky.actor.profile", Action: "create",
		Record: json.RawMessage(`{"displayName":"old","pronouns":"they/them"}`),
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
	var pronouns *string
	if err := pool.QueryRow(ctx,
		`SELECT display_name, pronouns, record_cid FROM bluesky_profiles WHERE did = $1`, create.DID).
		Scan(&dn, &pronouns, &cid); err != nil {
		t.Fatalf("select: %v", err)
	}
	if dn != "new" || cid != "c2" {
		t.Errorf("after update: display_name=%q record_cid=%q; want new, c2", dn, cid)
	}
	if pronouns != nil {
		t.Errorf("pronouns = %q, want NULL after omission", *pronouns)
	}
}

func TestBlueskyProfile_ReplayedEventPreservesIndexedAt(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
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
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
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
	pool := testdb.WithSchema(t, craftskyProfilesDDL)
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

func TestImageSafetyBlueskyProfileKeepsLastClearImagesIndependently(t *testing.T) {
	pool := testdb.WithSchema(t, craftskyProfilesDDL+imageScanTapPreStateDDL)
	migration, err := os.ReadFile("../../migrations/000073_image_safety.up.sql")
	if err != nil {
		t.Fatalf("read image safety migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
		t.Fatalf("apply image safety migration: %v", err)
	}

	ctx := context.Background()
	key := imagesafety.ScanKey{ScannerID: "fixture", PolicyVersion: "policy-1", CorpusVersion: "corpus-1"}
	const (
		oldAvatar = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
		oldBanner = "bafkreidjq52a7nre4puzipwf3gwfkgnxftvbwnp3jppfogo7her2g3ai64"
		newAvatar = "bafkreibm6jgql3m7ta4szj3q5wo7fkiny2hzqoqs4dbc65d5u3r3itkqae"
		newBanner = "bafkreic5jbn7z4xkzqh4mmyv5x5xjhzlb2y6h5b56tn3m76kt4qaxwji5u"
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO image_scan_results(
			id,blob_cid,scanner_id,policy_version,corpus_version,state,completed_at
		) VALUES
			('20000000-0000-4000-8000-000000000001',$1,$5,$6,$7,'clear',now()),
			('20000000-0000-4000-8000-000000000002',$2,$5,$6,$7,'clear',now()),
			('20000000-0000-4000-8000-000000000003',$3,$5,$6,$7,'pending',NULL),
			('20000000-0000-4000-8000-000000000004',$4,$5,$6,$7,'clear',now())
	`, oldAvatar, oldBanner, newAvatar, newBanner, key.ScannerID, key.PolicyVersion, key.CorpusVersion); err != nil {
		t.Fatalf("seed scan results: %v", err)
	}

	idx := index.NewImageSafetyBlueskyProfile(pool, key)
	event := tap.Event{
		URI:        "at://did:plc:safe-profile/app.bsky.actor.profile/self",
		CID:        "profile-cid-1",
		DID:        "did:plc:safe-profile",
		Rkey:       "self",
		Collection: "app.bsky.actor.profile",
		Action:     "create",
		Record: json.RawMessage(`{
			"displayName":"First",
			"avatar":{"$type":"blob","ref":{"$link":"` + oldAvatar + `"},"mimeType":"image/jpeg","size":10},
			"banner":{"$type":"blob","ref":{"$link":"` + oldBanner + `"},"mimeType":"image/png","size":20}
		}`),
	}
	if err := idx.Handle(ctx, event); err != nil {
		t.Fatalf("project initial clear profile: %v", err)
	}
	assertProfileImages(t, pool, event.DID.String(), oldAvatar, oldBanner)

	event.Action = "update"
	event.CID = "profile-cid-2"
	event.Record = json.RawMessage(`{
		"displayName":"Second",
		"avatar":{"$type":"blob","ref":{"$link":"` + newAvatar + `"},"mimeType":"image/webp","size":30},
		"banner":{"$type":"blob","ref":{"$link":"` + newBanner + `"},"mimeType":"image/jpeg","size":40}
	}`)
	if err := idx.Handle(ctx, event); err != nil {
		t.Fatalf("project mixed-state replacement: %v", err)
	}
	assertProfileImages(t, pool, event.DID.String(), oldAvatar, newBanner)

	if _, err := pool.Exec(ctx, `
		UPDATE image_scan_results SET state='clear', completed_at=now(), updated_at=now()
		WHERE blob_cid=$1
	`, newAvatar); err != nil {
		t.Fatalf("clear replacement avatar: %v", err)
	}
	if err := idx.Handle(ctx, event); err != nil {
		t.Fatalf("replay current profile after clear: %v", err)
	}
	assertProfileImages(t, pool, event.DID.String(), newAvatar, newBanner)

	placeholder := event
	placeholder.URI = "at://did:plc:placeholder/app.bsky.actor.profile/self"
	placeholder.DID = "did:plc:placeholder"
	placeholder.CID = "profile-cid-placeholder"
	placeholder.Action = "create"
	placeholder.Record = json.RawMessage(`{
		"displayName":"Placeholder",
		"avatar":{"$type":"blob","ref":{"$link":"` + oldAvatar + `"},"mimeType":"image/jpeg","size":10}
	}`)
	if _, err := pool.Exec(ctx, `UPDATE image_scan_results SET state='pending', completed_at=NULL WHERE blob_cid=$1`, oldAvatar); err != nil {
		t.Fatal(err)
	}
	if err := idx.Handle(ctx, placeholder); err != nil {
		t.Fatalf("project profile without clear history: %v", err)
	}
	assertProfileImages(t, pool, placeholder.DID.String(), "", "")
}

func assertProfileImages(t *testing.T, pool *pgxpool.Pool, did, wantAvatar, wantBanner string) {
	t.Helper()
	var avatar, banner *string
	if err := pool.QueryRow(context.Background(), `
		SELECT avatar_cid,banner_cid FROM bluesky_profiles WHERE did=$1
	`, did).Scan(&avatar, &banner); err != nil {
		t.Fatalf("read serving profile images: %v", err)
	}
	if got := stringValue(avatar); got != wantAvatar {
		t.Fatalf("avatar=%q, want %q", got, wantAvatar)
	}
	if got := stringValue(banner); got != wantBanner {
		t.Fatalf("banner=%q, want %q", got, wantBanner)
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
