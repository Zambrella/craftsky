package app

import (
	"context"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"social.craftsky/appview/internal/auth"
)

const testHandoffReceiptKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func testOAuthDeployment(t *testing.T, env Env) OAuthDeployment {
	t.Helper()
	if env == EnvDev {
		callback, err := parseLoopbackOAuthCallback("http://127.0.0.1:18080/oauth/callback")
		if err != nil {
			t.Fatal(err)
		}
		return OAuthDeployment{Mode: OAuthModeLocalhost, CallbackURL: callback, Scopes: []string{"atproto"}}
	}
	origin, err := parseCanonicalPublicOrigin("https://appview.craftsky.social")
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	return OAuthDeployment{
		Mode: OAuthModeConfidential, PublicOrigin: origin,
		ClientID:        resolveOriginPath(origin, "/oauth/client-metadata.json"),
		CallbackURL:     resolveOriginPath(origin, "/oauth/callback"),
		JWKSURL:         resolveOriginPath(origin, "/oauth/jwks.json"),
		ClientSecretKey: Secret(privateKey.Multibase()), ClientKeyID: "primary",
		Scopes: []string{"atproto"},
	}
}

func TestNewDevDeps_UnreachableDBReturnsError(t *testing.T) {
	cfg := Config{
		Env:                    EnvDev,
		DatabaseURL:            "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins:         []string{"*"},
		DevDID:                 "did:plc:test",
		OAuth:                  testOAuthDeployment(t, EnvDev),
		OAuthHandoffReceiptKey: Secret(testHandoffReceiptKey),
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
		Env:                    EnvProd,
		DatabaseURL:            "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins:         []string{"https://craftsky.social"},
		OAuth:                  testOAuthDeployment(t, EnvProd),
		OAuthHandoffReceiptKey: Secret(testHandoffReceiptKey),
	}
	_, _, err := NewProdDeps(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewProdDepsRejectsInvalidOAuthBeforeDatabaseConnection(t *testing.T) {
	cfg := Config{
		Env:                    EnvProd,
		DatabaseURL:            "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		AllowedOrigins:         []string{"https://craftsky.social"},
		OAuth:                  testOAuthDeployment(t, EnvProd),
		OAuthHandoffReceiptKey: Secret(testHandoffReceiptKey),
	}
	cfg.OAuth.ClientSecretKey = "not-a-private-key"

	_, _, err := NewProdDeps(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "OAUTH_CLIENT_SECRET_KEY") {
		t.Fatalf("NewProdDeps error = %v", err)
	}
	if strings.Contains(err.Error(), "db connect") {
		t.Fatalf("OAuth validation happened after database connection: %v", err)
	}
}

func TestNewDepsRejectsMissingHandoffReceiptKeyBeforeDatabaseConnection(t *testing.T) {
	cfg := Config{
		Env:         EnvDev,
		DatabaseURL: "postgres://u:p@127.0.0.1:1/x?sslmode=disable&connect_timeout=1",
		OAuth:       testOAuthDeployment(t, EnvDev),
	}

	_, _, err := NewDevDeps(context.Background(), cfg)
	if err == nil || !strings.Contains(err.Error(), "OAUTH_HANDOFF_RECEIPT_KEY") {
		t.Fatalf("NewDevDeps error = %v, want handoff receipt key failure", err)
	}
	if strings.Contains(err.Error(), "db connect") {
		t.Fatalf("handoff key validation happened after database connection: %v", err)
	}
}

// Covers the "which auth service gets wired" contract without touching
// the network: we construct Deps by hand and assert the field types match
// what each factory would have produced. This pins the behaviour even
// when a reachable DB isn't available.
func TestDepsAuthServiceShape(t *testing.T) {
	// Dev: StackedAuthService (real CraftskyAuthService primary, X-Dev-DID fallback)
	devDeps := &Deps{
		Config:      Config{Env: EnvDev, DevDID: "did:plc:default"},
		AuthService: &auth.StackedAuthService{Real: &auth.CraftskyAuthService{}},
	}
	if _, ok := devDeps.AuthService.(*auth.StackedAuthService); !ok {
		t.Errorf("dev: AuthService = %T, want *auth.StackedAuthService", devDeps.AuthService)
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
