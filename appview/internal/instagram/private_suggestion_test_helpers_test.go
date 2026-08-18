package instagram

import (
	"context"
	"crypto/sha256"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedSuggestionImport(
	t *testing.T,
	pool *pgxpool.Pool,
	id uuid.UUID,
	owner syntax.DID,
	username string,
	now time.Time,
) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_imports (
			id, owner_did, state, source_type, following_count,
			created_at, updated_at
		) VALUES ($1, $2, 'active', 'manual', 1, $3, $3)
	`, id, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO instagram_graph_handles (
			import_id, username_normalized, matched, created_at
		) VALUES ($1, $2, false, $3)
	`, id, username, now); err != nil {
		t.Fatal(err)
	}
}

func seedSuggestionLink(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	username string,
	now time.Time,
) {
	t.Helper()
	digest := sha256.Sum256([]byte(owner))
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version, igsid_digest,
			username, username_normalized, discoverable, conflict_pending,
			verified_at, created_at, updated_at
		) VALUES ($1, $2, 'active', $3, 1, $4,
		          $5, $5, true, false, $6, $6, $6)
	`, uuid.New(), owner, "igsid-"+uuid.NewString(), digest[:], username, now); err != nil {
		t.Fatal(err)
	}
}

func ensureInstagramOwnerLifecyclePreState(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS craftsky_profiles (
			did TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE TABLE IF NOT EXISTS oauth_sessions (
			account_did TEXT NOT NULL,
			session_id TEXT NOT NULL,
			data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (account_did, session_id)
		);
		CREATE TABLE IF NOT EXISTS oauth_auth_requests (
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
		CREATE TABLE IF NOT EXISTS craftsky_sessions (
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
		CREATE TABLE IF NOT EXISTS account_deletion_operations (
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
	`); err != nil {
		t.Fatalf("create Instagram owner lifecycle migration pre-state: %v", err)
	}
}
