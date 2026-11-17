package db_test

import (
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestPromotionalGrantsMigrationRollbackPreservesGrantedAccessAndHistory(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := t.Context()
	down, err := testdb.ReadMigration("000126_billing_promotional_grants.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := testdb.ReadMigration("000126_billing_promotional_grants.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	// Empty rollback and reapplication preserve the deployment recovery path.
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id) VALUES ('10000000-0000-4000-8000-000000000126','did:plc:rollback-gift','20000000-0000-4000-8000-000000000126');
		INSERT INTO billing_promotional_grants(billing_account_id,project_id,revenuecat_subscription_id,environment,tier,gives_access,accepted_generation)
		VALUES ('10000000-0000-4000-8000-000000000126','project','gift','production','plus',true,1);
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err == nil {
		t.Fatal("rollback discarded an active gift")
	}
	var active bool
	if err := pool.QueryRow(ctx, `SELECT gives_access FROM appview_effective_subscription_access('production') WHERE owner_did='did:plc:rollback-gift'`).Scan(&active); err != nil || !active {
		t.Fatalf("failed rollback changed access: active=%t err=%v", active, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE billing_promotional_grants SET gives_access=false`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(down)); err == nil {
		t.Fatal("rollback discarded gift audit history")
	}
	var retained int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM billing_promotional_grants`).Scan(&retained); err != nil || retained != 1 {
		t.Fatalf("failed rollback changed history: rows=%d err=%v", retained, err)
	}
}
