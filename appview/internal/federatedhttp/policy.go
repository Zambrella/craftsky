package federatedhttp

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// Resolver is the narrow DNS seam used by the destination policy. A
// *net.Resolver satisfies it.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// Policy validates URL syntax and every resolved destination address.
type Policy struct {
	resolver Resolver
}

// NewPolicy constructs a fail-closed policy. A nil resolver uses the process
// resolver. The injectable seam supports deterministic security tests without
// creating a production bypass for address classification.
func NewPolicy(resolver Resolver) *Policy {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	return &Policy{resolver: resolver}
}

// ValidateOrigin validates an atproto Resource Server or Authorization Server
// origin: HTTPS, no credentials, path, query, fragment, or explicit default
// port. Non-default ports are allowed by the atproto OAuth profile.
func (p *Policy) ValidateOrigin(ctx context.Context, raw string) (*url.URL, error) {
	u, err := parseHTTPSURL(raw)
	if err != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery {
		return nil, rejected(err)
	}
	if err := p.validateDestination(ctx, u.Hostname()); err != nil {
		return nil, err
	}
	return u, nil
}

// ValidateURL validates a general federated request URL. Unlike an origin or
// metadata-supplied OAuth endpoint, an XRPC request may contain both a path and
// query string.
func (p *Policy) ValidateURL(ctx context.Context, raw string) (*url.URL, error) {
	u, err := parseHTTPSURL(raw)
	if err != nil {
		return nil, rejected(err)
	}
	if err := p.validateDestination(ctx, u.Hostname()); err != nil {
		return nil, err
	}
	return u, nil
}

// ValidateOAuthEndpoint validates a metadata-supplied OAuth endpoint and
// requires it to share the issuer's canonical origin. Endpoints may have a
// path, but not a query or fragment.
func (p *Policy) ValidateOAuthEndpoint(ctx context.Context, issuer, endpoint string) (*url.URL, error) {
	issuerURL, err := p.ValidateOrigin(ctx, issuer)
	if err != nil {
		return nil, err
	}
	endpointURL, err := parseHTTPSURL(endpoint)
	if err != nil || endpointURL.RawQuery != "" || endpointURL.ForceQuery {
		return nil, rejected(err)
	}
	if !sameOrigin(issuerURL, endpointURL) {
		return nil, rejected(nil)
	}
	if err := p.validateDestination(ctx, endpointURL.Hostname()); err != nil {
		return nil, err
	}
	return endpointURL, nil
}

func parseHTTPSURL(raw string) (*url.URL, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return nil, ErrDestinationRejected
	}
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Opaque != "" || !strings.EqualFold(u.Scheme, "https") {
		return nil, ErrDestinationRejected
	}
	if u.User != nil || u.Host == "" || u.Hostname() == "" || u.Fragment != "" {
		return nil, ErrDestinationRejected
	}
	if strings.HasSuffix(u.Host, ":") || strings.HasSuffix(u.Hostname(), ".") {
		return nil, ErrDestinationRejected
	}
	hostname := strings.ToLower(u.Hostname())
	if !validASCIIHostname(hostname) {
		return nil, ErrDestinationRejected
	}
	port := u.Port()
	if port == "443" {
		return nil, ErrDestinationRejected
	}
	if port != "" {
		n, err := strconv.ParseUint(port, 10, 16)
		if err != nil || n == 0 {
			return nil, ErrDestinationRejected
		}
	}
	u.Scheme = "https"
	if port == "" {
		if strings.Contains(hostname, ":") {
			u.Host = "[" + hostname + "]"
		} else {
			u.Host = hostname
		}
	} else {
		u.Host = net.JoinHostPort(hostname, port)
	}
	return u, nil
}

func validASCIIHostname(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Zone() == ""
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || !asciiAlphaNumeric(label[0]) || !asciiAlphaNumeric(label[len(label)-1]) {
			return false
		}
		for i := 1; i < len(label)-1; i++ {
			if !asciiAlphaNumeric(label[i]) && label[i] != '-' {
				return false
			}
		}
	}
	return true
}

func asciiAlphaNumeric(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}

func sameOrigin(a, b *url.URL) bool {
	return a.Scheme == b.Scheme && strings.EqualFold(a.Host, b.Host)
}

func (p *Policy) validateDestination(ctx context.Context, host string) error {
	addresses, err := p.resolve(ctx, host)
	if err != nil {
		return err
	}
	for _, address := range addresses {
		if !IsPublicAddress(address) {
			return rejected(nil)
		}
	}
	return nil
}

func (p *Policy) resolve(ctx context.Context, host string) ([]netip.Addr, error) {
	if literal, err := netip.ParseAddr(host); err == nil {
		return []netip.Addr{literal}, nil
	}
	addresses, err := p.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, failure("", err)
	}
	if len(addresses) == 0 {
		return nil, &Error{Kind: KindUpstreamFailure, Cause: fmt.Errorf("empty DNS answer")}
	}
	return addresses, nil
}

func rejected(cause error) error {
	return &Error{Kind: KindDestinationRejected, Cause: cause}
}
