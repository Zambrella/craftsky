package db_test

import (
	"context"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestSubscriptionBusinessMigrationDropsLegacyAccountTypeTable(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	var present bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname='craftsky_account_types')`).Scan(&present); err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("testing-only account type table remains after migration")
	}
	if err := testdb.ApplyMigrations(ctx, pool, "000078_subscription_business_access.down.sql"); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname='craftsky_account_types')`).Scan(&present); err != nil || !present {
		t.Fatalf("rollback table present=%v, error=%v", present, err)
	}
}

func TestSubscriptionBusinessMigrationDropsSeededTestingFlags(t *testing.T) {
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if err := testdb.ApplyMigrations(ctx, pool, "000061_business_account_types.up.sql", "000076_subscription_accounts.up.sql"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_account_types(owner_did,account_type) VALUES ('did:plc:legacy-business','business'),('did:plc:legacy-regular','regular')`); err != nil {
		t.Fatal(err)
	}
	if err := testdb.ApplyMigrations(ctx, pool, "000078_subscription_business_access.up.sql"); err != nil {
		t.Fatal(err)
	}
	var present bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=current_schema() AND c.relname='craftsky_account_types')`).Scan(&present); err != nil || present {
		t.Fatalf("table after up=%t, err=%v", present, err)
	}
}
