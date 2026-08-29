package relationships_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessMembershipDepartureAndRejoinRetainsAndRestoresState(t *testing.T) {
	accountTypesMigration, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read account types migration: %v", err)
	}
	businessRecordsMigration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read business records migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY,
			state TEXT NOT NULL CHECK (state IN ('active', 'departed', 'deletion_pending', 'deleting', 'terminal')),
			generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL,
			transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL,
			terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE owner_effect_attempts (
			owner_did TEXT NOT NULL,
			owner_generation BIGINT NOT NULL,
			remote_outcome TEXT NOT NULL,
			projection_disposition TEXT NOT NULL,
			repeat_forbidden BOOLEAN NOT NULL,
			completed_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE OR REPLACE FUNCTION appview_owner_is_active(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((
				SELECT state = 'active' FROM owner_lifecycles WHERE owner_did = candidate_did
			), false)
		$$;
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY,
			crafts TEXT[] NOT NULL DEFAULT '{}',
			record_cid TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE bluesky_profiles (did TEXT PRIMARY KEY);
		CREATE TABLE atproto_blocks (
			uri TEXT PRIMARY KEY,
			blocker_did TEXT NOT NULL,
			subject_did TEXT NOT NULL
		);
		CREATE TABLE moderation_outputs (
			id TEXT PRIMARY KEY,
			source_did TEXT NOT NULL,
			subject_type TEXT NOT NULL,
			subject_did TEXT NOT NULL,
			subject_uri TEXT,
			value TEXT NOT NULL,
			action TEXT NOT NULL,
			expires_at TIMESTAMPTZ,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`+string(accountTypesMigration)+string(businessRecordsMigration))
	ctx := context.Background()
	owner := syntax.DID("did:plc:business-lifecycle-owner")
	visitor := syntax.DID("did:plc:business-lifecycle-visitor")
	rkey := syntax.RecordKey("3mslifecycle01")
	eventURI := syntax.ATURI("at://did:plc:business-lifecycle-owner/social.craftsky.business.event/3mslifecycle01")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	eventRecord, err := json.Marshal(map[string]any{
		"$type":     "social.craftsky.business.event",
		"name":      "Retained business event",
		"startsAt":  asOf.Add(time.Hour).Format(time.RFC3339),
		"endsAt":    asOf.Add(2 * time.Hour).Format(time.RFC3339),
		"roles":     []string{"vendor"},
		"createdAt": asOf.Add(-24 * time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("marshal event record: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did, state, generation, auth_epoch, transition_reason,
			transitioned_at, created_at, updated_at
		) VALUES ($1, 'active', 1, 1, 'testSetup', now(), now(), now())
	`, owner); err != nil {
		t.Fatalf("seed active lifecycle: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'membership-cid')`, owner); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES ($1, 'at://did:plc:business-lifecycle-owner/social.craftsky.business.profile/self',
			'business-profile-cid', '{"$type":"social.craftsky.business.profile","tagline":"Persisted details"}', '3msprofile001')
	`, owner); err != nil {
		t.Fatalf("seed business profile: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_events(
			uri, owner_did, rkey, cid, raw_record, source_revision, starts_at, ends_at, created_at, status
		) VALUES ($2, $1, $3, 'business-event-cid', $4, '3msevent001', $5, $6, $7, NULL)
	`, owner, eventURI, rkey, eventRecord, asOf.Add(time.Hour), asOf.Add(2*time.Hour), asOf.Add(-24*time.Hour)); err != nil {
		t.Fatalf("seed business event: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_record_tombstones(uri, owner_did, collection, source_revision)
		VALUES ('at://did:plc:business-lifecycle-owner/social.craftsky.business.event/3msdeleted001',
			$1, 'social.craftsky.business.event', '3mstombstone1')
	`, owner); err != nil {
		t.Fatalf("seed business tombstone: %v", err)
	}

	memberships := relationships.NewStore(pool)
	store := business.NewStore(pool)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new lifecycle fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return asOf })
	if err != nil {
		t.Fatalf("new lifecycle store: %v", err)
	}
	membershipProjection := index.NewCraftskyProfile(pool, lifecycleBackfiller{}, nil)
	assertBusinessMembershipServing(t, memberships, store, owner, visitor, rkey, asOf, true)

	departed, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 1, To: ownerlifecycle.StateDeparted, Reason: "profileDeleted",
	})
	if err != nil {
		t.Fatalf("transition departed: %v", err)
	}
	if err := membershipProjection.Handle(ctx, tap.Event{
		URI: "at://did:plc:business-lifecycle-owner/social.craftsky.actor.profile/self",
		DID: owner, Collection: "social.craftsky.actor.profile", Rkey: "self", Action: "delete",
	}); err != nil {
		t.Fatalf("project membership departure: %v", err)
	}
	assertBusinessMembershipServing(t, memberships, store, owner, visitor, rkey, asOf, false)
	assertRetainedBusinessLifecycleRows(t, pool, owner)

	if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation, To: ownerlifecycle.StateActive, Reason: "profileCreated",
	}); err != nil {
		t.Fatalf("transition active: %v", err)
	}
	if err := membershipProjection.Handle(ctx, tap.Event{
		URI: "at://did:plc:business-lifecycle-owner/social.craftsky.actor.profile/self",
		DID: owner, Collection: "social.craftsky.actor.profile", Rkey: "self",
		CID: "membership-cid-rejoined", Action: "create", Record: json.RawMessage(`{"crafts":[]}`),
	}); err != nil {
		t.Fatalf("project membership rejoin: %v", err)
	}
	assertBusinessMembershipServing(t, memberships, store, owner, visitor, rkey, asOf, true)
	assertRetainedBusinessLifecycleRows(t, pool, owner)
}

type lifecycleBackfiller struct{}

func (lifecycleBackfiller) Backfill(context.Context, syntax.DID) error { return nil }

func assertBusinessMembershipServing(
	t *testing.T,
	memberships *relationships.Store,
	store *business.Store,
	owner, visitor syntax.DID,
	rkey syntax.RecordKey,
	asOf time.Time,
	wantServed bool,
) {
	t.Helper()
	ctx := context.Background()
	current, err := memberships.IsCurrentMember(ctx, owner)
	if err != nil {
		t.Fatalf("read current membership: %v", err)
	}
	accountType, err := store.ReadAccountType(ctx, owner)
	if err != nil {
		t.Fatalf("read persisted account type: %v", err)
	}
	if got := business.IsBusinessClassified(current, accountType); got != wantServed {
		t.Fatalf("business classification = %v, want %v", got, wantServed)
	}

	profile, err := store.ReadEligibleProfile(ctx, owner)
	if err != nil {
		t.Fatalf("read eligible business profile: %v", err)
	}
	_, eventErr := store.ReadEvent(ctx, business.EventReadInput{
		CallerDID: visitor,
		OwnerDID:  owner,
		Rkey:      rkey,
		AsOf:      asOf,
	})
	events, err := store.ListUpcomingEvents(ctx, business.UpcomingEventListInput{
		CallerDID: visitor,
		OwnerDID:  owner,
		AsOf:      asOf,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("list upcoming events: %v", err)
	}
	if wantServed {
		if profile == nil || profile.Tagline != "Persisted details" || eventErr != nil || len(events) != 1 {
			t.Fatalf("served business state = profile %+v, direct error %v, events %+v", profile, eventErr, events)
		}
		return
	}
	if profile != nil || !errors.Is(eventErr, business.ErrEventNotFound) || len(events) != 0 {
		t.Fatalf("departed business state leaked = profile %+v, direct error %v, events %+v", profile, eventErr, events)
	}
}

func assertRetainedBusinessLifecycleRows(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) {
	t.Helper()
	var accountTypes, profiles, events, tombstones int
	row := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM craftsky_account_types WHERE owner_did = $1),
			(SELECT count(*) FROM craftsky_business_profiles WHERE owner_did = $1),
			(SELECT count(*) FROM craftsky_business_events WHERE owner_did = $1),
			(SELECT count(*) FROM craftsky_business_record_tombstones WHERE owner_did = $1)
	`, owner)
	if err := row.Scan(&accountTypes, &profiles, &events, &tombstones); err != nil {
		t.Fatalf("count retained business rows: %v", err)
	}
	if accountTypes != 1 || profiles != 1 || events != 1 || tombstones != 1 {
		t.Fatalf("retained account-type/profile/event/tombstone rows = %d/%d/%d/%d, want 1/1/1/1",
			accountTypes, profiles, events, tombstones)
	}
}
