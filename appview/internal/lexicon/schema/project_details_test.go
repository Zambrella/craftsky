package schema_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"social.craftsky/appview/internal/lexicon/schema"
)

func TestProjectPatternList(t *testing.T) {
	const four = `{"name":"Bodice"},{"name":"Sleeves"},{"name":"Skirt"},{"name":"Pockets"}`
	const five = four + `,{"name":"Collar","designer":"Mina","publisher":"Woven Press","url":"https://example.org/collar","difficulty":"social.craftsky.feed.defs#intermediate","selfDrafted":true}`
	for _, tc := range []struct {
		name   string
		fields string
		valid  bool
	}{
		{"no pattern", ``, true},
		{"legacy pattern", `,"pattern":{"name":"Old pattern"}`, true},
		{"empty list with legacy pattern", `,"patterns":[],"pattern":{"name":"Old pattern"}`, true},
		{"five ordered pattern objects", `,"patterns":[` + five + `]`, true},
		{"six pattern objects", `,"patterns":[` + five + `,{"name":"Lining"}]`, false},
		{"object instead of list", `,"patterns":{"name":"Bodice"}`, false},
		{"string instead of pattern object", `,"patterns":["Bodice"]`, false},
		{"malformed credits in a later entry", `,"patterns":[` + four + `,{"publisher":4}]`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-11-20T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#sewing"` + tc.fields + `}}}`)
			if err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post"); (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
		})
	}
}

func TestProjectToolSizeLists(t *testing.T) {
	for _, tool := range []struct{ craft, scalar, list string }{
		{"knitting", "needleSizeMm", "needleSizesMm"},
		{"crochet", "hookSizeMm", "hookSizesMm"},
	} {
		t.Run(tool.craft, func(t *testing.T) {
			for _, tc := range []struct {
				name  string
				value string
				field string
				valid bool
			}{
				{"legacy scalar", `"4.0mm"`, tool.scalar, true},
				{"empty list", `[]`, tool.list, true},
				{"two sizes", `["4.0mm","4.5mm"]`, tool.list, true},
				{"unfamiliar bounded item", `["` + strings.Repeat("a", 20) + `"]`, tool.list, true},
				{"oversized item", `["` + strings.Repeat("é", 11) + `"]`, tool.list, false},
				{"numeric item", `[4]`, tool.list, false},
				{"ten entries", `["1.0mm","2.0mm","2.5mm","3.0mm","3.5mm","4.0mm","4.5mm","5.0mm","5.5mm","6.0mm"]`, tool.list, true},
				{"eleven entries", `["1.0mm","2.0mm","2.5mm","3.0mm","3.5mm","4.0mm","4.5mm","5.0mm","5.5mm","6.0mm","6.5mm"]`, tool.list, false},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := projectDetailsRecord(tool.craft, fmt.Sprintf(`%q:%s`, tc.field, tc.value))
					if err := schema.ValidateCraftskyRecord(raw, "social.craftsky.feed.post"); (err == nil) != tc.valid {
						t.Fatalf("valid=%v, error=%v", tc.valid, err)
					}
				})
			}
		})
	}
}

func TestProjectDetailGroupShapes(t *testing.T) {
	technique := `{"technique":"cables"}`
	subtype := `{"subtype":"stranded"}`
	for _, tc := range []struct {
		name    string
		craft   string
		valid   string
		invalid string
	}{
		{"knitting methods", "knitting", `"methods":["hand","machine"]`, `"methods":"hand"`},
		{"crochet methods", "crochet", `"methods":["standardCrochet","tunisianCrochet"]`, `"methods":[0]`},
		{"felting methods", "felting", `"methods":["needle","wet"]`, `"methods":"needle"`},
		{"independent embroidery method and style", "embroidery", `"method":"machine","style":"crossStitch"`, `"style":["crossStitch"]`},
		{"custom item subtype", "weaving", `"projectType":"social.craftsky.project.defs#homeGoods","projectSubtype":"Lampshade"`, `"projectSubtype":"` + strings.Repeat("é", 51) + `"`},
		{"knitting technique subtypes", "knitting", `"techniques":[{"technique":"colourwork","subtypes":[{"subtype":"stranded"}],"futureLabel":"value"}]`, `"techniques":[{"technique":"colourwork","subtypes":["stranded"]}]`},
		{"technique list limit", "knitting", `"techniques":[` + strings.Repeat(technique+",", 19) + technique + `]`, `"techniques":[` + strings.Repeat(technique+",", 20) + technique + `]`},
		{"subtype list limit", "knitting", `"techniques":[{"technique":"colourwork","subtypes":[` + strings.Repeat(subtype+",", 9) + subtype + `]}]`, `"techniques":[{"technique":"colourwork","subtypes":[` + strings.Repeat(subtype+",", 10) + subtype + `]}]`},
		{"crochet techniques", "crochet", `"techniques":[{"technique":"filet-crochet"}]`, `"techniques":["filet-crochet"]`},
		{"sewing techniques", "sewing", `"techniques":[{"technique":"french-seams"}]`, `"techniques":[{}]`},
		{"embroidery style groups", "embroidery", `"methods":[{"method":"hand","styles":[{"style":"sashiko"}]},{"method":"machine","styles":[{"style":"3d-puff"}]}]`, `"methods":[{"styles":[{"style":"sashiko"}]}]`},
		{"spinning ply count", "spinning", `"plyCount":2`, `"plyCount":0`},
		{"dyeing material form", "dyeing", `"materialForm":"roving"`, `"materialForm":["roving"]`},
		{"punch needle loop height", "punchNeedle", `"loopHeight":{"amount":"4.5","unit":"mm"}`, `"loopHeight":{"amount":"4.5"}`},
		{"macrame cord entries", "macrame", `"cords":[{"material":"cotton","diameterMm":"4.0"}]`, `"cords":[{"diameterMm":4}]`},
		{"weaving material roles", "weaving", `"materials":[{"role":"warp","text":"linen"},{"role":"weft","text":"wool"}]`, `"materials":[{"text":"linen"}]`},
		{"felting needle gauges", "felting", `"needleGauges":[{"gauge":"36"},{"gauge":"40"}]`, `"needleGauges":[36]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := schema.ValidateCraftskyRecord(projectDetailsRecord(tc.craft, tc.valid), "social.craftsky.feed.post"); err != nil {
				t.Fatalf("valid detail shape rejected: %v", err)
			}
			if err := schema.ValidateCraftskyRecord(projectDetailsRecord(tc.craft, tc.invalid), "social.craftsky.feed.post"); err == nil {
				t.Fatal("malformed detail shape accepted")
			}
		})
	}
}

func projectDetailsRecord(craft string, fields string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"$type":"social.craftsky.feed.post","text":"project","sponsored":false,"createdAt":"2026-11-20T00:00:00Z","project":{"common":{"craftType":"social.craftsky.feed.defs#%s"},"details":{"$type":"social.craftsky.project.%s#details",%s}}}`, craft, craft, fields))
}
