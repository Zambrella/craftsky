package craftsky

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	comatproto "github.com/bluesky-social/indigo/api/atproto"
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
	field := reflect.ValueOf(embed).Elem().FieldByName("EmbedExternal")
	if !field.IsValid() || field.IsNil() {
		t.Fatal("generated embed did not select app.bsky.embed.external")
	}
	if got := field.Type().String(); got != "*bsky.EmbedExternal" {
		t.Fatalf("generated external type = %s, want *bsky.EmbedExternal", got)
	}
}

// IT-006: standard video is an additive branch of the optional open post embed
// union and uses Indigo's generated app.bsky.embed.video type.
func TestFeedPostVideoEmbedContract(t *testing.T) {
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

	fixtures := []struct {
		name string
		json string
	}{
		{
			name: "minimal",
			json: `{
				"$type":"social.craftsky.feed.post",
				"text":"Video post",
				"createdAt":"2026-09-03T00:00:00Z",
				"embed":{
					"$type":"app.bsky.embed.video",
					"video":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":123}
				}
			}`,
		},
		{
			name: "full",
			json: `{
				"$type":"social.craftsky.feed.post",
				"text":"Accessible video post",
				"createdAt":"2026-09-03T00:00:00Z",
				"embed":{
					"$type":"app.bsky.embed.video",
					"video":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":300000000},
					"alt":"Hands knitting a scarf",
					"aspectRatio":{"width":16,"height":9},
					"captions":[{"lang":"en","file":{"$type":"blob","ref":{"$link":"bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"},"mimeType":"text/vtt","size":42}}]
				}
			}`,
		},
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			var post FeedPost
			if err := json.Unmarshal([]byte(fixture.json), &post); err != nil {
				t.Fatalf("decode video post: %v", err)
			}
			assertGeneratedVideoEmbed(t, post.Embed)

			jsonRoundTrip, err := json.Marshal(&post)
			if err != nil {
				t.Fatalf("marshal video post as JSON: %v", err)
			}
			var jsonWire map[string]any
			if err := json.Unmarshal(jsonRoundTrip, &jsonWire); err != nil {
				t.Fatalf("decode video JSON round trip: %v", err)
			}
			if got := jsonWire["embed"].(map[string]any)["$type"]; got != "app.bsky.embed.video" {
				t.Fatalf("video JSON $type = %v", got)
			}

			var cbor bytes.Buffer
			if err := post.MarshalCBOR(&cbor); err != nil {
				t.Fatalf("marshal video post as CBOR: %v", err)
			}
			var decoded FeedPost
			if err := decoded.UnmarshalCBOR(bytes.NewReader(cbor.Bytes())); err != nil {
				t.Fatalf("decode video CBOR round trip: %v", err)
			}
			assertGeneratedVideoEmbed(t, decoded.Embed)
		})
	}
}

func assertGeneratedVideoEmbed(t *testing.T, embed *FeedPost_Embed) {
	t.Helper()
	if embed == nil {
		t.Fatal("generated embed is nil")
	}
	field := reflect.ValueOf(embed).Elem().FieldByName("EmbedVideo")
	if !field.IsValid() || field.IsNil() {
		t.Fatal("generated embed did not select app.bsky.embed.video")
	}
	if got := field.Type().String(); got != "*bsky.EmbedVideo" {
		t.Fatalf("generated video type = %s, want *bsky.EmbedVideo", got)
	}
}

// REG-003: adding a standard external variant does not change local quote
// dispatch or the optional nature of the embed field.
func TestFeedPostEmbedCompatibility(t *testing.T) {
	t.Parallel()

	quote := FeedPost{
		LexiconTypeID: "social.craftsky.feed.post",
		Text:          "Quoting a useful post",
		CreatedAt:     "2026-08-25T00:00:00Z",
		Embed: &FeedPost_Embed{FeedPost_QuoteEmbed: &FeedPost_QuoteEmbed{
			Record: &comatproto.RepoStrongRef{
				Uri: "at://did:plc:author/social.craftsky.feed.post/3mquote",
				Cid: "bafy-quote",
			},
		}},
	}
	encodedQuote, err := json.Marshal(&quote)
	if err != nil {
		t.Fatalf("marshal quote JSON: %v", err)
	}
	var quoteWire map[string]any
	if err := json.Unmarshal(encodedQuote, &quoteWire); err != nil {
		t.Fatalf("decode quote JSON: %v", err)
	}
	if got := quoteWire["embed"].(map[string]any)["$type"]; got != "social.craftsky.feed.post#quoteEmbed" {
		t.Fatalf("quote JSON $type = %v", got)
	}

	var cbor bytes.Buffer
	if err := quote.MarshalCBOR(&cbor); err != nil {
		t.Fatalf("marshal quote CBOR: %v", err)
	}
	var decodedQuote FeedPost
	if err := decodedQuote.UnmarshalCBOR(bytes.NewReader(cbor.Bytes())); err != nil {
		t.Fatalf("decode quote CBOR: %v", err)
	}
	if decodedQuote.Embed == nil || decodedQuote.Embed.FeedPost_QuoteEmbed == nil {
		t.Fatal("generated embed did not retain local quote variant")
	}

	ordinary := FeedPost{
		LexiconTypeID: "social.craftsky.feed.post",
		Text:          "Ordinary post",
		CreatedAt:     "2026-08-25T00:00:00Z",
	}
	encodedOrdinary, err := json.Marshal(&ordinary)
	if err != nil {
		t.Fatalf("marshal ordinary post: %v", err)
	}
	var ordinaryWire map[string]any
	if err := json.Unmarshal(encodedOrdinary, &ordinaryWire); err != nil {
		t.Fatalf("decode ordinary JSON: %v", err)
	}
	if _, ok := ordinaryWire["embed"]; ok {
		t.Fatalf("ordinary post unexpectedly contains embed: %s", encodedOrdinary)
	}
}
