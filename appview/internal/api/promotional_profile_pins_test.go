package api_test

import (
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/subscriptions"
	"social.craftsky/appview/internal/testdb"
)

func TestPromotionalProfilePinsHideAtExpiryAndDoNotReturnAfterRenewal(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := ownerlifecycle.WithExpectedGeneration(t.Context(), 1)
	owner := syntax.DID("did:plc:gift-pins")
	rkey := syntax.RecordKey("one")
	for _, statement := range []string{
		`INSERT INTO craftsky_profiles(did,record_cid) VALUES ('did:plc:gift-pins','cid')`,
		`INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES ('did:plc:gift-pins','active',1,1,'test',now(),now(),now())`,
		`INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at) VALUES ('at://did:plc:gift-pins/social.craftsky.feed.post/one','did:plc:gift-pins','one','cid','one','{}',now())`,
		`INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES ('at://did:plc:gift-pins/social.craftsky.feed.post/one','post','cid','clear')`,
	} {
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	access := subscriptions.NewStore(pool)
	account, _, err := access.EnsureAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := subscriptions.NewCatalog(subscriptions.CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]subscriptions.ProductMapping{"plus": {AppID: "app", Tier: subscriptions.TierPlus}}})
	if err != nil {
		t.Fatal(err)
	}
	generation := int64(0)
	apply := func(snapshots ...subscriptions.ProviderSubscriptionSnapshot) {
		t.Helper()
		generation++
		token := uuid.New()
		if _, err := pool.Exec(ctx, `UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=now()+interval '1 minute' WHERE id=$1`, account.ID, generation, token); err != nil {
			t.Fatal(err)
		}
		if err := access.ApplySnapshot(ctx, subscriptions.SnapshotClaim{BillingAccountID: account.ID, Generation: generation, LeaseToken: token}, subscriptions.CompleteSnapshot{Complete: true, Subscriptions: snapshots}, catalog); err != nil {
			t.Fatal(err)
		}
	}
	end := time.Now().Add(time.Hour)
	gift := subscriptions.ProviderSubscriptionSnapshot{ID: "gift", Store: "promotional", Environment: "production", GivesAccess: true, EndsAt: &end, Entitlements: []subscriptions.ProviderEntitlementSnapshot{{ID: "plus-entitlement", ProjectID: "project", LookupKey: "plus"}}}
	apply(gift)
	pins := api.NewProfilePinStore(pool, api.ProfilePinStoreOptions{RequirePlus: true})
	if _, err := pins.Pin(ctx, owner, owner, rkey); err != nil {
		t.Fatal(err)
	}
	assertVisible := func(want bool) {
		t.Helper()
		state, err := pins.Read(ctx, owner)
		if err != nil || (state.StandardPostURI != nil) != want {
			t.Fatalf("self pins=%+v err=%v wantVisible=%t", state, err, want)
		}
		pin, err := pins.ReadProfileListPin(ctx, owner, "", api.ProfilePinSlotStandard, nil)
		if err != nil || (pin != nil) != want {
			t.Fatalf("public pin=%+v err=%v wantVisible=%t", pin, err, want)
		}
	}
	assertVisible(true)
	// No webhook or reconciliation is needed for a known, non-renewing expiry
	// to hide pins and reject new protected mutations.
	if _, err := pool.Exec(ctx, `UPDATE billing_promotional_grants SET expires_at=now()-interval '1 second' WHERE billing_account_id=$1`, account.ID); err != nil {
		t.Fatal(err)
	}
	assertVisible(false)
	if _, err := pins.Pin(ctx, owner, owner, rkey); !errors.Is(err, subscriptions.ErrFeatureAccessRequired) {
		t.Fatalf("expired gift pin error=%v", err)
	}
	apply(gift)
	assertVisible(false)
	var retained int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did=$1`, owner).Scan(&retained); err != nil || retained != 0 {
		t.Fatalf("renewal restored stale pins: rows=%d err=%v", retained, err)
	}
	if _, err := pins.Pin(ctx, owner, owner, rkey); err != nil {
		t.Fatal(err)
	}
	paid := subscriptions.ProviderSubscriptionSnapshot{ID: "paid-plus", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true, AutoRenewalStatus: "will_renew"}
	apply(gift, paid)
	apply(paid) // Removing the gift preserves pins while paid access remains.
	assertVisible(true)
	apply()
	assertVisible(false)
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM profile_pins WHERE owner_did=$1`, owner).Scan(&retained); err != nil || retained != 0 {
		t.Fatalf("access loss retained pins: rows=%d err=%v", retained, err)
	}
}
