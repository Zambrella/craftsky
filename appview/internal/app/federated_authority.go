package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

type authoritativeOAuthVerifier struct {
	directory identity.Directory
	metadata  oauthAuthorityMetadataResolver
}

func newAuthoritativeOAuthVerifier(
	directory identity.Directory,
	resolver *oauth.Resolver,
) (*authoritativeOAuthVerifier, error) {
	if directory == nil || resolver == nil || resolver.Client == nil {
		return nil, errors.New("authoritative OAuth verifier dependencies are unavailable")
	}
	return newAuthoritativeOAuthVerifierWithMetadata(
		directory,
		&freshOAuthAuthorityMetadataResolver{source: resolver},
	)
}

func newAuthoritativeOAuthVerifierWithMetadata(
	directory identity.Directory,
	metadata oauthAuthorityMetadataResolver,
) (*authoritativeOAuthVerifier, error) {
	if directory == nil || metadata == nil {
		return nil, errors.New("authoritative OAuth verifier dependencies are unavailable")
	}
	return &authoritativeOAuthVerifier{directory: directory, metadata: metadata}, nil
}

func newOAuthAuthorityVerifiers(
	directory identity.Directory,
	source oauthMetadataSource,
	cacheTTL time.Duration,
	cacheCapacity int,
	fetchTimeout time.Duration,
	observer oauthMetadataCacheObserver,
) (*authoritativeOAuthVerifier, *authoritativeOAuthVerifier, error) {
	freshMetadata := &freshOAuthAuthorityMetadataResolver{source: source}
	fresh, err := newAuthoritativeOAuthVerifierWithMetadata(directory, freshMetadata)
	if err != nil {
		return nil, nil, err
	}
	cachedMetadata, err := newCachedOAuthAuthorityMetadataResolver(
		source, cacheTTL, cacheCapacity, fetchTimeout, time.Now, observer,
	)
	if err != nil {
		return nil, nil, err
	}
	operations, err := newAuthoritativeOAuthVerifierWithMetadata(directory, cachedMetadata)
	if err != nil {
		return nil, nil, err
	}
	return fresh, operations, nil
}

func (verifier *authoritativeOAuthVerifier) ResolveCurrent(
	ctx context.Context,
	did syntax.DID,
) (auth.OAuthAuthority, error) {
	if verifier == nil || verifier.directory == nil || verifier.metadata == nil || did == "" {
		return auth.OAuthAuthority{}, errors.New("authoritative OAuth verifier is unavailable")
	}
	resolved, err := verifier.directory.LookupDID(ctx, did)
	if err != nil {
		return auth.OAuthAuthority{}, fmt.Errorf("resolve current OAuth DID authority: %w", err)
	}
	if resolved == nil || resolved.DID != did || resolved.PDSEndpoint() == "" {
		return auth.OAuthAuthority{}, errors.New("resolved current OAuth DID authority is invalid")
	}
	pds := resolved.PDSEndpoint()
	issuer, err := verifier.metadata.ResolveIssuer(ctx, pds)
	if err != nil {
		return auth.OAuthAuthority{}, fmt.Errorf("resolve current OAuth authorization server: %w", err)
	}
	return auth.OAuthAuthority{
		DID: did, PDSOrigin: pds, IssuerOrigin: issuer,
	}, nil
}
