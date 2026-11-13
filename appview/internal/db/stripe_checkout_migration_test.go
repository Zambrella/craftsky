package db_test

import (
	"strings"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestStripeCheckoutMigrationPreservesRecoveryOnRollback(t *testing.T) {
	read := func(name string) string {
		t.Helper()
		contents, err := testdb.ReadMigration(name)
		if err != nil {
			t.Fatal(err)
		}
		return string(contents)
	}
	pool := testdb.WithSchema(t, read("000076_subscription_accounts.up.sql"))
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id)
		VALUES('10000000-0000-4000-8000-000000000125','did:plc:checkout-migration','20000000-0000-4000-8000-000000000125')
	`); err != nil {
		t.Fatal(err)
	}
	up := read("000125_stripe_web_checkouts.up.sql")
	down := read("000125_stripe_web_checkouts.down.sql")
	if _, err := pool.Exec(t.Context(), up); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO billing_web_checkouts(billing_account_id,environment,operation_id,tier,stripe_account_id,price_id,success_url,cancel_url,created_at)
		VALUES('10000000-0000-4000-8000-000000000125','sandbox','30000000-0000-4000-8000-000000000125','plus','acct_test','price_plus','https://app.example/success','https://app.example/cancel',now())
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), down); err == nil || !strings.Contains(err.Error(), "resolve and remove Stripe checkout operations") {
		t.Fatalf("rollback did not protect the pending provider operation: %v", err)
	}
	var pending bool
	if err := pool.QueryRow(t.Context(), `SELECT web_checkout_pending FROM billing_accounts`).Scan(&pending); err != nil || !pending {
		t.Fatalf("rollback lost the native purchase guard: pending=%t, err=%v", pending, err)
	}
	var retained int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM billing_web_checkouts`).Scan(&retained); err != nil || retained != 1 {
		t.Fatalf("rollback lost provider recovery: rows=%d, err=%v", retained, err)
	}
	// Once provider recovery records have been deliberately resolved, an empty
	// catalog rollback remains reversible and preserves the customer's identity.
	if _, err := pool.Exec(t.Context(), `DELETE FROM billing_web_checkouts`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), down+up); err != nil {
		t.Fatalf("empty checkout migration cycle: %v", err)
	}
	var customer string
	if err := pool.QueryRow(t.Context(), `SELECT revenuecat_app_user_id::text,web_checkout_pending FROM billing_accounts`).Scan(&customer, &pending); err != nil || customer != "20000000-0000-4000-8000-000000000125" || pending {
		t.Fatalf("migration changed customer identity or retained a false pending flag: customer=%q, pending=%t, err=%v", customer, pending, err)
	}
}
