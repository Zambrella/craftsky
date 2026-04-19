package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"social.craftsky/appview/internal/auth"
)

func TestCraftskyAuthService_HappyPath(t *testing.T) {
	pool := withAuthSchema(t)
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`); err != nil {
		t.Fatal(err)
	}
	store := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	token, err := store.Create(context.Background(), "did:plc:a", "s1", "")
	if err != nil {
		t.Fatal(err)
	}
	svc := &auth.CraftskyAuthService{Store: store}
	info, err := svc.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if info.DID != "did:plc:a" || info.SessionID != "s1" {
		t.Fatalf("unexpected: %+v", info)
	}
}

func TestCraftskyAuthService_EmptyToken(t *testing.T) {
	svc := &auth.CraftskyAuthService{Store: nil} // Store not touched on empty
	_, err := svc.Authenticate(context.Background(), "")
	if !errors.Is(err, auth.ErrAuthTokenInvalid) {
		t.Fatalf("want ErrAuthTokenInvalid, got %v", err)
	}
}

func TestCraftskyAuthService_RevokedOrUnknownToken(t *testing.T) {
	pool := withAuthSchema(t)
	store := auth.NewCraftskySessionStore(pool, 5*time.Minute)
	svc := &auth.CraftskyAuthService{Store: store}
	_, err := svc.Authenticate(context.Background(), "never-issued")
	if !errors.Is(err, auth.ErrAuthTokenInvalid) {
		t.Fatalf("want ErrAuthTokenInvalid, got %v", err)
	}
}
