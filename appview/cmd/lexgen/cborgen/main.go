// cborgen runs cbor-gen against the generated craftsky lexicon package
// to produce internal/lexicon/craftsky/cbor_gen.go. Invoked by `just
// lexgen` after the indigo lexgen step. See
// docs/superpowers/specs/2026-04-26-lexicon-codegen-design.md §3.1.
//
// When a lexicon edit introduces a new top-level struct or open-union
// variant, add the type below and re-run `just lexgen`.
package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"

	craftsky "social.craftsky/appview/internal/lexicon/craftsky"
)

func main() {
	gen := cbg.Gen{MaxStringLength: 1_000_000}

	if err := gen.WriteMapEncodersToFile(
		"internal/lexicon/craftsky/cbor_gen.go",
		"craftsky",
		craftsky.ActorProfile{},
		craftsky.FeedLike{},
		craftsky.FeedRepost{},
		craftsky.FeedPost{},
		craftsky.FeedPost_Image{},
		craftsky.FeedPost_Pattern{},
		craftsky.FeedPost_Project{},
		craftsky.FeedPost_ProjectCommon{},
		craftsky.FeedPost_QuoteEmbed{},
		craftsky.FeedPost_ReplyRef{},
		craftsky.ProjectSewing_Details{},
	); err != nil {
		panic(err)
	}
}
