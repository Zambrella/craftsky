package postpublication

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestPublicationMentionsUseAllEffectivePatterns(t *testing.T) {
	const common = `"craftType":"social.craftsky.feed.defs#sewing",
		"pattern":{"publisher":"@stale","publisherFacets":[{"index":{"byteStart":0,"byteEnd":6},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:stale"}]}]},
		"patterns":[
			{"designer":"@mina","designerFacets":[{"index":{"byteStart":0,"byteEnd":5},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:mina"}]}]},
			{"publisher":"@mina @loom","publisherFacets":[{"index":{"byteStart":0,"byteEnd":5},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:mina"}]},{"index":{"byteStart":6,"byteEnd":11},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:loom"}]}]}
		]`
	raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"Two patterns","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","project":{"common":{` + common + `}}}`)
	facts, err := factsForRecord(raw, true)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(facts.Mentions)
	if !slices.Equal(facts.Mentions, []string{"did:plc:loom", "did:plc:mina"}) {
		t.Fatalf("wrong publication mentions: %v", facts.Mentions)
	}
}
