package federatedhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestLimitedBodyReturnsTypedErrorToJSONDecoder(t *testing.T) {
	t.Parallel()

	body := &trackedBody{Reader: strings.NewReader(`{"a":123}`)}
	limited := &limitedReadCloser{
		body:      body,
		remaining: 8,
		purpose:   PurposeOAuthRequest,
	}
	var out map[string]any
	err := json.NewDecoder(limited).Decode(&out)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("Decode() error = %v, want response too large", err)
	}
	if !body.closed {
		t.Fatal("oversized JSON body was not closed")
	}
}

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return nil
}

func TestBoundaryTransportCapsSuccessAndErrorBodies(t *testing.T) {
	t.Parallel()

	const limit = int64(8)
	policy := NewPolicy(staticResolver{
		"pds.example": {netip.MustParseAddr("93.184.216.34")},
	})

	tests := []struct {
		name          string
		status        int
		body          string
		contentLength int64
		wantLarge     bool
	}{
		{name: "success exactly at limit", status: http.StatusOK, body: "12345678", contentLength: 8},
		{name: "success one byte over known length", status: http.StatusOK, body: "123456789", contentLength: 9, wantLarge: true},
		{name: "success one byte over streaming", status: http.StatusOK, body: "123456789", contentLength: -1, wantLarge: true},
		{name: "error one byte over streaming", status: http.StatusBadGateway, body: "123456789", contentLength: -1, wantLarge: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := &trackedBody{Reader: strings.NewReader(tt.body)}
			transport := &boundaryTransport{
				base: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode:    tt.status,
						ContentLength: tt.contentLength,
						Body:          body,
					}, nil
				}),
				policy:        policy,
				purpose:       PurposePDSJSON,
				responseLimit: limit,
			}
			request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://pds.example/xrpc", nil)
			if err != nil {
				t.Fatalf("NewRequestWithContext(): %v", err)
			}

			response, err := transport.RoundTrip(request)
			if err == nil && response != nil {
				_, err = io.ReadAll(response.Body)
				_ = response.Body.Close()
			}
			if tt.wantLarge {
				if !errors.Is(err, ErrResponseTooLarge) {
					t.Fatalf("body error = %v, want response too large", err)
				}
				if !body.closed {
					t.Fatal("oversized response body was not closed")
				}
				return
			}
			if err != nil {
				t.Fatalf("body read: %v", err)
			}
		})
	}
}

func TestBoundaryTransportClosesResponseBodyWhenRoundTripperErrors(t *testing.T) {
	t.Parallel()

	body := &trackedBody{Reader: strings.NewReader("upstream error body")}
	transport := &boundaryTransport{
		base: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusBadGateway, Body: body}, errors.New("network failed")
		}),
		policy: NewPolicy(staticResolver{
			"pds.example": {netip.MustParseAddr("93.184.216.34")},
		}),
		purpose:       PurposePDSJSON,
		responseLimit: 8,
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://pds.example/xrpc", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(): %v", err)
	}

	_, err = transport.RoundTrip(request)
	if err == nil {
		t.Fatal("RoundTrip() error = nil, want upstream failure")
	}
	if !body.closed {
		t.Fatal("response body was not closed when RoundTripper returned an error")
	}
}
