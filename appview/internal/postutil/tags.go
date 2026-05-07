// Package postutil holds small lexicon-derived helpers shared between
// the firehose indexer and the API handlers. Keeping them in a neutral
// package avoids an api → index import cycle and ensures both surfaces
// produce identical materialised values.
package postutil

import (
	"strings"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
)

// ExtractTags walks facets and pulls hashtag-feature tags. Lowercase,
// trim, drop empties, dedupe (preserve first-seen order). Always
// returns a non-nil slice — callers store this in a NOT NULL column.
func ExtractTags(facets []*appbsky.RichtextFacet) []string {
	if len(facets) == 0 {
		return []string{}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, facet := range facets {
		if facet == nil {
			continue
		}
		for _, feat := range facet.Features {
			if feat == nil || feat.RichtextFacet_Tag == nil {
				continue
			}
			t := strings.ToLower(strings.TrimSpace(feat.RichtextFacet_Tag.Tag))
			if t == "" {
				continue
			}
			if _, dup := seen[t]; dup {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}
