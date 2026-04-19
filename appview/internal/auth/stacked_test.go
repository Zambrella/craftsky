package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/auth"
)

// TestStackedAuth_RealTokenWins verifies that a valid Craftsky bearer
// token resolves via the real service even when X-Dev-DID is also set.
// The real path must take precedence so OAuth-issued tokens never get
// silently overridden by a stray dev header.
func TestStackedAuth_RealTokenWins(t *testing.T) {
	pool := withAuthSchema(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:real', 'oauth-1', '{}')`); err != nil {
		t.Fatal(err)
	}
	craftsky := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	token, err := craftsky.Create(ctx, "did:plc:real", "oauth-1", "")
	if err != nil {
		t.Fatal(err)
	}
	stacked := &auth.StackedAuthService{Real: &auth.CraftskyAuthService{Store: craftsky}}

	// Dev header set, but real token is valid: real wins.
	ctxWithDev := auth.WithDevDID(ctx, "did:plc:dev-shortcut")
	info, err := stacked.Authenticate(ctxWithDev, token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if info.DID != "did:plc:real" || info.SessionID != "oauth-1" {
		t.Fatalf("expected real DID, got %+v", info)
	}
}

// TestStackedAuth_FallsBackToDevDID covers the dev shortcut: invalid
// bearer + X-Dev-DID present → returns the dev DID.
func TestStackedAuth_FallsBackToDevDID(t *testing.T) {
	pool := withAuthSchema(t)
	craftsky := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	stacked := &auth.StackedAuthService{Real: &auth.CraftskyAuthService{Store: craftsky}}

	ctx := auth.WithDevDID(context.Background(), "did:plc:dev-shortcut")
	info, err := stacked.Authenticate(ctx, "any-garbage-token")
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if info.DID != "did:plc:dev-shortcut" {
		t.Fatalf("expected dev shortcut DID, got %q", info.DID)
	}
	if info.SessionID != "" {
		t.Fatalf("expected empty SessionID for dev shortcut, got %q", info.SessionID)
	}
}

// TestStackedAuth_NoDevHeader_BadTokenStaysBad guards the security
// invariant: without an X-Dev-DID header, a random token must not
// authenticate.
func TestStackedAuth_NoDevHeader_BadTokenStaysBad(t *testing.T) {
	pool := withAuthSchema(t)
	craftsky := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	stacked := &auth.StackedAuthService{Real: &auth.CraftskyAuthService{Store: craftsky}}

	_, err := stacked.Authenticate(context.Background(), "random-not-a-real-token")
	if !errors.Is(err, auth.ErrAuthTokenInvalid) {
		t.Fatalf("want ErrAuthTokenInvalid, got %v", err)
	}
}

// TestStackedAuth_EmptyDevDIDHeaderRejects guards against the user
// accidentally setting X-Dev-DID to "" — ParseHeader strips, but if
// somehow an empty string reached the context, we should not treat it
// as a valid identity.
func TestStackedAuth_EmptyDevDIDHeaderRejects(t *testing.T) {
	pool := withAuthSchema(t)
	craftsky := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	stacked := &auth.StackedAuthService{Real: &auth.CraftskyAuthService{Store: craftsky}}

	ctx := auth.WithDevDID(context.Background(), "")
	_, err := stacked.Authenticate(ctx, "garbage")
	if !errors.Is(err, auth.ErrAuthTokenInvalid) {
		t.Fatalf("want ErrAuthTokenInvalid for empty dev header, got %v", err)
	}
}
