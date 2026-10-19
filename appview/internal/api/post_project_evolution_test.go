package api_test

import (
	"encoding/json"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestProjectExtensionRoundTripAndKnownFieldClearing(t *testing.T) {
	raw := `{"futureProject":{"n":9007199254740993},"common":{"craftType":"social.craftsky.feed.defs#knitting","futureCommon":{"n":9007199254740993},"title":"old","pattern":{"name":"pattern","futurePattern":true},"materials":[{"text":"wool","futureMaterial":true}],"additionalCrafts":[{"craftType":"social.craftsky.feed.defs#sewing","futureCraft":true}]},"details":{"$type":"social.craftsky.project.future#details","futureDetails":true}}`
	var project api.Project
	if err := json.Unmarshal([]byte(raw), &project); err != nil {
		t.Fatal(err)
	}
	project.Common.Title = nil
	encoded, err := json.Marshal(api.PostResponse{Project: &project})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"futureProject", "futureCommon", "futurePattern", "futureMaterial", "futureCraft", "futureDetails", "9007199254740993"} {
		if !strings.Contains(string(encoded), field) {
			t.Fatalf("lost %s: %s", field, encoded)
		}
	}
	if strings.Contains(string(encoded), `"title":"old"`) {
		t.Fatalf("cleared title resurrected: %s", encoded)
	}
	project.Common.ExtraFields["craftType"] = json.RawMessage(`"overwrite"`)
	encoded, err = json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "overwrite") {
		t.Fatal("extension overwrote typed field")
	}
}

func TestProjectPrimaryAndAdditionalCraftAuthoring(t *testing.T) {
	for _, tc := range []struct {
		name, details, additional string
		valid                     bool
	}{
		{"matching", `{"$type":"social.craftsky.project.knitting#details"}`, `[{"craftType":"social.craftsky.feed.defs#sewing"}]`, true},
		{"unknown variant", `{"$type":"social.craftsky.project.future#details","future":true}`, `[]`, true},
		{"mismatched", `{"$type":"social.craftsky.project.sewing#details"}`, `[]`, false},
		{"secondary cannot own details", `{"$type":"social.craftsky.project.sewing#details"}`, `[{"craftType":"social.craftsky.feed.defs#sewing"}]`, false},
		{"primary repeated", `{"$type":"social.craftsky.project.knitting#details"}`, `[{"craftType":"social.craftsky.feed.defs#knitting"}]`, false},
		{"secondary repeated", `{"$type":"social.craftsky.project.knitting#details"}`, `[{"craftType":"social.craftsky.feed.defs#sewing"},{"craftType":"social.craftsky.feed.defs#sewing"}]`, false},
		{"unsupported secondary", `{"$type":"social.craftsky.project.knitting#details"}`, `[{"craftType":"unknown"}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"text":"project","sponsored":false,"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting","additionalCrafts":` + tc.additional + `},"details":` + tc.details + `}}`
			req, err := api.DecodePostCreate(strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			if err = api.ValidatePostCreate(req); (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
}

func TestPostImageAltUnicodeBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, alt string
		valid     bool
	}{
		{"CJK", strings.Repeat("縫", 1000), true},
		{"emoji", strings.Repeat("🧶", 1000), true},
		{"graphemes", strings.Repeat("a", 1001), false},
		{"bytes", strings.Repeat("a"+strings.Repeat("\u0301", 6), 1000), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := api.DecodePostCreate(strings.NewReader(`{"text":"project","sponsored":false,"images":[{"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":123}}]}`))
			if err != nil {
				t.Fatal(err)
			}
			req.Images[0].Alt = tc.alt
			if err = api.ValidatePostCreate(req); (err == nil) != tc.valid {
				t.Fatalf("valid=%t err=%v", tc.valid, err)
			}
		})
	}
}
