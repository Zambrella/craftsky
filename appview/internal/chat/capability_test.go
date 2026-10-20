package chat

import (
	"slices"
	"social.craftsky/appview/internal/auth"
	"testing"
)

// UT-001 / FR-001, RULE-004, NFR-001: individual optional operations do
// not borrow the authority of unrelated granted operations.
func TestCapabilitiesKeepReportAndDeclarationIndependent(t *testing.T) {
	full := append([]string{"atproto"}, auth.OptionalChatScopes()...)
	for _, test := range []struct {
		name                       string
		scopes                     []string
		read, send, report, policy bool
	}{
		{"base", auth.DefaultScopes(), false, false, false, false},
		{"full", full, true, true, true, true},
		{"without report", slices.Delete(slices.Clone(full), len(full)-2, len(full)-1), true, true, false, true},
		{"without declaration", slices.Clone(full[:len(full)-1]), true, true, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := EvaluateCapabilities(test.scopes)
			if got.CanRead != test.read || got.CanSend != test.send || got.CanReport != test.report || got.CanSetPolicy != test.policy {
				t.Fatalf("capability result: %+v", got)
			}
		})
	}
}
