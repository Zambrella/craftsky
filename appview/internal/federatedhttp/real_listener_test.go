package federatedhttp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"
)

type mappedListenerDialer struct {
	target string

	mu        sync.Mutex
	addresses []string
}

func (dialer *mappedListenerDialer) DialContext(
	ctx context.Context,
	network string,
	address string,
) (net.Conn, error) {
	dialer.mu.Lock()
	dialer.addresses = append(dialer.addresses, address)
	dialer.mu.Unlock()
	return (&net.Dialer{}).DialContext(ctx, network, dialer.target)
}

func (dialer *mappedListenerDialer) calls() []string {
	dialer.mu.Lock()
	defer dialer.mu.Unlock()
	return append([]string(nil), dialer.addresses...)
}

type acceptingListener struct {
	net.Listener

	mu       sync.Mutex
	accepted int
	done     chan struct{}
}

func newAcceptingListener(t *testing.T) *acceptingListener {
	t.Helper()
	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener := &acceptingListener{Listener: base, done: make(chan struct{})}
	go func() {
		defer close(listener.done)
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			listener.mu.Lock()
			listener.accepted++
			listener.mu.Unlock()
			_ = connection.Close()
		}
	}()
	t.Cleanup(func() {
		_ = listener.Close()
		select {
		case <-listener.done:
		case <-time.After(time.Second):
			t.Error("instrumented listener goroutine did not stop")
		}
	})
	return listener
}

func (listener *acceptingListener) count() int {
	listener.mu.Lock()
	defer listener.mu.Unlock()
	return listener.accepted
}

func TestRealListenerRejectsPrivateMixedAndReboundDestinationsBeforeDial(t *testing.T) {
	listener := newAcceptingListener(t)

	tests := []struct {
		name     string
		purpose  Purpose
		path     string
		resolver Resolver
	}{
		{
			name:    "private PAR",
			purpose: PurposeOAuthRequest,
			path:    "/oauth/par",
			resolver: staticResolver{
				"target.example": {netip.MustParseAddr("127.0.0.1")},
			},
		},
		{
			name:    "private token",
			purpose: PurposeOAuthRequest,
			path:    "/oauth/token",
			resolver: staticResolver{
				"target.example": {netip.MustParseAddr("127.0.0.1")},
			},
		},
		{
			name:    "private revocation",
			purpose: PurposeOAuthRequest,
			path:    "/oauth/revoke",
			resolver: staticResolver{
				"target.example": {netip.MustParseAddr("127.0.0.1")},
			},
		},
		{
			name:    "private PDS",
			purpose: PurposePDSJSON,
			path:    "/xrpc/com.atproto.repo.getRecord",
			resolver: staticResolver{
				"target.example": {netip.MustParseAddr("127.0.0.1")},
			},
		},
		{
			name:    "mixed DNS",
			purpose: PurposeOAuthRequest,
			path:    "/oauth/token",
			resolver: staticResolver{
				"target.example": {
					netip.MustParseAddr("93.184.216.34"),
					netip.MustParseAddr("127.0.0.1"),
				},
			},
		},
		{
			name:    "DNS rebound",
			purpose: PurposePDSJSON,
			path:    "/xrpc/com.atproto.repo.getRecord",
			resolver: &sequenceResolver{answers: [][]netip.Addr{
				{netip.MustParseAddr("93.184.216.34")},
				{netip.MustParseAddr("127.0.0.1")},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dialer := &mappedListenerDialer{target: listener.Addr().String()}
			boundary, err := NewTestBoundary(
				DefaultTransportProfile(),
				TestNetworkDependencies{Resolver: test.resolver, Dialer: dialer},
			)
			if err != nil {
				t.Fatal(err)
			}
			defer boundary.CloseIdleConnections()
			profile, err := DefaultProfile(test.purpose)
			if err != nil {
				t.Fatal(err)
			}
			profile.TotalTimeout = time.Second
			client, err := boundary.Client(profile)
			if err != nil {
				t.Fatal(err)
			}

			request, err := http.NewRequestWithContext(
				context.Background(), http.MethodGet,
				"https://target.example"+test.path+"?secret=value", nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Do(request)
			if !errors.Is(err, ErrDestinationRejected) || Classify(err) != KindDestinationRejected {
				t.Fatalf("request error = %v, want destination rejection", err)
			}
			var boundaryError *Error
			if !errors.As(err, &boundaryError) {
				t.Fatalf("request error = %v, want typed boundary error", err)
			}
			for _, forbidden := range []string{"target.example", "127.0.0.1", "secret", "value"} {
				if strings.Contains(boundaryError.Error(), forbidden) {
					t.Fatalf("redacted boundary error exposed %q: %v", forbidden, boundaryError)
				}
			}
			if calls := dialer.calls(); len(calls) != 0 {
				t.Fatalf("base dialer calls = %v, want zero", calls)
			}
			if accepted := listener.count(); accepted != 0 {
				t.Fatalf("listener connections = %d, want zero", accepted)
			}
		})
	}
}
