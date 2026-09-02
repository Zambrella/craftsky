package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessEligibilityDoesNotDependOnDeclaration(t *testing.T) {
	accountMigration, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read account type migration: %v", err)
	}
	recordMigration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read business record migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY,
			record_cid TEXT NOT NULL
		);
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
	`+string(accountMigration)+string(recordMigration))

	ctx := context.Background()
	owner := syntax.DID("did:plc:owner")
	visitor := syntax.DID("did:plc:visitor")
	rkey := syntax.RecordKey("3mseligible001")
	eventURI := syntax.ATURI("at://did:plc:owner/social.craftsky.business.event/3mseligible001")
	asOf := time.Date(2026, time.August, 29, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES ($1,'active',1,1,'test',now(),now(),now())
	`, owner); err != nil {
		t.Fatalf("seed current lifecycle: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed current membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, owner); err != nil {
		t.Fatalf("seed business account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_events
			(uri, owner_did, rkey, cid, raw_record, source_revision, starts_at, ends_at, created_at, status)
		VALUES ($1, $2, $3, 'event-cid', $4::jsonb, '3msrevision001', $5, $6, $7, NULL)
	`, eventURI, owner, rkey, `{
		"$type":"social.craftsky.business.event",
		"name":"Declaration-independent fair",
		"startsAt":"2026-08-30T10:00:00Z",
		"endsAt":"2026-08-30T12:00:00Z",
		"roles":["vendor"],
		"createdAt":"2026-08-01T09:00:00Z"
	}`, asOf.Add(22*time.Hour), asOf.Add(24*time.Hour), asOf.Add(-28*24*time.Hour)); err != nil {
		t.Fatalf("seed eligible event projection: %v", err)
	}

	store := business.NewStore(pool)
	profileHandler := api.GetProfileHandler(
		&fakeStore{row: &api.ProfileRow{
			DID: owner.String(), Crafts: []string{}, CreatedAt: asOf, IsCraftskyProfile: true,
		}},
		store,
		fakeResolver{handleFor: "owner.test"},
		nilLogger(),
	)
	accountTypeHandler := businessAccountTypeHandler(store, owner, businessLifecycleReader{
		owner: {Owner: owner, State: ownerlifecycle.StateActive, Generation: 1},
	})

	assertBusinessEligibilityState(t, profileHandler, store, owner, visitor, rkey, asOf, "business", false, true)

	effects := &businessProfileEffects{}
	putDeclaration := api.PutBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return effects, nil
	})
	response := serveBusinessProfileRequest(t, putDeclaration, http.MethodPut, `{"tagline":"Projected declaration"}`, "*")
	if response.Code != http.StatusOK {
		t.Fatalf("add declaration status=%d body=%s", response.Code, response.Body.String())
	}
	declarationRaw, err := json.Marshal(effects.record)
	if err != nil {
		t.Fatalf("marshal projected declaration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES ($1, 'at://did:plc:owner/social.craftsky.business.profile/self', $2, $3, '3msprofile001')
	`, owner, effects.cid, declarationRaw); err != nil {
		t.Fatalf("project declaration: %v", err)
	}
	assertBusinessEligibilityState(t, profileHandler, store, owner, visitor, rkey, asOf, "business", true, true)

	response = serveBusinessAccountType(accountTypeHandler, `{"accountType":"regular"}`, true)
	if response.Code != http.StatusOK {
		t.Fatalf("set regular status=%d body=%s", response.Code, response.Body.String())
	}
	assertBusinessEligibilityState(t, profileHandler, store, owner, visitor, rkey, asOf, "regular", false, false)
	assertBusinessProjectionRowCounts(t, pool, owner, 1, 1)

	deleteDeclaration := api.DeleteBusinessProfileHandler(func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		return effects, nil
	})
	response = serveBusinessProfileRequest(t, deleteDeclaration, http.MethodDelete, "", businessProfileCID1)
	if response.Code != http.StatusOK {
		t.Fatalf("delete declaration status=%d body=%s", response.Code, response.Body.String())
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_business_profiles WHERE owner_did=$1`, owner); err != nil {
		t.Fatalf("project declaration deletion: %v", err)
	}
	assertBusinessEligibilityState(t, profileHandler, store, owner, visitor, rkey, asOf, "regular", false, false)

	response = serveBusinessAccountType(accountTypeHandler, `{"accountType":"business"}`, true)
	if response.Code != http.StatusOK {
		t.Fatalf("restore business status=%d body=%s", response.Code, response.Body.String())
	}
	assertBusinessEligibilityState(t, profileHandler, store, owner, visitor, rkey, asOf, "business", false, true)
	assertBusinessProjectionRowCounts(t, pool, owner, 0, 1)
}

func assertBusinessEligibilityState(
	t *testing.T,
	profileHandler http.Handler,
	store *business.Store,
	owner, visitor syntax.DID,
	rkey syntax.RecordKey,
	asOf time.Time,
	wantAccountType string,
	wantDeclaration, wantEvent bool,
) {
	t.Helper()
	response := serveBusinessProfileRead(profileHandler, owner)
	if response.Code != http.StatusOK {
		t.Fatalf("profile status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	if body["accountType"] != wantAccountType {
		t.Fatalf("accountType=%v, want %q", body["accountType"], wantAccountType)
	}
	declaration, hasDeclaration := body["business"].(map[string]any)
	if hasDeclaration != wantDeclaration {
		t.Fatalf("business=%#v, presence=%v, want presence=%v", body["business"], hasDeclaration, wantDeclaration)
	}
	if wantDeclaration && declaration["tagline"] != "Projected declaration" {
		t.Fatalf("business declaration=%#v", declaration)
	}

	event, directErr := store.ReadEvent(context.Background(), business.EventReadInput{
		CallerDID: visitor, OwnerDID: owner, Rkey: rkey, AsOf: asOf,
	})
	upcoming, listErr := store.ListUpcomingEvents(context.Background(), business.UpcomingEventListInput{
		CallerDID: visitor, OwnerDID: owner, AsOf: asOf, Limit: 10,
	})
	if wantEvent {
		if directErr != nil || event.Name != "Declaration-independent fair" {
			t.Fatalf("public direct event=(%+v, %v), want eligible", event, directErr)
		}
		if listErr != nil || len(upcoming) != 1 || upcoming[0].Rkey != rkey {
			t.Fatalf("public upcoming events=(%+v, %v), want event %s", upcoming, listErr, rkey)
		}
		return
	}
	if !errors.Is(directErr, business.ErrEventNotFound) {
		t.Fatalf("public direct event error=%v, want ErrEventNotFound", directErr)
	}
	if listErr != nil || len(upcoming) != 0 {
		t.Fatalf("public upcoming events=(%+v, %v), want suppressed", upcoming, listErr)
	}
}

func assertBusinessProjectionRowCounts(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, wantProfiles, wantEvents int) {
	t.Helper()
	var profiles, events int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_business_profiles WHERE owner_did=$1`, owner).Scan(&profiles); err != nil {
		t.Fatalf("count raw business profile rows: %v", err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_business_events WHERE owner_did=$1`, owner).Scan(&events); err != nil {
		t.Fatalf("count raw business event rows: %v", err)
	}
	if profiles != wantProfiles || events != wantEvents {
		t.Fatalf("raw profile/event rows=%d/%d, want %d/%d", profiles, events, wantProfiles, wantEvents)
	}
}
