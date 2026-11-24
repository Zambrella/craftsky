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

	community "social.craftsky/appview/internal/lexicon/community"
	craftsky "social.craftsky/appview/internal/lexicon/craftsky"
)

func main() {
	gen := cbg.Gen{MaxStringLength: 1_000_000}

	if err := gen.WriteMapEncodersToFile(
		"internal/lexicon/community/cbor_gen.go",
		"community",
		community.LocationAddress{},
	); err != nil {
		panic(err)
	}

	if err := gen.WriteMapEncodersToFile(
		"internal/lexicon/craftsky/cbor_gen.go",
		"craftsky",
		craftsky.ActorProfile{},
		craftsky.BusinessDefs_Action{},
		craftsky.BusinessDefs_AspectRatio{},
		craftsky.BusinessDefs_Image{},
		craftsky.BusinessDefs_Price{},
		craftsky.BusinessDefs_Product{},
		craftsky.BusinessEvent{},
		craftsky.BusinessProfile{},
		craftsky.FeedLike{},
		craftsky.FeedRepost{},
		craftsky.FeedPost{},
		craftsky.FeedPost_AspectRatio{},
		craftsky.FeedPost_Craft{},
		craftsky.FeedPost_ExternalImport{},
		craftsky.FeedPost_Image{},
		craftsky.FeedPost_QuoteEmbed{},
		craftsky.FeedPost_ReplyRef{},
		craftsky.ProjectCrochet_Details{},
		craftsky.ProjectCrochet_TechniqueSelection{},
		craftsky.ProjectDefs_AdditionalCraft{},
		craftsky.ProjectDefs_Gauge{},
		craftsky.ProjectDefs_LengthMeasurement{},
		craftsky.ProjectDefs_Material{},
		craftsky.ProjectDefs_MassMeasurement{},
		craftsky.ProjectDefs_Pattern{},
		craftsky.ProjectDefs_Project{},
		craftsky.ProjectDefs_ProjectCommon{},
		craftsky.ProjectDyeing_Details{},
		craftsky.ProjectEmbroidery_Details{},
		craftsky.ProjectEmbroidery_MethodSelection{},
		craftsky.ProjectEmbroidery_StyleSelection{},
		craftsky.ProjectFelting_Details{},
		craftsky.ProjectFelting_NeedleGauge{},
		craftsky.ProjectKnitting_Details{},
		craftsky.ProjectKnitting_TechniqueSelection{},
		craftsky.ProjectKnitting_TechniqueSubtype{},
		craftsky.ProjectMacrame_Cord{},
		craftsky.ProjectMacrame_Details{},
		craftsky.ProjectMacrame_KnotSelection{},
		craftsky.ProjectPunchNeedle_Details{},
		craftsky.ProjectQuilting_Details{},
		craftsky.ProjectSewing_Details{},
		craftsky.ProjectSewing_TechniqueSelection{},
		craftsky.ProjectSpinning_Details{},
		craftsky.ProjectSpinning_DraftingTechnique{},
		craftsky.ProjectSpinning_FinishedYarn{},
		craftsky.ProjectSpinning_PlyingTechnique{},
		craftsky.ProjectWeaving_Details{},
		craftsky.ProjectWeaving_Material{},
		craftsky.ProjectWeaving_Sett{},
	); err != nil {
		panic(err)
	}
}
