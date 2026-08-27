package linkpreview

import (
	"errors"
	"testing"
)

// UT-009: unsafe syntax and non-public literals are rejected before the
// resolver or dialer can become involved.
func TestValidateURLSyntax(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		wantURL  string
		wantFail bool
	}{
		{name: "HTTPS", raw: "https://Example.COM/pattern?q=1", wantURL: "https://example.com/pattern?q=1"},
		{name: "HTTP", raw: "http://example.com:80/pattern", wantURL: "http://example.com/pattern"},
		{name: "explicit HTTPS default port", raw: "https://example.com:443/pattern", wantURL: "https://example.com/pattern"},
		{name: "public IPv4 literal", raw: "https://93.184.216.34/pattern", wantURL: "https://93.184.216.34/pattern"},
		{name: "public IPv6 literal", raw: "https://[2606:4700:4700::1111]/pattern", wantURL: "https://[2606:4700:4700::1111]/pattern"},
		{name: "input fragment ignored", raw: "https://example.com/pattern#section", wantURL: "https://example.com/pattern"},
		{name: "FTP", raw: "ftp://example.com/pattern", wantFail: true},
		{name: "file", raw: "file:///etc/passwd", wantFail: true},
		{name: "data", raw: "data:text/plain,hello", wantFail: true},
		{name: "userinfo", raw: "https://member:secret@example.com/pattern", wantFail: true},
		{name: "nonstandard HTTP port", raw: "http://example.com:8080/pattern", wantFail: true},
		{name: "nonstandard HTTPS port", raw: "https://example.com:8443/pattern", wantFail: true},
		{name: "empty host", raw: "https:///pattern", wantFail: true},
		{name: "malformed host", raw: "https://bad_host.example/pattern", wantFail: true},
		{name: "loopback IPv4", raw: "https://127.0.0.1/pattern", wantFail: true},
		{name: "private IPv4", raw: "https://10.0.0.1/pattern", wantFail: true},
		{name: "link-local IPv4", raw: "https://169.254.1.1/pattern", wantFail: true},
		{name: "loopback IPv6", raw: "https://[::1]/pattern", wantFail: true},
		{name: "mapped loopback IPv6", raw: "https://[::ffff:127.0.0.1]/pattern", wantFail: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateURL(tt.raw)
			if tt.wantFail {
				if !errors.Is(err, ErrNotAllowed) {
					t.Fatalf("ValidateURL(%q) error = %v, want ErrNotAllowed", tt.raw, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateURL(%q): %v", tt.raw, err)
			}
			if got.String() != tt.wantURL {
				t.Fatalf("ValidateURL(%q) = %q, want %q", tt.raw, got, tt.wantURL)
			}
		})
	}
}
