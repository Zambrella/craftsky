package db_test

import (
	"context"
	"os"
	"testing"

	"social.craftsky/appview/internal/testdb"
)

func TestPDSMigrationCallbackAuthorityMigrationUpDownUp(t *testing.T) {
	providerUp, err := os.ReadFile("../../migrations/000064_provider_first_registration.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile("../../migrations/000066_pds_migration_identity.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile("../../migrations/000066_pds_migration_identity.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	pool := testdb.WithSchema(t, providerRegistrationMigrationBaseline)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(providerUp)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		CREATE TABLE oauth_sessions(
			account_did TEXT NOT NULL,
			session_id TEXT NOT NULL,
			data JSONB NOT NULL,
			PRIMARY KEY(account_did,session_id)
		);
		INSERT INTO oauth_auth_requests(
			state,data,handoff_mode,device_id,purpose,owner_did,owner_generation,
			auth_epoch,request_uri,request_state,consumed_at
		) VALUES
			('backfilled','{"authserver_url":"https://issuer-a.example"}','verified_link','device-1',
			 'login','did:plc:login',1,1,'urn:request:backfilled','consumed',now()),
			('incomplete','{}','verified_link','device-2','login','did:plc:login',1,1,
			 'urn:request:incomplete','ready',NULL);
		INSERT INTO oauth_sessions(account_did,session_id,data)
		VALUES('did:plc:login','backfilled','{"host_url":"https://pds-a.example"}');
		CREATE TABLE account_deletion_operations(
			id UUID PRIMARY KEY,
			confirmation_handle_hash BYTEA
		);
		INSERT INTO account_deletion_operations(id,confirmation_handle_hash)
		VALUES('10000000-0000-4000-8000-000000000013',decode(repeat('13',32),'hex'));
	`); err != nil {
		t.Fatal(err)
	}

	for pass := 1; pass <= 2; pass++ {
		if _, err := pool.Exec(ctx, string(up)); err != nil {
			t.Fatalf("apply up pass %d: %v", pass, err)
		}
		var resource, issuer string
		if err := pool.QueryRow(ctx, `
			SELECT resource_server_origin,authorization_server_issuer
			FROM oauth_auth_requests WHERE state='backfilled'
		`).Scan(&resource, &issuer); err != nil {
			t.Fatal(err)
		}
		if resource != "https://pds-a.example" || issuer != "https://issuer-a.example" {
			t.Fatalf("backfilled authority = %q/%q", resource, issuer)
		}
		var incomplete int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_requests WHERE state='incomplete'`).Scan(&incomplete); err != nil {
			t.Fatal(err)
		}
		if incomplete != 0 {
			t.Fatal("unrecoverable pre-migration login request was retained")
		}
		if !columnExists(t, pool, "account_deletion_operations", "confirmation_did_hash") ||
			columnExists(t, pool, "account_deletion_operations", "confirmation_handle_hash") {
			t.Fatal("up migration did not rename deletion confirmation hash to DID")
		}
		var confirmationHash []byte
		if err := pool.QueryRow(ctx, `SELECT confirmation_did_hash FROM account_deletion_operations`).Scan(&confirmationHash); err != nil {
			t.Fatal(err)
		}
		if len(confirmationHash) != 32 || confirmationHash[0] != 0x13 {
			t.Fatal("up migration did not preserve deletion confirmation hash")
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO oauth_auth_requests(
				state,data,handoff_mode,device_id,purpose,owner_did,owner_generation,
				auth_epoch,request_uri,request_state
			) VALUES('missing-authority','{}','verified_link','device-3','login',
			         'did:plc:login',1,1,'urn:request:missing-authority','ready')
		`); !isCheckViolation(err) {
			t.Fatalf("missing callback authority error = %v, want check violation", err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO oauth_auth_requests(
				state,data,handoff_mode,device_id,purpose,owner_did,owner_generation,
				auth_epoch,request_uri,request_state,exchange_attempt_id,
				exchange_started_at,exchange_finished_at,consumed_at,
				resource_server_origin,authorization_server_issuer
			) VALUES('callback-cleanup','{}','verified_link','device-4','login',
			         'did:plc:login',1,1,'urn:request:callback-cleanup','cleanup_pending',
			         '20000000-0000-0000-0000-000000000066',now(),now(),now(),
			         'https://pds-a.example','https://issuer-a.example');
			INSERT INTO oauth_unverified_credentials(request_state,data,status,eligible_at,expires_at)
			VALUES('callback-cleanup','{}','pending',now(),now()+interval '1 hour');
		`); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(down)); err != nil {
			t.Fatalf("apply down pass %d: %v", pass, err)
		}
		if columnExists(t, pool, "oauth_auth_requests", "resource_server_origin") ||
			columnExists(t, pool, "oauth_auth_requests", "authorization_server_issuer") {
			t.Fatal("callback authority columns remained after down migration")
		}
		if !columnExists(t, pool, "account_deletion_operations", "confirmation_handle_hash") ||
			columnExists(t, pool, "account_deletion_operations", "confirmation_did_hash") {
			t.Fatal("down migration did not restore deletion confirmation hash name")
		}
		var cleanupRows int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM oauth_auth_requests WHERE state='callback-cleanup'`).Scan(&cleanupRows); err != nil {
			t.Fatal(err)
		}
		if cleanupRows != 0 {
			t.Fatal("down migration retained unrepresentable login cleanup state")
		}
	}
}
