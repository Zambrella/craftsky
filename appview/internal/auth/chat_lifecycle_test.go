package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/auth"
)

type noActivitySessionLookup interface {
	LookupWithoutActivity(context.Context, string) (auth.AuthInfo, error)
}

type noActivityAuthentication interface {
	AuthenticateWithoutActivity(context.Context, string) (auth.AuthInfo, error)
}

// IT-005 / FR-016, NFR-005 / AC-014, AC-022.
func TestChatAutomaticAuthServicesUseNoActivityLookup(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, time.Minute)
	seedActiveOAuthSession(t, pool, "did:plc:chat-service-owner", "chat-service-parent")
	token, err := store.Create(t.Context(), "did:plc:chat-service-owner", "chat-service-parent", "device")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(t.Context(), `UPDATE craftsky_sessions SET last_seen_at=now()-interval '10 days',idle_expires_at=now()+interval '1 day' WHERE token_hash=$1`, hash[:]); err != nil {
		t.Fatal(err)
	}
	var seen, expiry time.Time
	if err := pool.QueryRow(t.Context(), `SELECT last_seen_at,idle_expires_at FROM craftsky_sessions WHERE token_hash=$1`, hash[:]).Scan(&seen, &expiry); err != nil {
		t.Fatal(err)
	}
	real := &auth.CraftskyAuthService{Store: store}
	for name, service := range map[string]any{"real": real, "stacked": &auth.StackedAuthService{Real: real}} {
		t.Run(name, func(t *testing.T) {
			automatic, ok := service.(noActivityAuthentication)
			if !ok {
				t.Fatal("auth service has no no-activity contract")
			}
			info, err := automatic.AuthenticateWithoutActivity(t.Context(), token)
			if err != nil || info.DID != "did:plc:chat-service-owner" {
				t.Fatalf("automatic auth = %v", err)
			}
			if _, err := automatic.AuthenticateWithoutActivity(t.Context(), "unknown"); !errors.Is(err, auth.ErrAuthTokenInvalid) {
				t.Fatalf("invalid automatic auth = %v", err)
			}
		})
	}
	var afterSeen, afterExpiry time.Time
	if err := pool.QueryRow(t.Context(), `SELECT last_seen_at,idle_expires_at FROM craftsky_sessions WHERE token_hash=$1`, hash[:]).Scan(&afterSeen, &afterExpiry); err != nil {
		t.Fatal(err)
	}
	if !seen.Equal(afterSeen) || !expiry.Equal(afterExpiry) {
		t.Fatal("auth service extended inactivity")
	}
}

// IT-005 / FR-016, NFR-005 / AC-014, AC-022.
func TestChatAutomaticAuthenticationDoesNotExtendInactivity(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, time.Minute)
	seedActiveOAuthSession(t, pool, "did:plc:chat-idle-owner", "chat-idle-parent")
	token, err := store.Create(t.Context(), "did:plc:chat-idle-owner", "chat-idle-parent", "chat-device")
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(token))
	if _, err := pool.Exec(t.Context(), `UPDATE craftsky_sessions SET last_seen_at=now()-interval '10 days',idle_expires_at=now()+interval '1 day' WHERE token_hash=$1`, hash[:]); err != nil {
		t.Fatal(err)
	}
	readActivity := func() (time.Time, time.Time) {
		t.Helper()
		var seen, expiry time.Time
		if err := pool.QueryRow(t.Context(), `SELECT last_seen_at,idle_expires_at FROM craftsky_sessions WHERE token_hash=$1`, hash[:]).Scan(&seen, &expiry); err != nil {
			t.Fatal(err)
		}
		return seen, expiry
	}
	seen, expiry := readActivity()
	lookup, ok := any(store).(noActivitySessionLookup)
	if !ok {
		t.Fatal("automatic authentication has no no-activity lookup")
	}
	for range 3 {
		info, err := lookup.LookupWithoutActivity(t.Context(), token)
		if err != nil || info.DID != "did:plc:chat-idle-owner" || info.SessionID != "chat-idle-parent" {
			t.Fatalf("automatic authentication = %v", err)
		}
	}
	gotSeen, gotExpiry := readActivity()
	if !gotSeen.Equal(seen) || !gotExpiry.Equal(expiry) {
		t.Fatal("automatic authentication extended idle activity")
	}
	if _, err := store.Lookup(t.Context(), token); err != nil {
		t.Fatal(err)
	}
	gotSeen, gotExpiry = readActivity()
	if !gotSeen.After(seen) || !gotExpiry.After(expiry) {
		t.Fatal("explicit user activity no longer extends the normal session")
	}
	if _, err := pool.Exec(t.Context(), `UPDATE craftsky_sessions SET last_seen_at=now()-interval '40 days',idle_expires_at=now()-interval '1 day' WHERE token_hash=$1`, hash[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := lookup.LookupWithoutActivity(t.Context(), token); !errors.Is(err, auth.ErrCraftskySessionNotFound) {
		t.Fatalf("expired automatic authentication = %v", err)
	}
}
