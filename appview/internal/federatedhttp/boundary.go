package federatedhttp

import (
	"context"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
)

// Boundary owns the destination policy and reusable connection pool shared by
// all OAuth and PDS clients.
type Boundary struct {
	policy    *Policy
	transport *http.Transport
}

// TestNetworkDependencies is the deterministic-network seam used by
// listener-backed integration tests. Production composition must use
// NewBoundary. URL validation, public-address classification, TLS chain
// validation, and original-hostname verification remain enabled.
type TestNetworkDependencies struct {
	Resolver   Resolver
	Dialer     Dialer
	TLSRootCAs *x509.CertPool
}

// NewBoundary constructs a shared federated HTTP boundary.
func NewBoundary(profile TransportProfile) (*Boundary, error) {
	return newBoundary(profile, NewPolicy(nil), nil)
}

// NewTestBoundary constructs the hardened boundary with explicit test network
// dependencies. Missing resolver/dialer dependencies fail closed.
func NewTestBoundary(
	profile TransportProfile,
	network TestNetworkDependencies,
) (*Boundary, error) {
	if network.Resolver == nil || network.Dialer == nil {
		return nil, fmt.Errorf("federated http: resolver and dialer are required")
	}
	boundary, err := newBoundary(
		profile,
		NewPolicy(network.Resolver),
		network.Dialer,
	)
	if err != nil {
		return nil, err
	}
	if network.TLSRootCAs != nil {
		boundary.transport.TLSClientConfig = boundary.transport.TLSClientConfig.Clone()
		boundary.transport.TLSClientConfig.RootCAs = network.TLSRootCAs
	}
	return boundary, nil
}

func newBoundary(profile TransportProfile, policy *Policy, dialer Dialer) (*Boundary, error) {
	transport, err := newHardenedTransport(profile, policy, dialer)
	if err != nil {
		return nil, err
	}
	return &Boundary{policy: policy, transport: transport}, nil
}

// Client returns a purpose-specific client backed by the Boundary's shared
// destination policy and connection pool.
func (b *Boundary) Client(profile Profile) (*http.Client, error) {
	if err := validateProfile(profile); err != nil {
		return nil, err
	}
	if b == nil || b.policy == nil || b.transport == nil {
		return nil, fmt.Errorf("federated http: boundary is not initialized")
	}
	return &http.Client{
		Transport: &boundaryTransport{
			base:          b.transport,
			policy:        b.policy,
			purpose:       profile.Purpose,
			responseLimit: profile.ResponseLimit,
		},
		Timeout:       profile.TotalTimeout,
		CheckRedirect: checkRedirect(profile, b.policy),
	}, nil
}

// ValidateURL prevalidates a general federated request with the same policy
// used by all clients and at dial time.
func (b *Boundary) ValidateURL(ctx context.Context, raw string) (*url.URL, error) {
	if b == nil || b.policy == nil {
		return nil, &Error{Kind: KindDestinationRejected}
	}
	return b.policy.ValidateURL(ctx, raw)
}

// ValidateOrigin prevalidates a Resource Server or Authorization Server
// origin with the same policy used at dial time.
func (b *Boundary) ValidateOrigin(ctx context.Context, raw string) (*url.URL, error) {
	if b == nil || b.policy == nil {
		return nil, &Error{Kind: KindDestinationRejected}
	}
	return b.policy.ValidateOrigin(ctx, raw)
}

// ValidateOAuthEndpoint binds a metadata-supplied endpoint to its issuer and
// prevalidates its destination using the shared policy.
func (b *Boundary) ValidateOAuthEndpoint(ctx context.Context, issuer, endpoint string) (*url.URL, error) {
	if b == nil || b.policy == nil {
		return nil, &Error{Kind: KindDestinationRejected}
	}
	return b.policy.ValidateOAuthEndpoint(ctx, issuer, endpoint)
}

// CloseIdleConnections releases idle sockets during process shutdown.
func (b *Boundary) CloseIdleConnections() {
	if b != nil && b.transport != nil {
		b.transport.CloseIdleConnections()
	}
}
