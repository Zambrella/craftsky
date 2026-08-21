package api_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestProfilePinStoreUnpinRejectsStaleOwnerGeneration(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, profilePinStoreTestDDL+string(migration))
	ctx := context.Background()
	owner := syntax.DID("did:plc:alice")
	target := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-a")
	if _, err := pool.Exec(ctx, `
		INSERT INTO profile_pins(owner_did,slot,post_uri,state_token,created_at,updated_at)
		VALUES($1,'standard',$2,'00000000-0000-4000-8000-000000000001',now(),now())
	`, owner, target); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE owner_lifecycles SET generation=2 WHERE owner_did=$1`, owner); err != nil {
		t.Fatal(err)
	}

	_, err = api.NewProfilePinStore(pool).Unpin(
		ownerlifecycle.WithExpectedGeneration(ctx, 1),
		owner,
		target,
	)
	if !errors.Is(err, ownerlifecycle.ErrGenerationChanged) {
		t.Fatalf("Unpin error = %v, want ErrGenerationChanged", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did=$1`, owner).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("profile pins = %d, want 1", count)
	}
}

const profilePinStoreTestDDL = `
CREATE TABLE craftsky_profiles (
    did        TEXT NOT NULL PRIMARY KEY,
    record_cid TEXT NOT NULL
);
CREATE TABLE owner_lifecycles (
	owner_did TEXT PRIMARY KEY,
	state TEXT NOT NULL,
	generation BIGINT NOT NULL,
	auth_epoch BIGINT NOT NULL DEFAULT 1,
	transition_reason TEXT NOT NULL DEFAULT 'test',
	transitioned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	terminal_at TIMESTAMPTZ,
	purge_completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE FUNCTION seed_active_profile_pin_owner() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
	INSERT INTO owner_lifecycles(owner_did,state,generation)
	VALUES(NEW.did,'active',1)
	ON CONFLICT (owner_did) DO NOTHING;
	RETURN NEW;
END
$$;
CREATE TRIGGER seed_active_profile_pin_owner
AFTER INSERT ON craftsky_profiles
FOR EACH ROW EXECUTE FUNCTION seed_active_profile_pin_owner();
CREATE TABLE craftsky_posts (
    uri              TEXT    NOT NULL PRIMARY KEY,
    did              TEXT    NOT NULL,
    rkey             TEXT    NOT NULL,
    cid               TEXT    NOT NULL,
    reply_root_uri    TEXT,
    reply_parent_uri  TEXT,
    quote_uri         TEXT,
    is_project        BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (did, rkey)
);
CREATE TABLE craftsky_project_posts (
    uri TEXT NOT NULL PRIMARY KEY REFERENCES craftsky_posts(uri) ON DELETE CASCADE
);
CREATE TABLE moderation_outputs (
    id           TEXT        NOT NULL PRIMARY KEY,
    source_did   TEXT        NOT NULL,
    subject_type TEXT        NOT NULL,
    subject_did  TEXT        NOT NULL,
    subject_uri  TEXT,
    value        TEXT        NOT NULL,
    action       TEXT        NOT NULL,
    expires_at   TIMESTAMPTZ,
    indexed_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO craftsky_profiles (did, record_cid)
VALUES ('did:plc:alice', 'alice-cid'), ('did:plc:bob', 'bob-cid');
INSERT INTO craftsky_posts (
    uri, did, rkey, cid, reply_root_uri, reply_parent_uri, quote_uri, is_project
)
VALUES
	    ('at://did:plc:alice/social.craftsky.feed.post/standard-a', 'did:plc:alice', 'standard-a', 'cid-standard-a', NULL, NULL, NULL, false),
	    ('at://did:plc:alice/social.craftsky.feed.post/standard-b', 'did:plc:alice', 'standard-b', 'cid-standard-b', NULL, NULL, NULL, false),
	    ('at://did:plc:alice/social.craftsky.feed.post/standard-c', 'did:plc:alice', 'standard-c', 'cid-standard-c', NULL, NULL, NULL, false),
	    ('at://did:plc:alice/social.craftsky.feed.post/quote', 'did:plc:alice', 'quote', 'cid-quote', NULL, NULL, 'at://did:plc:bob/social.craftsky.feed.post/other', false),
	    ('at://did:plc:alice/social.craftsky.feed.post/comment', 'did:plc:alice', 'comment', 'cid-comment', 'root', 'root', NULL, false),
	    ('at://did:plc:alice/social.craftsky.feed.post/partial-reply', 'did:plc:alice', 'partial-reply', 'cid-partial', 'root', NULL, NULL, false),
	    ('at://did:plc:alice/social.craftsky.feed.post/project-a', 'did:plc:alice', 'project-a', 'cid-project-a', NULL, NULL, NULL, true),
	    ('at://did:plc:alice/social.craftsky.feed.post/project-b', 'did:plc:alice', 'project-b', 'cid-project-b', NULL, NULL, NULL, true),
	    ('at://did:plc:alice/social.craftsky.feed.post/project-quote', 'did:plc:alice', 'project-quote', 'cid-project-quote', NULL, NULL, 'quoted', true),
	    ('at://did:plc:alice/social.craftsky.feed.post/project-missing', 'did:plc:alice', 'project-missing', 'cid-project-missing', NULL, NULL, NULL, true),
	    ('at://did:plc:alice/social.craftsky.feed.post/standard-materialized', 'did:plc:alice', 'standard-materialized', 'cid-standard-materialized', NULL, NULL, NULL, false),
	    ('at://did:plc:alice/social.craftsky.feed.post/hidden', 'did:plc:alice', 'hidden', 'cid-hidden', NULL, NULL, NULL, false),
	    ('at://did:plc:bob/social.craftsky.feed.post/other', 'did:plc:bob', 'other', 'cid-other', NULL, NULL, NULL, false),
	    ('at://did:plc:dave/social.craftsky.feed.post/nonmember', 'did:plc:dave', 'nonmember', 'cid-nonmember', NULL, NULL, NULL, false);
INSERT INTO craftsky_project_posts (uri)
VALUES
	    ('at://did:plc:alice/social.craftsky.feed.post/project-a'),
	    ('at://did:plc:alice/social.craftsky.feed.post/project-b'),
	    ('at://did:plc:alice/social.craftsky.feed.post/project-quote'),
	    ('at://did:plc:alice/social.craftsky.feed.post/standard-materialized');
INSERT INTO moderation_outputs (
    id, source_did, subject_type, subject_did, subject_uri, value, action
) VALUES (
    'hidden-post', 'did:plc:moderator', 'post', 'did:plc:alice',
    'at://did:plc:alice/social.craftsky.feed.post/hidden', 'hide', 'apply'
);
`

func TestProfilePinStorePersistsIndependentIdempotentAndReplacementStates(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, profilePinStoreTestDDL+string(migration))
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	owner := syntax.DID("did:plc:alice")

	times := []time.Time{
		time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 8, 5, 10, 1, 0, 0, time.UTC),
		time.Date(2026, 8, 5, 10, 2, 0, 0, time.UTC),
		time.Date(2026, 8, 5, 10, 3, 0, 0, time.UTC),
	}
	ids := []uuid.UUID{
		uuid.MustParse("00000000-0000-4000-8000-000000000001"),
		uuid.MustParse("00000000-0000-4000-8000-000000000002"),
		uuid.MustParse("00000000-0000-4000-8000-000000000003"),
		uuid.MustParse("00000000-0000-4000-8000-000000000004"),
	}
	nowCall := 0
	idCall := 0
	store := api.NewProfilePinStore(pool, api.ProfilePinStoreOptions{
		Now: func() time.Time {
			value := times[nowCall]
			nowCall++
			return value
		},
		NewID: func() uuid.UUID {
			value := ids[idCall]
			idCall++
			return value
		},
	})

	empty, err := store.Read(ctx, owner)
	if err != nil {
		t.Fatalf("read empty pin state: %v", err)
	}
	assertProfilePinState(t, empty, "", "")

	standardA, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-a"))
	if err != nil {
		t.Fatalf("pin standard A: %v", err)
	}
	if standardA.Slot != api.ProfilePinSlotStandard || standardA.Operation != api.ProfilePinOperationPin {
		t.Fatalf("standard A mutation = slot %q operation %q", standardA.Slot, standardA.Operation)
	}
	assertProfilePinState(t, standardA.State, "at://did:plc:alice/social.craftsky.feed.post/standard-a", "")

	var firstToken string
	var firstCreatedAt, firstUpdatedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state_token::text, created_at, updated_at
		FROM profile_pins
		WHERE owner_did = $1 AND slot = 'standard'
	`, owner).Scan(&firstToken, &firstCreatedAt, &firstUpdatedAt); err != nil {
		t.Fatalf("read first standard pin internals: %v", err)
	}

	sameStandard, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-a"))
	if err != nil {
		t.Fatalf("pin standard A idempotently: %v", err)
	}
	if sameStandard.Operation != api.ProfilePinOperationNoop {
		t.Fatalf("same-target operation = %q, want noop", sameStandard.Operation)
	}
	assertProfilePinState(t, sameStandard.State, "at://did:plc:alice/social.craftsky.feed.post/standard-a", "")
	var sameToken string
	var sameCreatedAt, sameUpdatedAt time.Time
	if err := pool.QueryRow(ctx, `
		SELECT state_token::text, created_at, updated_at
		FROM profile_pins
		WHERE owner_did = $1 AND slot = 'standard'
	`, owner).Scan(&sameToken, &sameCreatedAt, &sameUpdatedAt); err != nil {
		t.Fatalf("read idempotent standard pin internals: %v", err)
	}
	if sameToken != firstToken || !sameCreatedAt.Equal(firstCreatedAt) || !sameUpdatedAt.Equal(firstUpdatedAt) {
		t.Fatalf("same-target pin rewrote internals: token %q/%q created %s/%s updated %s/%s", sameToken, firstToken, sameCreatedAt, firstCreatedAt, sameUpdatedAt, firstUpdatedAt)
	}

	projectA, err := store.Pin(ctx, owner, owner, syntax.RecordKey("project-a"))
	if err != nil {
		t.Fatalf("pin project A: %v", err)
	}
	if projectA.Slot != api.ProfilePinSlotProject || projectA.Operation != api.ProfilePinOperationPin {
		t.Fatalf("project A mutation = slot %q operation %q", projectA.Slot, projectA.Operation)
	}
	assertProfilePinState(t, projectA.State,
		"at://did:plc:alice/social.craftsky.feed.post/standard-a",
		"at://did:plc:alice/social.craftsky.feed.post/project-a",
	)

	standardB, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-b"))
	if err != nil {
		t.Fatalf("replace with standard B: %v", err)
	}
	if standardB.Slot != api.ProfilePinSlotStandard || standardB.Operation != api.ProfilePinOperationReplace {
		t.Fatalf("standard B mutation = slot %q operation %q", standardB.Slot, standardB.Operation)
	}
	assertProfilePinState(t, standardB.State,
		"at://did:plc:alice/social.craftsky.feed.post/standard-b",
		"at://did:plc:alice/social.craftsky.feed.post/project-a",
	)

	var (
		pinCount         int
		replacementToken string
	)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did = $1`, owner).Scan(&pinCount); err != nil {
		t.Fatalf("count final pins: %v", err)
	}
	if pinCount != 2 {
		t.Fatalf("final pin count = %d, want 2", pinCount)
	}
	if err := pool.QueryRow(ctx, `
		SELECT state_token::text FROM profile_pins
		WHERE owner_did = $1 AND slot = 'standard'
	`, owner).Scan(&replacementToken); err != nil {
		t.Fatalf("read replacement token: %v", err)
	}
	if replacementToken == firstToken {
		t.Fatal("replacement retained the previous state token")
	}
	if nowCall != 3 || idCall != 3 {
		t.Fatalf("clock/id calls = %d/%d, want 3/3 for two inserts and one replacement", nowCall, idCall)
	}
}

func TestProfilePinStoreKeepsOwnersIsolatedAcrossReloadAndMembershipRemoval(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, profilePinStoreTestDDL+string(migration))
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	store := api.NewProfilePinStore(pool)
	if _, err := store.Pin(ctx, alice, alice, syntax.RecordKey("standard-a")); err != nil {
		t.Fatalf("pin Alice: %v", err)
	}
	if _, err := store.Pin(ctx, bob, bob, syntax.RecordKey("other")); err != nil {
		t.Fatalf("pin Bob: %v", err)
	}

	reloaded := api.NewProfilePinStore(pool)
	aliceState, err := reloaded.Read(ctx, alice)
	if err != nil {
		t.Fatalf("reload Alice: %v", err)
	}
	bobState, err := reloaded.Read(ctx, bob)
	if err != nil {
		t.Fatalf("reload Bob: %v", err)
	}
	assertProfilePinState(t, aliceState, "at://did:plc:alice/social.craftsky.feed.post/standard-a", "")
	assertProfilePinState(t, bobState, "at://did:plc:bob/social.craftsky.feed.post/other", "")

	if _, err := reloaded.Pin(ctx, alice, alice, syntax.RecordKey("standard-b")); err != nil {
		t.Fatalf("replace Alice: %v", err)
	}
	bobAfterAliceMutation, err := reloaded.Read(ctx, bob)
	if err != nil {
		t.Fatalf("read Bob after Alice mutation: %v", err)
	}
	assertProfilePinState(t, bobAfterAliceMutation, "at://did:plc:bob/social.craftsky.feed.post/other", "")

	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did = $1`, alice); err != nil {
		t.Fatalf("remove Alice membership: %v", err)
	}
	aliceAfterRemoval, err := reloaded.Read(ctx, alice)
	if err != nil {
		t.Fatalf("read removed Alice: %v", err)
	}
	bobAfterRemoval, err := reloaded.Read(ctx, bob)
	if err != nil {
		t.Fatalf("read Bob after Alice removal: %v", err)
	}
	assertProfilePinState(t, aliceAfterRemoval, "", "")
	assertProfilePinState(t, bobAfterRemoval, "at://did:plc:bob/social.craftsky.feed.post/other", "")
}

func TestProfilePinStoreSerializesReplacementAndTargetSpecificUnpin(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, profilePinStoreTestDDL+string(migration))
	baseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx := ownerlifecycle.WithExpectedGeneration(baseCtx, 1)
	owner := syntax.DID("did:plc:alice")
	store := api.NewProfilePinStore(pool)

	if _, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-a")); err != nil {
		t.Fatalf("seed standard A pin: %v", err)
	}

	barrier, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin target barrier: %v", err)
	}
	defer func() { _ = barrier.Rollback(context.Background()) }()
	if _, err := barrier.Exec(ctx, `
		SELECT uri FROM craftsky_posts
		WHERE uri = 'at://did:plc:alice/social.craftsky.feed.post/standard-b'
		FOR UPDATE
	`); err != nil {
		t.Fatalf("lock standard B target: %v", err)
	}

	type pinResult struct {
		mutation api.ProfilePinMutationResult
		err      error
	}
	standardBResult := make(chan pinResult, 1)
	go func() {
		mutation, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-b"))
		standardBResult <- pinResult{mutation: mutation, err: err}
	}()

	standardC, err := store.Pin(ctx, owner, owner, syntax.RecordKey("standard-c"))
	if err != nil {
		t.Fatalf("commit standard C while B target is blocked: %v", err)
	}
	assertProfilePinState(t, standardC.State, "at://did:plc:alice/social.craftsky.feed.post/standard-c", "")

	if err := barrier.Commit(ctx); err != nil {
		t.Fatalf("release standard B target barrier: %v", err)
	}
	standardB := <-standardBResult
	if standardB.err != nil {
		t.Fatalf("commit standard B last: %v", standardB.err)
	}
	assertProfilePinState(t, standardB.mutation.State, "at://did:plc:alice/social.craftsky.feed.post/standard-b", "")

	stale, err := store.Unpin(ctx, owner, syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-c"))
	if err != nil {
		t.Fatalf("stale unpin standard C: %v", err)
	}
	if stale.Operation != api.ProfilePinOperationNoop {
		t.Fatalf("stale unpin operation = %q, want noop", stale.Operation)
	}
	assertProfilePinState(t, stale.State, "at://did:plc:alice/social.craftsky.feed.post/standard-b", "")

	removed, err := store.Unpin(ctx, owner, syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/standard-b"))
	if err != nil {
		t.Fatalf("unpin current standard B: %v", err)
	}
	if removed.Slot != api.ProfilePinSlotStandard || removed.Operation != api.ProfilePinOperationUnpin {
		t.Fatalf("current unpin = slot %q operation %q", removed.Slot, removed.Operation)
	}
	assertProfilePinState(t, removed.State, "", "")
}

func assertProfilePinState(t *testing.T, state api.ProfilePinState, standardURI, projectURI string) {
	t.Helper()
	if got := optionalATURIString(state.StandardPostURI); got != standardURI {
		t.Fatalf("standard pin = %q, want %q", got, standardURI)
	}
	if got := optionalATURIString(state.ProjectPostURI); got != projectURI {
		t.Fatalf("project pin = %q, want %q", got, projectURI)
	}
}

func optionalATURIString(uri *syntax.ATURI) string {
	if uri == nil {
		return ""
	}
	return uri.String()
}
