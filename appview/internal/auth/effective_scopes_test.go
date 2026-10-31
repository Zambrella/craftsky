package auth

import (
	"net/url"
	"slices"
	"strings"
	"testing"
)

// UT-001 / BR-002, RULE-004, NFR-001 / AC-002, AC-019.
func TestChatOptionalScopesPreserveBaseSession(t *testing.T) {
	callback, err := url.Parse("http://127.0.0.1:18350/oauth/callback")
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := BuildClientArtifacts(ClientConfigInput{
		Mode: ClientModeLocalhost, CallbackURL: *callback, Scopes: DefaultScopes(),
	})
	if err != nil {
		t.Fatal(err)
	}
	sendScope := "rpc:chat.bsky.convo.sendMessage?aud=did:web:api.bsky.chat%23bsky_chat"
	if !slices.Contains(strings.Fields(artifacts.Metadata.Scope), sendScope) {
		t.Fatal("consumer metadata does not advertise optional chat authority")
	}
	if !slices.Equal(artifacts.Config.Scopes, DefaultScopes()) {
		t.Fatal("optional advertised permissions changed ordinary login requirements")
	}
	baseGrant := strings.Join(DefaultScopes(), " ")
	if !validRegistrationTokenScopes(baseGrant, artifacts.Config.Scopes) {
		t.Fatal("base-only grant rejected for ordinary login/refresh")
	}
	if validRegistrationTokenScopes(baseGrant, append(DefaultScopes(), sendScope)) {
		t.Fatal("base-only grant authorized sending a private message")
	}
	clientID, err := url.Parse(artifacts.Config.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if clientID.Query().Get("scope") != artifacts.Metadata.Scope {
		t.Fatal("localhost discovery identity does not advertise the same permissions")
	}
}

// UT-001 / RULE-004, NFR-001 / AC-019.
func TestChatCapabilityNormalizesRPCScopes(t *testing.T) {
	const method = "chat.bsky.convo.sendMessage"
	const audience = "did:web:api.bsky.chat%23bsky_chat"
	required := []string{"atproto", "rpc:" + method + "?aud=" + audience}
	for _, test := range []struct {
		name, grant string
		want        bool
	}{
		{"positional", "rpc:" + method + "?aud=" + audience, true},
		{"combined methods", "rpc?lxm=chat.bsky.convo.getLog&lxm=" + method + "&aud=" + audience, true},
		{"reordered and encoded", "rpc?aud=did%3Aweb%3Aapi.bsky.chat%23bsky_chat&lxm=chat.bsky.convo.%73endMessage", true},
		{"encoded positional", "rpc:chat.bsky.convo.%73endMessage?aud=" + audience, true},
		{"wrong service", "rpc:" + method + "?aud=did:web:other.example%23bsky_chat", false},
		{"wrong fragment", "rpc:" + method + "?aud=did:web:api.bsky.chat%23other", false},
		{"wrong method", "rpc:chat.bsky.convo.getLog?aud=" + audience, false},
		{"wildcard audience", "rpc:" + method + "?aud=*", false},
		{"wildcard method", "rpc?lxm=*&aud=" + audience, false},
		{"partial wildcard", "rpc:chat.bsky.convo.*?aud=" + audience, false},
		{"missing audience", "rpc:" + method, false},
		{"duplicate audience", "rpc:" + method + "?aud=" + audience + "&aud=" + audience, false},
		{"positional plus parameter", "rpc:" + method + "?lxm=" + method + "&aud=" + audience, false},
		{"unknown parameter", "rpc?lxm=" + method + "&aud=" + audience + "&extra=ignored", false},
		{"malformed encoding", "rpc?lxm=" + method + "&aud=%XX", false},
		{"full chat include", "include:chat.bsky.authFullChatClient?aud=" + audience, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := validRegistrationTokenScopes("atproto "+test.grant, required); got != test.want {
				t.Fatalf("grant recognized = %v, want %v", got, test.want)
			}
		})
	}
}
