package federatedhttp

import (
	"context"
	"errors"
	"net/netip"
	"testing"
)

type staticResolver map[string][]netip.Addr

func (r staticResolver) LookupNetIP(_ context.Context, network, host string) ([]netip.Addr, error) {
	if network != "ip" {
		return nil, errors.New("unexpected network")
	}
	addrs, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	return addrs, nil
}

func TestPolicyValidateOrigin(t *testing.T) {
	t.Parallel()

	policy := NewPolicy(staticResolver{
		"pds.example": {netip.MustParseAddr("93.184.216.34")},
	})

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "canonical public origin", raw: "HTTPS://PDS.EXAMPLE:8443", want: "https://pds.example:8443"},
		{name: "http", raw: "http://pds.example"},
		{name: "userinfo", raw: "https://user:pass@pds.example"},
		{name: "path", raw: "https://pds.example/xrpc"},
		{name: "query", raw: "https://pds.example?target=internal"},
		{name: "fragment", raw: "https://pds.example#fragment"},
		{name: "explicit default port", raw: "https://pds.example:443"},
		{name: "empty port", raw: "https://pds.example:"},
		{name: "trailing dot", raw: "https://pds.example."},
		{name: "unicode host", raw: "https://pds.exämple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := policy.ValidateOrigin(context.Background(), tt.raw)
			if tt.want == "" {
				if !errors.Is(err, ErrDestinationRejected) {
					t.Fatalf("ValidateOrigin(%q) error = %v, want destination rejection", tt.raw, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateOrigin(%q): %v", tt.raw, err)
			}
			if got.String() != tt.want {
				t.Fatalf("ValidateOrigin(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestPolicyValidateOAuthEndpoint(t *testing.T) {
	t.Parallel()

	policy := NewPolicy(staticResolver{
		"as.example":    {netip.MustParseAddr("93.184.216.34")},
		"other.example": {netip.MustParseAddr("93.184.216.35")},
	})

	tests := []struct {
		name     string
		issuer   string
		endpoint string
		want     string
	}{
		{
			name:     "same origin endpoint",
			issuer:   "https://as.example:8443",
			endpoint: "HTTPS://AS.EXAMPLE:8443/oauth/token",
			want:     "https://as.example:8443/oauth/token",
		},
		{name: "different host", issuer: "https://as.example", endpoint: "https://other.example/oauth/token"},
		{name: "different port", issuer: "https://as.example:8443", endpoint: "https://as.example:9443/oauth/token"},
		{name: "endpoint query", issuer: "https://as.example", endpoint: "https://as.example/oauth/token?next=x"},
		{name: "endpoint fragment", issuer: "https://as.example", endpoint: "https://as.example/oauth/token#x"},
		{name: "issuer path", issuer: "https://as.example/issuer", endpoint: "https://as.example/oauth/token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := policy.ValidateOAuthEndpoint(context.Background(), tt.issuer, tt.endpoint)
			if tt.want == "" {
				if !errors.Is(err, ErrDestinationRejected) {
					t.Fatalf("ValidateOAuthEndpoint() error = %v, want destination rejection", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateOAuthEndpoint(): %v", err)
			}
			if got.String() != tt.want {
				t.Fatalf("ValidateOAuthEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPublicAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		address string
		public  bool
	}{
		{address: "8.8.8.8", public: true},
		{address: "1.1.1.1", public: true},
		{address: "2606:4700:4700::1111", public: true},
		{address: "0.0.0.0"},
		{address: "10.0.0.1"},
		{address: "100.64.0.1"},
		{address: "127.0.0.1"},
		{address: "169.254.169.254"},
		{address: "172.16.0.1"},
		{address: "192.0.0.1"},
		{address: "192.0.2.1"},
		{address: "192.168.0.1"},
		{address: "198.18.0.1"},
		{address: "198.51.100.1"},
		{address: "203.0.113.1"},
		{address: "224.0.0.1"},
		{address: "240.0.0.1"},
		{address: "::"},
		{address: "::1"},
		{address: "::ffff:127.0.0.1"},
		{address: "::ffff:169.254.169.254"},
		{address: "64:ff9b::1"},
		{address: "100::1"},
		{address: "2001:db8::1"},
		{address: "2002::1"},
		{address: "3fff::1"},
		{address: "5f00::1"},
		{address: "fc00::1"},
		{address: "fe80::1"},
		{address: "ff00::1"},
	}

	for _, tt := range tests {
		t.Run(tt.address, func(t *testing.T) {
			t.Parallel()
			if got := IsPublicAddress(netip.MustParseAddr(tt.address)); got != tt.public {
				t.Fatalf("IsPublicAddress(%s) = %v, want %v", tt.address, got, tt.public)
			}
		})
	}
}

func TestPolicyRejectsMixedDNSAnswers(t *testing.T) {
	t.Parallel()

	policy := NewPolicy(staticResolver{
		"mixed.example": {
			netip.MustParseAddr("93.184.216.34"),
			netip.MustParseAddr("127.0.0.1"),
		},
	})

	_, err := policy.ValidateOrigin(context.Background(), "https://mixed.example")
	if !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("ValidateOrigin() error = %v, want destination rejection", err)
	}
}
