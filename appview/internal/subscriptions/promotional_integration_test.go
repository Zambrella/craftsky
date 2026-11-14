package subscriptions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/testdb"
)

type promotionalFixture struct {
	store      *Store
	pool       *pgxpool.Pool
	owner      syntax.DID
	account    BillingAccount
	catalog    *Catalog
	generation int64
}

func newPromotionalFixture(t *testing.T) *promotionalFixture {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	store := NewStore(pool)
	owner := syntax.DID("did:plc:promotional-owner")
	account, _, err := store.EnsureAccount(t.Context(), owner)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := NewCatalog(CatalogConfig{ProjectID: "project", AppIDs: []string{"app"}, Products: map[string]ProductMapping{
		"plus": {AppID: "app", Tier: TierPlus}, "business": {AppID: "app", Tier: TierBusiness},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return &promotionalFixture{store: store, pool: pool, owner: owner, account: account, catalog: catalog}
}

func (f *promotionalFixture) apply(t *testing.T, snapshots ...ProviderSubscriptionSnapshot) {
	t.Helper()
	f.generation++
	token := uuid.New()
	if _, err := f.pool.Exec(t.Context(), `UPDATE billing_accounts SET requested_generation=$2,claimed_generation=$2,lease_token=$3,lease_expires_at=now()+interval '1 minute' WHERE id=$1`, f.account.ID, f.generation, token); err != nil {
		t.Fatal(err)
	}
	if err := f.store.ApplySnapshot(t.Context(), SnapshotClaim{BillingAccountID: f.account.ID, Generation: f.generation, LeaseToken: token}, CompleteSnapshot{Complete: true, Subscriptions: snapshots}, f.catalog); err != nil {
		t.Fatal(err)
	}
}

func promotionalSnapshot(tier Tier, expiry *time.Time) ProviderSubscriptionSnapshot {
	return ProviderSubscriptionSnapshot{ID: "gift", Store: "promotional", Environment: "production", Status: "active", GivesAccess: true, EndsAt: expiry,
		Entitlements: []ProviderEntitlementSnapshot{{ID: "entitlement", ProjectID: "project", LookupKey: string(tier)}}}
}

func paidSnapshot(id string, tier Tier) ProviderSubscriptionSnapshot {
	return ProviderSubscriptionSnapshot{ID: id, ProductID: string(tier), Store: "app_store", Environment: "production", Status: "active", GivesAccess: true, AutoRenewalStatus: "will_renew"}
}

func (f *promotionalFixture) assertAccess(t *testing.T, at time.Time, want Tier) {
	t.Helper()
	access, err := f.store.SelfAccess(t.Context(), f.owner, at)
	if err != nil || access.EffectiveTier != want || access.GivesAccess != want.Paid() {
		t.Fatalf("access=%+v err=%v want=%s", access, err, want)
	}
	if want.Paid() && (access.AssignedTier == nil || *access.AssignedTier != want) {
		t.Fatalf("effective and assigned tiers disagree: %+v", access)
	}
}

func TestPromotionalGrantsAuthorizeEntitlementsExpireAndRevokeWithoutPaidRows(t *testing.T) {
	f := newPromotionalFixture(t)
	now := time.Now().UTC()
	ends := now.Add(6 * time.Hour)
	periodEnd := now.Add(3 * time.Hour)
	grant := promotionalSnapshot(TierPlus, &ends)
	grant.CurrentPeriodEndsAt = &periodEnd
	f.apply(t, grant)
	f.assertAccess(t, now, TierPlus)
	f.assertAccess(t, periodEnd, TierFree)
	state, err := f.store.OwnerState(t.Context(), f.owner, now)
	if err != nil || len(state.Subscriptions) != 0 || len(state.Licenses) != 0 || len(state.PromotionalGrants) != 1 {
		t.Fatalf("gift billing state=%+v err=%v", state, err)
	}
	if gift := state.PromotionalGrants[0]; gift.Tier != TierPlus || !gift.GivesAccess || gift.ExpiresAt == nil || !gift.ExpiresAt.Equal(periodEnd) {
		t.Fatalf("gift=%+v", gift)
	}
	stateAtExpiry, err := f.store.OwnerState(t.Context(), f.owner, periodEnd)
	if err != nil || stateAtExpiry.PromotionalGrants[0].GivesAccess {
		t.Fatalf("expired state=%+v err=%v", stateAtExpiry, err)
	}
	batch, err := f.store.EffectiveTiers(t.Context(), []syntax.DID{f.owner, "did:plc:other"})
	if err != nil || batch[f.owner] != TierPlus || batch["did:plc:other"] != TierFree {
		t.Fatalf("batch=%v err=%v", batch, err)
	}
	if err := f.store.WithPlusAccess(t.Context(), f.owner, func(_ context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	expired := now.Add(-time.Second)
	f.apply(t, promotionalSnapshot(TierPlus, &expired))
	if err := f.store.WithPlusAccess(t.Context(), f.owner, func(_ context.Context) error {
		t.Fatal("timed-out gift ran protected effect")
		return nil
	}); !errors.Is(err, ErrFeatureAccessRequired) {
		t.Fatalf("expired gift effect=%v", err)
	}

	// A lifetime Business entitlement remains non-renewing and has no product ID.
	grant = promotionalSnapshot(TierBusiness, nil)
	f.apply(t, grant)
	f.assertAccess(t, now.AddDate(10, 0, 0), TierBusiness)
	grant.GivesAccess = false
	f.apply(t, grant)
	f.assertAccess(t, now, TierFree)
	f.apply(t, promotionalSnapshot(TierBusiness, nil))
	f.apply(t) // A complete empty snapshot revokes the gift, retaining its history.
	f.assertAccess(t, now, TierFree)
	var rows, paidRows int
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM billing_promotional_grants WHERE billing_account_id=$1`, f.account.ID).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(t.Context(), `SELECT count(*) FROM provider_subscriptions WHERE billing_account_id=$1`, f.account.ID).Scan(&paidRows); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || paidRows != 0 {
		t.Fatalf("history=%d paid=%d", rows, paidRows)
	}
	if err := f.store.WithPlusAccess(t.Context(), f.owner, func(_ context.Context) error { t.Fatal("expired gift ran protected effect"); return nil }); !errors.Is(err, ErrFeatureAccessRequired) {
		t.Fatalf("revoked effect=%v", err)
	}
}

func TestPromotionalGrantsCannotUseUnknownSharedOrForeignEntitlementsOrSandbox(t *testing.T) {
	f := newPromotionalFixture(t)
	for _, test := range []struct{ name, key, project, environment string }{
		{"shared", "paid", "project", "production"},
		{"unknown", "future", "project", "production"},
		{"foreign", "business", "another-project", "production"},
		{"sandbox", "business", "project", "sandbox"},
	} {
		t.Run(test.name, func(t *testing.T) {
			grant := promotionalSnapshot(TierBusiness, nil)
			grant.Environment = test.environment
			grant.Entitlements[0].LookupKey = test.key
			grant.Entitlements[0].ProjectID = test.project
			f.apply(t, grant)
			f.assertAccess(t, time.Now(), TierFree)
		})
	}
	f.apply(t, promotionalSnapshot(TierBusiness, nil))
	sandbox := NewStoreForEnvironment(f.pool, "sandbox")
	access, err := sandbox.SelfAccess(t.Context(), f.owner, time.Now())
	if err != nil || access.GivesAccess || access.AssignedTier != nil {
		t.Fatalf("sandbox copied production gift: %+v err=%v", access, err)
	}
	state, err := sandbox.OwnerState(t.Context(), f.owner, time.Now())
	if err != nil || len(state.PromotionalGrants) != 0 {
		t.Fatalf("sandbox gift state=%+v err=%v", state, err)
	}
	productionStore, productionCatalog := f.store, f.catalog
	f.store = sandbox
	f.catalog, err = NewCatalog(CatalogConfig{ProjectID: "project", Environment: "sandbox"})
	if err != nil {
		t.Fatal(err)
	}
	// A sandbox reconciliation can observe the same customer's production gift
	// without mirroring it or closing the independently accepted production row.
	f.apply(t, promotionalSnapshot(TierBusiness, nil))
	f.store, f.catalog = productionStore, productionCatalog
	f.assertAccess(t, time.Now(), TierBusiness)
}

func TestPromotionalAccessCombinesWithPaidAndDoesNotHidePaidDuplicateAnomalies(t *testing.T) {
	f := newPromotionalFixture(t)
	paidPlus := paidSnapshot("paid-plus", TierPlus)
	paidBusiness := paidSnapshot("paid-business", TierBusiness)
	giftPlus := promotionalSnapshot(TierPlus, nil)
	giftBusiness := promotionalSnapshot(TierBusiness, nil)
	f.apply(t, paidPlus, giftBusiness)
	f.assertAccess(t, time.Now(), TierBusiness)
	f.apply(t, paidBusiness, giftPlus)
	f.assertAccess(t, time.Now(), TierBusiness)
	f.apply(t, paidPlus, giftPlus)
	f.assertAccess(t, time.Now(), TierPlus)
	f.apply(t, paidPlus)
	f.assertAccess(t, time.Now(), TierPlus)
	// Cross-tier paid duplicates remain denied; an independent gift only grants
	// its own tier rather than rescuing the denied Business subscription.
	f.apply(t, paidPlus, paidBusiness, giftPlus)
	f.assertAccess(t, time.Now(), TierPlus)
	state, err := f.store.OwnerState(t.Context(), f.owner, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, sub := range state.Subscriptions {
		if sub.GivesAccess && sub.Anomaly == "none" {
			t.Fatalf("paid duplicate accepted: %+v", sub)
		}
	}
	f.apply(t, paidPlus, paidBusiness)
	f.assertAccess(t, time.Now(), TierFree)
	duplicatePlus := paidPlus
	duplicatePlus.ID = "duplicate-plus"
	f.apply(t, paidPlus, duplicatePlus)
	f.assertAccess(t, time.Now(), TierFree)
}

func TestPromotionalOnlyAccessAndLegacyPromotionalRowsAllowCheckoutAndDeletion(t *testing.T) {
	store, pool, fake, owner, account, now := checkoutFixture(t)
	if _, err := pool.Exec(t.Context(), `INSERT INTO billing_promotional_grants(billing_account_id,project_id,revenuecat_subscription_id,environment,tier,gives_access,accepted_generation) VALUES($1,'project','gift','production','business',true,1)`, account.ID); err != nil {
		t.Fatal(err)
	}
	// Prior versions stored an unsupported promotional subscription. Keep it for
	// history while ignoring its unknown renewal fields as a paid obligation.
	if _, err := pool.Exec(t.Context(), `INSERT INTO provider_subscriptions(billing_account_id,project_id,revenuecat_subscription_id,product_id,store,environment,status,gives_access,accepted_generation,anomaly) VALUES($1,'project','old-gift','','promotional','production','active',true,1,'unsupported')`, account.ID); err != nil {
		t.Fatal(err)
	}
	state, err := store.OwnerState(t.Context(), owner, now)
	if err != nil || len(state.Subscriptions) != 0 || len(state.Licenses) != 0 || len(state.PromotionalGrants) != 1 {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	if _, err := store.WebCheckoutURL(t.Context(), owner, TierPlus, now); err != nil || len(fake.requests) != 1 {
		t.Fatalf("gift blocked checkout: %v calls=%d", err, len(fake.requests))
	}
	// Close the unpaid checkout through the existing verified provider path.
	fake.session.Status = "expired"
	if err := store.ResolveWebCheckoutForDeletion(t.Context(), owner, now); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(t.Context())
	participant := NewDeletionParticipant()
	if _, err := participant.BeginDeletion(t.Context(), tx, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(t.Context(), `UPDATE billing_accounts SET reconciled_generation=requested_generation WHERE id=$1`, account.ID); err != nil {
		t.Fatal(err)
	}
	if closed, err := participant.ConfirmDeletion(t.Context(), tx, owner, now); err != nil || !closed {
		t.Fatalf("gift blocked deletion: closed=%t err=%v", closed, err)
	}
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	var active bool
	if err := pool.QueryRow(t.Context(), `SELECT gives_access FROM billing_promotional_grants WHERE billing_account_id=$1`, account.ID).Scan(&active); err != nil || active {
		t.Fatalf("closed gift active=%t err=%v", active, err)
	}
	access, err := store.SelfAccess(t.Context(), owner, now)
	if err != nil || access.GivesAccess {
		t.Fatalf("deleted owner access=%+v err=%v", access, err)
	}
}

func TestPromotionalGrantIdentityCannotMoveBetweenBillingCustomers(t *testing.T) {
	f := newPromotionalFixture(t)
	f.apply(t, promotionalSnapshot(TierBusiness, nil))
	other := syntax.DID("did:plc:other-gift-customer")
	account, _, err := f.store.EnsureAccount(t.Context(), other)
	if err != nil {
		t.Fatal(err)
	}
	token := uuid.New()
	if _, err := f.pool.Exec(t.Context(), `UPDATE billing_accounts SET claimed_generation=requested_generation,lease_token=$2,lease_expires_at=now()+interval '1 minute' WHERE id=$1`, account.ID, token); err != nil {
		t.Fatal(err)
	}
	claim := SnapshotClaim{BillingAccountID: account.ID, Generation: account.RequestedGeneration, LeaseToken: token}
	err = f.store.ApplySnapshot(t.Context(), claim, CompleteSnapshot{Complete: true, Subscriptions: []ProviderSubscriptionSnapshot{promotionalSnapshot(TierBusiness, nil)}}, f.catalog)
	if !errors.Is(err, ErrStaleSnapshot) {
		t.Fatalf("cross-customer grant transfer error=%v", err)
	}
	f.assertAccess(t, time.Now(), TierBusiness)
	access, err := f.store.SelfAccess(t.Context(), other, time.Now())
	if err != nil || access.GivesAccess {
		t.Fatalf("destination received foreign gift=%+v err=%v", access, err)
	}
	var actualOwner uuid.UUID
	if err := f.pool.QueryRow(t.Context(), `SELECT billing_account_id FROM billing_promotional_grants WHERE revenuecat_subscription_id='gift'`).Scan(&actualOwner); err != nil || actualOwner != f.account.ID {
		t.Fatalf("grant owner=%s err=%v", actualOwner, err)
	}
	current, err := f.store.OwnerAccount(t.Context(), other)
	if err != nil || current.ReconciledGeneration != 0 {
		t.Fatalf("failed apply advanced destination generation=%+v err=%v", current, err)
	}
}
