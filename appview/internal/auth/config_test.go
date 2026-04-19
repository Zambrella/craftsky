package auth_test

import (
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"

	"social.craftsky/appview/internal/auth"
)

func TestBuildClientConfig_Localhost(t *testing.T) {
	cfg, err := auth.BuildClientConfig("", "", "", []string{"atproto"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IsConfidential() {
		t.Fatal("localhost config should not be confidential")
	}
	if !strings.HasPrefix(cfg.ClientID, "http://localhost?") {
		t.Fatalf("unexpected client_id: %q", cfg.ClientID)
	}
}

func TestBuildClientConfig_Confidential(t *testing.T) {
	priv, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	keyMB := priv.Multibase()
	cfg, err := auth.BuildClientConfig("appview.example", keyMB, "primary", []string{"atproto"})
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsConfidential() {
		t.Fatal("expected confidential client")
	}
	expectedID := "https://appview.example/oauth/client-metadata.json"
	if cfg.ClientID != expectedID {
		t.Fatalf("client_id: %q want %q", cfg.ClientID, expectedID)
	}
	jwks := cfg.PublicJWKS()
	if len(jwks.Keys) != 1 {
		t.Fatalf("expected 1 key in JWKS, got %d", len(jwks.Keys))
	}
}
