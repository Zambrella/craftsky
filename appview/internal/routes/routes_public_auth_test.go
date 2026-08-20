package routes

import "testing"

func TestNewOAuthHandlersCarriesDevSchemeCapability(t *testing.T) {
	handlers := newOAuthHandlers(oauthRouteDependencies{allowDevScheme: true})
	if !handlers.AllowDevScheme {
		t.Fatal("development OAuth scheme capability was dropped by route composition")
	}
}
