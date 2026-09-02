package business

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestBusinessDestinationProcessingDoesNotFetchOrResolve(t *testing.T) {
	transport := &failFastCountingTransport{}
	resolver := &failFastCountingResolver{}
	originalTransport := http.DefaultTransport
	originalResolver := net.DefaultResolver
	http.DefaultTransport = transport
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial:     resolver.dial,
	}
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
		net.DefaultResolver = originalResolver
	})

	webDestinations := []string{
		"https://products.example/item?color=blue#buy",
		"HTTPS://xn--bcher-kva.example:65535/event",
		"http://products.example/item",
		"custom://products.example/item",
		"https://user:password@products.example/item",
		"https:///hostless",
		"https://localhost/item",
		"https://127.0.0.1/item",
		"https://bücher.example/item",
	}
	for _, destination := range webDestinations {
		_ = ValidateWebDestination(destination)
	}
	mailDestinations := []string{
		"mailto:First.Last+orders@EXAMPLE.COM",
		"MAILTO:person@example.com",
		"mailto:person name@example.com",
		"mailto:person%2Bshop@example.com",
		"mailto:person@example.com?subject=private",
		"mailto:one@example.com,two@example.com",
	}
	for _, destination := range mailDestinations {
		_ = ValidateMailtoDestination(destination)
	}

	for _, action := range []Action{
		{Type: "visit-website", Destination: webDestinations[0]},
		{Type: "email", Destination: mailDestinations[0]},
	} {
		if err := ValidateActions([]Action{action}); err != nil {
			t.Fatalf("validate action catalog: %v", err)
		}
	}
	if err := ValidateEventMedia(EventWrite{
		EventURI: webDestinations[0], RegistrationURI: webDestinations[1],
	}); err != nil {
		t.Fatalf("validate event destinations: %v", err)
	}
	_ = ValidateEventMedia(EventWrite{EventURI: webDestinations[2], RegistrationURI: webDestinations[3]})
	_ = HydrateIndependentEventDestinations(webDestinations[0], webDestinations[4])

	profileRaw, err := json.Marshal(map[string]any{
		"primaryAction": map[string]any{"type": "email", "destination": mailDestinations[0]},
		"products": []map[string]any{
			{"title": "safe product", "uri": webDestinations[0]},
			{"title": "unsafe product", "uri": webDestinations[4]},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HydrateProfile(profileRaw); err != nil {
		t.Fatalf("hydrate profile destinations: %v", err)
	}

	eventRaw, err := json.Marshal(map[string]any{
		"name": "Destination safety event", "startsAt": "2026-09-02T10:00:00Z",
		"endsAt": "2026-09-02T18:00:00Z", "createdAt": "2026-09-01T12:00:00Z",
		"roles": []string{"vendor"}, "eventUri": webDestinations[0],
		"registrationUri": webDestinations[4],
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := HydrateEvent(eventRaw); err != nil {
		t.Fatalf("hydrate event destinations: %v", err)
	}

	if calls := transport.calls.Load(); calls != 0 {
		t.Fatalf("business destination processing made %d default HTTP transport calls, want zero", calls)
	}
	if calls := resolver.calls.Load(); calls != 0 {
		t.Fatalf("business destination processing made %d default DNS resolver calls, want zero", calls)
	}
}

type failFastCountingTransport struct {
	calls atomic.Int64
}

func (transport *failFastCountingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls.Add(1)
	return nil, errors.New("unexpected business destination HTTP fetch")
}

type failFastCountingResolver struct {
	calls atomic.Int64
}

func (resolver *failFastCountingResolver) dial(context.Context, string, string) (net.Conn, error) {
	resolver.calls.Add(1)
	return nil, errors.New("unexpected business destination DNS resolution")
}
