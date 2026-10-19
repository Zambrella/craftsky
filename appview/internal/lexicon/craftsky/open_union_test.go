package craftsky_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	craftsky "social.craftsky/appview/internal/lexicon/craftsky"
)

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
