package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestOnboardingCompletionMigrationUpDownAndReapply(t *testing.T) {
	t.Parallel()
	up, err := os.ReadFile("../../migrations/000062_account_onboarding_completion.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000062_account_onboarding_completion.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()

	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply migration 000062: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_onboarding_completions(account_did)
		VALUES('did:plc:alice')
	`); err != nil {
		t.Fatalf("insert migrated completion: %v", err)
	}
	var completedAtPresent bool
	if err := pool.QueryRow(ctx, `
		SELECT completed_at IS NOT NULL
		FROM account_onboarding_completions
		WHERE account_did='did:plc:alice'
	`).Scan(&completedAtPresent); err != nil || !completedAtPresent {
		t.Fatalf("read migrated completion = %t, %v", completedAtPresent, err)
	}

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("roll back migration 000062: %v", err)
	}
	var tableAbsent bool
	if err := pool.QueryRow(ctx, `
		SELECT to_regclass(current_schema() || '.account_onboarding_completions') IS NULL
	`).Scan(&tableAbsent); err != nil || !tableAbsent {
		t.Fatalf("down migration table absent = %t, %v", tableAbsent, err)
	}

	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("reapply migration 000062: %v", err)
	}
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM account_onboarding_completions`).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("reapplied completion rows = %d, %v", rows, err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO account_onboarding_completions(account_did)
		VALUES('did:plc:bob')
	`); err != nil {
		t.Fatalf("insert after reapply: %v", err)
	}
}
