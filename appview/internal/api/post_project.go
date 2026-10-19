package api

import (
	"encoding/json"
	"social.craftsky/appview/internal/projectcatalog"
)

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
	ExtraFields map[string]json.RawMessage `json:"-"`
	Common      ProjectCommon              `json:"common"`
	Details     json.RawMessage            `json:"details,omitempty"`
}

type ProjectCommon struct {
	ExtraFields      map[string]json.RawMessage `json:"-"`
	AdditionalCrafts []ProjectCraft             `json:"additionalCrafts,omitempty"`
	CraftType        string                     `json:"craftType"`
	Status           *string                    `json:"status,omitempty"`
	Title            *string                    `json:"title,omitempty"`
	Duration         *string                    `json:"duration,omitempty"`
	Pattern          *ProjectPattern            `json:"pattern,omitempty"`
	Materials        []ProjectMaterial          `json:"materials,omitempty"`
	Colors           []string                   `json:"colors,omitempty"`
	DesignTags       []string                   `json:"designTags,omitempty"`
	Tags             []string                   `json:"tags,omitempty"`
}

type ProjectCraft struct {
	ExtraFields map[string]json.RawMessage `json:"-"`
	CraftType   string                     `json:"craftType"`
}

type ProjectMaterial struct {
	ExtraFields map[string]json.RawMessage `json:"-"`
	Text        string                     `json:"text"`
	Facets      json.RawMessage            `json:"facets,omitempty"`
}

type ProjectPattern struct {
	ExtraFields     map[string]json.RawMessage `json:"-"`
	URL             *string                    `json:"url,omitempty"`
	Name            *string                    `json:"name,omitempty"`
	NameFacets      json.RawMessage            `json:"nameFacets,omitempty"`
	Difficulty      *string                    `json:"difficulty,omitempty"`
	Designer        *string                    `json:"designer,omitempty"`
	DesignerFacets  json.RawMessage            `json:"designerFacets,omitempty"`
	Publisher       *string                    `json:"publisher,omitempty"`
	PublisherFacets json.RawMessage            `json:"publisherFacets,omitempty"`
	SelfDrafted     *bool                      `json:"selfDrafted,omitempty"`
}

// Recognized details belong to the primary craft. Additional crafts describe
// participation, and do not reinterpret the primary detail object.
func validateProjectCrafts(fields map[string]string, project *Project) {
	var details struct {
		Type string `json:"$type"`
	}
	if len(project.Details) > 0 && json.Unmarshal(project.Details, &details) == nil {
		if craft, known := projectcatalog.DetailCraft(details.Type); known && craft != project.Common.CraftType {
			fields["project.details"] = "must correspond to the primary craft type"
		}
	}
	seen := map[string]bool{project.Common.CraftType: true}
	for _, craft := range project.Common.AdditionalCrafts {
		if !IsSupportedProjectCraftType(craft.CraftType) || seen[craft.CraftType] {
			fields["project.common.additionalCrafts"] = "must contain distinct supported crafts other than the primary craft"
		}
		seen[craft.CraftType] = true
	}
}
