package auth

import (
	"strings"
	"testing"
)

func TestOAuthScopesRequireGranularConsent(t *testing.T) {
	for _, broad := range []string{"transition:generic", "transition:chat.bsky", "repo:*", "repo:*?action=delete", "repo?collection=*", "repo:%2A", "repo?collection=%2A&action=create"} {
		t.Run(broad, func(t *testing.T) {
			scopes := append(DefaultScopes(), broad)
			if ValidateClientScopes(scopes) == nil {
				t.Fatal("accepted broad configured authority")
			}
			if validRegistrationTokenScopes(strings.Join(scopes, " "), DefaultScopes()) {
				t.Fatal("accepted broad granted authority")
			}
		})
	}
	if !validRegistrationTokenScopes(strings.Join(DefaultScopes(), " "), DefaultScopes()) {
		t.Fatal("rejected the current granular grant")
	}
	if validRegistrationTokenScopes("atproto", DefaultScopes()) {
		t.Fatal("accepted a grant missing required permissions")
	}
}

const resolvedCraftskyRepoScope = "repo?collection=social.craftsky.actor.profile&collection=social.craftsky.feed.post&collection=social.craftsky.feed.like&collection=social.craftsky.feed.repost&collection=social.craftsky.business.profile&collection=social.craftsky.business.event"

func TestResolvedPermissionSetGrantCoversRequiredAuthority(t *testing.T) {
	required := DefaultScopes()
	direct := append([]string{required[0]}, required[2:]...)
	raw := strings.Join(append(direct, resolvedCraftskyRepoScope), " ")
	if !validRegistrationTokenScopes(raw, required) {
		t.Fatal("rejected provider-resolved permission set grant")
	}
	for _, invalid := range []string{
		strings.Replace(raw, "&collection=social.craftsky.business.event", "", 1),
		strings.Replace(raw, resolvedCraftskyRepoScope, resolvedCraftskyRepoScope+"&action=create&action=update", 1),
		strings.Replace(raw, "social.craftsky.business.event", "other.example.event", 1),
		strings.Replace(raw, resolvedCraftskyRepoScope, resolvedCraftskyRepoScope+"&unknown=parameter", 1),
		strings.Replace(raw, resolvedCraftskyRepoScope, "repo:*", 1),
		strings.Replace(raw, resolvedCraftskyRepoScope, "repo?collection=*", 1),
	} {
		if validRegistrationTokenScopes(invalid, required) {
			t.Fatal("accepted incomplete or unbounded resolved authority")
		}
	}
	// The same authority can be split into separate operation grants.
	separate := resolvedCraftskyRepoScope + "&action=create " + resolvedCraftskyRepoScope + "&action=update " + resolvedCraftskyRepoScope + "&action=delete"
	if !validRegistrationTokenScopes(strings.Join(direct, " ")+" "+separate, required) {
		t.Fatal("rejected complete split action grants")
	}
}
