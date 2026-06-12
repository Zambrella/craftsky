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
	return extractTags(facets, nil)
}

// ExtractTagsForText walks facets and pulls hashtag-feature tags only from
// facets whose byte slice is valid for text. This is the safe helper for
// indexing user records where facet arrays are scoped to sibling strings.
func ExtractTagsForText(text string, facets []*appbsky.RichtextFacet) []string {
	return extractTags(facets, func(facet *appbsky.RichtextFacet) bool {
		return validFacetByteRange(text, facet)
	})
}

func extractTags(facets []*appbsky.RichtextFacet, valid func(*appbsky.RichtextFacet) bool) []string {
	if len(facets) == 0 {
		return []string{}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, facet := range facets {
		if facet == nil || (valid != nil && !valid(facet)) {
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

// ExtractMentionDIDs walks facets and pulls mention-feature DIDs. Trim, drop
// empties, dedupe (preserve first-seen order). Always returns a non-nil slice.
func ExtractMentionDIDs(facets []*appbsky.RichtextFacet) []string {
	return extractMentionDIDs(facets, nil)
}

// ExtractMentionDIDsForText walks facets and pulls mention-feature DIDs only
// from facets whose byte slice is valid for text.
func ExtractMentionDIDsForText(text string, facets []*appbsky.RichtextFacet) []string {
	return extractMentionDIDs(facets, func(facet *appbsky.RichtextFacet) bool {
		return validFacetByteRange(text, facet)
	})
}

func extractMentionDIDs(facets []*appbsky.RichtextFacet, valid func(*appbsky.RichtextFacet) bool) []string {
	if len(facets) == 0 {
		return []string{}
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, facet := range facets {
		if facet == nil || (valid != nil && !valid(facet)) {
			continue
		}
		for _, feat := range facet.Features {
			if feat == nil || feat.RichtextFacet_Mention == nil {
				continue
			}
			did := strings.TrimSpace(feat.RichtextFacet_Mention.Did)
			if did == "" {
				continue
			}
			if _, dup := seen[did]; dup {
				continue
			}
			seen[did] = struct{}{}
			out = append(out, did)
		}
	}
	return out
}

func validFacetByteRange(text string, facet *appbsky.RichtextFacet) bool {
	if facet == nil || facet.Index == nil {
		return false
	}
	start := facet.Index.ByteStart
	end := facet.Index.ByteEnd
	return start >= 0 && end > start && end <= int64(len([]byte(text)))
}

// MergeMentionDIDs trims, drops empties, and dedupes multiple DID sets while
// preserving first-seen order. It always returns a non-nil slice.
func MergeMentionDIDs(didSets ...[]string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, dids := range didSets {
		for _, raw := range dids {
			did := strings.TrimSpace(raw)
			if did == "" {
				continue
			}
			if _, ok := seen[did]; ok {
				continue
			}
			seen[did] = struct{}{}
			out = append(out, did)
		}
	}
	return out
}

// MergeTags lowercases, trims, drops empties, and dedupes multiple tag sets
// while preserving first-seen order. It always returns a non-nil slice so
// callers can store the result in NOT NULL array columns and JSON responses can
// serialize empty tags as [].
func MergeTags(tagSets ...[]string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, tags := range tagSets {
		for _, raw := range tags {
			tag := strings.ToLower(strings.TrimSpace(raw))
			if tag == "" {
				continue
			}
			if _, ok := seen[tag]; ok {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
		}
	}
	return out
}
