package api

import (
	"strings"
	"testing"
)

func TestProtocolProfileAcceptsGraphemeLimitedName(t *testing.T) {
	name := strings.Repeat("🧶", 10) + strings.Repeat("👨‍👩‍👧‍👦", 10)
	err := ValidateProfilePut(ProfilePutRequest{DisplayName: &name})
	t.Logf("20-grapheme display name: %v", err)
	if err != nil {
		t.Fatal("valid grapheme-limited name rejected")
	}
}
