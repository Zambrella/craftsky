package federatedhttp

import (
	"fmt"
	"time"
)

// TransportProfile contains the lowerable connection and pool budgets shared
// by every purpose client in a Boundary.
type TransportProfile struct {
	DialTimeout            time.Duration
	TLSHandshakeTimeout    time.Duration
	ResponseHeaderTimeout  time.Duration
	ExpectContinueTimeout  time.Duration
	IdleConnTimeout        time.Duration
	MaxIdleConns           int
	MaxIdleConnsPerHost    int
	MaxConnsPerHost        int
	MaxResponseHeaderBytes int64
}

// DefaultTransportProfile returns the hard security ceilings used by the
// shared connection pool. Operators may lower, but not raise or disable, them.
func DefaultTransportProfile() TransportProfile {
	return TransportProfile{
		DialTimeout:            5 * time.Second,
		TLSHandshakeTimeout:    5 * time.Second,
		ResponseHeaderTimeout:  10 * time.Second,
		ExpectContinueTimeout:  time.Second,
		IdleConnTimeout:        30 * time.Second,
		MaxIdleConns:           32,
		MaxIdleConnsPerHost:    4,
		MaxConnsPerHost:        8,
		MaxResponseHeaderBytes: 64 << 10,
	}
}

func validateTransportProfile(profile TransportProfile) error {
	maximum := DefaultTransportProfile()
	durations := []struct {
		value time.Duration
		max   time.Duration
	}{
		{value: profile.DialTimeout, max: maximum.DialTimeout},
		{value: profile.TLSHandshakeTimeout, max: maximum.TLSHandshakeTimeout},
		{value: profile.ResponseHeaderTimeout, max: maximum.ResponseHeaderTimeout},
		{value: profile.ExpectContinueTimeout, max: maximum.ExpectContinueTimeout},
		{value: profile.IdleConnTimeout, max: maximum.IdleConnTimeout},
	}
	for _, duration := range durations {
		if duration.value <= 0 || duration.value > duration.max {
			return fmt.Errorf("federated http: transport duration must be positive and at most its hard maximum")
		}
	}
	if profile.MaxIdleConns <= 0 || profile.MaxIdleConns > maximum.MaxIdleConns ||
		profile.MaxIdleConnsPerHost <= 0 || profile.MaxIdleConnsPerHost > maximum.MaxIdleConnsPerHost ||
		profile.MaxConnsPerHost <= 0 || profile.MaxConnsPerHost > maximum.MaxConnsPerHost {
		return fmt.Errorf("federated http: connection pool setting must be positive and at most its hard maximum")
	}
	if profile.MaxIdleConnsPerHost > profile.MaxConnsPerHost || profile.MaxIdleConnsPerHost > profile.MaxIdleConns {
		return fmt.Errorf("federated http: idle connection limits exceed pool limits")
	}
	if profile.MaxResponseHeaderBytes <= 0 || profile.MaxResponseHeaderBytes > maximum.MaxResponseHeaderBytes {
		return fmt.Errorf("federated http: response header limit must be positive and at most its hard maximum")
	}
	return nil
}
