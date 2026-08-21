package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestAccountDeletionIntentExpiryMigrationUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000047_account_deletion_intent_expiry.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000047_account_deletion_intent_expiry.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, `
		CREATE TABLE account_deletion_operations (
			id UUID PRIMARY KEY,
			state TEXT NOT NULL,
			intent_expires_at TIMESTAMPTZ
		);
	`)
	ctx := context.Background()
	for pass := 0; pass < 2; pass++ {
		if _, err := pool.Exec(ctx, string(up)); err != nil {
			t.Fatalf("apply up pass %d: %v", pass+1, err)
		}
		var definition string
		if err := pool.QueryRow(ctx, `
			SELECT pg_get_indexdef(indexrelid)
			FROM pg_index
			WHERE indexrelid='account_deletion_operations_intent_expiry_idx'::regclass
		`).Scan(&definition); err != nil {
			t.Fatalf("read index definition: %v", err)
		}
		if !strings.Contains(definition, "(intent_expires_at, id)") ||
			!strings.Contains(definition, "WHERE (state = 'intent'::text)") {
			t.Fatalf("unexpected index definition: %s", definition)
		}
		if _, err := pool.Exec(ctx, string(down)); err != nil {
			t.Fatalf("apply down pass %d: %v", pass+1, err)
		}
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass('account_deletion_operations_intent_expiry_idx') IS NOT NULL`).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Fatal("intent expiry index remained after down")
		}
	}
}
