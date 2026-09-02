package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
)

type businessOwnershipMembership bool

func (current businessOwnershipMembership) IsCurrentMember(context.Context, syntax.DID) (bool, error) {
	return bool(current), nil
}

func TestRegularMemberCanPrepareOnlyOwnBusinessRecords(t *testing.T) {
	accountMigration, err := os.ReadFile("../../migrations/000061_business_account_types.up.sql")
	if err != nil {
		t.Fatalf("read account type migration: %v", err)
	}
	recordMigration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read business record migration: %v", err)
	}
	pool := testdb.WithSchema(t, `
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
	other := syntax.DID("did:plc:other")
	visitor := syntax.DID("did:plc:visitor")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed current membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'regular')`, owner); err != nil {
		t.Fatalf("seed regular account type: %v", err)
	}
	lifecycles := businessLifecycleReader{
		owner: {Owner: owner, State: ownerlifecycle.StateActive, Generation: 7},
		other: {Owner: other, State: ownerlifecycle.StateActive, Generation: 3},
	}

	profileEffects := &businessProfileEffects{}
	profileFactoryCalls := 0
	profileFactory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		profileFactoryCalls++
		return profileEffects, nil
	}
	putProfile := businessOwnershipMutationHandler(
		api.PutBusinessProfileHandler(profileFactory), owner, true, lifecycles,
	)
	createdProfile := serveBusinessOwnershipRequest(
		putProfile, http.MethodPut, "/v1/profiles/me/business", `{"tagline":"Preparing"}`, "*", true, "", "",
	)
	if createdProfile.Code != http.StatusOK {
		t.Fatalf("create declaration status=%d body=%s", createdProfile.Code, createdProfile.Body.String())
	}
	editedProfile := serveBusinessOwnershipRequest(
		putProfile, http.MethodPut, "/v1/profiles/me/business", `{"tagline":"Prepared"}`, businessProfileCID1, true, "", "",
	)
	if editedProfile.Code != http.StatusOK {
		t.Fatalf("edit declaration status=%d body=%s", editedProfile.Code, editedProfile.Body.String())
	}
	if profileFactoryCalls != 2 || len(profileEffects.puts) != 2 || profileEffects.record["tagline"] != "Prepared" {
		t.Fatalf("declaration PDS effects calls=%d puts=%d record=%#v", profileFactoryCalls, len(profileEffects.puts), profileEffects.record)
	}

	eventEffects := newBusinessEventEffects()
	eventFactoryCalls := 0
	eventFactory := func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
		eventFactoryCalls++
		return eventEffects, nil
	}
	postEvent := businessOwnershipMutationHandler(
		api.PostBusinessEventHandler(eventFactory, func() time.Time { return businessEventNow }), owner, true, lifecycles,
	)
	createdEvent := serveBusinessOwnershipRequest(
		postEvent, http.MethodPost, "/v1/events", validBusinessEventBody(false), "", true, "", "",
	)
	if createdEvent.Code != http.StatusCreated {
		t.Fatalf("create event status=%d body=%s", createdEvent.Code, createdEvent.Body.String())
	}
	eventResponse := decodeBusinessEventMutationResponse(t, createdEvent)
	rkey := syntax.RecordKey(eventResponse.Rkey)
	putEvent := businessOwnershipMutationHandler(
		api.PutBusinessEventHandler(eventFactory, func() time.Time { return businessEventNow }), owner, true, lifecycles,
	)
	editedEvent := serveBusinessOwnershipRequest(
		putEvent,
		http.MethodPut,
		"/v1/events/"+owner.String()+"/"+rkey.String(),
		strings.Replace(validBusinessEventBody(false), "Fiber Fair", "Fiber Fair Prepared", 1),
		businessEventCID1,
		true,
		owner.String(),
		rkey.String(),
	)
	if editedEvent.Code != http.StatusOK {
		t.Fatalf("edit event status=%d body=%s", editedEvent.Code, editedEvent.Body.String())
	}
	if eventFactoryCalls != 2 || len(eventEffects.puts) != 2 || eventEffects.records[rkey]["name"] != "Fiber Fair Prepared" {
		t.Fatalf("event PDS effects calls=%d puts=%d record=%#v", eventFactoryCalls, len(eventEffects.puts), eventEffects.records[rkey])
	}

	profileRaw, err := json.Marshal(profileEffects.record)
	if err != nil {
		t.Fatalf("marshal declaration projection: %v", err)
	}
	eventRaw, err := json.Marshal(eventEffects.records[rkey])
	if err != nil {
		t.Fatalf("marshal event projection: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES ($1, 'at://did:plc:owner/social.craftsky.business.profile/self', $2, $3, '3msownerprofile')
	`, owner, profileEffects.cid, profileRaw); err != nil {
		t.Fatalf("project prepared declaration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_events
			(uri, owner_did, rkey, cid, raw_record, source_revision, starts_at, ends_at, created_at, status)
		VALUES ($2, $1, $3, $4, $5, '3msownerevent', '2026-09-02T10:00:00Z', '2026-09-02T18:00:00Z',
			'2026-09-01T12:34:56Z', 'scheduled')
	`, owner, eventResponse.URI, rkey, eventEffects.cids[rkey], eventRaw); err != nil {
		t.Fatalf("project prepared event: %v", err)
	}

	store := business.NewStore(pool)
	profileHandler := api.GetProfileHandler(
		&fakeStore{row: &api.ProfileRow{DID: owner.String(), Crafts: []string{}, CreatedAt: businessEventNow, IsCraftskyProfile: true}},
		store,
		fakeResolver{handleFor: "owner.test"},
		nilLogger(),
	)
	profile := serveBusinessProfileRead(profileHandler, owner)
	var profileBody map[string]any
	if err := json.Unmarshal(profile.Body.Bytes(), &profileBody); err != nil {
		t.Fatalf("decode regular profile: %v", err)
	}
	if profileBody["accountType"] != "regular" {
		t.Fatalf("accountType=%v, want regular", profileBody["accountType"])
	}
	if _, exposed := profileBody["business"]; exposed {
		t.Fatalf("regular profile exposed declaration: %s", profile.Body.String())
	}
	if event, err := store.ReadEvent(ctx, business.EventReadInput{
		CallerDID: visitor, OwnerDID: owner, Rkey: rkey, AsOf: businessEventNow,
	}); !errors.Is(err, business.ErrEventNotFound) {
		t.Fatalf("regular public event=(%+v, %v), want suppressed", event, err)
	}
	upcoming, err := store.ListUpcomingEvents(ctx, business.UpcomingEventListInput{
		CallerDID: visitor, OwnerDID: owner, AsOf: businessEventNow, Limit: 10,
	})
	if err != nil || len(upcoming) != 0 {
		t.Fatalf("regular public upcoming events=(%+v, %v), want suppressed", upcoming, err)
	}

	t.Run("ineligible callers are rejected before PDS access", func(t *testing.T) {
		tests := []struct {
			name          string
			handler       http.Handler
			method        string
			target        string
			body          string
			ifMatch       string
			authenticated bool
			did           string
			rkey          string
			wantStatus    int
			factoryCalls  *int
		}{
			{
				name: "unauthenticated declaration create", handler: businessOwnershipMutationHandler(
					api.PutBusinessProfileHandler(profileFactory), owner, true, lifecycles,
				), method: http.MethodPut, target: "/v1/profiles/me/business", body: `{}`, ifMatch: "*",
				wantStatus: http.StatusUnauthorized, factoryCalls: &profileFactoryCalls,
			},
			{
				name: "non-current event create", handler: businessOwnershipMutationHandler(
					api.PostBusinessEventHandler(eventFactory, func() time.Time { return businessEventNow }), owner, false, lifecycles,
				), method: http.MethodPost, target: "/v1/events", body: validBusinessEventBody(false), authenticated: true,
				wantStatus: http.StatusNotFound, factoryCalls: &eventFactoryCalls,
			},
			{
				name: "different DID event edit", handler: businessOwnershipMutationHandler(
					api.PutBusinessEventHandler(eventFactory, func() time.Time { return businessEventNow }), other, true, lifecycles,
				), method: http.MethodPut, target: "/v1/events/" + owner.String() + "/" + rkey.String(),
				body: validBusinessEventBody(false), ifMatch: businessEventCID2, authenticated: true,
				did: owner.String(), rkey: rkey.String(), wantStatus: http.StatusForbidden, factoryCalls: &eventFactoryCalls,
			},
		}
		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				before := *test.factoryCalls
				response := serveBusinessOwnershipRequest(
					test.handler, test.method, test.target, test.body, test.ifMatch,
					test.authenticated, test.did, test.rkey,
				)
				if response.Code != test.wantStatus {
					t.Fatalf("status=%d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
				}
				if *test.factoryCalls != before {
					t.Fatalf("request reached PDS effect factory: calls %d -> %d", before, *test.factoryCalls)
				}
			})
		}
	})
}

func businessOwnershipMutationHandler(
	handler http.Handler,
	did syntax.DID,
	current businessOwnershipMembership,
	lifecycles businessLifecycleReader,
) http.Handler {
	handler = middleware.CurrentMember(current, nilLogger(), lifecycles)(handler)
	return middleware.Authenticated(
		&auth.MockAuthService{DefaultDID: did}, nilLogger(), middleware.DevAuthPolicy{Mode: middleware.DevAuthDisabled},
	)(handler)
}

func serveBusinessOwnershipRequest(
	handler http.Handler,
	method, target, body, ifMatch string,
	authenticated bool,
	did, rkey string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if authenticated {
		request.Header.Set("Authorization", "Bearer test-session")
	}
	if ifMatch != "" {
		request.Header.Set("If-Match", ifMatch)
	}
	if did != "" {
		request.SetPathValue("did", did)
	}
	if rkey != "" {
		request.SetPathValue("rkey", rkey)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
