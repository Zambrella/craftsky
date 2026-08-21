package federatedhttp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

func newHardenedTransport(profile TransportProfile, policy *Policy, dialer Dialer) (*http.Transport, error) {
	if err := validateTransportProfile(profile); err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, fmt.Errorf("federated http: destination policy is required")
	}
	if dialer == nil {
		dialer = &net.Dialer{
			Timeout:   profile.DialTimeout,
			KeepAlive: 30 * time.Second,
		}
	}

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("federated http: default transport has unexpected type")
	}
	transport := defaultTransport.Clone()
	transport.Proxy = nil
	transport.DialContext = policy.DialContext(dialer)
	transport.ForceAttemptHTTP2 = true
	transport.TLSHandshakeTimeout = profile.TLSHandshakeTimeout
	transport.ResponseHeaderTimeout = profile.ResponseHeaderTimeout
	transport.ExpectContinueTimeout = profile.ExpectContinueTimeout
	transport.IdleConnTimeout = profile.IdleConnTimeout
	transport.MaxIdleConns = profile.MaxIdleConns
	transport.MaxIdleConnsPerHost = profile.MaxIdleConnsPerHost
	transport.MaxConnsPerHost = profile.MaxConnsPerHost
	transport.MaxResponseHeaderBytes = profile.MaxResponseHeaderBytes
	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	} else {
		transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		transport.TLSClientConfig.MinVersion = tls.VersionTLS12
	}
	return transport, nil
}
