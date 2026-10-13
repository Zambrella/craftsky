package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestScopeDiagnosticRetainsOnlyFixedFacts(t *testing.T) {
	media := DefaultScopes()[5]
	tests := []struct{ raw, reason, summary string }{
		{"atproto", "scope_required_missing", "missing=craftsky"},
		{"atproto atproto", "scope_duplicate_token", "granted=2"},
		{"atproto  private-response-canary", "scope_empty_token", "granted=3"},
		{"atproto Bearer\tprivate-token-canary", "scope_invalid_character", "granted=2"},
		{"atproto transition:generic", "scope_broad_authority", "granted=2"},
	}
	for _, test := range tests {
		err := scopeGrantDiagnostic(test.raw, DefaultScopes())
		var diagnostic *OAuthCallbackDiagnostic
		if !errors.As(err, &diagnostic) || !errors.Is(err, ErrOAuthFlowInvalid) || diagnostic.Reason() != test.reason {
			t.Fatalf("wrong diagnostic for %q", test.reason)
		}
		message := diagnostic.DiagnosticMessage()
		if !strings.Contains(message, test.summary) || strings.Contains(message, "private-") || strings.Contains(message, "Bearer") {
			t.Fatalf("unsafe/unhelpful diagnostic: %s", message)
		}
	}
	err := scopeGrantDiagnostic("atproto blob?accept=text/vtt&accept=video/mp4&accept=image/webp&accept=image/png&accept=image/jpeg", []string{"atproto", media})
	diagnostic := err.(*OAuthCallbackDiagnostic)
	if !strings.Contains(diagnostic.DiagnosticMessage(), "alternate_forms=1") {
		t.Fatal("equivalent scope formatting was not identified")
	}
	if validRegistrationTokenScopes("atproto blob?accept=text/vtt&accept=video/mp4&accept=image/webp&accept=image/png&accept=image/jpeg", []string{"atproto", media}) {
		t.Fatal("diagnostics changed authorization decisions")
	}
}
