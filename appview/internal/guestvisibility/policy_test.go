package guestvisibility_test

import (
	"encoding/json"
	"testing"

	"social.craftsky/appview/internal/guestvisibility"
)

// UT-001 / FR-010, RULE-002 / AC-010.
func TestGuestVisibilityRecordPreference(t *testing.T) {
	for _, tc := range []struct {
		name, record string
		want         guestvisibility.Preference
	}{
		{"absent labels", `{"$type":"app.bsky.actor.profile"}`, guestvisibility.Public},
		{"empty labels", `{"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[]}}`, guestvisibility.Public},
		{"other labels", `{"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{"val":"bot"}]}}`, guestvisibility.Public},
		{"opted out", `{"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{"val":"bot"},{"val":"!no-unauthenticated"}]}}`, guestvisibility.Hidden},
		{"unknown union", `{"labels":{"$type":"unknown","values":[]}}`, guestvisibility.Unknown},
		{"missing values", `{"labels":{"$type":"com.atproto.label.defs#selfLabels"}}`, guestvisibility.Unknown},
		{"malformed", `{"labels":"broken"}`, guestvisibility.Unknown},
		{"null record", `null`, guestvisibility.Unknown},
		{"null label", `{"labels":null}`, guestvisibility.Unknown},
		{"malformed value", `{"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{}]}}`, guestvisibility.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := guestvisibility.ParsePreference(json.RawMessage(tc.record))
			if got != tc.want {
				t.Fatalf("preference=%v, want %v", got, tc.want)
			}
			if got.AllowsGuest() != (tc.want == guestvisibility.Public) {
				t.Fatal("only verified public state may allow guests")
			}
		})
	}
	if guestvisibility.Unknown.AllowsGuest() {
		t.Fatal("unknown visibility exposed")
	}
}
