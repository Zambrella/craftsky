package craftsky

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// IT-001, REG-001: the optional generated contract remains minimal,
// self-asserted, open to future source tokens, and absent from ordinary posts.
func TestFeedPostExternalImportContract(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../../../lexicon/social/craftsky/feed/post.json")
	if err != nil {
		t.Fatalf("read post lexicon: %v", err)
	}
	var schema struct {
		Defs map[string]struct {
			Description string `json:"description"`
			Required    []string
			Properties  map[string]struct {
				Type        string   `json:"type"`
				Ref         string   `json:"ref"`
				MaxLength   int      `json:"maxLength"`
				KnownValues []string `json:"knownValues"`
				Description string   `json:"description"`
			}
			Record struct {
				Properties map[string]struct {
					Ref string `json:"ref"`
				}
			}
		}
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("decode post lexicon: %v", err)
	}

	main := schema.Defs["main"]
	if got := main.Record.Properties["externalImport"].Ref; got != "#externalImport" {
		t.Fatalf("main.externalImport.ref = %q, want #externalImport", got)
	}
	externalImport := schema.Defs["externalImport"]
	if len(externalImport.Required) != 1 || externalImport.Required[0] != "source" {
		t.Fatalf("externalImport.required = %v, want [source]", externalImport.Required)
	}
	source := externalImport.Properties["source"]
	if source.Type != "string" || source.MaxLength != 64 {
		t.Fatalf("externalImport.source type/maxLength = %q/%d, want string/64", source.Type, source.MaxLength)
	}
	if len(source.KnownValues) != 1 || source.KnownValues[0] != "instagram" {
		t.Fatalf("externalImport.source.knownValues = %v, want [instagram]", source.KnownValues)
	}
	description := strings.ToLower(externalImport.Description + " " + source.Description)
	if !strings.Contains(description, "self-asserted") || !strings.Contains(description, "does not verify") {
		t.Fatalf("provenance description must explicitly be self-asserted and non-verifying: %q", description)
	}

	sourceToken := "instagram"
	post := FeedPost{
		LexiconTypeID: "social.craftsky.feed.post",
		Text:          "historical post",
		CreatedAt:     "2020-01-02T03:04:05Z",
		ExternalImport: &FeedPost_ExternalImport{
			Source: sourceToken,
		},
	}
	encoded, err := json.Marshal(post)
	if err != nil {
		t.Fatalf("marshal generated post: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("decode generated post: %v", err)
	}
	gotExternal, ok := wire["externalImport"].(map[string]any)
	if !ok || gotExternal["source"] != "instagram" {
		t.Fatalf("generated externalImport = %#v", wire["externalImport"])
	}

	post.ExternalImport = nil
	encoded, err = json.Marshal(post)
	if err != nil {
		t.Fatalf("marshal ordinary generated post: %v", err)
	}
	if strings.Contains(string(encoded), "externalImport") {
		t.Fatalf("ordinary generated post unexpectedly contains externalImport: %s", encoded)
	}
}
