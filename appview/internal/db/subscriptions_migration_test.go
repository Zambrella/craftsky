package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestSubscriptionMigrationsUpDownUpPreserveExistingState(t *testing.T) {
	read := func(path string) string {
		t.Helper()
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(contents)
	}
	accountsUp := read("../../migrations/000069_subscription_accounts.up.sql")
	accountsDown := read("../../migrations/000069_subscription_accounts.down.sql")
	eventsUp := read("../../migrations/000070_revenuecat_events.up.sql")
	eventsDown := read("../../migrations/000070_revenuecat_events.down.sql")
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (did TEXT PRIMARY KEY);
		CREATE TABLE existing_authority_sentinels (kind TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO craftsky_profiles VALUES('did:plc:migration-owner');
		INSERT INTO existing_authority_sentinels VALUES('auth','auth-preserved'),('session','session-preserved'),('deletion','deletion-preserved');
	`)
	ctx := context.Background()
	for pass := 1; pass <= 2; pass++ {
		if _, err := pool.Exec(ctx, accountsUp+eventsUp); err != nil {
			t.Fatalf("apply subscription up pass %d: %v", pass, err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO billing_accounts(id,owner_did,revenuecat_app_user_id)
			VALUES('10000000-0000-4000-8000-000000000071','did:plc:migration-owner','20000000-0000-4000-8000-000000000071')
		`); err != nil {
			t.Fatalf("exercise subscription constraints pass %d: %v", pass, err)
		}
		if _, err := pool.Exec(ctx, eventsDown+accountsDown); err != nil {
			t.Fatalf("apply subscription down pass %d: %v", pass, err)
		}
		var tablesRemain bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass('billing_accounts') IS NOT NULL OR to_regclass('revenuecat_events') IS NOT NULL`).Scan(&tablesRemain); err != nil {
			t.Fatal(err)
		}
		if tablesRemain {
			t.Fatal("subscription tables remained after down migration")
		}
		var sentinelCount int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM existing_authority_sentinels WHERE value LIKE '%-preserved'`).Scan(&sentinelCount); err != nil || sentinelCount != 3 {
			t.Fatalf("existing authority sentinels = %d, error %v", sentinelCount, err)
		}
	}
}
