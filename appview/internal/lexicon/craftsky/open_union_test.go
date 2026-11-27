package craftsky_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	craftsky "social.craftsky/appview/internal/lexicon/craftsky"
)

func TestProjectPatternsJSONAndCBORRoundTrip(t *testing.T) {
	raw := []byte(`{
		"craftType":"social.craftsky.feed.defs#sewing",
		"pattern":{"name":"Legacy pattern"},
		"patterns":[
			{"name":"Sleeves","url":"https://example.org/sleeves","difficulty":"social.craftsky.feed.defs#intermediate","publisher":"Woven Press"},
			{"name":"#Collar","nameFacets":[{"index":{"byteStart":0,"byteEnd":7},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Collar"}]}],"designer":"Mina","selfDrafted":true}
		]
	}`)
	var value craftsky.ProjectDefs_ProjectCommon
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	var cbor bytes.Buffer
	if err := value.MarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	var decoded craftsky.ProjectDefs_ProjectCommon
	if err := decoded.UnmarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	var want, got map[string]any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip got %s, want %s", encoded, raw)
	}
}

func TestAdditiveProjectDetailsJSONAndCBORRoundTrip(t *testing.T) {
	for _, tc := range []struct{ craft, fields string }{
		{"knitting", `"needleSizeMm":"4.0mm","needleSizesMm":["4.0mm","4.5mm"],"methods":["hand","machine"],"techniques":[{"technique":"cables"},{"technique":"colourwork","subtypes":[{"subtype":"intarsia"},{"subtype":"stranded"}]}]`},
		{"embroidery", `"methods":[{"method":"hand","styles":[{"style":"sashiko"}]},{"method":"machine","styles":[{"style":"threeDimensionalPuff"}]}]`},
		{"weaving", `"materials":[{"role":"warp","text":"linen"},{"role":"weft","text":"wool"}],"sett":{"amount":"12.5","unit":"endsPerInch"}`},
	} {
		t.Run(tc.craft, func(t *testing.T) {
			raw := []byte(`{"$type":"social.craftsky.project.` + tc.craft + `#details",` + tc.fields + `}`)
			var value craftsky.ProjectDefs_Project_Details
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			if value.Unknown != nil {
				t.Fatal("expected a generated, known project details variant")
			}
			var cbor bytes.Buffer
			if err := value.MarshalCBOR(&cbor); err != nil {
				t.Fatal(err)
			}
			var decoded craftsky.ProjectDefs_Project_Details
			if err := decoded.UnmarshalCBOR(&cbor); err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(&decoded)
			if err != nil {
				t.Fatal(err)
			}
			var want, got map[string]any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatal(err)
			}
			if decoded.Unknown != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("round trip got %s, want %s", encoded, raw)
			}
		})
	}
}

func TestUnknownProjectUnionJSONAndCBORRoundTrip(t *testing.T) {
	raw := []byte(`{"$type":"social.craftsky.project.future#details","largeInteger":9007199254740993,"nested":{"text":"future"}}`)
	var value craftsky.ProjectDefs_Project_Details
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	var cbor bytes.Buffer
	if err := value.MarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	var decoded craftsky.ProjectDefs_Project_Details
	if err := decoded.UnmarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(&decoded)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"9007199254740993", "future#details", "nested"} {
		if !strings.Contains(string(encoded), part) {
			t.Fatalf("lost %s: %s", part, encoded)
		}
	}
	if err = json.Unmarshal([]byte(`{"$type":"social.craftsky.project.knitting#details"}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Unknown != nil {
		t.Fatal("stale unknown variant retained after decoding a known one")
	}
}
