package auth

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
)

// ClientMode selects the explicit localhost or confidential OAuth client path.
type ClientMode string

const (
	ClientModeLocalhost    ClientMode = "localhost"
	ClientModeConfidential ClientMode = "confidential"
)

// ClientConfigInput is the validated deployment identity consumed at startup.
type ClientConfigInput struct {
	Mode            ClientMode
	ClientID        url.URL
	CallbackURL     url.URL
	JWKSURL         url.URL
	ClientSecretKey string
	ClientKeyID     string
	Scopes          []string
}

// ClientArtifacts contains one OAuth runtime config and its fixed discovery
// documents. Handlers serialize these snapshots without consulting a request.
type ClientArtifacts struct {
	Config   oauth.ClientConfig
	Metadata oauth.ClientMetadata
	JWKS     oauth.JWKS
}

// BuildClientArtifacts constructs and validates every OAuth discovery
// artifact before the HTTP listener or background workers start.
func BuildClientArtifacts(input ClientConfigInput) (ClientArtifacts, error) {
	if err := validateClientScopes(input.Scopes); err != nil {
		return ClientArtifacts{}, err
	}
	var cfg oauth.ClientConfig
	switch input.Mode {
	case ClientModeLocalhost:
		if err := validateLocalhostCallback(input.CallbackURL); err != nil {
			return ClientArtifacts{}, err
		}
		if input.ClientSecretKey != "" || input.ClientKeyID != "" || input.ClientID.String() != "" || input.JWKSURL.String() != "" {
			return ClientArtifacts{}, errors.New("localhost OAuth mode cannot configure confidential-client fields")
		}
		cfg = oauth.NewLocalhostConfig(input.CallbackURL.String(), append([]string(nil), input.Scopes...))
	case ClientModeConfidential:
		if err := validateConfidentialURLs(input); err != nil {
			return ClientArtifacts{}, err
		}
		if input.ClientSecretKey == "" {
			return ClientArtifacts{}, errors.New("OAUTH_CLIENT_SECRET_KEY is required for confidential OAuth")
		}
		if strings.TrimSpace(input.ClientKeyID) == "" || strings.TrimSpace(input.ClientKeyID) != input.ClientKeyID {
			return ClientArtifacts{}, errors.New("OAUTH_CLIENT_SECRET_KEY_ID is required for confidential OAuth")
		}
		priv, err := atcrypto.ParsePrivateMultibase(input.ClientSecretKey)
		if err != nil {
			return ClientArtifacts{}, errors.New("OAUTH_CLIENT_SECRET_KEY is not a valid P-256 private key")
		}
		cfg = oauth.NewPublicConfig(input.ClientID.String(), input.CallbackURL.String(), append([]string(nil), input.Scopes...))
		if err := cfg.SetClientSecret(priv, input.ClientKeyID); err != nil {
			return ClientArtifacts{}, errors.New("OAUTH_CLIENT_SECRET_KEY must be a P-256 private key")
		}
	default:
		return ClientArtifacts{}, fmt.Errorf("unknown OAuth client mode %q", input.Mode)
	}

	metadata := cfg.ClientMetadata()
	if input.Mode == ClientModeConfidential {
		jwksURL := input.JWKSURL.String()
		metadata.JWKSURI = &jwksURL
	}
	if err := metadata.Validate(cfg.ClientID); err != nil {
		return ClientArtifacts{}, fmt.Errorf("validate OAuth client metadata: %w", err)
	}
	return ClientArtifacts{Config: cfg, Metadata: metadata, JWKS: cfg.PublicJWKS()}, nil
}

func validateClientScopes(scopes []string) error {
	seen := make(map[string]struct{}, len(scopes))
	hasATProto := false
	for _, scope := range scopes {
		if scope == "" || strings.ContainsAny(scope, " \t\r\n") {
			return errors.New("OAUTH_SCOPES contains an invalid scope")
		}
		for _, char := range scope {
			if char < 0x21 || char > 0x7e {
				return errors.New("OAUTH_SCOPES contains an invalid scope")
			}
		}
		if _, exists := seen[scope]; exists {
			return errors.New("OAUTH_SCOPES contains a duplicate scope")
		}
		seen[scope] = struct{}{}
		hasATProto = hasATProto || scope == "atproto"
	}
	if !hasATProto {
		return errors.New("OAUTH_SCOPES must include atproto")
	}
	return nil
}

func validateLocalhostCallback(callback url.URL) error {
	port, err := strconv.Atoi(callback.Port())
	if err != nil || port < 1 || port > 65535 || callback.Scheme != "http" ||
		callback.Hostname() != "127.0.0.1" || callback.Path != "/oauth/callback" ||
		callback.Opaque != "" || callback.User != nil || callback.RawPath != "" ||
		callback.RawQuery != "" || callback.ForceQuery || callback.Fragment != "" {
		return errors.New("OAUTH_CALLBACK_URL is not a strict loopback callback")
	}
	return nil
}

func validateConfidentialURLs(input ClientConfigInput) error {
	if !isExactConfidentialEndpoint(input.ClientID, "/oauth/client-metadata.json") ||
		!isExactConfidentialEndpoint(input.CallbackURL, "/oauth/callback") ||
		!isExactConfidentialEndpoint(input.JWKSURL, "/oauth/jwks.json") ||
		input.ClientID.Host != input.CallbackURL.Host || input.ClientID.Host != input.JWKSURL.Host {
		return errors.New("confidential OAuth URLs must share the canonical HTTPS origin")
	}
	return nil
}

func isExactConfidentialEndpoint(endpoint url.URL, path string) bool {
	return endpoint.Scheme == "https" && endpoint.Host != "" && endpoint.Port() == "" &&
		endpoint.Host == strings.ToLower(endpoint.Host) && endpoint.Path == path &&
		endpoint.Opaque == "" && endpoint.User == nil && endpoint.RawPath == "" &&
		endpoint.RawQuery == "" && !endpoint.ForceQuery && endpoint.Fragment == ""
}
