package chat

import (
	"strings"
	"testing"
)

// UT-002 / FR-005 / AC-006. Byte bounds remain independent of graphemes.
func TestMessageTextValidation(t *testing.T) {
	byteBoundary := "a" + strings.Repeat("\u0301", 4999)
	for _, test := range []struct {
		name, text string
		valid      bool
	}{
		{"empty", "", false},
		{"whitespace", " \n\t\u2003", false},
		{"byte order mark only", "\ufeff", false},
		{"ordinary", " knitting plans ", true},
		{"999 graphemes", strings.Repeat("x", 999), true},
		{"1000 graphemes", strings.Repeat("x", 1000), true},
		{"1001 graphemes", strings.Repeat("x", 1001), false},
		{"1000 emoji", strings.Repeat("🧶", 1000), true},
		{"1001 emoji", strings.Repeat("🧶", 1001), false},
		{"combining clusters", strings.Repeat("e\u0301", 1000), true},
		{"ZWJ clusters", strings.Repeat("👩‍👩‍👧‍👦", 400), true},
		{"9999 bytes", byteBoundary, true},
		{"10000 bytes", byteBoundary + "b", true},
		{"10001 bytes", byteBoundary + "bc", false},
		{"malformed UTF8", string([]byte{0xff}), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidateMessageText(test.text) == nil; got != test.valid {
				t.Fatalf("valid=%t want=%t", got, test.valid)
			}
		})
	}
}
