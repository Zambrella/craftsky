package craftsky

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// IT-008: the post's open embed union reuses the generated standard external
// type for JSON and CBOR records.
func TestFeedPostExternalEmbedContract(t *testing.T) {
	t.Parallel()

	rawSchema, err := os.ReadFile("../../../../lexicon/social/craftsky/feed/post.json")
	if err != nil {
		t.Fatalf("read post lexicon: %v", err)
	}
	var schema struct {
		Defs map[string]struct {
			Record struct {
				Properties map[string]struct {
					Type   string   `json:"type"`
					Refs   []string `json:"refs"`
					Closed bool     `json:"closed"`
				}
			}
		}
	}
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		t.Fatalf("decode post lexicon: %v", err)
	}
	embed := schema.Defs["main"].Record.Properties["embed"]
	if embed.Type != "union" || embed.Closed {
		t.Fatalf("main.embed must remain an open union: %#v", embed)
	}
	if !slices.Contains(embed.Refs, "app.bsky.embed.external") {
		t.Fatalf("main.embed.refs = %v, want app.bsky.embed.external", embed.Refs)
	}

	fixture := []byte(`{
		"$type":"social.craftsky.feed.post",
		"text":"Useful pattern",
		"createdAt":"2026-08-25T00:00:00Z",
		"embed":{
			"$type":"app.bsky.embed.external",
			"external":{
				"uri":"https://example.com/pattern",
				"title":"Example pattern",
				"description":"A useful pattern"
			}
		}
	}`)
	var post FeedPost
	if err := json.Unmarshal(fixture, &post); err != nil {
		t.Fatalf("decode external post: %v", err)
	}
	assertGeneratedExternalEmbed(t, post.Embed)

	jsonRoundTrip, err := json.Marshal(&post)
	if err != nil {
		t.Fatalf("marshal external post as JSON: %v", err)
	}
	var jsonWire map[string]any
	if err := json.Unmarshal(jsonRoundTrip, &jsonWire); err != nil {
		t.Fatalf("decode external JSON round trip: %v", err)
	}
	if got := jsonWire["embed"].(map[string]any)["$type"]; got != "app.bsky.embed.external" {
		t.Fatalf("external JSON $type = %v", got)
	}

	var cbor bytes.Buffer
	if err := post.MarshalCBOR(&cbor); err != nil {
		t.Fatalf("marshal external post as CBOR: %v", err)
	}
	var decoded FeedPost
	if err := decoded.UnmarshalCBOR(bytes.NewReader(cbor.Bytes())); err != nil {
		t.Fatalf("decode external CBOR round trip: %v", err)
	}
	assertGeneratedExternalEmbed(t, decoded.Embed)
}

func assertGeneratedExternalEmbed(t *testing.T, embed *FeedPost_Embed) {
	t.Helper()
	if embed == nil {
		t.Fatal("generated embed is nil")
	}
	if embed.EmbedExternal == nil {
		t.Fatal("generated embed did not select app.bsky.embed.external")
	}
}

// IT-006: standard video is an additive branch of the optional open post embed
// union and uses Indigo's generated app.bsky.embed.video type.
func TestFeedPostVideoEmbedSchemaContract(t *testing.T) {
	t.Parallel()

	rawSchema, err := os.ReadFile("../../../../lexicon/social/craftsky/feed/post.json")
	if err != nil {
		t.Fatalf("read post lexicon: %v", err)
	}
	var schema struct {
		Defs map[string]struct {
			Record struct {
				Properties map[string]struct {
					Type   string   `json:"type"`
					Refs   []string `json:"refs"`
					Closed bool     `json:"closed"`
				}
			}
		}
	}
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		t.Fatalf("decode post lexicon: %v", err)
	}
	embed := schema.Defs["main"].Record.Properties["embed"]
	if embed.Type != "union" || embed.Closed {
		t.Fatalf("main.embed must remain an open union: %#v", embed)
	}
	if !slices.Contains(embed.Refs, "app.bsky.embed.video") {
		t.Fatalf("main.embed.refs = %v, want app.bsky.embed.video", embed.Refs)
	}

}

// REG-003: adding a standard external variant does not change local quote
// dispatch or the optional nature of the embed field.
func TestFeedPostEmbedCompatibility(t *testing.T) {
	t.Parallel()

	rawSchema, err := os.ReadFile("../../../../lexicon/social/craftsky/feed/post.json")
	if err != nil {
		t.Fatalf("read post lexicon: %v", err)
	}
	var schema struct {
		Defs map[string]struct {
			Record struct {
				Required   []string
				Properties map[string]struct {
					Refs []string `json:"refs"`
				}
			}
		}
	}
	if err := json.Unmarshal(rawSchema, &schema); err != nil {
		t.Fatalf("decode post lexicon: %v", err)
	}
	main := schema.Defs["main"].Record
	if slices.Contains(main.Required, "embed") {
		t.Fatal("main.embed must remain optional")
	}
	if refs := main.Properties["embed"].Refs; !slices.Contains(refs, "#quoteEmbed") {
		t.Fatalf("main.embed.refs = %v, want local quote variant", refs)
	}
}
