package federatedhttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

func TestOAuthMetadataTransportRejectsCrossOriginEndpoint(t *testing.T) {
	t.Parallel()

	closed := false
	transport := &oauthMetadataTransport{
		next: roundTripFunc(func(*http.Request) (*http.Response, error) {
			body := `{
				"issuer":"https://auth.example",
				"authorization_endpoint":"https://auth.example/authorize",
				"token_endpoint":"https://attacker.example/token",
				"pushed_authorization_request_endpoint":"https://auth.example/par"
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: &trackingReadCloser{
					Reader: strings.NewReader(body),
					closed: &closed,
				},
			}, nil
		}),
		policy: NewPolicy(staticResolver{
			"auth.example":     {netip.MustParseAddr("93.184.216.34")},
			"attacker.example": {netip.MustParseAddr("93.184.216.35")},
		}),
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://auth.example/.well-known/oauth-authorization-server",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err := transport.RoundTrip(request)
	if response != nil {
		t.Fatal("metadata response escaped validation")
	}
	if !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("error = %v, want destination rejection", err)
	}
	if !closed {
		t.Fatal("rejected metadata body was not closed")
	}
}

func TestOAuthMetadataTransportAcceptsSameOriginEndpointsAndRestoresBody(t *testing.T) {
	t.Parallel()

	body := `{
		"issuer":"https://auth.example",
		"authorization_endpoint":"https://auth.example/authorize",
		"token_endpoint":"https://auth.example/token",
		"pushed_authorization_request_endpoint":"https://auth.example/par",
		"revocation_endpoint":"https://auth.example/revoke"
	}`
	transport := &oauthMetadataTransport{
		next: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
		policy: NewPolicy(staticResolver{
			"auth.example": {netip.MustParseAddr("93.184.216.34")},
		}),
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://auth.example/.well-known/oauth-authorization-server",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer response.Body.Close()
	got, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(got) != body {
		t.Fatalf("restored body = %q, want %q", string(got), body)
	}
}

func TestOAuthMetadataTransportRejectsPrivateAuthorizationServer(t *testing.T) {
	t.Parallel()

	transport := &oauthMetadataTransport{
		next: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"authorization_servers":["https://127.0.0.1"]}`,
				)),
			}, nil
		}),
		policy: NewPolicy(staticResolver{
			"pds.example": {netip.MustParseAddr("93.184.216.34")},
		}),
	}
	request, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"https://pds.example/.well-known/oauth-protected-resource",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response, err := transport.RoundTrip(request)
	if response != nil {
		t.Fatal("metadata response escaped validation")
	}
	if !errors.Is(err, ErrDestinationRejected) {
		t.Fatalf("error = %v, want destination rejection", err)
	}
}

type trackingReadCloser struct {
	io.Reader
	closed *bool
}

func (reader *trackingReadCloser) Close() error {
	*reader.closed = true
	return nil
}
