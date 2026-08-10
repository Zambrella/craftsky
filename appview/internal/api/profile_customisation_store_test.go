package api_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const profileCustomisationStoreTestDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
`

func TestProfileCustomisationStorePersistsCompleteOwnerScopedValues(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read profile customisation migration: %v", err)
	}
	pool := testdb.WithSchema(t, profileCustomisationStoreTestDDL+string(migration))
	ctx := context.Background()
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	now := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	store := api.NewProfileCustomisationStore(pool, api.ProfileCustomisationStoreOptions{
		Now: func() time.Time { return now },
	})

	got, err := store.Read(ctx, alice)
	if err != nil {
		t.Fatalf("read missing customisation: %v", err)
	}
	if got != api.DefaultProfileCustomisation {
		t.Fatalf("missing customisation = %+v, want defaults %+v", got, api.DefaultProfileCustomisation)
	}

	aliceFirst := api.ProfileCustomisation{
		Colour:     "teal",
		Border:     "thin",
		Background: "x2",
	}
	got, err = store.Put(ctx, alice, aliceFirst)
	if err != nil {
		t.Fatalf("put first customisation: %v", err)
	}
	if got != aliceFirst {
		t.Fatalf("first put = %+v, want %+v", got, aliceFirst)
	}
	if retry, err := store.Put(ctx, alice, aliceFirst); err != nil || retry != aliceFirst {
		t.Fatalf("idempotent retry = %+v, %v", retry, err)
	}

	bobValue := api.ProfileCustomisation{
		Colour:     "rose",
		Border:     "thick",
		Background: "scallopdark",
	}
	if got, err := store.Put(ctx, bob, bobValue); err != nil || got != bobValue {
		t.Fatalf("put Bob customisation = %+v, %v", got, err)
	}

	aliceReplacement := api.ProfileCustomisation{
		Colour:     "amber",
		Border:     "medium",
		Background: "bayerdark",
	}
	now = now.Add(time.Minute)
	if got, err := store.Put(ctx, alice, aliceReplacement); err != nil || got != aliceReplacement {
		t.Fatalf("replace Alice customisation = %+v, %v", got, err)
	}

	reloaded := api.NewProfileCustomisationStore(pool)
	for owner, want := range map[syntax.DID]api.ProfileCustomisation{
		alice: aliceReplacement,
		bob:   bobValue,
	} {
		got, err := reloaded.Read(ctx, owner)
		if err != nil {
			t.Fatalf("read reloaded %s: %v", owner, err)
		}
		if got != want {
			t.Fatalf("reloaded %s = %+v, want %+v", owner, got, want)
		}
	}

	var createdAt, updatedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT created_at, updated_at
		FROM profile_customisations
		WHERE owner_did = $1
	`, alice).Scan(&createdAt, &updatedAt); err != nil {
		t.Fatalf("read Alice timestamps: %v", err)
	}
	if !createdAt.Equal(time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)) || !updatedAt.Equal(now) {
		t.Fatalf("Alice timestamps = (%s, %s), want creation preserved and update advanced", createdAt, updatedAt)
	}
}

func TestProfileCustomisationStoreFallsBackPerPersistedField(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000036_profile_customisation.up.sql")
	if err != nil {
		t.Fatalf("read profile customisation migration: %v", err)
	}
	pool := testdb.WithSchema(t, profileCustomisationStoreTestDDL+string(migration))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_customisations (
			owner_did, colour, profile_border, profile_background
		) VALUES ($1, 'retired-colour', 'thick', 'cubedark')
	`, owner); err != nil {
		t.Fatalf("seed retired customisation: %v", err)
	}

	got, err := api.NewProfileCustomisationStore(pool).Read(ctx, owner)
	if err != nil {
		t.Fatalf("read retired customisation: %v", err)
	}
	want := api.ProfileCustomisation{
		Colour:     "cobalt",
		Border:     "thick",
		Background: "cubedark",
	}
	if got != want {
		t.Fatalf("effective stored customisation = %+v, want %+v", got, want)
	}
}
