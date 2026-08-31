package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestProviderRegistrationMigrationAuthorityShapesUpDownUp(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000061_provider_first_registration.up.sql")
	if err != nil {
		t.Fatalf("read provider registration up migration: %v", err)
	}
	down, err := os.ReadFile("../../migrations/000061_provider_first_registration.down.sql")
	if err != nil {
		t.Fatalf("read provider registration down migration: %v", err)
	}

	pool := testdb.WithSchema(t, providerRegistrationMigrationBaseline)
	ctx := context.Background()
	for pass := 1; pass <= 2; pass++ {
		if _, err := pool.Exec(ctx, string(up)); err != nil {
			t.Fatalf("apply up pass %d: %v", pass, err)
		}

		assertProviderRegistrationAuthorityShapes(t, pool)

		if _, err := pool.Exec(ctx, string(down)); err != nil {
			t.Fatalf("apply down pass %d: %v", pass, err)
		}
		if columnExists(t, pool, "oauth_auth_requests", "registration_provider_origin") {
			t.Fatal("registration_provider_origin remained after down migration")
		}
		if columnExists(t, pool, "oauth_auth_requests", "registration_issuer") {
			t.Fatal("registration_issuer remained after down migration")
		}

		var registrationRows int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM oauth_auth_requests WHERE purpose='registration'
		`).Scan(&registrationRows); err != nil {
			t.Fatalf("count registration rows after down pass %d: %v", pass, err)
		}
		if registrationRows != 0 {
			t.Fatalf("registration rows after down pass %d = %d, want 0", pass, registrationRows)
		}
		if pass == 1 {
			if _, err := pool.Exec(ctx, `DELETE FROM oauth_auth_requests`); err != nil {
				t.Fatalf("reset request fixtures after down pass %d: %v", pass, err)
			}
		}
	}
}

func TestProviderRegistrationMigrationAddsOwnerlessCredentialQuarantine(t *testing.T) {
	up, err := os.ReadFile("../../migrations/000061_provider_first_registration.up.sql")
	if err != nil {
		t.Fatalf("read provider registration up migration: %v", err)
	}

	pool := testdb.WithSchema(t, providerRegistrationMigrationBaseline)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, string(up)); err != nil {
		t.Fatalf("apply up migration: %v", err)
	}

	if !columnExists(t, pool, "oauth_unverified_credentials", "data") {
		t.Fatal("oauth_unverified_credentials.data missing after up migration")
	}
	if columnExists(t, pool, "oauth_unverified_credentials", "owner_did") {
		t.Fatal("unverified credential quarantine must not require owner authority")
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests(
			state,data,handoff_mode,device_id,purpose,request_uri,
			registration_provider_origin,registration_issuer,
			request_state,exchange_attempt_id,exchange_started_at,consumed_at
		) VALUES(
			'registration-cleanup','{}','verified_link','device-1','registration',
			'urn:request:registration-cleanup','https://bsky.social','https://auth.bsky.app',
			'exchange_started','20000000-0000-0000-0000-000000000001',now(),now()
		)
	`); err != nil {
		t.Fatalf("insert exchanging ownerless registration: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO oauth_unverified_credentials(
			request_state,data,status,eligible_at,expires_at
		) VALUES(
			'registration-cleanup','{"accessToken":"server-only"}','held',now() + interval '1 minute',now() + interval '1 hour'
		)
	`); err != nil {
		t.Fatalf("quarantine ownerless registration credential: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		UPDATE oauth_unverified_credentials
		SET status='pending',eligible_at=now(),updated_at=now()
		WHERE request_state='registration-cleanup';
		UPDATE oauth_auth_requests
		SET request_state='cleanup_pending',exchange_finished_at=now()
		WHERE state='registration-cleanup';
	`); err != nil {
		t.Fatalf("mark registration credential cleanup pending: %v", err)
	}

	var requestStatus, credentialStatus string
	if err := pool.QueryRow(ctx, `
		SELECT r.request_state,c.status
		FROM oauth_auth_requests r
		JOIN oauth_unverified_credentials c ON c.request_state=r.state
		WHERE r.state='registration-cleanup'
	`).Scan(&requestStatus, &credentialStatus); err != nil {
		t.Fatalf("read cleanup state: %v", err)
	}
	if requestStatus != "cleanup_pending" || credentialStatus != "pending" {
		t.Fatalf("cleanup states = %q/%q, want cleanup_pending/pending", requestStatus, credentialStatus)
	}
}

func assertProviderRegistrationAuthorityShapes(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	tests := []struct {
		name         string
		purpose      string
		state        string
		requestState string
		ownerDID     any
		generation   any
		authEpoch    any
		provider     any
		issuer       any
		wantValid    bool
	}{
		{
			name: "login requires complete owner authority", purpose: "login", state: "login-valid",
			requestState: "ready", ownerDID: "did:plc:login", generation: int64(1), authEpoch: int64(2), wantValid: true,
		},
		{
			name: "login rejects ownerless authority", purpose: "login", state: "login-ownerless",
			requestState: "ready", wantValid: false,
		},
		{
			name: "deletion requires complete owner authority", purpose: "accountDeletion", state: "deletion-valid",
			requestState: "ready", ownerDID: "did:plc:deletion", generation: int64(3), authEpoch: int64(4), wantValid: true,
		},
		{
			name: "ready registration is ownerless", purpose: "registration", state: "registration-ownerless",
			requestState: "ready",
			provider:     "https://bsky.social", issuer: "https://auth.bsky.app", wantValid: true,
		},
		{
			name: "ready registration rejects complete authority", purpose: "registration", state: "registration-ready-bound",
			requestState: "ready", ownerDID: "did:plc:registration", generation: int64(5), authEpoch: int64(6),
			provider: "https://pds.registration.test", issuer: "https://auth.registration.test", wantValid: false,
		},
		{
			name: "registration rejects null provider origin", purpose: "registration", state: "registration-null-provider",
			requestState: "ready", issuer: "https://auth.registration.test", wantValid: false,
		},
		{
			name: "registration rejects null issuer", purpose: "registration", state: "registration-null-issuer",
			requestState: "ready", provider: "https://pds.registration.test", wantValid: false,
		},
		{
			name: "registration rejects blank provider origin", purpose: "registration", state: "registration-blank-provider",
			requestState: "ready", provider: "  ", issuer: "https://auth.registration.test", wantValid: false,
		},
		{
			name: "registration rejects blank issuer", purpose: "registration", state: "registration-blank-issuer",
			requestState: "ready", provider: "https://pds.registration.test", issuer: "   ", wantValid: false,
		},
		{
			name: "registration rejects partial authority", purpose: "registration", state: "registration-partial",
			requestState: "exchange_started",
			ownerDID:     "did:plc:registration", provider: "https://bsky.social", issuer: "https://auth.bsky.app", wantValid: false,
		},
		{
			name: "exchanging registration accepts ownerless authority", purpose: "registration", state: "registration-exchanging-ownerless",
			requestState: "exchange_started",
			provider:     "https://pds.registration.test", issuer: "https://auth.registration.test", wantValid: true,
		},
		{
			name: "exchanging registration accepts complete authority", purpose: "registration", state: "registration-bound",
			requestState: "exchange_started",
			ownerDID:     "did:plc:registration", generation: int64(5), authEpoch: int64(6),
			provider: "https://bsky.social", issuer: "https://auth.bsky.app", wantValid: true,
		},
		{
			name: "consumed registration accepts complete authority", purpose: "registration", state: "registration-consumed",
			requestState: "consumed",
			ownerDID:     "did:plc:registration", generation: int64(5), authEpoch: int64(6),
			provider: "https://pds.registration.test", issuer: "https://auth.registration.test", wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deletionOwner := any(nil)
			deletionJob := any(nil)
			exchangeAttempt := any(nil)
			exchangeStartedAt := any(nil)
			consumedAt := any(nil)
			if tt.purpose == "accountDeletion" {
				deletionOwner = tt.ownerDID
				deletionJob = "10000000-0000-0000-0000-000000000001"
			}
			if tt.requestState == "exchange_started" {
				exchangeAttempt = "20000000-0000-0000-0000-000000000001"
				exchangeStartedAt = "2026-08-30T00:00:00Z"
				consumedAt = "2026-08-30T00:00:00Z"
			}
			if tt.requestState == "consumed" {
				consumedAt = "2026-08-30T00:00:00Z"
			}
			_, err := pool.Exec(ctx, `
					INSERT INTO oauth_auth_requests(
						state,data,handoff_mode,device_id,purpose,
						account_deletion_owner_did,account_deletion_job_id,
						owner_did,owner_generation,auth_epoch,request_uri,
						registration_provider_origin,registration_issuer,request_state,
						exchange_attempt_id,exchange_started_at,consumed_at
					) VALUES(
						$1,'{}','verified_link','device-1',$2,$3,$4,$5,$6,$7,$8,$9,$10,
						$11,$12,$13,$14
					)
				`, tt.state, tt.purpose, deletionOwner, deletionJob, tt.ownerDID, tt.generation,
				tt.authEpoch, "urn:request:"+tt.state, tt.provider, tt.issuer, tt.requestState,
				exchangeAttempt, exchangeStartedAt, consumedAt)
			if tt.wantValid && err != nil {
				t.Fatalf("insert valid shape: %v", err)
			}
			if !tt.wantValid && !isCheckViolation(err) {
				t.Fatalf("invalid shape error = %v, want check violation", err)
			}
		})
	}
}

const providerRegistrationMigrationBaseline = `
	CREATE TABLE owner_lifecycles (owner_did TEXT PRIMARY KEY);
	INSERT INTO owner_lifecycles(owner_did) VALUES
		('did:plc:login'), ('did:plc:deletion'), ('did:plc:registration');

	CREATE TABLE oauth_auth_requests (
		state TEXT PRIMARY KEY,
		data JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		handoff_mode TEXT NOT NULL,
		loopback_redirect_uri TEXT,
		purpose TEXT NOT NULL DEFAULT 'login',
		device_id TEXT NOT NULL,
		account_deletion_owner_did TEXT,
		account_deletion_job_id UUID,
		owner_did TEXT NOT NULL REFERENCES owner_lifecycles(owner_did),
		owner_generation BIGINT NOT NULL,
		auth_epoch BIGINT NOT NULL,
		request_uri TEXT NOT NULL UNIQUE,
		request_state TEXT NOT NULL DEFAULT 'ready',
		exchange_attempt_id UUID,
		exchange_started_at TIMESTAMPTZ,
		exchange_finished_at TIMESTAMPTZ,
		consumed_at TIMESTAMPTZ,
		CONSTRAINT oauth_auth_requests_purpose_check
			CHECK (purpose IN ('login', 'accountDeletion')),
		CONSTRAINT oauth_auth_requests_account_deletion_metadata_check
			CHECK (
				(purpose = 'login' AND account_deletion_owner_did IS NULL AND account_deletion_job_id IS NULL)
				OR
				(purpose = 'accountDeletion' AND account_deletion_owner_did IS NOT NULL AND account_deletion_job_id IS NOT NULL)
			),
		CONSTRAINT oauth_auth_requests_authority_check
			CHECK (owner_generation > 0 AND auth_epoch > 0),
		CONSTRAINT oauth_auth_requests_state_check
			CHECK (request_state IN ('ready', 'exchange_started', 'exchange_failed', 'exchange_ambiguous', 'consumed', 'revoked')),
		CONSTRAINT oauth_auth_requests_attempt_shape_check
			CHECK (
				(request_state = 'ready'
					AND exchange_attempt_id IS NULL
					AND exchange_started_at IS NULL
					AND exchange_finished_at IS NULL
					AND consumed_at IS NULL)
				OR
				(request_state = 'exchange_started'
					AND exchange_attempt_id IS NOT NULL
					AND exchange_started_at IS NOT NULL
					AND exchange_finished_at IS NULL
					AND consumed_at IS NOT NULL)
				OR
				(request_state IN ('exchange_failed', 'exchange_ambiguous')
					AND exchange_attempt_id IS NOT NULL
					AND exchange_started_at IS NOT NULL
					AND exchange_finished_at IS NOT NULL
					AND consumed_at IS NOT NULL)
				OR
				(request_state IN ('consumed', 'revoked') AND consumed_at IS NOT NULL)
			)
	);
	CREATE INDEX oauth_auth_requests_owner_epoch_idx
		ON oauth_auth_requests(owner_did, auth_epoch, request_state, created_at);
`
