package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"social.craftsky/appview/internal/lexicon/schema"
)

func TestReservedPostCraftsContract(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, field string
		valid       bool
	}{
		{"absent", "", true},
		{"empty", `,"crafts":[]`, true},
		{"three", `,"crafts":[{"craftType":"social.craftsky.feed.defs#sewing"},{"craftType":"social.craftsky.feed.defs#embroidery"},{"craftType":"social.craftsky.feed.defs#quilting"}]`, true},
		{"future token", `,"crafts":[{"craftType":"social.craftsky.feed.defs#weaving"}]`, true},
		{"four", `,"crafts":[{"craftType":"a"},{"craftType":"b"},{"craftType":"c"},{"craftType":"d"}]`, false},
		{"missing craft type", `,"crafts":[{}]`, false},
		{"empty craft type", `,"crafts":[{"craftType":""}]`, false},
		{"scalar entry", `,"crafts":["sewing"]`, false},
		{"null entry", `,"crafts":[null]`, false},
		{"oversized craft type", `,"crafts":[{"craftType":"` + strings.Repeat("a", 101) + `"}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello","sponsored":false,"createdAt":"2026-10-19T00:00:00Z"` + tc.field + `}`)
			if err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post"); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}
