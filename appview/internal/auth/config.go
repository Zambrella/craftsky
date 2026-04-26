package auth

import (
	"fmt"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

// BuildClientConfig produces an indigo oauth.ClientConfig.
//
//   - hostname == ""               → localhost/public client via NewLocalhostConfig
//     (callback http://127.0.0.1:18080/oauth/callback — matches the host
//     port docker-compose.yml publishes the appview on).
//   - hostname != "" and key == "" → public client at that hostname (test scenario).
//   - hostname != "" and key != "" → confidential client via SetClientSecret.
//
// The key is multibase-encoded P-256 (matching what `cli oauth-keygen` emits).
func BuildClientConfig(hostname, clientSecretKey, clientKeyID string, scopes []string) (oauth.ClientConfig, error) {
	if hostname == "" {
		return oauth.NewLocalhostConfig("http://127.0.0.1:18080/oauth/callback", scopes), nil
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
