package index_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/index"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessProfileProjectionAndSafeHydration(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000062_business_records.up.sql")
	if err != nil {
		t.Fatalf("read business records migration: %v", err)
	}
	pool := testdb.WithSchema(t, string(migration))
	ctx := context.Background()
	raw := json.RawMessage(`{
		"$type":"social.craftsky.business.profile",
		"businessTypes":["teacher","future-business-type","dyer"],
		"offerings":["classes","future-offering","yarn"],
		"tagline":"Natural dyes and classes",
		"hoursNote":"Open by appointment",
		"serviceArea":"Ships within the UK",
		"location":{"country":"gb","locality":"Edinburgh","street":"not hydrated"},
		"primaryAction":{"type":"shop","destination":"http://unsafe.example"},
		"products":[{"title":"Indigo kit","uri":"https://example.com/kit","price":{"amount":"12.00","currency":"ZZZ"}}],
		"futureExtension":{"enabled":true}
	}`)
	event := tap.Event{
		URI:        syntax.ATURI("at://did:plc:alice/social.craftsky.business.profile/self"),
		CID:        syntax.CID("bafyreiprofilev1"),
		DID:        syntax.DID("did:plc:alice"),
		Collection: syntax.NSID("social.craftsky.business.profile"),
		Rkey:       syntax.RecordKey("self"),
		Action:     "create",
		Record:     raw,
		Rev:        syntax.TID("3mprofile00001"),
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin projection: %v", err)
	}
	outcome, err := index.NewCraftskyBusinessProfile().Project(ctx, tx, event)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("project profile: %v", err)
	}
	if outcome.Kind != tap.OutcomeApplied {
		_ = tx.Rollback(ctx)
		t.Fatalf("projection outcome = %+v", outcome)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit projection: %v", err)
	}

	var gotURI, gotCID, gotRevision string
	var stored json.RawMessage
	if err := pool.QueryRow(ctx, `
		SELECT uri, cid, source_revision, raw_record
		FROM craftsky_business_profiles
		WHERE owner_did=$1
	`, event.DID).Scan(&gotURI, &gotCID, &gotRevision, &stored); err != nil {
		t.Fatalf("read projected profile: %v", err)
	}
	if gotURI != string(event.URI) || gotCID != string(event.CID) || gotRevision != string(event.Rev) {
		t.Fatalf("projected identity = %q/%q/%q", gotURI, gotCID, gotRevision)
	}
	assertJSONEquivalent(t, stored, raw)

	view, err := business.HydrateProfile(stored)
	if err != nil {
		t.Fatalf("hydrate projected profile: %v", err)
	}
	if view.Tagline != "Natural dyes and classes" || view.HoursNote != "Open by appointment" || view.ServiceArea != "Ships within the UK" {
		t.Fatalf("hydrated text = %+v", view)
	}
	wantTypes := []business.OpenValue{{Value: "dyer", Known: true}, {Value: "teacher", Known: true}, {Value: "future-business-type", Known: false}}
	wantOfferings := []business.OpenValue{{Value: "yarn", Known: true}, {Value: "classes", Known: true}, {Value: "future-offering", Known: false}}
	if !reflect.DeepEqual(view.BusinessTypes, wantTypes) || !reflect.DeepEqual(view.Offerings, wantOfferings) {
		t.Fatalf("hydrated catalogs = types:%+v offerings:%+v", view.BusinessTypes, view.Offerings)
	}
	if view.Location == nil || view.Location.Country != "GB" || view.Location.Locality == nil || *view.Location.Locality != "Edinburgh" {
		t.Fatalf("hydrated location = %+v", view.Location)
	}
	if view.PrimaryAction != nil {
		t.Fatalf("unsafe action hydrated: %+v", view.PrimaryAction)
	}
	if len(view.Products) != 1 || view.Products[0].Title != "Indigo kit" || view.Products[0].Price != nil {
		t.Fatalf("hydrated products = %+v", view.Products)
	}
}

func assertJSONEquivalent(t *testing.T, got, want json.RawMessage) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode stored JSON: %v", err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("decode wanted JSON: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("stored JSON = %s, want %s", got, want)
	}
}
