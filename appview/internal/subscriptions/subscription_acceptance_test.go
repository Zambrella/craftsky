package subscriptions

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/testdb"
)

func TestPlusAndBusinessLicensesAreIndependentlyAssigned(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE owner_lifecycles (
			owner_did TEXT PRIMARY KEY, state TEXT NOT NULL, generation BIGINT NOT NULL,
			auth_epoch BIGINT NOT NULL, transition_reason TEXT NOT NULL,
			transitioned_at TIMESTAMPTZ NOT NULL, terminal_at TIMESTAMPTZ,
			purge_completed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		CREATE TABLE craftsky_sessions (
			token_hash BYTEA PRIMARY KEY, account_did TEXT NOT NULL,
			last_device_id TEXT, last_seen_at TIMESTAMPTZ NOT NULL,
			idle_expires_at TIMESTAMPTZ NOT NULL, lifecycle_state TEXT NOT NULL,
			revoked_at TIMESTAMPTZ
		);
		CREATE FUNCTION appview_owner_is_active(candidate_did TEXT)
		RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
			SELECT COALESCE((SELECT state='active' FROM owner_lifecycles WHERE owner_did=candidate_did),false)
		$$;
	`+string(migration))
	ctx := context.Background()
	store := NewStore(pool)
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:acceptance-owner")
	targetA := syntax.DID("did:plc:acceptance-a")
	targetB := syntax.DID("did:plc:acceptance-b")
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did) VALUES($1),($2),($3)`, owner, targetA, targetB); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)
		VALUES($1,'active',1,1,'test',$4,$4,$4),($2,'active',1,1,'test',$4,$4,$4),($3,'active',1,1,'test',$4,$4,$4)
	`, owner, targetA, targetB, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_sessions(token_hash,account_did,last_device_id,last_seen_at,idle_expires_at,lifecycle_state)
		VALUES(decode('21','hex'),$1,'shared-device',$3,$3::timestamptz+interval '30 days','active'),
		      (decode('22','hex'),$2,'shared-device',$3,$3::timestamptz+interval '30 days','active')
	`, targetA, targetB, now); err != nil {
		t.Fatal(err)
	}
	account, _, err := store.EnsureAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{
		"plus": {AppID: "app", Tier: TierPlus}, "business": {AppID: "app", Tier: TierBusiness},
	}})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{
		{ID: "plus-subscription", ProductID: "plus", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
		{ID: "business-subscription", ProductID: "business", Store: "app_store", Environment: "production", Status: "active", GivesAccess: true},
	}}
	for generation := int64(1); generation <= 2; generation++ {
		token := uuid.New()
		if _, err := pool.Exec(ctx, `
			UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=$4 WHERE id=$1
		`, account.ID, generation, token, now.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
		if err := store.ApplySnapshot(ctx, SnapshotClaim{BillingAccountID: account.ID, Generation: generation, LeaseToken: token}, snapshot, catalog); err != nil {
			t.Fatal(err)
		}
	}

	licenses := map[Tier]uuid.UUID{}
	rows, err := pool.Query(ctx, `SELECT tier,id,assigned_did FROM billing_licenses ORDER BY tier`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var tier Tier
		var id uuid.UUID
		var assigned *syntax.DID
		if err := rows.Scan(&tier, &id, &assigned); err != nil {
			t.Fatal(err)
		}
		if assigned != nil {
			t.Fatalf("new %s license assigned to %v", tier, assigned)
		}
		licenses[tier] = id
	}
	if len(licenses) != 2 {
		t.Fatalf("licenses = %#v, want Plus and Business", licenses)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenses[TierPlus], TargetDID: targetA, DeviceID: "shared-device", Now: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenses[TierBusiness], TargetDID: targetB, DeviceID: "shared-device", Now: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Assign(ctx, AssignParams{OwnerDID: owner, LicenseID: licenses[TierBusiness], TargetDID: targetA, DeviceID: "shared-device", Now: now}); !errors.Is(err, ErrDIDAlreadyAssigned) {
		t.Fatalf("overlapping assignment error = %v, want ErrDIDAlreadyAssigned", err)
	}
	for did, tier := range map[syntax.DID]Tier{targetA: TierPlus, targetB: TierBusiness} {
		access, err := store.SelfAccess(ctx, did, now)
		if err != nil || access.EffectiveTier != tier {
			t.Fatalf("%s access = %+v, error %v", did, access, err)
		}
	}
}
