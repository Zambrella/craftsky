package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/middleware"
)

func TestAccountHostingLinks(t *testing.T) {
	for _, tc := range []struct{ endpoint, provider, manage, password string }{
		{"https://bsky.social", "Bluesky", "https://bsky.social/account", "https://bsky.app/settings/account"},
		{"https://morel.us-east.host.bsky.network", "Bluesky", "https://bsky.social/account", "https://bsky.app/settings/account"},
		{"https://pds.example/", "pds.example", "https://pds.example/account", ""},
		{"https://host.bsky.network.attacker.example", "host.bsky.network.attacker.example", "https://host.bsky.network.attacker.example/account", ""},
		{"https://pds.example:8443", "pds.example:8443", "https://pds.example:8443/account", ""},
		{"http://pds.example", "", "", ""},
		{"https://user:secret@pds.example", "", "", ""},
		{"https://pds.example/path", "", "", ""},
		{"https://pds.example?secret=canary", "", "", ""},
		{"https://pds.example/#fragment", "", "", ""},
		{"", "", "", ""},
	} {
		t.Run(tc.endpoint, func(t *testing.T) {
			// A custom handle must not prevent detection of Bluesky hosting.
			info := accountHostingInfo("did:plc:alice", "alice.example", tc.endpoint)
			if info.HostingProvider != tc.provider || info.AccountManagementURL != tc.manage || info.PasswordChangeURL != tc.password {
				t.Fatalf("unexpected provider links: %#v", info)
			}
		})
	}
	// A Bluesky-style handle does not prove Bluesky hosting.
	info := accountHostingInfo("did:plc:alice", "alice.bsky.social", "https://pds.example")
	if info.PasswordChangeURL != "" {
		t.Fatal("inferred host from handle")
	}
}

func TestWhoAmIHostingThroughResolverWrappers(t *testing.T) {
	directory := DirectoryHandleResolver{Directory: &fakeDirectory{didResult: &identity.Identity{
		DID: syntax.DID("did:plc:alice"), Handle: syntax.Handle("alice.example"),
		Services: map[string]identity.ServiceEndpoint{"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: "https://morel.us-east.host.bsky.network"}},
	}}}
	observer := &recordingIdentityObserver{}
	resolver := NewObservedHandleResolver(DevHandleResolver{Primary: directory}, "cached", observer)
	req := httptest.NewRequest("GET", "/v1/whoami", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:alice"))
	response := httptest.NewRecorder()
	WhoAmIHandler(resolver, testLogger(t)).ServeHTTP(response, req)
	var body WhoAmIResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if response.Code != 200 || body.HostingProvider != "Bluesky" || body.PDSURL == "" || body.DID != "did:plc:alice" {
		t.Fatalf("bad response: %s", response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("account info must not be cached")
	}
	if len(observer.observations) != 1 {
		t.Fatal("missing identity observation")
	}
}
