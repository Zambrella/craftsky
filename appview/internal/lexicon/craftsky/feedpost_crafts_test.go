package craftsky

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestPostCraftsJSONAndCBORRoundTrip(t *testing.T) {
	t.Parallel()
	post := FeedPost{LexiconTypeID: "social.craftsky.feed.post", Text: "hello", CreatedAt: "2026-10-19T00:00:00Z",
		Crafts: []*FeedPost_Craft{{CraftType: "social.craftsky.feed.defs#sewing"}, {CraftType: "social.craftsky.feed.defs#weaving"}},
	}
	wire, err := json.Marshal(&post)
	if err != nil {
		t.Fatal(err)
	}
	var fromJSON FeedPost
	if err := json.Unmarshal(wire, &fromJSON); err != nil {
		t.Fatal(err)
	}
	var cbor bytes.Buffer
	if err := post.MarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	var fromCBOR FeedPost
	if err := fromCBOR.UnmarshalCBOR(&cbor); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(post.Crafts, fromJSON.Crafts) || !reflect.DeepEqual(post.Crafts, fromCBOR.Crafts) {
		t.Fatalf("craft metadata lost: JSON=%+v CBOR=%+v", fromJSON.Crafts, fromCBOR.Crafts)
	}
	post.Crafts = nil
	wire, err = json.Marshal(&post)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wire, []byte(`"crafts"`)) {
		t.Fatalf("ordinary post gained crafts: %s", wire)
	}
}
