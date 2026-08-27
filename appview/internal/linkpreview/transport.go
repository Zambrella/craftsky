package linkpreview

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var ErrNotAllowed = errors.New("link preview destination is not allowed")

type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// NewPinnedTransport constructs a direct transport whose socket hook resolves,
// validates, and pins each connection without consulting ambient proxy config.
func NewPinnedTransport(resolver Resolver, dialer Dialer) *http.Transport {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	}
	return &http.Transport{
		Proxy:                  nil,
		DialContext:            pinnedDialContext(resolver, dialer),
		ForceAttemptHTTP2:      true,
		DisableCompression:     true,
		MaxIdleConns:           16,
		MaxIdleConnsPerHost:    2,
		MaxConnsPerHost:        2,
		IdleConnTimeout:        30 * time.Second,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  6 * time.Second,
		ExpectContinueTimeout:  time.Second,
		MaxResponseHeaderBytes: 64 << 10,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}
}

func pinnedDialContext(resolver Resolver, dialer Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, ErrNotAllowed
		}
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "80" && port != "443" {
			return nil, ErrNotAllowed
		}
		var addresses []netip.Addr
		if literal, err := netip.ParseAddr(host); err == nil {
			addresses = []netip.Addr{literal}
		} else {
			addresses, err = resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
		}
		validated, err := ValidateAddresses(addresses)
		if err != nil {
			return nil, err
		}
		var lastErr error
		for _, resolved := range validated {
			resolved = resolved.Unmap()
			if network == "tcp4" && !resolved.Is4() || network == "tcp6" && !resolved.Is6() {
				continue
			}
			connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.String(), port))
			if err == nil {
				return connection, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = ErrNotAllowed
		}
		return nil, lastErr
	}
}

// ValidateURL applies destination syntax policy before any DNS resolution or
// dialing. Input fragments are intentionally discarded from transport URLs.
func ValidateURL(raw string) (*url.URL, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return nil, ErrNotAllowed
	}
	u, err := url.Parse(raw)
	if err != nil || !u.IsAbs() || u.Opaque != "" || u.User != nil {
		return nil, ErrNotAllowed
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrNotAllowed
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.HasSuffix(host, ".") || !validHostname(host) {
		return nil, ErrNotAllowed
	}
	port := u.Port()
	if strings.HasSuffix(u.Host, ":") || port != "" &&
		(u.Scheme == "http" && port != "80" || u.Scheme == "https" && port != "443") {
		return nil, ErrNotAllowed
	}
	if literal, err := netip.ParseAddr(host); err == nil {
		if literal.Zone() != "" || !isPublicAddress(literal) {
			return nil, ErrNotAllowed
		}
		host = literal.Unmap().String()
	}
	if port == "" || u.Scheme == "http" && port == "80" || u.Scheme == "https" && port == "443" {
		if strings.Contains(host, ":") {
			u.Host = "[" + host + "]"
		} else {
			u.Host = host
		}
	} else {
		u.Host = net.JoinHostPort(host, port)
	}
	u.Fragment = ""
	u.RawFragment = ""
	return u, nil
}

func validHostname(host string) bool {
	if len(host) > 253 {
		return false
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return true
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

func asciiAlphaNumeric(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= '0' && value <= '9'
}
