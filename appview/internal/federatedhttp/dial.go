package federatedhttp

import (
	"context"
	"fmt"
	"net"
	"strconv"
)

// Dialer is the narrow connection seam used after DNS validation. A
// *net.Dialer satisfies it.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DialContext returns a net/http-compatible dial function which re-resolves
// the hostname, rejects the entire answer set if any address is non-public,
// and passes only a validated numeric address to the underlying dialer. TLS
// still verifies the original request hostname because this hook changes only
// the socket destination.
func (p *Policy) DialContext(dialer Dialer) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if dialer == nil || network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, rejected(nil)
		}
		host, port, err := net.SplitHostPort(address)
		if err != nil || host == "" || port == "" {
			return nil, rejected(err)
		}
		portNumber, err := strconv.ParseUint(port, 10, 16)
		if err != nil || portNumber == 0 {
			return nil, rejected(err)
		}

		addresses, err := p.resolve(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, resolved := range addresses {
			if !IsPublicAddress(resolved) {
				return nil, rejected(nil)
			}
		}

		var lastErr error
		for _, resolved := range addresses {
			resolved = resolved.Unmap()
			if network == "tcp4" && !resolved.Is4() || network == "tcp6" && !resolved.Is6() {
				continue
			}
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(resolved.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = fmt.Errorf("no address for requested network")
		}
		return nil, failure("", lastErr)
	}
}
