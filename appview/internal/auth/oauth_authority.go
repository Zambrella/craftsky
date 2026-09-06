package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

var ErrOAuthAuthorityStale = errors.New("OAuth authority is stale")

// OAuthAuthority identifies the resource and issuer currently authorized by a DID.
type OAuthAuthority struct {
	DID          syntax.DID
	PDSOrigin    string
	IssuerOrigin string
}

type OAuthAuthorityVerifier interface {
	ResolveCurrent(context.Context, syntax.DID) (OAuthAuthority, error)
}

func expectedOAuthAuthority(data oauth.ClientSessionData) OAuthAuthority {
	return OAuthAuthority{
		DID:          data.AccountDID,
		PDSOrigin:    data.HostURL,
		IssuerOrigin: data.AuthServerURL,
	}
}

func verifyOAuthAuthority(current, expected OAuthAuthority) error {
	if current.DID != expected.DID {
		return fmt.Errorf("resolved OAuth authority DID mismatch")
	}
	currentPDS, err := canonicalAuthorityURL(current.PDSOrigin)
	if err != nil {
		return fmt.Errorf("current PDS authority: %w", err)
	}
	expectedPDS, err := canonicalAuthorityURL(expected.PDSOrigin)
	if err != nil {
		return fmt.Errorf("expected PDS authority: %w", err)
	}
	currentIssuer, err := canonicalAuthorityURL(current.IssuerOrigin)
	if err != nil {
		return fmt.Errorf("current OAuth issuer authority: %w", err)
	}
	expectedIssuer, err := canonicalAuthorityURL(expected.IssuerOrigin)
	if err != nil {
		return fmt.Errorf("expected OAuth issuer authority: %w", err)
	}
	if currentPDS != expectedPDS || currentIssuer != expectedIssuer {
		return ErrOAuthAuthorityStale
	}
	return nil
}

func canonicalAuthorityURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("invalid authority URL")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimSuffix(parsed.EscapedPath(), "/")
	parsed.RawPath = ""
	return parsed.String(), nil
}
