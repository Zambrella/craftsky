package api

import "encoding/json"

const (
	ProjectCraftTypeKnitting   = "social.craftsky.feed.defs#knitting"
	ProjectCraftTypeCrochet    = "social.craftsky.feed.defs#crochet"
	ProjectCraftTypeSewing     = "social.craftsky.feed.defs#sewing"
	ProjectCraftTypeEmbroidery = "social.craftsky.feed.defs#embroidery"
	ProjectCraftTypeQuilting   = "social.craftsky.feed.defs#quilting"
)

func IsSupportedProjectCraftType(craftType string) bool {
	switch craftType {
	case ProjectCraftTypeKnitting,
		ProjectCraftTypeCrochet,
		ProjectCraftTypeSewing,
		ProjectCraftTypeEmbroidery,
		ProjectCraftTypeQuilting:
		return true
	default:
		return false
	}
}

// Project mirrors social.craftsky.project.defs#project for /v1 post request
// and response bodies. Details stay as raw JSON so future open-union variants
// can round-trip without AppView codegen knowing their shape.
type Project struct {
	Common  ProjectCommon   `json:"common"`
	Details json.RawMessage `json:"details,omitempty"`
}

type ProjectCommon struct {
	CraftType  string          `json:"craftType"`
	Status     *string         `json:"status,omitempty"`
	Title      *string         `json:"title,omitempty"`
	Duration   *string         `json:"duration,omitempty"`
	Pattern    *ProjectPattern `json:"pattern,omitempty"`
	Materials  []string        `json:"materials,omitempty"`
	Colors     []string        `json:"colors,omitempty"`
	DesignTags []string        `json:"designTags,omitempty"`
	Tags       []string        `json:"tags,omitempty"`
}

type ProjectPattern struct {
	URL        *string `json:"url,omitempty"`
	Name       *string `json:"name,omitempty"`
	Difficulty *string `json:"difficulty,omitempty"`
	Designer   *string `json:"designer,omitempty"`
	Publisher  *string `json:"publisher,omitempty"`
}
