package auth_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

// withAuthSchema creates a private schema, runs the OAuth DDL inside it,
// and returns a pool scoped to that schema via search_path. Dropped via
// t.Cleanup. Mirrors the withSchema helper in internal/index tests.
//
// IMPORTANT: includes the sibling columns on oauth_auth_requests
// (handoff_mode, loopback_redirect_uri) per Appendix A's decision.
func withAuthSchema(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("TEST_DATABASE_URL and DATABASE_URL both unset")
	}
	ctx := context.Background()
	bootstrap, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("bootstrap pool: %v", err)
	}
	schema := fmt.Sprintf("test_auth_%d", rand.Uint32())
	if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = bootstrap.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		bootstrap.Close()
	})

	ddl := `
		CREATE TABLE ` + schema + `.oauth_sessions (
			account_did TEXT NOT NULL,
			session_id  TEXT NOT NULL,
			data        JSONB NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (account_did, session_id)
		);
		CREATE TABLE ` + schema + `.oauth_auth_requests (
			state                  TEXT NOT NULL PRIMARY KEY,
			data                   JSONB NOT NULL,
			created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
			handoff_mode           TEXT NOT NULL DEFAULT 'deep_link',
			loopback_redirect_uri  TEXT
		);
		CREATE TABLE ` + schema + `.craftsky_sessions (
			token_hash        BYTEA NOT NULL PRIMARY KEY,
			account_did       TEXT NOT NULL,
			oauth_session_id  TEXT NOT NULL,
			device_label      TEXT,
			last_device_id    TEXT,
			created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
			revoked_at        TIMESTAMPTZ,
			FOREIGN KEY (account_did, oauth_session_id)
				REFERENCES ` + schema + `.oauth_sessions (account_did, session_id)
				ON DELETE CASCADE
		);`
	if _, err := bootstrap.Exec(ctx, ddl); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	cfg, _ := pgxpool.ParseConfig(url)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("scoped pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func testStoreConfig() auth.StoreConfig {
	return auth.StoreConfig{
		SessionExpiry:     180 * 24 * time.Hour,
		SessionInactivity: 30 * 24 * time.Hour,
		AuthRequestExpiry: 30 * time.Minute,
		Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
	}
}

func TestStore_SaveGetSession(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	sess := oauth.ClientSessionData{
		AccountDID: syntax.DID("did:plc:abc"),
		SessionID:  "sess-1",
		HostURL:    "https://pds.example.com",
	}
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	got, err := store.GetSession(ctx, sess.AccountDID, sess.SessionID)
	if err != nil {
		t.Fatalf("GetSession: %v", err)
	}
	if got.HostURL != sess.HostURL {
		t.Fatalf("HostURL: got %q want %q", got.HostURL, sess.HostURL)
	}
}

func TestStore_DeleteSession(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	sess := oauth.ClientSessionData{
		AccountDID: syntax.DID("did:plc:del"),
		SessionID:  "sess-del",
		HostURL:    "https://pds.example.com",
	}
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	if err := store.DeleteSession(ctx, sess.AccountDID, sess.SessionID); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	_, err := store.GetSession(ctx, sess.AccountDID, sess.SessionID)
	if err == nil {
		t.Fatal("expected ErrOAuthSessionNotFound after delete, got nil")
	}
	if !isNotFound(err) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", err)
	}
}

func TestStore_SaveGetAuthRequest(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	info := oauth.AuthRequestData{
		State:         "state-abc",
		AuthServerURL: "https://bsky.social",
		PKCEVerifier:  "verifier-xyz",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}
	got, err := store.GetAuthRequestInfo(ctx, info.State)
	if err != nil {
		t.Fatalf("GetAuthRequestInfo: %v", err)
	}
	if got.AuthServerURL != info.AuthServerURL {
		t.Fatalf("AuthServerURL: got %q want %q", got.AuthServerURL, info.AuthServerURL)
	}
	if got.PKCEVerifier != info.PKCEVerifier {
		t.Fatalf("PKCEVerifier: got %q want %q", got.PKCEVerifier, info.PKCEVerifier)
	}
}

func TestStore_DeleteAuthRequest(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	info := oauth.AuthRequestData{
		State:         "state-del",
		AuthServerURL: "https://bsky.social",
	}
	if err := store.SaveAuthRequestInfo(ctx, info); err != nil {
		t.Fatalf("SaveAuthRequestInfo: %v", err)
	}
	if err := store.DeleteAuthRequestInfo(ctx, info.State); err != nil {
		t.Fatalf("DeleteAuthRequestInfo: %v", err)
	}
	_, err := store.GetAuthRequestInfo(ctx, info.State)
	if err == nil {
		t.Fatal("expected ErrOAuthSessionNotFound after delete, got nil")
	}
	if !isNotFound(err) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", err)
	}
}

func TestStore_ExpiredSessionsCleanedUp(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	// Insert directly with a forged created_at well outside the 180-day window.
	_, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions (account_did, session_id, data, created_at, updated_at)
		VALUES ('did:plc:expired', 'sess-expired',
		        '{"account_did":"did:plc:expired","session_id":"sess-expired","host_url":"https://pds.example.com"}'::jsonb,
		        now() - interval '200 days',
		        now() - interval '200 days')
	`)
	if err != nil {
		t.Fatalf("insert expired session: %v", err)
	}

	// GetSession triggers lazy cleanup, then the row should be gone.
	_, getErr := store.GetSession(ctx, syntax.DID("did:plc:expired"), "sess-expired")
	if getErr == nil {
		t.Fatal("expected ErrOAuthSessionNotFound for expired session, got nil")
	}
	if !isNotFound(getErr) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", getErr)
	}

	// Verify the row is actually gone.
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM oauth_sessions WHERE account_did = 'did:plc:expired'`,
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after cleanup, got %d", count)
	}
}

func TestStore_InactiveSessionsCleanedUp(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	// Insert with created_at recent but updated_at beyond the 30-day inactivity window.
	_, err := pool.Exec(ctx, `
		INSERT INTO oauth_sessions (account_did, session_id, data, created_at, updated_at)
		VALUES ('did:plc:inactive', 'sess-inactive',
		        '{"account_did":"did:plc:inactive","session_id":"sess-inactive","host_url":"https://pds.example.com"}'::jsonb,
		        now() - interval '10 days',
		        now() - interval '60 days')
	`)
	if err != nil {
		t.Fatalf("insert inactive session: %v", err)
	}

	_, getErr := store.GetSession(ctx, syntax.DID("did:plc:inactive"), "sess-inactive")
	if getErr == nil {
		t.Fatal("expected ErrOAuthSessionNotFound for inactive session, got nil")
	}
	if !isNotFound(getErr) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", getErr)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM oauth_sessions WHERE account_did = 'did:plc:inactive'`,
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after inactivity cleanup, got %d", count)
	}
}

func TestStore_ExpiredAuthRequestsCleanedUp(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	// Insert directly with a forged created_at beyond the 30-minute window.
	_, err := pool.Exec(ctx, `
		INSERT INTO oauth_auth_requests (state, data, created_at)
		VALUES ('state-expired',
		        '{"state":"state-expired","authserver_url":"https://bsky.social","scopes":[],"request_uri":"","authserver_token_endpoint":"","pkce_verifier":"","dpop_authserver_nonce":"","dpop_privatekey_multibase":""}'::jsonb,
		        now() - interval '60 minutes')
	`)
	if err != nil {
		t.Fatalf("insert expired auth request: %v", err)
	}

	_, getErr := store.GetAuthRequestInfo(ctx, "state-expired")
	if getErr == nil {
		t.Fatal("expected ErrOAuthSessionNotFound for expired auth request, got nil")
	}
	if !isNotFound(getErr) {
		t.Fatalf("expected ErrOAuthSessionNotFound, got: %v", getErr)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM oauth_auth_requests WHERE state = 'state-expired'`,
	).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after auth request cleanup, got %d", count)
	}
}

func TestStore_SaveSessionUpdatesTimestamp(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	ctx := context.Background()

	sess := oauth.ClientSessionData{
		AccountDID: syntax.DID("did:plc:ts"),
		SessionID:  "sess-ts",
		HostURL:    "https://pds.example.com",
	}
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("initial SaveSession: %v", err)
	}

	var before time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updated_at FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		sess.AccountDID.String(), sess.SessionID,
	).Scan(&before); err != nil {
		t.Fatalf("read updated_at before: %v", err)
	}

	// Small sleep to ensure the clock advances enough for the DB timestamp to differ.
	time.Sleep(50 * time.Millisecond)

	// Update the session with a different field value.
	sess.HostURL = "https://pds2.example.com"
	if err := store.SaveSession(ctx, sess); err != nil {
		t.Fatalf("second SaveSession: %v", err)
	}

	var after time.Time
	if err := pool.QueryRow(ctx,
		`SELECT updated_at FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		sess.AccountDID.String(), sess.SessionID,
	).Scan(&after); err != nil {
		t.Fatalf("read updated_at after: %v", err)
	}

	if !after.After(before) {
		t.Fatalf("updated_at did not advance: before=%v after=%v", before, after)
	}
}

// isNotFound is a thin alias around errors.Is for the not-found sentinel,
// kept so the assertion sites read cleanly. The whole point of having
// ErrOAuthSessionNotFound exported is that callers can errors.Is against it —
// don't use string comparison.
func isNotFound(err error) bool {
	return errors.Is(err, auth.ErrOAuthSessionNotFound)
}
