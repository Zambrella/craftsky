package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/auth"
)

func TestCraftskySession_Create_ReturnsTokenAndRow(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 15*time.Minute)
	ctx := context.Background()

	// Seed the oauth_sessions FK row.
	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}

	token, err := store.Create(ctx, "did:plc:a", "s1", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	hash := sha256.Sum256([]byte(token))
	var did, sessID string
	var revokedAt *time.Time
	err = pool.QueryRow(ctx,
		`SELECT account_did, oauth_session_id, revoked_at FROM craftsky_sessions WHERE token_hash = $1`,
		hash[:]).Scan(&did, &sessID, &revokedAt)
	if err != nil {
		t.Fatalf("SELECT craftsky_sessions: %v", err)
	}
	if did != "did:plc:a" {
		t.Errorf("account_did: got %q want %q", did, "did:plc:a")
	}
	if sessID != "s1" {
		t.Errorf("oauth_session_id: got %q want %q", sessID, "s1")
	}
	if revokedAt != nil {
		t.Errorf("revoked_at: expected NULL, got %v", revokedAt)
	}
}

func TestCraftskySession_Lookup_HappyPath(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 15*time.Minute)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}

	token, err := store.Create(ctx, "did:plc:a", "s1", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	info, err := store.Lookup(ctx, token)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if info.DID != "did:plc:a" {
		t.Errorf("DID: got %q want %q", info.DID, "did:plc:a")
	}
	if info.SessionID != "s1" {
		t.Errorf("SessionID: got %q want %q", info.SessionID, "s1")
	}
}

func TestCraftskySession_Lookup_Unknown(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 15*time.Minute)
	ctx := context.Background()

	_, err := store.Lookup(ctx, "never-issued")
	if !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("expected ErrCraftskySessionNotFound, got: %v", err)
	}
}

func TestCraftskySession_Lookup_Revoked(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 15*time.Minute)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}

	token, err := store.Create(ctx, "did:plc:a", "s1", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.Revoke(ctx, token); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	_, lookupErr := store.Lookup(ctx, token)
	if !errors.Is(lookupErr, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("expected ErrCraftskySessionNotFound after revoke, got: %v", lookupErr)
	}
}

func TestCraftskySession_RevokeAll_SetsRevokedAtOnAllRowsForDID(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 15*time.Minute)
	ctx := context.Background()

	// Seed two oauth_sessions rows for the same DID with different session IDs.
	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:b', 'sA', '{}'), ('did:plc:b', 'sB', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}

	token1, err := store.Create(ctx, "did:plc:b", "sA", "")
	if err != nil {
		t.Fatalf("Create token1: %v", err)
	}
	token2, err := store.Create(ctx, "did:plc:b", "sB", "")
	if err != nil {
		t.Fatalf("Create token2: %v", err)
	}

	if err := store.RevokeAll(ctx, "did:plc:b"); err != nil {
		t.Fatalf("RevokeAll: %v", err)
	}

	// Both rows should have non-null revoked_at.
	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM craftsky_sessions WHERE account_did = 'did:plc:b' AND revoked_at IS NOT NULL`,
	).Scan(&count); err != nil {
		t.Fatalf("count revoked: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 revoked rows, got %d", count)
	}

	// Lookup should return not-found for both tokens.
	if _, err := store.Lookup(ctx, token1); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Errorf("token1: expected ErrCraftskySessionNotFound, got: %v", err)
	}
	if _, err := store.Lookup(ctx, token2); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Errorf("token2: expected ErrCraftskySessionNotFound, got: %v", err)
	}
}

func TestCraftskySession_LastSeenThrottled(t *testing.T) {
	pool := withAuthSchema(t)
	// Use a 1-hour throttle so the second Lookup is always within the window.
	store := auth.NewCraftskySessionStore(pool, 1*time.Hour)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:c', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}

	token, err := store.Create(ctx, "did:plc:c", "s1", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Small sleep so the first Lookup's last_seen_at write produces a timestamp
	// measurably different from created_at.
	time.Sleep(5 * time.Millisecond)

	// First Lookup — in-memory map is empty, so maybeTouchLastSeen writes.
	if _, err := store.Lookup(ctx, token); err != nil {
		t.Fatalf("first Lookup: %v", err)
	}

	hash := sha256.Sum256([]byte(token))
	var lastSeen1 time.Time
	if err := pool.QueryRow(ctx,
		`SELECT last_seen_at FROM craftsky_sessions WHERE token_hash = $1`, hash[:],
	).Scan(&lastSeen1); err != nil {
		t.Fatalf("read last_seen_at after first Lookup: %v", err)
	}

	// Second Lookup — still within the 1-hour throttle window; should NOT write.
	if _, err := store.Lookup(ctx, token); err != nil {
		t.Fatalf("second Lookup: %v", err)
	}

	var lastSeen2 time.Time
	if err := pool.QueryRow(ctx,
		`SELECT last_seen_at FROM craftsky_sessions WHERE token_hash = $1`, hash[:],
	).Scan(&lastSeen2); err != nil {
		t.Fatalf("read last_seen_at after second Lookup: %v", err)
	}

	if !lastSeen1.Equal(lastSeen2) {
		t.Errorf("last_seen_at changed on throttled lookup: %v → %v", lastSeen1, lastSeen2)
	}
}

func TestCraftskySession_FKCascadeFromOAuthSessionDelete(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 15*time.Minute)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:d', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}

	if _, err := store.Create(ctx, "did:plc:d", "s1", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Delete the parent oauth_sessions row; the FK ON DELETE CASCADE should
	// remove the craftsky_sessions row automatically.
	_, err = pool.Exec(ctx,
		`DELETE FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
		"did:plc:d", "s1")
	if err != nil {
		t.Fatalf("delete oauth_sessions: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM craftsky_sessions WHERE account_did = $1`, "did:plc:d",
	).Scan(&count); err != nil {
		t.Fatalf("count craftsky_sessions: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 craftsky_sessions rows after FK cascade, got %d", count)
	}
}

func TestCraftskySession_TouchDeviceID_PersistsValue(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 0) // throttle disabled
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}
	if _, err := store.Create(ctx, "did:plc:a", "s1", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.TouchDeviceID(ctx, "did:plc:a", "s1", "dev-xyz"); err != nil {
		t.Fatalf("TouchDeviceID: %v", err)
	}

	var got *string
	err = pool.QueryRow(ctx,
		`SELECT last_device_id FROM craftsky_sessions WHERE account_did = $1`,
		"did:plc:a").Scan(&got)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if got == nil || *got != "dev-xyz" {
		t.Errorf("last_device_id = %v, want dev-xyz", got)
	}
}

func TestCraftskySession_TouchDeviceID_ThrottlesRepeats(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, time.Hour)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:b', 's2', '{}')`)
	if err != nil {
		t.Fatalf("seed oauth_sessions: %v", err)
	}
	if _, err := store.Create(ctx, "did:plc:b", "s2", ""); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.TouchDeviceID(ctx, "did:plc:b", "s2", "dev-first"); err != nil {
		t.Fatalf("TouchDeviceID 1: %v", err)
	}
	if err := store.TouchDeviceID(ctx, "did:plc:b", "s2", "dev-second"); err != nil {
		t.Fatalf("TouchDeviceID 2: %v", err)
	}

	var got *string
	err = pool.QueryRow(ctx,
		`SELECT last_device_id FROM craftsky_sessions WHERE account_did = $1`,
		"did:plc:b").Scan(&got)
	if err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if got == nil || *got != "dev-first" {
		t.Errorf("last_device_id = %v, want dev-first (throttle should have blocked the second write)", got)
	}
}
