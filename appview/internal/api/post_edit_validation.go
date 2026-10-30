package api

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

func validatePostEditRichText(candidate PostCreateRequest) error {
	if err := validatePostEditFacets(candidate.Text, candidate.Facets, "facets"); err != nil {
		return err
	}
	if candidate.Project == nil {
		return nil
	}
	common := candidate.Project.Common
	for index, material := range common.Materials {
		if err := validatePostEditFacets(material.Text, material.Facets, fmt.Sprintf("project.common.materials[%d].facets", index)); err != nil {
			return err
		}
	}
	if pattern := common.Pattern; pattern != nil {
		for _, field := range []struct {
			name   string
			text   *string
			facets json.RawMessage
		}{
			{"nameFacets", pattern.Name, pattern.NameFacets},
			{"designerFacets", pattern.Designer, pattern.DesignerFacets},
			{"publisherFacets", pattern.Publisher, pattern.PublisherFacets},
		} {
			text := ""
			if field.text != nil {
				text = *field.text
			}
			if err := validatePostEditFacets(text, field.facets, "project.common.pattern."+field.name); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePostEditFacets(text string, raw json.RawMessage, field string) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var facets []struct {
		Index *struct {
			Start int64 `json:"byteStart"`
			End   int64 `json:"byteEnd"`
		} `json:"index"`
	}
	invalid := func() error {
		return &FieldError{Code: "validation_failed", Fields: map[string]string{field: "must use UTF-8 byte ranges within the corresponding text"}}
	}
	if err := json.Unmarshal(raw, &facets); err != nil {
		return invalid()
	}
	for _, facet := range facets {
		if facet.Index == nil {
			return invalid()
		}
		start, end := facet.Index.Start, facet.Index.End
		if start < 0 || end <= start || end > int64(len(text)) || !utf8.ValidString(text[start:end]) {
			return invalid()
		}
	}
	return nil
}
