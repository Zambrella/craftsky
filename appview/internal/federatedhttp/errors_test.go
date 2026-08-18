package federatedhttp

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type errorResolver struct {
	err error
}

func TestBoundaryPolicyErrorsCarryPurposeWithoutCallingUpstream(t *testing.T) {
	t.Parallel()

	upstreamCalls := 0
	transport := &boundaryTransport{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			upstreamCalls++
			return nil, errors.New("must not be called")
		}),
		policy: NewPolicy(staticResolver{
			"private.example": {netip.MustParseAddr("127.0.0.1")},
		}),
		purpose:       PurposePDSJSON,
		responseLimit: 1,
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://private.example/xrpc?token=secret", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(): %v", err)
	}
	_, err = transport.RoundTrip(request)
	if !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("RoundTrip() error = %v, want destination rejection", err)
	}
	var boundaryErr *Error
	if !errors.As(err, &boundaryErr) {
		t.Fatalf("RoundTrip() error type = %T, want *Error", err)
	}
	if boundaryErr.Purpose != PurposePDSJSON {
		t.Fatalf("error purpose = %q, want %q", boundaryErr.Purpose, PurposePDSJSON)
	}
	if upstreamCalls != 0 {
		t.Fatalf("upstream calls = %d, want zero", upstreamCalls)
	}
	if strings.Contains(err.Error(), "private.example") || strings.Contains(err.Error(), "token=secret") {
		t.Fatalf("error exposes target details: %q", err)
	}
}

func (r errorResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return nil, r.err
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "remote 192.0.2.1 timed out" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestPolicyClassifiesAndRedactsResolverFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		kind Kind
	}{
		{name: "canceled", err: context.Canceled, kind: KindCanceled},
		{name: "deadline", err: context.DeadlineExceeded, kind: KindTimeout},
		{name: "network timeout", err: timeoutError{}, kind: KindTimeout},
		{name: "upstream", err: errors.New("lookup pds.example returned 192.0.2.1"), kind: KindUpstreamFailure},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			policy := NewPolicy(errorResolver{err: tt.err})
			_, err := policy.ValidateURL(context.Background(), "https://pds.example/xrpc?token=secret")
			if got := Classify(err); got != tt.kind {
				t.Fatalf("Classify() = %q, want %q (error %v)", got, tt.kind, err)
			}
			for _, secret := range []string{"pds.example", "192.0.2.1", "token=secret"} {
				if strings.Contains(err.Error(), secret) {
					t.Fatalf("error %q exposes %q", err, secret)
				}
			}
		})
	}
}

func TestClassifyBoundaryKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err  error
		kind Kind
	}{
		{err: &Error{Kind: KindDestinationRejected}, kind: KindDestinationRejected},
		{err: &Error{Kind: KindRedirectRejected}, kind: KindRedirectRejected},
		{err: &Error{Kind: KindResponseTooLarge}, kind: KindResponseTooLarge},
	}
	for _, tt := range tests {
		if got := Classify(tt.err); got != tt.kind {
			t.Fatalf("Classify(%v) = %q, want %q", tt.err, got, tt.kind)
		}
	}
}
