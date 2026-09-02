package api_test

import (
	"context"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/testdb"
)

func TestBusinessProfileStoreEligibility(t *testing.T) {
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY,
			record_cid TEXT NOT NULL
		);
		CREATE TABLE craftsky_account_types (
			owner_did TEXT PRIMARY KEY,
			account_type TEXT NOT NULL CHECK (account_type IN ('regular', 'business'))
		);
		CREATE TABLE craftsky_business_profiles (
			owner_did TEXT PRIMARY KEY,
			uri TEXT NOT NULL UNIQUE,
			cid TEXT NOT NULL,
			raw_record JSONB NOT NULL,
			source_revision TEXT NOT NULL,
			indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
	`)
	ctx := context.Background()
	did := syntax.DID("did:plc:alice")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did, record_cid) VALUES ($1, 'profile-cid')`, did); err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did, account_type) VALUES ($1, 'business')`, did); err != nil {
		t.Fatalf("seed account type: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_business_profiles(owner_did, uri, cid, raw_record, source_revision)
		VALUES ($1, 'at://did:plc:alice/social.craftsky.business.profile/self', 'business-cid',
			'{"$type":"social.craftsky.business.profile","tagline":"Visible"}'::jsonb, '3mprofile00001')
	`, did); err != nil {
		t.Fatalf("seed business profile: %v", err)
	}

	store := business.NewStore(pool)
	view, err := store.ReadEligibleProfile(ctx, did)
	if err != nil || view == nil || view.Tagline != "Visible" {
		t.Fatalf("business member profile = (%+v, %v)", view, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_account_types SET account_type='regular' WHERE owner_did=$1`, did); err != nil {
		t.Fatalf("set regular: %v", err)
	}
	if view, err := store.ReadEligibleProfile(ctx, did); err != nil || view != nil {
		t.Fatalf("regular member profile = (%+v, %v), want nil", view, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE craftsky_account_types SET account_type='business' WHERE owner_did=$1`, did); err != nil {
		t.Fatalf("restore business type: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM craftsky_profiles WHERE did=$1`, did); err != nil {
		t.Fatalf("remove membership: %v", err)
	}
	if view, err := store.ReadEligibleProfile(ctx, did); err != nil || view != nil {
		t.Fatalf("non-member profile = (%+v, %v), want nil", view, err)
	}
	var retained int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_business_profiles WHERE owner_did=$1`, did).Scan(&retained); err != nil || retained != 1 {
		t.Fatalf("retained profile count = %d, error %v", retained, err)
	}
}
