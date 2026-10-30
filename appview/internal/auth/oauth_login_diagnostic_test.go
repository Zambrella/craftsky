package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestIdentityDirectoryDiagnosticsSelectOnlyPinnedStaticFailures(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"identity not found in cache after coalesce returned", "cache_result_missing"},
		{"unexpected control-flow error", "directory_control_flow"},
		{"at-identifier neither a Handle nor a DID", "identifier_type_invalid"},
		{"private-provider-canary https://private.example/?token=secret", "OAuth login failed at identity"},
	} {
		original := errors.New(test.input)
		diagnostic := loginDiagnostic("identity", original).(*OAuthLoginDiagnostic)
		if diagnostic.Unwrap() != original || !strings.Contains(diagnostic.DiagnosticMessage(), test.want) {
			t.Fatal("original cause or selected reason lost")
		}
		if strings.Contains(diagnostic.DiagnosticMessage(), "private-") {
			t.Fatal("private prose escaped")
		}
	}
}
