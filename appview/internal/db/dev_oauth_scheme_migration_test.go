package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestDevOAuthSchemeMigrationUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000052_dev_oauth_scheme.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000052_dev_oauth_scheme.down.sql")
	if err != nil {
		t.Fatal(err)
	}

	pool := testdb.WithSchema(t, `
		CREATE TABLE oauth_auth_requests (
			id TEXT PRIMARY KEY,
			handoff_mode TEXT NOT NULL,
			loopback_redirect_uri TEXT,
			device_id TEXT NOT NULL,
			CONSTRAINT oauth_auth_requests_handoff_check CHECK (
				handoff_mode IN ('verified_link', 'loopback')
				AND btrim(device_id) <> ''
				AND (
					(handoff_mode = 'verified_link' AND loopback_redirect_uri IS NULL)
					OR
					(handoff_mode = 'loopback' AND loopback_redirect_uri IS NOT NULL)
				)
			)
		);
	`)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply development OAuth scheme up: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests(id,handoff_mode,loopback_redirect_uri,device_id)
		VALUES('dev','dev_scheme',NULL,'device-dev')
	`); err != nil {
		t.Fatalf("insert development scheme request: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests(id,handoff_mode,loopback_redirect_uri,device_id)
		VALUES('dev-with-uri','dev_scheme','http://127.0.0.1:1234/callback','device-dev')
	`); !isCheckViolation(err) {
		t.Fatalf("development scheme with client URI error = %v, want check violation", err)
	}

	if _, err := pool.Exec(ctx, string(down)); err != nil {
		t.Fatalf("apply development OAuth scheme down: %v", err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_requests WHERE handoff_mode='dev_scheme'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("development OAuth requests after down = %d, want 0", remaining)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests(id,handoff_mode,loopback_redirect_uri,device_id)
		VALUES('dev-after-down','dev_scheme',NULL,'device-dev')
	`); !isCheckViolation(err) {
		t.Fatalf("development scheme after down error = %v, want check violation", err)
	}
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply development OAuth scheme second up: %v", err)
	}
}
