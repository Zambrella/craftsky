package guestvisibility_test

import (
	"encoding/json"
	"social.craftsky/appview/internal/guestvisibility"
	"testing"
)

// UT-007 / FR-009 / AC-009.
func TestGuestVisibilityPatchPreservesProfileAndOtherLabels(t *testing.T) {
	before := json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"Original","future":{"number":9007199254740993},"labels":{"$type":"com.atproto.label.defs#selfLabels","extra":"keep","values":[{"val":"bot","extra":true},{"val":"!no-unauthenticated"},{"val":"!no-unauthenticated"}]}}`)
	for _, hidden := range []bool{false, true} {
		result, err := guestvisibility.SetPreference(before, hidden)
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]json.RawMessage
		if err := json.Unmarshal(result, &got); err != nil {
			t.Fatal(err)
		}
		if string(got["future"]) != `{"number":9007199254740993}` || string(got["displayName"]) != `"Original"` {
			t.Fatalf("profile changed: %s", result)
		}
		var labels struct {
			Extra  string                       `json:"extra"`
			Values []map[string]json.RawMessage `json:"values"`
		}
		if err := json.Unmarshal(got["labels"], &labels); err != nil {
			t.Fatal(err)
		}
		count := 0
		bot := false
		for _, v := range labels.Values {
			if string(v["val"]) == `"!no-unauthenticated"` {
				count++
			}
			if string(v["val"]) == `"bot"` && string(v["extra"]) == "true" {
				bot = true
			}
		}
		want := 0
		if hidden {
			want = 1
		}
		if labels.Extra != "keep" || !bot || count != want {
			t.Fatalf("labels changed: %s", result)
		}
		again, err := guestvisibility.SetPreference(result, hidden)
		if err != nil || string(again) != string(result) {
			t.Fatalf("patch is not idempotent: %s %v", again, err)
		}
	}
	if _, err := guestvisibility.SetPreference(json.RawMessage(`{"labels":{"$type":"unknown","values":[]}}`), true); err == nil {
		t.Fatal("unknown label union overwritten")
	}
}

func TestGuestVisibilityPatchStillRejectsMalformedLabelsWithOptOut(t *testing.T) {
	for _, raw := range []string{
		`{"labels":{"$type":"unknown","values":[{"val":"!no-unauthenticated"}]}}`,
		`{"labels":{"$type":"com.atproto.label.defs#selfLabels","values":[{}, {"val":"!no-unauthenticated"}]}}`,
	} {
		if _, err := guestvisibility.SetPreference(json.RawMessage(raw), false); err == nil {
			t.Fatal("malformed labels were overwritten")
		}
	}
}
