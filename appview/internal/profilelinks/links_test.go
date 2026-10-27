package profilelinks

import (
	"encoding/json"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/lexicon/schema"
	"strings"
	"testing"
)

func TestDestinationPolicy(t *testing.T) {
	for _, raw := range []string{"http://example.com/", "javascript:alert(1)", "https://user:secret@example.com/", "https://bit.ly/abc", "https://www.tinyurl.com/a"} {
		if _, err := Normalize(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	if got, err := Normalize(" example.com/shop "); err != nil || got != "https://example.com/shop" {
		t.Fatalf("normalization=%q %v", got, err)
	}
	for _, tc := range []struct{ url, name string }{
		{"https://ko-fi.com/maker", "Ko-fi"}, {"https://etsy.com/shop/maker", "Etsy"},
		{"https://ko-fi.com/", ""}, {"https://ko-fi.com.example.net/maker", ""},
		{"https://paypal.com/paypalme/maker", "PayPal"}, {"https://paypal.com/about", ""},
		{"https://maker.itch.io/", "itch.io"},
	} {
		provider := Support(tc.url)
		got := ""
		if provider != nil {
			got = provider.Name
		}
		if got != tc.name {
			t.Fatalf("%s => %q", tc.url, got)
		}
	}
}

func TestPublishedSchemaAndEditorTitleRemainDistinct(t *testing.T) {
	record := map[string]any{"$type": Collection.String(), "url": "https://example.com/", "createdAt": "2026-10-27T12:00:00Z", "title": strings.Repeat("🧶", 40)}
	raw, _ := json.Marshal(record)
	if err := schema.ValidateCraftskyRecord(raw, Collection.String()); err != nil {
		t.Fatal(err)
	}
	if got := Title(record["title"].(string), 30); got != strings.Repeat("🧶", 30) {
		t.Fatal("editor split or exceeded graphemes")
	}
	record["title"] = strings.Repeat("a", 41)
	raw, _ = json.Marshal(record)
	if schema.ValidateCraftskyRecord(raw, Collection.String()) == nil {
		t.Fatal("accepted overlong title")
	}
}

func TestRecordCIDUsesCanonicalProtocolData(t *testing.T) {
	a := json.RawMessage(`{"$type":"app.bsky.actor.link","url":"https://example.com/","createdAt":"2026-10-27T12:00:00Z"}`)
	b := json.RawMessage(`{"createdAt":"2026-10-27T12:00:00Z", "url":"https://example.com/", "$type":"app.bsky.actor.link"}`)
	ac, err := RecordCID(a)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := RecordCID(b)
	if err != nil || ac != bc {
		t.Fatal("CID depends on JSON order or whitespace")
	}
	if !OwnRef(syntax.DID("did:plc:owner"), "at://did:plc:owner/app.bsky.actor.link/3m5abc2345672") || OwnRef("did:plc:owner", "at://did:plc:other/app.bsky.actor.link/3m5abc2345672") {
		t.Fatal("cross-repository reference accepted")
	}
}
