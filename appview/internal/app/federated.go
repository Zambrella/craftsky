package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/identity"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/federatedhttp"
)

type federatedClients struct {
	boundary               *federatedhttp.Boundary
	metadata               *http.Client
	oauth                  *http.Client
	pdsJSON                *http.Client
	pdsBlob                *http.Client
	directory              identity.Directory
	authoritativeDirectory identity.Directory
}

func newFederatedClients(config FederatedHTTPConfig) (*federatedClients, error) {
	boundary, err := federatedhttp.NewBoundary(config.Transport)
	if err != nil {
		return nil, fmt.Errorf("build federated boundary: %w", err)
	}
	return newFederatedClientsWithBoundary(config, boundary)
}

// newFederatedClientsWithBoundary keeps deterministic network injection at
// the composition boundary for listener-backed integration tests. Production
// construction always enters through newFederatedClients above.
func newFederatedClientsWithBoundary(
	config FederatedHTTPConfig,
	boundary *federatedhttp.Boundary,
) (*federatedClients, error) {
	if boundary == nil {
		return nil, fmt.Errorf("build federated boundary: boundary is required")
	}
	metadata, err := boundary.OAuthMetadataClient(config.OAuthMetadata)
	if err != nil {
		return nil, fmt.Errorf("build OAuth metadata client: %w", err)
	}
	oauth, err := boundary.Client(config.OAuthRequest)
	if err != nil {
		return nil, fmt.Errorf("build OAuth request client: %w", err)
	}
	pdsJSON, err := boundary.Client(config.PDSJSON)
	if err != nil {
		return nil, fmt.Errorf("build PDS JSON client: %w", err)
	}
	pdsBlob, err := boundary.Client(config.PDSUpload)
	if err != nil {
		return nil, fmt.Errorf("build PDS upload client: %w", err)
	}

	dnsDialer := &net.Dialer{Timeout: 3 * time.Second}
	baseDirectory := &identity.BaseDirectory{
		PLCURL:     identity.DefaultPLCURL,
		HTTPClient: *metadata,
		Resolver: net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dnsDialer.DialContext(ctx, network, address)
			},
		},
		// Direct authoritative-name-server retry turns an identity string
		// into another outbound destination. Keep resolution on the configured
		// system resolver, whose answer is still checked again by Boundary at
		// each HTTP dial.
		TryAuthoritativeDNS:   false,
		SkipDNSDomainSuffixes: []string{".bsky.social"},
		UserAgent:             "craftsky-appview",
	}
	authoritativeDirectory := identity.Directory(baseDirectory)
	directory := identity.NewCacheDirectory(
		authoritativeDirectory,
		250_000,
		24*time.Hour,
		2*time.Minute,
		5*time.Minute,
	)
	return &federatedClients{
		boundary:  boundary,
		metadata:  metadata,
		oauth:     oauth,
		pdsJSON:   pdsJSON,
		pdsBlob:   pdsBlob,
		directory: directory, authoritativeDirectory: authoritativeDirectory,
	}, nil
}

func (clients *federatedClients) validateSessionDestinations(
	ctx context.Context,
	hostURL string,
	authServerURL string,
	tokenEndpoint string,
	revocationEndpoint string,
) error {
	if clients == nil || clients.boundary == nil {
		return fmt.Errorf("federated clients unavailable")
	}
	if _, err := clients.boundary.ValidateOrigin(ctx, hostURL); err != nil {
		return err
	}
	if _, err := clients.boundary.ValidateOrigin(ctx, authServerURL); err != nil {
		return err
	}
	if _, err := clients.boundary.ValidateOAuthEndpoint(ctx, authServerURL, tokenEndpoint); err != nil {
		return err
	}
	if revocationEndpoint != "" {
		if _, err := clients.boundary.ValidateOAuthEndpoint(ctx, authServerURL, revocationEndpoint); err != nil {
			return err
		}
	}
	return nil
}

// newPDSClient binds an Indigo OAuth session to the purpose-specific PDS
// clients. Token refresh/revocation remains on the OAuth-purpose client while
// record JSON and blob responses use separate bounded clients.
func (clients *federatedClients) newPDSClient(
	ctx context.Context,
	session *oauth.ClientSession,
	onExpired func(context.Context),
) (auth.PDSClient, error) {
	if clients == nil || session == nil || session.Data == nil || session.Config == nil {
		return nil, errors.New("federated PDS session unavailable")
	}
	if err := clients.validateSessionDestinations(
		ctx,
		session.Data.HostURL,
		session.Data.AuthServerURL,
		session.Data.AuthServerTokenEndpoint,
		session.Data.AuthServerRevocationEndpoint,
	); err != nil {
		return nil, err
	}
	session.Client = clients.oauth
	jsonAPI := session.APIClient()
	jsonAPI.Client = clients.pdsJSON
	uploadAPIValue := *jsonAPI
	uploadAPIValue.Client = clients.pdsBlob
	return auth.NewIndigoPDSClient(jsonAPI, &uploadAPIValue, onExpired)
}

// newPendingPDSClient reconstructs the callback's narrowly authorized pending
// parent without calling Indigo's ordinary ResumeSession path, which correctly
// rejects pending_handoff parents. It intentionally has no persistence callback:
// the freshly issued access token is used only for bounded onboarding work and
// cannot silently refresh the pending parent.
func (clients *federatedClients) newPendingPDSClient(
	ctx context.Context,
	config *oauth.ClientConfig,
	data oauth.ClientSessionData,
) (auth.PDSClient, error) {
	if config == nil {
		return nil, errors.New("pending OAuth client configuration unavailable")
	}
	privateKey, err := atcrypto.ParsePrivateMultibase(data.DPoPPrivateKeyMultibase)
	if err != nil {
		return nil, fmt.Errorf("parse pending DPoP key: %w", err)
	}
	session := &oauth.ClientSession{
		Client: clients.oauth, Config: config, Data: &data, DPoPPrivateKey: privateKey,
	}
	return clients.newPDSClient(ctx, session, nil)
}
