package middleware

import (
	"net/http/httptest"
	"testing"
)

func TestClientAddressResolverTrustsOnlyKnownProxyChain(t *testing.T) {
	t.Parallel()

	resolver, err := NewClientAddressResolver([]string{"10.0.0.0/8"}, 64)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		remoteAddr string
		forwarded  string
		xff        string
		want       string
	}{
		{
			name:       "untrusted socket ignores spoofed forwarding",
			remoteAddr: "203.0.113.4:1234",
			xff:        "198.51.100.8",
			want:       "203.0.113.4/32",
		},
		{
			name:       "walks trusted XFF peers from the edge",
			remoteAddr: "10.0.0.3:443",
			xff:        "198.51.100.8, 10.0.0.2",
			want:       "198.51.100.8/32",
		},
		{
			name:       "stops at first untrusted XFF peer",
			remoteAddr: "10.0.0.3:443",
			xff:        "192.0.2.9, 198.51.100.8, 10.0.0.2",
			want:       "198.51.100.8/32",
		},
		{
			name:       "malformed XFF fails closed to socket peer",
			remoteAddr: "10.0.0.3:443",
			xff:        "198.51.100.8, definitely-not-an-address",
			want:       "10.0.0.3/32",
		},
		{
			name:       "Forwarded takes precedence and supports quoted host port",
			remoteAddr: "10.0.0.3:443",
			forwarded:  `for=198.51.100.9, for="10.0.0.2:8443"`,
			xff:        "192.0.2.1",
			want:       "198.51.100.9/32",
		},
		{
			name:       "IPv4 mapped socket is normalized",
			remoteAddr: "[::ffff:203.0.113.7]:80",
			want:       "203.0.113.7/32",
		},
		{
			name:       "IPv6 clients share configured prefix",
			remoteAddr: "[2001:db8:abcd:1234::99]:80",
			want:       "2001:db8:abcd:1234::/64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest("GET", "http://appview.test/v1/whoami", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwarded != "" {
				req.Header.Set("Forwarded", tt.forwarded)
			}
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			got, err := resolver.Resolve(req)
			if err != nil {
				t.Fatal(err)
			}
			if got.String() != tt.want {
				t.Fatalf("Resolve() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestClientAddressResolverRejectsAmbiguousTrustConfiguration(t *testing.T) {
	t.Parallel()

	for _, cidr := range []string{"0.0.0.0/0", "::/0", "0.0.0.1/1", "::1/1", "not-a-prefix", "224.0.0.0/4"} {
		t.Run(cidr, func(t *testing.T) {
			t.Parallel()
			if _, err := NewClientAddressResolver([]string{cidr}, 64); err == nil {
				t.Fatalf("NewClientAddressResolver(%q) error = nil", cidr)
			}
		})
	}
}

func TestClientAddressResolverCombinesRepeatedForwardingHeaders(t *testing.T) {
	t.Parallel()

	resolver, err := NewClientAddressResolver([]string{"10.0.0.0/8"}, 64)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "http://appview.test/v1/whoami", nil)
	request.RemoteAddr = "10.0.0.3:443"
	request.Header.Add("X-Forwarded-For", "192.0.2.9")
	request.Header.Add("X-Forwarded-For", "198.51.100.8, 10.0.0.2")

	got, err := resolver.Resolve(request)
	if err != nil {
		t.Fatal(err)
	}
	if got.String() != "198.51.100.8/32" {
		t.Fatalf("Resolve() = %s, want first untrusted hop from the combined chain", got)
	}
}
