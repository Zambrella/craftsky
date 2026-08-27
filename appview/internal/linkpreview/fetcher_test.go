package linkpreview

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"testing"
)

// UT-011: redirects are manual and bounded, every destination is revalidated,
// and only a fragment supplied by a redirect reaches the final URL.
func TestFetcherRedirectPolicyAndFragments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		start        string
		redirects    []string
		wantFinal    string
		wantFail     bool
		wantRequests int
	}{
		{name: "input fragment removed", start: "https://public.example/start#source", wantFinal: "https://public.example/start", wantRequests: 1},
		{name: "redirect without fragment", start: "https://public.example/start#source", redirects: []string{"/next"}, wantFinal: "https://public.example/next", wantRequests: 2},
		{name: "redirect fragment wins", start: "https://public.example/start#source", redirects: []string{"/next#destination"}, wantFinal: "https://public.example/next#destination", wantRequests: 2},
		{name: "five redirects", start: "https://public.example/0", redirects: []string{"/1", "/2", "/3", "/4", "/5"}, wantFinal: "https://public.example/5", wantRequests: 6},
		{name: "six redirects rejected", start: "https://public.example/0", redirects: []string{"/1", "/2", "/3", "/4", "/5", "/6"}, wantFail: true, wantRequests: 6},
		{name: "forbidden next hop", start: "https://public.example/start", redirects: []string{"http://127.0.0.1/private"}, wantFail: true, wantRequests: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doer := &redirectDoer{locations: tt.redirects}
			fetcher := NewFetcher(staticResolver{
				"public.example": {netip.MustParseAddr("8.8.8.8")},
			}, doer)
			response, finalURL, err := fetcher.Fetch(context.Background(), tt.start)
			if tt.wantFail {
				if !errors.Is(err, ErrNotAllowed) {
					t.Fatalf("Fetch() error = %v, want ErrNotAllowed", err)
				}
			} else {
				if err != nil {
					t.Fatalf("Fetch(): %v", err)
				}
				response.Body.Close()
				if got := finalURL.String(); got != tt.wantFinal {
					t.Fatalf("final URL = %q, want %q", got, tt.wantFinal)
				}
			}
			if len(doer.requests) != tt.wantRequests {
				t.Fatalf("request count = %d, want %d", len(doer.requests), tt.wantRequests)
			}
		})
	}
}

type staticResolver map[string][]netip.Addr

func (r staticResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	addresses, ok := r[host]
	if !ok {
		return nil, errors.New("unconfigured test host")
	}
	return append([]netip.Addr(nil), addresses...), nil
}

type redirectDoer struct {
	locations []string
	requests  []string
}

func (d *redirectDoer) Do(request *http.Request) (*http.Response, error) {
	d.requests = append(d.requests, request.URL.String())
	index := len(d.requests) - 1
	if index < len(d.locations) {
		return &http.Response{
			StatusCode: http.StatusFound,
			Header:     http.Header{"Location": {d.locations[index]}},
			Body:       io.NopCloser(&emptyReader{}),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(&emptyReader{}),
	}, nil
}

type emptyReader struct{}

func (*emptyReader) Read([]byte) (int, error) { return 0, io.EOF }
