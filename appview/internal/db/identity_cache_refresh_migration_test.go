package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestIdentityCacheRefreshMigrationUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000053_identity_cache_refresh.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000053_identity_cache_refresh.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, "")
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply identity refresh up: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_refresh_state(
			did,next_attempt_at,attempt_count,last_result
		) VALUES('did:plc:test',now(),1,'retry')
	`); err != nil {
		t.Fatalf("insert identity refresh state: %v", err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply identity refresh down: %v", err)
	}
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('atproto_identity_refresh_state') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("identity refresh state table remained after down")
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply identity refresh second up: %v", err)
	}
}

func TestTapIdentityRefreshTriggerMigrationUpDownUp(t *testing.T) {
	base, err := os.ReadFile("../../migrations/000053_identity_cache_refresh.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000054_tap_identity_refresh_trigger.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000054_tap_identity_refresh_trigger.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(base))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply Tap identity refresh trigger up: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_refresh_state(
			did,next_attempt_at,attempt_count,last_result,tap_event_id
		) VALUES('did:plc:test',now(),0,'pending',42)
	`); err != nil {
		t.Fatalf("insert pending Tap identity refresh: %v", err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply Tap identity refresh trigger down: %v", err)
	}
	var result string
	if err := pool.QueryRow(ctx, `SELECT last_result FROM atproto_identity_refresh_state WHERE did='did:plc:test'`).Scan(&result); err != nil || result != "retry" {
		t.Fatalf("down-migrated result=%q err=%v", result, err)
	}
	var eventColumnExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_schema=current_schema()
			  AND table_name='atproto_identity_refresh_state'
			  AND column_name='tap_event_id'
		)
	`).Scan(&eventColumnExists); err != nil {
		t.Fatal(err)
	}
	if eventColumnExists {
		t.Fatal("tap_event_id remained after down migration")
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply Tap identity refresh trigger second up: %v", err)
	}
}

func TestTapIdentityRefreshVersionMigrationUpDownUp(t *testing.T) {
	base53, err := os.ReadFile("../../migrations/000053_identity_cache_refresh.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	base54, err := os.ReadFile("../../migrations/000054_tap_identity_refresh_trigger.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000057_tap_identity_refresh_version.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000057_tap_identity_refresh_version.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, string(base53))
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(base54)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_refresh_state(
			did,next_attempt_at,attempt_count,last_result,tap_event_id
		) VALUES('did:plc:test',now(),0,'pending',700)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply refresh version up: %v", err)
	}
	var version int64
	if err := pool.QueryRow(ctx, `
		SELECT refresh_version FROM atproto_identity_refresh_state
		WHERE did='did:plc:test'
	`).Scan(&version); err != nil || version != 1 {
		t.Fatalf("backfilled refresh version=%d err=%v", version, err)
	}
	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply refresh version down: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply refresh version second up: %v", err)
	}
}
