package craftsky

import (
	"os"
	"strings"
	"testing"
)

func TestIT008ProductionExternalAuthoringUsesGeneratedUnionTypes(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"../../api/post_create.go",
		"../../scheduledposts/publication_processor.go",
	} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(source), "postrecord.ExternalEmbed(") {
			t.Fatalf("%s does not use the shared generated external authoring boundary", path)
		}
	}
	helper, err := os.ReadFile("../../postrecord/external.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(helper)
	for _, generatedType := range []string{
		"craftsky.FeedPost_Embed",
		"appbsky.EmbedExternal",
		"appbsky.EmbedExternal_External",
	} {
		if !strings.Contains(source, generatedType) {
			t.Fatalf("generated authoring helper does not construct %s", generatedType)
		}
	}
}
