package auth

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/identity"
	"log/slog"
	"social.craftsky/appview/internal/ctxkeys"
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

func TestProviderCallbackReasonSelectsOnlyKnownCode(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"access_denied", "provider_callback_error_access_denied"},
		{"server_error", "provider_callback_error_server_error"},
		{"PRIVATE_CODE https://private.invalid/?code=secret", "provider_callback_error_unknown"},
	} {
		err := providerCallbackDiagnostic(test.input)
		var diagnostic *OAuthCallbackDiagnostic
		if !errors.As(err, &diagnostic) || !errors.Is(err, ErrOAuthFlowInvalid) || diagnostic.Reason() != test.want {
			t.Fatal("wrong callback reason")
		}
		if strings.Contains(diagnostic.DiagnosticMessage(), "PRIVATE_") || strings.Contains(diagnostic.DiagnosticMessage(), "secret") {
			t.Fatal("private provider prose escaped")
		}
	}
}

func TestExpectedAuthFailuresClassifiedAtIssueBoundary(t *testing.T) {
	for _, test := range []struct {
		name               string
		cause              error
		wantKey, wantValue string
	}{
		{"provider denial", providerCallbackDiagnostic("access_denied"), "result", "expected"},
		{"unknown provider refusal", providerCallbackDiagnostic("private-provider-canary"), "error_category", "finalization"},
		{"nonexistent handle", loginDiagnostic("identity", identity.ErrHandleNotFound), "error_category", "validation"},
		{"identity dependency failure", loginDiagnostic("identity", errors.New("private-provider-canary")), "error_category", "finalization"},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			ctx := ctxkeys.WithDiagnosticBoundary(context.Background(), func(_ context.Context, cause error, fields map[string]any) []slog.Attr {
				called = true
				if cause != test.cause {
					t.Fatal("original cause replaced")
				}
				if fields[test.wantKey] != test.wantValue {
					t.Fatalf("boundary facts = %v", fields)
				}
				return nil
			})
			authLogErrorAttrs(ctx, "", "oauth.callback", "finalization", test.cause)
			if !called {
				t.Fatal("diagnostic owner was not invoked")
			}
		})
	}
}
