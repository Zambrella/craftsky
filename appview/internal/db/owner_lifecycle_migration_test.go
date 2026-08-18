package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestOwnerLifecycleMigrationsUpDownUp(t *testing.T) {
	up38 := readOwnerLifecycleMigration(t, "../../migrations/000038_owner_auth_lifecycle.up.sql")
	down38 := readOwnerLifecycleMigration(t, "../../migrations/000038_owner_auth_lifecycle.down.sql")
	up39 := readOwnerLifecycleMigration(t, "../../migrations/000039_owner_effects_terminal_purge.up.sql")
	down39 := readOwnerLifecycleMigration(t, "../../migrations/000039_owner_effects_terminal_purge.down.sql")

	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles (
			did TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE oauth_sessions (
			account_did TEXT NOT NULL,
			session_id TEXT NOT NULL,
			data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (account_did, session_id)
		);
		CREATE TABLE oauth_auth_requests (
			state TEXT PRIMARY KEY,
			data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			handoff_mode TEXT NOT NULL DEFAULT 'deep_link',
			loopback_redirect_uri TEXT,
			purpose TEXT NOT NULL DEFAULT 'login',
			device_id TEXT,
			account_deletion_owner_did TEXT,
			account_deletion_job_id UUID
		);
		CREATE TABLE craftsky_sessions (
			token_hash BYTEA PRIMARY KEY,
			account_did TEXT NOT NULL,
			oauth_session_id TEXT NOT NULL,
			device_label TEXT,
			last_device_id TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			revoked_at TIMESTAMPTZ,
			FOREIGN KEY (account_did, oauth_session_id)
				REFERENCES oauth_sessions(account_did, session_id) ON DELETE CASCADE
		);
		CREATE TABLE account_deletion_operations (
			id UUID PRIMARY KEY,
			owner_did TEXT NOT NULL UNIQUE,
			state TEXT NOT NULL,
			accepted_at TIMESTAMPTZ,
			reauth_oauth_session_id TEXT,
			deletion_oauth_session_id TEXT,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			next_attempt_at TIMESTAMPTZ,
			error_category TEXT,
			intent_proof_hash BYTEA,
			confirmation_handle_hash BYTEA,
			intent_expires_at TIMESTAMPTZ,
			lease_owner TEXT,
			lease_token UUID,
			lease_expires_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			FOREIGN KEY (owner_did, deletion_oauth_session_id)
				REFERENCES oauth_sessions(account_did, session_id),
			FOREIGN KEY (owner_did, reauth_oauth_session_id)
				REFERENCES oauth_sessions(account_did, session_id)
		);
		INSERT INTO craftsky_profiles(did) VALUES ('did:plc:existing-member');
	`)
	applyOwnerLifecycleMigration(t, pool, "38 up", up38)
	assertOwnerLifecycleSchema(t, pool)
	assertAuthLifecycleSchema(t, pool)
	assertAuthRequestHandoffDefault(t, pool, nil)
	assertExistingMemberLifecycleBackfill(t, pool)
	applyOwnerLifecycleMigration(t, pool, "39 up", up39)
	assertOwnerEffectSchema(t, pool)
	assertOwnerLifecycleConstraints(t, pool)

	applyOwnerLifecycleMigration(t, pool, "39 down", down39)
	assertRelationAbsent(t, pool, "owner_effect_attempts")
	assertRelationAbsent(t, pool, "owner_purge_components")
	applyOwnerLifecycleMigration(t, pool, "38 down", down38)
	assertRelationAbsent(t, pool, "owner_lifecycles")
	assertRelationAbsent(t, pool, "oauth_handoff_receipts")
	assertRelationAbsent(t, pool, "oauth_handoff_exchanges")
	assertRelationAbsent(t, pool, "auth_auxiliary_cleanup_jobs")
	legacyDefault := "'deep_link'::text"
	assertAuthRequestHandoffDefault(t, pool, &legacyDefault)

	applyOwnerLifecycleMigration(t, pool, "38 second up", up38)
	applyOwnerLifecycleMigration(t, pool, "39 second up", up39)
	assertOwnerLifecycleSchema(t, pool)
	assertAuthLifecycleSchema(t, pool)
	assertOwnerEffectSchema(t, pool)
}

func assertAuthRequestHandoffDefault(t *testing.T, pool *pgxpool.Pool, want *string) {
	t.Helper()
	var got *string
	if err := pool.QueryRow(context.Background(), `
		SELECT column_default
		FROM information_schema.columns
		WHERE table_schema=current_schema()
		  AND table_name='oauth_auth_requests'
		  AND column_name='handoff_mode'
	`).Scan(&got); err != nil {
		t.Fatalf("read oauth_auth_requests.handoff_mode default: %v", err)
	}
	if (got == nil) != (want == nil) || got != nil && *got != *want {
		t.Fatalf("oauth_auth_requests.handoff_mode default = %v, want %v", got, want)
	}
}

func assertAuthLifecycleSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	assertColumnExists(t, pool, "account_deletion_operations", "deletion_credential_generation")
	for _, column := range []string{
		"lifecycle_state", "owner_generation", "auth_epoch", "row_version",
		"absolute_expires_at", "deletion_operation_id", "deletion_credential_generation",
		"revocation_requested_at", "cleanup_attempts", "cleanup_next_attempt_at",
		"cleanup_lease_token", "cleanup_lease_expires_at", "cleanup_last_category",
	} {
		assertColumnExists(t, pool, "oauth_sessions", column)
	}
	for _, column := range []string{
		"owner_did", "owner_generation", "auth_epoch", "request_uri", "request_state",
		"exchange_attempt_id", "exchange_started_at", "exchange_finished_at", "consumed_at",
	} {
		assertColumnExists(t, pool, "oauth_auth_requests", column)
	}
	for _, column := range []string{
		"lifecycle_state", "auth_epoch", "idle_expires_at", "last_device_seen_at",
	} {
		assertColumnExists(t, pool, "craftsky_sessions", column)
	}
	for _, relation := range []string{
		"oauth_handoff_exchanges", "oauth_handoff_receipts", "auth_auxiliary_cleanup_jobs",
	} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `SELECT to_regclass($1) IS NOT NULL`, relation).Scan(&exists); err != nil {
			t.Fatalf("inspect relation %s: %v", relation, err)
		}
		if !exists {
			t.Errorf("relation %s does not exist", relation)
		}
	}
	for _, index := range []string{
		"oauth_sessions_revocation_claim_idx",
		"oauth_sessions_expiry_idx",
		"oauth_sessions_owner_epoch_idx",
		"oauth_auth_requests_owner_epoch_idx",
		"oauth_auth_requests_state_created_idx",
		"oauth_handoff_exchanges_expiry_idx",
		"oauth_handoff_receipts_expiry_idx",
		"craftsky_sessions_expiry_idx",
		"auth_auxiliary_cleanup_claim_idx",
	} {
		assertIndexExists(t, pool, index)
	}
}

func assertExistingMemberLifecycleBackfill(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var state string
	var generation, authEpoch int64
	if err := pool.QueryRow(context.Background(), `
		SELECT state,generation,auth_epoch
		FROM owner_lifecycles
		WHERE owner_did='did:plc:existing-member'
	`).Scan(&state, &generation, &authEpoch); err != nil {
		t.Fatalf("read existing member lifecycle: %v", err)
	}
	if state != "active" || generation != 1 || authEpoch != 1 {
		t.Fatalf("existing member lifecycle = %s/%d/%d", state, generation, authEpoch)
	}
}

func readOwnerLifecycleMigration(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return contents
}

func applyOwnerLifecycleMigration(t *testing.T, pool *pgxpool.Pool, label string, sql []byte) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
		t.Fatalf("apply %s: %v", label, err)
	}
}

func assertOwnerLifecycleSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, column := range []string{
		"owner_did", "state", "generation", "auth_epoch", "transition_reason",
		"transitioned_at", "terminal_at", "purge_completed_at", "created_at", "updated_at",
	} {
		assertColumnExists(t, pool, "owner_lifecycles", column)
	}
	assertIndexExists(t, pool, "owner_lifecycles_terminal_pending_idx")
}

func assertOwnerEffectSchema(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, index := range []string{
		"owner_purge_components_claim_idx",
		"owner_effect_attempts_owner_generation_idx",
		"owner_effect_attempts_remote_key_idx",
		"owner_effect_attempts_unresolved_idx",
	} {
		assertIndexExists(t, pool, index)
	}
}

func assertOwnerLifecycleConstraints(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES ('did:plc:migration-owner','departed',1,1,'onboarding',now(),now(),now())
	`); err != nil {
		t.Fatalf("insert valid lifecycle: %v", err)
	}

	assertCheckViolation(t, pool, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES ('did:plc:bad-generation','departed',0,1,'test',now(),now(),now())
	`)
	assertCheckViolation(t, pool, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,terminal_at,created_at,updated_at
		) VALUES ('did:plc:bad-terminal','active',1,1,'test',now(),now(),now(),now())
	`)
	assertCheckViolation(t, pool, `
		UPDATE owner_lifecycles
		SET state='active',generation=1,transition_reason='activate',transitioned_at=now(),updated_at=now()
		WHERE owner_did='did:plc:migration-owner'
	`)
	if _, err := pool.Exec(ctx, `
		UPDATE owner_lifecycles
		SET state='terminal',generation=2,auth_epoch=2,transition_reason='identityDeleted',
		    transitioned_at=now(),terminal_at=now(),updated_at=now()
		WHERE owner_did='did:plc:migration-owner'
	`); err != nil {
		t.Fatalf("transition lifecycle terminal: %v", err)
	}
	assertCheckViolation(t, pool, `
		UPDATE owner_lifecycles
		SET terminal_at=terminal_at + interval '1 second',updated_at=now()
		WHERE owner_did='did:plc:migration-owner'
	`)
	assertCheckViolation(t, pool, `
		UPDATE owner_lifecycles
		SET state='departed',generation=3,transition_reason='replay',
		    transitioned_at=now(),terminal_at=NULL,updated_at=now()
		WHERE owner_did='did:plc:migration-owner'
	`)

	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_purge_components(
			owner_did,owner_generation,component,did_role,state,next_attempt_at,created_at,updated_at
		) VALUES ('did:plc:migration-owner',2,'profiles','owner','pending',now(),now(),now())
	`); err != nil {
		t.Fatalf("insert purge component: %v", err)
	}
	assertCheckViolation(t, pool, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,deterministic_key,
			request_fingerprint,remote_outcome,projection_disposition,repeat_forbidden,
			remote_deadline,created_at,updated_at
		) VALUES (
			'bad-fingerprint','did:plc:migration-owner',2,'pds_record','at://did:plc:migration-owner/social.craftsky.feed.post/one',
			decode('00','hex'),'prepared','pending',false,now()+interval '1 minute',now(),now()
		)
	`)
}

func assertColumnExists(t *testing.T, pool *pgxpool.Pool, table, column string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_schema=current_schema() AND table_name=$1 AND column_name=$2
		)
	`, table, column).Scan(&exists); err != nil {
		t.Fatalf("inspect %s.%s: %v", table, column, err)
	}
	if !exists {
		t.Errorf("column %s.%s does not exist", table, column)
	}
}

func assertIndexExists(t *testing.T, pool *pgxpool.Pool, index string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass($1) IS NOT NULL`, index).Scan(&exists); err != nil {
		t.Fatalf("inspect index %s: %v", index, err)
	}
	if !exists {
		t.Errorf("index %s does not exist", index)
	}
}

func assertRelationAbsent(t *testing.T, pool *pgxpool.Pool, relation string) {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), `SELECT to_regclass($1) IS NOT NULL`, relation).Scan(&exists); err != nil {
		t.Fatalf("inspect relation %s: %v", relation, err)
	}
	if exists {
		t.Errorf("relation %s still exists", relation)
	}
}

func assertCheckViolation(t *testing.T, pool *pgxpool.Pool, query string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), query)
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23514" {
		t.Fatalf("query error = %v, want PostgreSQL check violation", err)
	}
}
