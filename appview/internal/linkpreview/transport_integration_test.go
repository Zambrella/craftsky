package linkpreview

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"testing"
)

// IT-002: allowed sockets are pinned to a validated numeric address while HTTP
// retains the original Host, and forbidden answer sets never reach the dialer.
func TestPinnedTransportDialsValidatedAddress(t *testing.T) {
	t.Parallel()

	t.Run("public answer", func(t *testing.T) {
		resolver := staticResolver{
			"public.example": {netip.MustParseAddr("8.8.8.8")},
		}
		dialer := &pipeDialer{}
		transport := NewPinnedTransport(resolver, dialer)
		client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		fetcher := NewFetcher(resolver, client)
		response, _, err := fetcher.Fetch(context.Background(), "http://public.example/pattern")
		if err != nil {
			t.Fatalf("Fetch(): %v", err)
		}
		response.Body.Close()
		transport.CloseIdleConnections()
		if got := dialer.Addresses(); len(got) != 1 || got[0] != "8.8.8.8:80" {
			t.Fatalf("dial addresses = %v, want [8.8.8.8:80]", got)
		}
		if got := dialer.Hosts(); len(got) != 1 || got[0] != "public.example" {
			t.Fatalf("HTTP hosts = %v, want [public.example]", got)
		}
	})

	for name, answers := range map[string][]netip.Addr{
		"private": {netip.MustParseAddr("127.0.0.1")},
		"mixed": {
			netip.MustParseAddr("8.8.8.8"),
			netip.MustParseAddr("127.0.0.1"),
		},
	} {
		name, answers := name, answers
		t.Run(name, func(t *testing.T) {
			resolver := staticResolver{"blocked.example": answers}
			dialer := &pipeDialer{}
			transport := NewPinnedTransport(resolver, dialer)
			client := &http.Client{Transport: transport}
			fetcher := NewFetcher(resolver, client)
			if _, _, err := fetcher.Fetch(context.Background(), "http://blocked.example/pattern"); !errors.Is(err, ErrNotAllowed) {
				t.Fatalf("Fetch() error = %v, want ErrNotAllowed", err)
			}
			if got := dialer.Addresses(); len(got) != 0 {
				t.Fatalf("forbidden answer dialed %v", got)
			}
		})
	}
}

// IT-003: DNS is checked again at connection time and redirect targets are
// independently admitted before another socket can be opened.
func TestPinnedTransportRevalidatesDNSAndRedirects(t *testing.T) {
	t.Parallel()

	t.Run("DNS changes before dial", func(t *testing.T) {
		resolver := &sequenceResolver{answers: map[string][][]netip.Addr{
			"rebind.example": {
				{netip.MustParseAddr("8.8.8.8")},
				{netip.MustParseAddr("127.0.0.1")},
			},
		}}
		dialer := &pipeDialer{}
		transport := NewPinnedTransport(resolver, dialer)
		fetcher := NewFetcher(resolver, &http.Client{Transport: transport})
		if _, _, err := fetcher.Fetch(context.Background(), "http://rebind.example/start"); !errors.Is(err, ErrNotAllowed) {
			t.Fatalf("Fetch() error = %v, want ErrNotAllowed", err)
		}
		if got := dialer.Addresses(); len(got) != 0 {
			t.Fatalf("rebound destination dialed %v", got)
		}
	})

	t.Run("redirect target", func(t *testing.T) {
		resolver := staticResolver{
			"public.example":  {netip.MustParseAddr("8.8.8.8")},
			"private.example": {netip.MustParseAddr("127.0.0.1")},
		}
		dialer := &responseDialer{locations: []string{"http://private.example/secret"}}
		transport := NewPinnedTransport(resolver, dialer)
		client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}}
		fetcher := NewFetcher(resolver, client)
		if _, _, err := fetcher.Fetch(context.Background(), "http://public.example/start"); !errors.Is(err, ErrNotAllowed) {
			t.Fatalf("Fetch() error = %v, want ErrNotAllowed", err)
		}
		if got := dialer.Addresses(); len(got) != 1 || got[0] != "8.8.8.8:80" {
			t.Fatalf("redirect dials = %v, want only public first hop", got)
		}
	})
}

// IT-019: preview requests use direct pinned egress and a fixed, non-sensitive
// header set regardless of process proxy configuration.
func TestPinnedTransportIgnoresProxyAndForwardsOnlyFixedHeaders(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy-canary.invalid:8080")
	t.Setenv("HTTPS_PROXY", "http://proxy-canary.invalid:8080")

	resolver := staticResolver{"public.example": {netip.MustParseAddr("8.8.8.8")}}
	dialer := &responseDialer{}
	transport := NewPinnedTransport(resolver, dialer)
	if transport.Proxy != nil {
		t.Fatal("preview transport must not consult proxy environment variables")
	}
	client := &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, _, err := NewFetcher(resolver, client).Fetch(context.Background(), "http://public.example/pattern")
	if err != nil {
		t.Fatalf("Fetch(): %v", err)
	}
	response.Body.Close()

	requests := dialer.Requests()
	if len(requests) != 1 {
		t.Fatalf("captured requests = %d, want 1", len(requests))
	}
	request := requests[0]
	if got := request.Header.Get("User-Agent"); got != "CraftskyLinkPreview/1.0 (+https://craftsky.social)" {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := request.Header.Get("Accept"); got != "text/html, application/xhtml+xml" {
		t.Fatalf("Accept = %q", got)
	}
	for _, forbidden := range []string{"Accept-Language", "Authorization", "Cookie", "Referer"} {
		if got := request.Header.Get(forbidden); got != "" {
			t.Fatalf("outbound %s = %q, want absent", forbidden, got)
		}
	}
}

type pipeDialer struct {
	mu        sync.Mutex
	addresses []string
	hosts     []string
}

type sequenceResolver struct {
	mu      sync.Mutex
	answers map[string][][]netip.Addr
}

func (r *sequenceResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	answers := r.answers[host]
	if len(answers) == 0 {
		return nil, errors.New("unconfigured test host")
	}
	answer := answers[0]
	if len(answers) > 1 {
		r.answers[host] = answers[1:]
	}
	return append([]netip.Addr(nil), answer...), nil
}

type responseDialer struct {
	mu        sync.Mutex
	addresses []string
	locations []string
	requests  []*http.Request
}

func (d *responseDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	client, server := net.Pipe()
	d.mu.Lock()
	index := len(d.addresses)
	d.addresses = append(d.addresses, address)
	var location string
	if index < len(d.locations) {
		location = d.locations[index]
	}
	d.mu.Unlock()
	go func() {
		defer server.Close()
		request, err := http.ReadRequest(bufio.NewReader(server))
		if err != nil {
			return
		}
		d.mu.Lock()
		d.requests = append(d.requests, request)
		d.mu.Unlock()
		if location != "" {
			_, _ = server.Write([]byte("HTTP/1.1 302 Found\r\nLocation: " + location + "\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
			return
		}
		_, _ = server.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
	}()
	return client, nil
}

func (d *responseDialer) Addresses() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.addresses...)
}

func (d *responseDialer) Requests() []*http.Request {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]*http.Request(nil), d.requests...)
}

func (d *pipeDialer) DialContext(_ context.Context, _, address string) (net.Conn, error) {
	client, server := net.Pipe()
	d.mu.Lock()
	d.addresses = append(d.addresses, address)
	d.mu.Unlock()
	go func() {
		defer server.Close()
		request, err := http.ReadRequest(bufio.NewReader(server))
		if err != nil {
			return
		}
		d.mu.Lock()
		d.hosts = append(d.hosts, request.Host)
		d.mu.Unlock()
		_, _ = server.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\nConnection: close\r\n\r\n"))
	}()
	return client, nil
}

func (d *pipeDialer) Addresses() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.addresses...)
}

func (d *pipeDialer) Hosts() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.hosts...)
}
