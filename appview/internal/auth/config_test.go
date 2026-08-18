package auth_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atcrypto"

	"social.craftsky/appview/internal/auth"
)

func mustURL(t *testing.T, raw string) url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return *parsed
}

func TestBuildClientArtifactsProduction(t *testing.T) {
	priv, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:            auth.ClientModeConfidential,
		ClientID:        mustURL(t, "https://appview.craftsky.social/oauth/client-metadata.json"),
		CallbackURL:     mustURL(t, "https://appview.craftsky.social/oauth/callback"),
		JWKSURL:         mustURL(t, "https://appview.craftsky.social/oauth/jwks.json"),
		ClientSecretKey: priv.Multibase(),
		ClientKeyID:     "primary",
		Scopes:          []string{"atproto", "transition:generic"},
	})
	if err != nil {
		t.Fatalf("BuildClientArtifacts: %v", err)
	}
	if !artifacts.Config.IsConfidential() {
		t.Fatal("Config is public, want confidential")
	}
	if artifacts.Metadata.JWKSURI == nil || *artifacts.Metadata.JWKSURI != "https://appview.craftsky.social/oauth/jwks.json" {
		t.Fatalf("metadata jwks_uri = %v", artifacts.Metadata.JWKSURI)
	}
	if err := artifacts.Metadata.Validate(artifacts.Config.ClientID); err != nil {
		t.Fatalf("metadata validation: %v", err)
	}
	if len(artifacts.JWKS.Keys) != 1 || artifacts.JWKS.Keys[0].KeyID == nil || *artifacts.JWKS.Keys[0].KeyID != "primary" {
		t.Fatalf("JWKS = %#v", artifacts.JWKS)
	}
}

func TestBuildClientArtifactsLocalhost(t *testing.T) {
	callback := "http://127.0.0.1:18350/oauth/callback"
	artifacts, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:        auth.ClientModeLocalhost,
		CallbackURL: mustURL(t, callback),
		Scopes:      []string{"atproto"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifacts.Config.IsConfidential() {
		t.Fatal("localhost config should not be confidential")
	}
	if !strings.HasPrefix(artifacts.Config.ClientID, "http://localhost?") {
		t.Fatalf("unexpected client_id: %q", artifacts.Config.ClientID)
	}
	if artifacts.Config.CallbackURL != callback {
		t.Fatalf("callback URL = %q, want %q", artifacts.Config.CallbackURL, callback)
	}
}

func TestBuildClientArtifactsRejectsInvalidConfidentialKey(t *testing.T) {
	_, err := auth.BuildClientArtifacts(auth.ClientConfigInput{
		Mode:            auth.ClientModeConfidential,
		ClientID:        mustURL(t, "https://appview.craftsky.social/oauth/client-metadata.json"),
		CallbackURL:     mustURL(t, "https://appview.craftsky.social/oauth/callback"),
		JWKSURL:         mustURL(t, "https://appview.craftsky.social/oauth/jwks.json"),
		ClientSecretKey: "not-a-private-key",
		ClientKeyID:     "primary",
		Scopes:          []string{"atproto"},
	})
	if err == nil || !strings.Contains(err.Error(), "OAUTH_CLIENT_SECRET_KEY") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildClientArtifactsRejectsAmbiguousTypedURLs(t *testing.T) {
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name  string
		input auth.ClientConfigInput
	}{
		{
			name: "localhost port outside range",
			input: auth.ClientConfigInput{
				Mode:        auth.ClientModeLocalhost,
				CallbackURL: mustURL(t, "http://127.0.0.1:65536/oauth/callback"),
				Scopes:      []string{"atproto"},
			},
		},
		{
			name: "confidential endpoint query",
			input: auth.ClientConfigInput{
				Mode:            auth.ClientModeConfidential,
				ClientID:        mustURL(t, "https://appview.craftsky.social/oauth/client-metadata.json?poison=true"),
				CallbackURL:     mustURL(t, "https://appview.craftsky.social/oauth/callback"),
				JWKSURL:         mustURL(t, "https://appview.craftsky.social/oauth/jwks.json"),
				ClientSecretKey: privateKey.Multibase(),
				ClientKeyID:     "primary",
				Scopes:          []string{"atproto"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := auth.BuildClientArtifacts(test.input); err == nil {
				t.Fatal("BuildClientArtifacts error = nil")
			}
		})
	}
}
