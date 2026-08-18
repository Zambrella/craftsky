package federatedhttp

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync"
	"testing"
)

type sequenceResolver struct {
	mu      sync.Mutex
	answers [][]netip.Addr
	calls   int
}

func (r *sequenceResolver) LookupNetIP(_ context.Context, _ string, _ string) ([]netip.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	answer := r.answers[r.calls]
	r.calls++
	return answer, nil
}

type recordingDialer struct {
	mu        sync.Mutex
	addresses []string
}

func (d *recordingDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.addresses = append(d.addresses, address)
	return nil, nil
}

func TestPolicyDialContextRejectsDNSRebinding(t *testing.T) {
	t.Parallel()

	resolver := &sequenceResolver{answers: [][]netip.Addr{
		{netip.MustParseAddr("93.184.216.34")},
		{netip.MustParseAddr("127.0.0.1")},
	}}
	dialer := &recordingDialer{}
	policy := NewPolicy(resolver)

	if _, err := policy.ValidateURL(context.Background(), "https://pds.example/xrpc?repo=did%3Aplc%3Aexample"); err != nil {
		t.Fatalf("prevalidate public answer: %v", err)
	}
	_, err := policy.DialContext(dialer)(context.Background(), "tcp", "pds.example:443")
	if !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("DialContext() error = %v, want destination rejection", err)
	}
	if len(dialer.addresses) != 0 {
		t.Fatalf("DialContext() dialed %v after private rebinding", dialer.addresses)
	}
}

func TestPolicyDialContextPinsValidatedAddress(t *testing.T) {
	t.Parallel()

	dialer := &recordingDialer{}
	policy := NewPolicy(staticResolver{
		"pds.example": {netip.MustParseAddr("93.184.216.34")},
	})

	_, err := policy.DialContext(dialer)(context.Background(), "tcp", "pds.example:8443")
	if err != nil {
		t.Fatalf("DialContext(): %v", err)
	}
	if len(dialer.addresses) != 1 || dialer.addresses[0] != "93.184.216.34:8443" {
		t.Fatalf("DialContext() addresses = %v, want pinned public IP", dialer.addresses)
	}
}

func TestPolicyDialContextRejectsMixedAnswerBeforeDial(t *testing.T) {
	t.Parallel()

	dialer := &recordingDialer{}
	policy := NewPolicy(staticResolver{
		"pds.example": {
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("169.254.169.254"),
		},
	})

	_, err := policy.DialContext(dialer)(context.Background(), "tcp", "pds.example:443")
	if !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("DialContext() error = %v, want destination rejection", err)
	}
	if len(dialer.addresses) != 0 {
		t.Fatalf("DialContext() dialed %v for mixed DNS answer", dialer.addresses)
	}
}
