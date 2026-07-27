package auth

import (
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

const defaultLocalhostCallbackURL = "http://127.0.0.1:18080/oauth/callback"

// BuildClientConfig produces an indigo oauth.ClientConfig.
//
//   - hostname == ""               → localhost/public client via NewLocalhostConfig
//     using localhostCallbackURL (scripts/compose-dev aligns it with the
//     checkout's published AppView port).
//   - hostname != "" and key == "" → public client at that hostname (test scenario).
//   - hostname != "" and key != "" → confidential client via SetClientSecret.
//
// The key is multibase-encoded P-256 (matching what `cli oauth-keygen` emits).
func BuildClientConfig(hostname, localhostCallbackURL, clientSecretKey, clientKeyID string, scopes []string) (oauth.ClientConfig, error) {
	if hostname == "" {
		if localhostCallbackURL == "" {
			localhostCallbackURL = defaultLocalhostCallbackURL
		}
		return oauth.NewLocalhostConfig(localhostCallbackURL, scopes), nil
	}
	clientID := fmt.Sprintf("https://%s/oauth/client-metadata.json", hostname)
	callback := fmt.Sprintf("https://%s/oauth/callback", hostname)
	cfg := oauth.NewPublicConfig(clientID, callback, scopes)

	if clientSecretKey == "" {
		return cfg, nil
	}
	priv, err := atcrypto.ParsePrivateMultibase(clientSecretKey)
	if err != nil {
		return oauth.ClientConfig{}, fmt.Errorf("parse OAUTH_CLIENT_SECRET_KEY: %w", err)
	}
	if err := cfg.SetClientSecret(priv, clientKeyID); err != nil {
		return oauth.ClientConfig{}, fmt.Errorf("set client secret: %w", err)
	}
	return cfg, nil
}
