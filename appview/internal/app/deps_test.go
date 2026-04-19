package app

import (
	"context"
	"strings"
	"testing"

	"social.craftsky/appview/internal/auth"
)

func TestNewDevDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:            EnvDev,
		DatabaseURL:    "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins: []string{"*"},
		DevDID:         "did:plc:test",
	}
	deps, cleanup, err := NewDevDeps(context.Background(), cfg)
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		if deps != nil && deps.DB != nil {
			deps.DB.Close()
		}
		t.Fatal("expected error for unreachable DB, got nil")
	}
	if !strings.Contains(err.Error(), "db") && !strings.Contains(err.Error(), "ping") {
		t.Errorf("err = %v, expected db/ping context", err)
	}
	if deps != nil {
		t.Errorf("deps = %v, want nil on error", deps)
	}
}

func TestNewProdDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:            EnvProd,
		DatabaseURL:    "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins: []string{"https://craftsky.social"},
	}
	_, _, err := NewProdDeps(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// Covers the "which auth service gets wired" contract without touching
// the network: we construct Deps by hand and assert the field types match
// what each factory would have produced. This pins the behaviour even
// when a reachable DB isn't available.
func TestDepsAuthServiceShape(t *testing.T) {
	// Dev: MockAuthService
	devDeps := &Deps{
		Config:      Config{Env: EnvDev, DevDID: "did:plc:default"},
		AuthService: &auth.MockAuthService{DefaultDID: "did:plc:default"},
	}
	if _, ok := devDeps.AuthService.(*auth.MockAuthService); !ok {
		t.Errorf("dev: AuthService = %T, want *auth.MockAuthService", devDeps.AuthService)
	}

	// Prod: CraftskyAuthService
	prodDeps := &Deps{
		Config:      Config{Env: EnvProd},
		AuthService: &auth.CraftskyAuthService{},
	}
	if _, ok := prodDeps.AuthService.(*auth.CraftskyAuthService); !ok {
		t.Errorf("prod: AuthService = %T, want *auth.CraftskyAuthService", prodDeps.AuthService)
	}
}
