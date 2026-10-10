package postutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
)

// ErrAmbiguousTagFields contains no source payload or field values.
var ErrAmbiguousTagFields = errors.New("ambiguous_tag_fields")

// These are only fields that influence recognized hashtag-source decoding.
// Unknown fields and feature payloads retain best-effort semantics. Keep the
// migration's craftsky_tag_source_unambiguous rules aligned with these shapes.
var tagSourceFields = map[string]map[string]string{
	"post":     {"text": "", "facets": "facet[]", "project": "project"},
	"project":  {"common": "common"},
	"common":   {"craftType": "", "tags": "", "pattern": "pattern", "materials": "material[]"},
	"pattern":  {"name": "", "nameFacets": "facet[]", "designer": "", "designerFacets": "facet[]", "publisher": "", "publisherFacets": "facet[]"},
	"material": {"text": "", "facets": "facet[]"},
	"facet":    {"index": "index", "features": "feature[]"},
	"index":    {"byteStart": "", "byteEnd": ""},
	"feature":  {"$type": ""},
}

// HasAmbiguousTagFields examines original JSON tokens, including duplicate
// exact keys, before map decoding or JSONB storage can discard information.
// It is an ambiguity check, not a replacement for record/facet validation.
func HasAmbiguousTagFields(raw json.RawMessage) bool {
	return ambiguousTagFields(raw, "post")
}

func ambiguousTagFields(raw json.RawMessage, shape string) bool {
	if strings.HasSuffix(shape, "[]") {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return false
		}
		for _, item := range items {
			if ambiguousTagFields(item, strings.TrimSuffix(shape, "[]")) {
				return true
			}
		}
		return false
	}
	rules := tagSourceFields[shape]
	featureField := ""
	if shape == "feature" {
		var feature struct {
			Type string `json:"$type"`
		}
		_ = json.Unmarshal(raw, &feature)
		switch feature.Type {
		case "app.bsky.richtext.facet#tag":
			featureField = "tag"
		case "app.bsky.richtext.facet#mention":
			featureField = "did"
		case "app.bsky.richtext.facet#link":
			featureField = "uri"
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return false
	}
	seen := make(map[string]bool, len(rules)+1)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		key, ok := token.(string)
		if !ok {
			return false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return false
		}
		canonical, child := "", ""
		for field, childShape := range rules {
			if strings.EqualFold(key, field) {
				canonical, child = field, childShape
				break
			}
		}
		if featureField != "" && strings.EqualFold(key, featureField) {
			canonical = featureField
		}
		if canonical == "" {
			continue
		}
		if seen[canonical] {
			return true
		}
		seen[canonical] = true
		if child != "" && ambiguousTagFields(value, child) {
			return true
		}
	}
	return false
}
