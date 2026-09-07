package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestIdentityHandleInvalidAliasMigrationUpDownUp(t *testing.T) {
	base, err := os.ReadFile("../../migrations/000015_identity_handle_cache.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000067_identity_handle_invalid_alias.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000067_identity_handle_invalid_alias.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(base))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply invalid-handle alias up: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES
			('did:plc:one','handle.invalid','handle.invalid',now()),
			('did:plc:two','handle.invalid','handle.invalid',now())
	`); err != nil {
		t.Fatalf("store multiple invalid handles: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES
			('did:plc:valid-one','valid.example','valid.example',now()),
			('did:plc:valid-two','valid.example','valid.example',now())
	`); err == nil {
		t.Fatal("duplicate valid alias succeeded")
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply invalid-handle alias down: %v", err)
	}
	var invalidRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_identity_cache WHERE handle_lower='handle.invalid'`).Scan(&invalidRows); err != nil || invalidRows != 1 {
		t.Fatalf("invalid rows after down=%d err=%v, want 1", invalidRows, err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply invalid-handle alias second up: %v", err)
	}
}
