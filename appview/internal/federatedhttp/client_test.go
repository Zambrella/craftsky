package federatedhttp

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"
)

func TestBoundaryClientUsesFinitePurposeProfileAndHardenedTransport(t *testing.T) {
	t.Parallel()
	boundary, err := NewBoundary(DefaultTransportProfile())
	if err != nil {
		t.Fatalf("NewBoundary(): %v", err)
	}

	tests := []struct {
		purpose Purpose
		limit   int64
	}{
		{purpose: PurposeOAuthMetadata, limit: MaxOAuthMetadataResponseBytes},
		{purpose: PurposeOAuthRequest, limit: MaxOAuthResponseBytes},
		{purpose: PurposePDSJSON, limit: MaxPDSJSONResponseBytes},
		{purpose: PurposePDSUpload, limit: MaxPDSUploadResponseBytes},
	}

	for _, tt := range tests {
		t.Run(string(tt.purpose), func(t *testing.T) {
			t.Parallel()
			profile, err := DefaultProfile(tt.purpose)
			if err != nil {
				t.Fatalf("DefaultProfile(): %v", err)
			}
			if profile.TotalTimeout <= 0 || profile.TotalTimeout > 30*time.Second {
				t.Fatalf("TotalTimeout = %v, want finite hard maximum", profile.TotalTimeout)
			}
			if profile.ResponseLimit != tt.limit {
				t.Fatalf("ResponseLimit = %d, want %d", profile.ResponseLimit, tt.limit)
			}

			client, err := boundary.Client(profile)
			if err != nil {
				t.Fatalf("Boundary.Client(): %v", err)
			}
			if client.Timeout != profile.TotalTimeout {
				t.Fatalf("client.Timeout = %v, want %v", client.Timeout, profile.TotalTimeout)
			}
			boundary, ok := client.Transport.(*boundaryTransport)
			if !ok {
				t.Fatalf("client.Transport type = %T, want *boundaryTransport", client.Transport)
			}
			transport, ok := boundary.base.(*http.Transport)
			if !ok {
				t.Fatalf("base transport type = %T, want *http.Transport", boundary.base)
			}
			if transport.Proxy != nil {
				t.Fatal("transport.Proxy must not honor ambient proxy settings")
			}
			if transport.DialContext == nil || transport.TLSHandshakeTimeout <= 0 || transport.ResponseHeaderTimeout <= 0 {
				t.Fatal("transport connection, TLS, and response-header deadlines must be non-zero")
			}
			if transport.IdleConnTimeout <= 0 || transport.ExpectContinueTimeout <= 0 {
				t.Fatal("transport idle and expect-continue deadlines must be non-zero")
			}
			if transport.MaxConnsPerHost <= 0 || transport.MaxIdleConnsPerHost <= 0 || transport.MaxIdleConns <= 0 {
				t.Fatal("transport connection pools must be bounded")
			}
			if transport.MaxResponseHeaderBytes <= 0 {
				t.Fatal("transport response headers must be bounded")
			}
			if transport.TLSClientConfig == nil ||
				transport.TLSClientConfig.InsecureSkipVerify ||
				transport.TLSClientConfig.MinVersion < tls.VersionTLS12 {
				t.Fatal("transport must verify the original TLS hostname at TLS 1.2 or newer")
			}
		})
	}
}

func TestBoundaryUsesItsSharedPolicyForPrevalidation(t *testing.T) {
	t.Parallel()

	boundary, err := newBoundary(
		DefaultTransportProfile(),
		NewPolicy(staticResolver{
			"as.example":      {netip.MustParseAddr("93.184.216.34")},
			"private.example": {netip.MustParseAddr("127.0.0.1")},
		}),
		&recordingDialer{},
	)
	if err != nil {
		t.Fatalf("newBoundary(): %v", err)
	}
	if _, err := boundary.ValidateOrigin(context.Background(), "https://as.example"); err != nil {
		t.Fatalf("ValidateOrigin(public): %v", err)
	}
	if _, err := boundary.ValidateOAuthEndpoint(
		context.Background(),
		"https://as.example",
		"https://as.example/oauth/token",
	); err != nil {
		t.Fatalf("ValidateOAuthEndpoint(public): %v", err)
	}
	if _, err := boundary.ValidateURL(context.Background(), "https://private.example/xrpc"); !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("ValidateURL(private) error = %v, want destination rejection", err)
	}
}

func TestBoundaryNetworkInjectionRetainsDestinationPolicy(t *testing.T) {
	t.Parallel()

	dialer := &recordingDialer{}
	roots := x509.NewCertPool()
	boundary, err := NewTestBoundary(
		DefaultTransportProfile(),
		TestNetworkDependencies{
			Resolver: staticResolver{
				"public.example":  {netip.MustParseAddr("93.184.216.34")},
				"private.example": {netip.MustParseAddr("127.0.0.1")},
			},
			Dialer: dialer, TLSRootCAs: roots,
		},
	)
	if err != nil {
		t.Fatalf("NewTestBoundary(): %v", err)
	}
	defer boundary.CloseIdleConnections()
	if boundary.transport.TLSClientConfig == nil ||
		boundary.transport.TLSClientConfig.RootCAs != roots ||
		boundary.transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("test root injection weakened TLS certificate or hostname verification")
	}

	if _, err := boundary.ValidateURL(
		context.Background(), "https://public.example/xrpc",
	); err != nil {
		t.Fatalf("ValidateURL(public): %v", err)
	}
	if _, err := boundary.ValidateURL(
		context.Background(), "https://private.example/xrpc",
	); !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("ValidateURL(private) error = %v, want destination rejection", err)
	}
	if len(dialer.addresses) != 0 {
		t.Fatalf("prevalidation unexpectedly dialed %v", dialer.addresses)
	}

	if _, err := NewTestBoundary(
		DefaultTransportProfile(), TestNetworkDependencies{Resolver: staticResolver{}},
	); err == nil {
		t.Fatal("NewTestBoundary accepted a missing base dialer")
	}
}

func TestClientRedirectPolicyRevalidatesEveryHop(t *testing.T) {
	t.Parallel()

	profile, err := DefaultProfile(PurposeOAuthMetadata)
	if err != nil {
		t.Fatalf("DefaultProfile(): %v", err)
	}
	profile.MaxRedirects = 1
	policy := NewPolicy(staticResolver{
		"as.example":      {netip.MustParseAddr("93.184.216.34")},
		"other.example":   {netip.MustParseAddr("93.184.216.35")},
		"private.example": {netip.MustParseAddr("127.0.0.1")},
	})
	boundary, err := newBoundary(DefaultTransportProfile(), policy, &recordingDialer{})
	if err != nil {
		t.Fatalf("newBoundary(): %v", err)
	}
	client, err := boundary.Client(profile)
	if err != nil {
		t.Fatalf("Boundary.Client(): %v", err)
	}
	if client.CheckRedirect == nil {
		t.Fatal("CheckRedirect must enforce the federated redirect policy")
	}

	original := &http.Request{URL: mustURL(t, "https://as.example/start")}
	tests := []struct {
		name   string
		target string
		via    []*http.Request
		wantOK bool
	}{
		{name: "same origin", target: "https://as.example/next", via: []*http.Request{original}, wantOK: true},
		{name: "downgrade", target: "http://as.example/next", via: []*http.Request{original}},
		{name: "cross origin", target: "https://other.example/next", via: []*http.Request{original}},
		{name: "private destination", target: "https://private.example/next", via: []*http.Request{original}},
		{name: "redirect limit", target: "https://as.example/final", via: []*http.Request{original, original}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			request := &http.Request{URL: mustURL(t, tt.target)}
			err := client.CheckRedirect(request, tt.via)
			if tt.wantOK {
				if err != nil {
					t.Fatalf("CheckRedirect(): %v", err)
				}
				return
			}
			if !errors.Is(err, ErrRedirectRejected) {
				t.Fatalf("CheckRedirect() error = %v, want redirect rejection", err)
			}
		})
	}
}

func TestBoundarySharesHardenedTransportAcrossPurposeClients(t *testing.T) {
	t.Parallel()

	config := DefaultTransportProfile()
	boundary, err := NewBoundary(config)
	if err != nil {
		t.Fatalf("NewBoundary(): %v", err)
	}
	metadataProfile, err := DefaultProfile(PurposeOAuthMetadata)
	if err != nil {
		t.Fatalf("DefaultProfile(metadata): %v", err)
	}
	pdsProfile, err := DefaultProfile(PurposePDSJSON)
	if err != nil {
		t.Fatalf("DefaultProfile(PDS): %v", err)
	}
	metadataClient, err := boundary.Client(metadataProfile)
	if err != nil {
		t.Fatalf("metadata client: %v", err)
	}
	pdsClient, err := boundary.Client(pdsProfile)
	if err != nil {
		t.Fatalf("PDS client: %v", err)
	}

	metadataTransport := metadataClient.Transport.(*boundaryTransport)
	pdsTransport := pdsClient.Transport.(*boundaryTransport)
	if metadataTransport.base != pdsTransport.base {
		t.Fatal("purpose clients do not share the hardened connection pool")
	}
	if metadataTransport.policy != pdsTransport.policy {
		t.Fatal("purpose clients do not share the destination policy")
	}

	config.DialTimeout = 0
	if _, err := NewBoundary(config); err == nil {
		t.Fatal("NewBoundary() accepted a disabled dial timeout")
	}
}

func TestClientRedirectPolicyPreservesTimeoutClassification(t *testing.T) {
	t.Parallel()

	profile, err := DefaultProfile(PurposePDSJSON)
	if err != nil {
		t.Fatalf("DefaultProfile(): %v", err)
	}
	boundary, err := newBoundary(
		DefaultTransportProfile(),
		NewPolicy(errorResolver{err: context.DeadlineExceeded}),
		&recordingDialer{},
	)
	if err != nil {
		t.Fatalf("newBoundary(): %v", err)
	}
	client, err := boundary.Client(profile)
	if err != nil {
		t.Fatalf("Boundary.Client(): %v", err)
	}
	original := &http.Request{URL: mustURL(t, "https://pds.example/start")}
	request := &http.Request{URL: mustURL(t, "https://pds.example/next")}

	err = client.CheckRedirect(request, []*http.Request{original})
	if got := Classify(err); got != KindTimeout {
		t.Fatalf("Classify(CheckRedirect()) = %q, want %q (error %v)", got, KindTimeout, err)
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	return u
}
