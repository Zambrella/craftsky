package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type ClientAddressResolver struct {
	trustedProxyPrefixes []netip.Prefix
	ipv6PrefixBits       int
}

func NewClientAddressResolver(trustedProxyCIDRs []string, ipv6PrefixBits int) (*ClientAddressResolver, error) {
	if ipv6PrefixBits < 32 || ipv6PrefixBits > 128 {
		return nil, errors.New("HTTP client IPv6 prefix bits must be between 32 and 128")
	}
	resolver := &ClientAddressResolver{ipv6PrefixBits: ipv6PrefixBits}
	for _, raw := range trustedProxyCIDRs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy CIDR: %w", err)
		}
		prefix, err = normalizePrefix(prefix)
		prefix = prefix.Masked()
		if err != nil || prefix.Bits() == 0 || prefix.Addr().IsUnspecified() || prefix.Addr().IsMulticast() {
			return nil, fmt.Errorf("trusted proxy CIDR %q is ambiguous or unsafe", raw)
		}
		resolver.trustedProxyPrefixes = append(resolver.trustedProxyPrefixes, prefix)
	}
	return resolver, nil
}

func (r *ClientAddressResolver) Resolve(request *http.Request) (netip.Prefix, error) {
	if request == nil {
		return netip.Prefix{}, errors.New("HTTP request is required")
	}
	peer, err := parseRemoteAddr(request.RemoteAddr)
	if err != nil {
		return netip.Prefix{}, err
	}
	client := peer
	if r.isTrusted(peer) {
		if forwardedValues := request.Header.Values("Forwarded"); len(forwardedValues) > 0 {
			forwarded := strings.Join(forwardedValues, ",")
			if chain, ok := parseHTTPForwardedChain(forwarded); ok {
				client = r.walkTrustedChain(peer, chain)
			}
		} else if forwardedForValues := request.Header.Values("X-Forwarded-For"); len(forwardedForValues) > 0 {
			forwardedFor := strings.Join(forwardedForValues, ",")
			if chain, ok := parseHTTPXForwardedForChain(forwardedFor); ok {
				client = r.walkTrustedChain(peer, chain)
			}
		}
	}
	return r.limitPrefix(client), nil
}

func (r *ClientAddressResolver) walkTrustedChain(peer netip.Addr, chain []netip.Addr) netip.Addr {
	current := peer
	for index := len(chain) - 1; index >= 0 && r.isTrusted(current); index-- {
		current = chain[index]
	}
	return current
}

func (r *ClientAddressResolver) isTrusted(address netip.Addr) bool {
	address = address.Unmap()
	for _, prefix := range r.trustedProxyPrefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func (r *ClientAddressResolver) limitPrefix(address netip.Addr) netip.Prefix {
	address = address.Unmap()
	if address.Is4() {
		return netip.PrefixFrom(address, 32)
	}
	return netip.PrefixFrom(address, r.ipv6PrefixBits).Masked()
}

func parseRemoteAddr(raw string) (netip.Addr, error) {
	host, _, err := net.SplitHostPort(raw)
	if err != nil {
		host = raw
	}
	address, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parse HTTP peer address: %w", err)
	}
	return address.Unmap(), nil
}

func parseHTTPForwardedChain(raw string) ([]netip.Addr, bool) {
	elements := strings.Split(raw, ",")
	chain := make([]netip.Addr, 0, len(elements))
	for _, element := range elements {
		found := false
		for _, parameter := range strings.Split(element, ";") {
			key, value, ok := strings.Cut(parameter, "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "for") {
				continue
			}
			if found {
				return nil, false
			}
			address, ok := parseForwardedAddress(value)
			if !ok {
				return nil, false
			}
			chain = append(chain, address)
			found = true
		}
		if !found {
			return nil, false
		}
	}
	return chain, len(chain) > 0
}

func parseHTTPXForwardedForChain(raw string) ([]netip.Addr, bool) {
	elements := strings.Split(raw, ",")
	chain := make([]netip.Addr, 0, len(elements))
	for _, element := range elements {
		address, ok := parseForwardedAddress(element)
		if !ok {
			return nil, false
		}
		chain = append(chain, address)
	}
	return chain, len(chain) > 0
}

func parseForwardedAddress(raw string) (netip.Addr, bool) {
	value := strings.TrimSpace(raw)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		value = value[1 : len(value)-1]
	}
	if value == "" || strings.EqualFold(value, "unknown") || strings.HasPrefix(value, "_") {
		return netip.Addr{}, false
	}
	if address, err := netip.ParseAddr(strings.Trim(value, "[]")); err == nil {
		return address.Unmap(), true
	}
	addressPort, err := netip.ParseAddrPort(value)
	if err != nil {
		return netip.Addr{}, false
	}
	return addressPort.Addr().Unmap(), true
}

func normalizePrefix(prefix netip.Prefix) (netip.Prefix, error) {
	if !prefix.IsValid() {
		return netip.Prefix{}, errors.New("invalid prefix")
	}
	if prefix.Addr().Is4In6() {
		if prefix.Bits() < 96 {
			return netip.Prefix{}, errors.New("ambiguous IPv4-mapped prefix")
		}
		return netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96), nil
	}
	return prefix, nil
}
