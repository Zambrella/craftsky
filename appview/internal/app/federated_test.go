package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/federatedhttp"
)

func TestFederatedHTTPSHandleLookupPreservesFailuresAndVerifiesIdentity(t *testing.T) {
	t.Parallel()
	const handle syntax.Handle = "sample.bsky.social"
	const did syntax.DID = "did:plc:aaaaaaaaaaaaaaaaaaaaaaaa"
	for _, test := range []struct {
		name           string
		handleStatus   int
		declaredHandle string
		wantError      error
	}{
		{"missing handle", http.StatusNotFound, "", identity.ErrHandleNotFound},
		{"upstream failure", http.StatusServiceUnavailable, "", identity.ErrHandleResolutionFailed},
		{"declared handle differs", http.StatusOK, "someone-else.bsky.social", identity.ErrHandleMismatch},
		{"verified handle", http.StatusOK, handle.String(), nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := &identity.BaseDirectory{
				SkipDNSDomainSuffixes: []string{".bsky.social"},
				HTTPClient: http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					status, body := http.StatusOK, ""
					switch r.URL.String() {
					case "https://" + handle.String() + "/.well-known/atproto-did":
						status, body = test.handleStatus, did.String()
					case identity.DefaultPLCURL + "/" + did.String():
						if test.handleStatus != http.StatusOK {
							t.Fatal("DID lookup followed a failed handle lookup")
						}
						body = `{"id":"` + did.String() + `","alsoKnownAs":["at://` + test.declaredHandle + `"]}`
					default:
						t.Fatalf("unexpected lookup: %s", r.URL)
					}
					return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
				})},
			}
			resolved, err := (federatedIdentityDirectory{base}).Lookup(context.Background(), handle.AtIdentifier())
			if !errors.Is(err, test.wantError) {
				t.Fatalf("lookup error = %v, want %v", err, test.wantError)
			}
			if test.wantError == nil && (resolved == nil || resolved.DID != did || resolved.Handle != handle) {
				t.Fatalf("resolved identity = %+v", resolved)
			}
		})
	}
}

func TestNewFederatedClientsBuildsDistinctPurposeClients(t *testing.T) {
	t.Parallel()

	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		t.Fatalf("federatedHTTPConfigFromEnv: %v", err)
	}
	clients, err := newFederatedClients(config)
	if err != nil {
		t.Fatalf("newFederatedClients: %v", err)
	}
	defer clients.boundary.CloseIdleConnections()
	if clients.metadata == nil || clients.oauth == nil || clients.pdsJSON == nil ||
		clients.pdsBlob == nil || clients.pdsRepository == nil || clients.directory == nil {
		t.Fatal("federated client set is incomplete")
	}
	if clients.metadata == clients.oauth || clients.oauth == clients.pdsJSON ||
		clients.pdsJSON == clients.pdsBlob || clients.pdsBlob == clients.pdsRepository {
		t.Fatal("purpose clients unexpectedly share an http.Client wrapper")
	}
}

func TestNewFederatedClientsSeparatesCachedAndAuthoritativeIdentityDirectories(t *testing.T) {
	t.Parallel()

	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		t.Fatalf("defaultFederatedHTTPConfig: %v", err)
	}
	clients, err := newFederatedClients(config)
	if err != nil {
		t.Fatalf("newFederatedClients: %v", err)
	}
	defer clients.boundary.CloseIdleConnections()

	if clients.authoritativeDirectory == nil {
		t.Fatal("authoritative identity directory is nil")
	}
	if _, ok := clients.authoritativeDirectory.(federatedIdentityDirectory); !ok {
		t.Fatal("authoritative directory is missing HTTPS-only handle lookup")
	}
	if clients.directory == clients.authoritativeDirectory {
		t.Fatal("cached and authoritative identity directories unexpectedly share one object")
	}
	cached, ok := clients.directory.(*identity.CacheDirectory)
	if !ok {
		t.Fatalf("display directory type = %T, want *identity.CacheDirectory", clients.directory)
	}
	if cached.Inner != clients.authoritativeDirectory {
		t.Fatal("cached directory does not wrap the authoritative hardened directory")
	}
}

func TestFederatedClientsRejectStoredPrivateSessionDestination(t *testing.T) {
	t.Parallel()

	config, err := defaultFederatedHTTPConfig()
	if err != nil {
		t.Fatalf("federatedHTTPConfigFromEnv: %v", err)
	}
	clients, err := newFederatedClients(config)
	if err != nil {
		t.Fatalf("newFederatedClients: %v", err)
	}
	defer clients.boundary.CloseIdleConnections()
	err = clients.validateSessionDestinations(
		context.Background(),
		"https://127.0.0.1",
		"https://auth.example",
		"https://auth.example/token",
		"",
	)
	if !errors.Is(err, federatedhttp.ErrDestinationRejected) {
		t.Fatalf("error = %v, want destination rejection", err)
	}
}
