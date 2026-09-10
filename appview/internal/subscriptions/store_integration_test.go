package subscriptions

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/testdb"
)

func TestCoreBillingConstraints(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatalf("read subscription accounts migration: %v", err)
	}
	pool := testdb.WithSchema(t, string(up))
	ctx := context.Background()

	var accountID string
	err = pool.QueryRow(ctx, `
		INSERT INTO billing_accounts(owner_did, revenuecat_app_user_id)
		VALUES ('did:plc:billing-owner', '10000000-0000-0000-0000-000000000001')
		RETURNING id
	`).Scan(&accountID)
	if err != nil {
		t.Fatalf("insert billing account: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(owner_did, revenuecat_app_user_id)
		VALUES ('did:plc:billing-owner', '10000000-0000-0000-0000-000000000002')
	`); err == nil {
		t.Fatal("duplicate active billing owner succeeded")
	}

	var plusSubscriptionID, businessSubscriptionID string
	for _, subscription := range []struct {
		providerID string
		productID  string
		tier       string
	}{
		{providerID: "sub-plus", productID: "prod-plus", tier: "plus"},
		{providerID: "sub-business", productID: "prod-business", tier: "business"},
	} {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO provider_subscriptions(
				billing_account_id, project_id, revenuecat_subscription_id,
				product_id, app_id, store, environment, status, gives_access,
				mapped_tier, accepted_generation
			) VALUES ($1, 'proj-production', $2, $3, 'app-ios', 'app_store',
				'production', 'active', true, $4, 1)
			RETURNING id
		`, accountID, subscription.providerID, subscription.productID, subscription.tier).Scan(&id)
		if err != nil {
			t.Fatalf("insert %s provider subscription: %v", subscription.tier, err)
		}
		if subscription.tier == "plus" {
			plusSubscriptionID = id
		} else {
			businessSubscriptionID = id
		}
	}
	if plusSubscriptionID == businessSubscriptionID {
		t.Fatal("distinct provider subscriptions shared an identity")
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO provider_subscriptions(
			billing_account_id, project_id, revenuecat_subscription_id,
			product_id, store, environment, status, gives_access, accepted_generation
		) VALUES ($1, 'proj-production', 'sub-plus', 'prod-other', 'play_store',
			'production', 'active', true, 1)
	`, accountID); err == nil {
		t.Fatal("duplicate RevenueCat subscription identity succeeded")
	}

	var plusLicenseID, businessLicenseID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO billing_licenses(provider_subscription_id, tier, assigned_did, assigned_at)
		VALUES ($1, 'plus', 'did:plc:beneficiary', now()) RETURNING id
	`, plusSubscriptionID).Scan(&plusLicenseID); err != nil {
		t.Fatalf("insert plus license: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_licenses(provider_subscription_id, tier)
		VALUES ($1, 'plus')
	`, plusSubscriptionID); err == nil {
		t.Fatal("second license for one provider subscription succeeded")
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO billing_licenses(provider_subscription_id, tier)
		VALUES ($1, 'business') RETURNING id
	`, businessSubscriptionID).Scan(&businessLicenseID); err != nil {
		t.Fatalf("insert business license: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE billing_licenses
		SET assigned_did='did:plc:beneficiary', assigned_at=now()
		WHERE id=$1
	`, businessLicenseID); err == nil {
		t.Fatal("second license assignment for one DID succeeded")
	}

	var providerCount, licenseCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM provider_subscriptions`).Scan(&providerCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_licenses`).Scan(&licenseCount); err != nil {
		t.Fatal(err)
	}
	if providerCount != 2 || licenseCount != 2 {
		t.Fatalf("separate row counts = subscriptions %d, licenses %d; want 2 and 2", providerCount, licenseCount)
	}

	if _, err := pool.Exec(ctx, `DELETE FROM provider_subscriptions WHERE id=$1`, plusSubscriptionID); err != nil {
		t.Fatalf("delete provider subscription: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT id FROM billing_licenses WHERE id=$1`, plusLicenseID).Scan(new(string)); err != pgx.ErrNoRows {
		t.Fatalf("license cascade error = %v, want no rows", err)
	}
}

func TestEnsureBillingAccountIsExplicitStableAndConcurrent(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000069_subscription_accounts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(up))
	store := NewStore(pool)
	ctx := context.Background()
	owner := syntax.DID("did:plc:concurrent-billing-owner")

	if _, err := store.OwnerAccount(ctx, owner); !errors.Is(err, ErrBillingAccountNotFound) {
		t.Fatalf("ordinary owner read error = %v, want ErrBillingAccountNotFound", err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_accounts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("ordinary read created %d billing accounts", count)
	}

	const callers = 8
	results := make(chan BillingAccount, callers)
	errorsCh := make(chan error, callers)
	var ready sync.WaitGroup
	ready.Add(callers)
	start := make(chan struct{})
	for range callers {
		go func() {
			ready.Done()
			<-start
			account, _, err := store.EnsureAccount(ctx, owner)
			if err != nil {
				errorsCh <- err
				return
			}
			results <- account
		}()
	}
	ready.Wait()
	close(start)

	var stable BillingAccount
	for range callers {
		select {
		case err := <-errorsCh:
			t.Fatalf("concurrent ensure: %v", err)
		case account := <-results:
			if stable.ID == uuid.Nil {
				stable = account
			}
			if account.ID != stable.ID || account.RevenueCatAppUserID != stable.RevenueCatAppUserID {
				t.Fatalf("unstable account: got %+v, first %+v", account, stable)
			}
		}
	}

	repeated, created, err := store.EnsureAccount(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if created || repeated.ID != stable.ID || repeated.RevenueCatAppUserID != stable.RevenueCatAppUserID {
		t.Fatalf("repeated ensure = %+v, created=%t; first %+v", repeated, created, stable)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_accounts`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("concurrent ensure created %d accounts, want 1", count)
	}
}
