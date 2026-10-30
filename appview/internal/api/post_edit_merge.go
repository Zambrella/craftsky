package api

import (
	"bytes"
	"encoding/json"
	"errors"

	"social.craftsky/appview/internal/projectcatalog"
)

type postEditObjectMerger func(json.RawMessage, json.RawMessage) (json.RawMessage, error)

func mergePostEditAspectRatio(base, changes json.RawMessage) (json.RawMessage, error) {
	return mergeSupportedPostEditObject(base, changes, []string{"width", "height"}, nil)
}

// mergeSupportedPostEditObject applies only a server-selected field inventory.
// Unknown authored siblings remain in the authoritative raw object.
func mergeSupportedPostEditObject(base, changes json.RawMessage, allowed []string, nested map[string]postEditObjectMerger) (json.RawMessage, error) {
	fields := map[string]json.RawMessage{}
	if len(base) != 0 && !bytes.Equal(bytes.TrimSpace(base), []byte("null")) {
		if err := json.Unmarshal(base, &fields); err != nil {
			return nil, err
		}
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(changes, &patch); err != nil {
		return nil, err
	}
	if patch == nil {
		return nil, errors.New("post edit requires an object")
	}
	for _, pair := range [][2]string{{"text", "facets"}, {"name", "nameFacets"}, {"designer", "designerFacets"}, {"publisher", "publisherFacets"}} {
		changed, present := patch[pair[0]]
		if _, supplied := patch[pair[1]]; !present || supplied {
			continue
		}
		var originalText, changedText string
		_ = json.Unmarshal(fields[pair[0]], &originalText)
		_ = json.Unmarshal(changed, &changedText)
		if originalText != changedText {
			delete(fields, pair[1])
		}
	}
	for key, value := range patch {
		known := false
		for _, field := range allowed {
			if field == key {
				known = true
				break
			}
		}
		if !known {
			return nil, errors.New("unsupported post edit field")
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			delete(fields, key)
			continue
		}
		if merge := nested[key]; merge != nil {
			merged, err := merge(fields[key], value)
			if err != nil {
				return nil, err
			}
			fields[key] = merged
		} else {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}

func mergePostProjectEdit(base, changes json.RawMessage) (json.RawMessage, error) {
	if len(base) == 0 || bytes.Equal(bytes.TrimSpace(base), []byte("null")) || bytes.Equal(bytes.TrimSpace(changes), []byte("null")) {
		return nil, errors.New("post type cannot be changed")
	}
	var original struct {
		Details struct {
			Type string `json:"$type"`
		}
	}
	if err := json.Unmarshal(base, &original); err != nil {
		return nil, err
	}
	if original.Details.Type != "" {
		if _, supported := projectcatalog.DetailCraft(original.Details.Type); !supported {
			var patch map[string]json.RawMessage
			if err := json.Unmarshal(changes, &patch); err != nil {
				return nil, err
			}
			if _, changed := patch["details"]; changed {
				return nil, errors.New("unsupported project details must be preserved")
			}
		}
	}
	return mergeSupportedPostEditObject(base, changes, []string{"common", "details"}, map[string]postEditObjectMerger{
		"details": mergePostDetailsEdit,
		"common": func(base, changes json.RawMessage) (json.RawMessage, error) {
			return mergeSupportedPostEditObject(base, changes, projectCommonKeys, map[string]postEditObjectMerger{
				"materials":        mergePostEditEntries(projectMaterialKeys),
				"additionalCrafts": mergePostEditEntries(projectCraftKeys),
				"pattern": func(base, changes json.RawMessage) (json.RawMessage, error) {
					return mergeSupportedPostEditObject(base, changes, projectPatternKeys, nil)
				},
			})
		},
	})
}

func mergePostDetailsEdit(base, changes json.RawMessage) (json.RawMessage, error) {
	var original, patch struct {
		Type string `json:"$type"`
	}
	if len(base) > 0 {
		if err := json.Unmarshal(base, &original); err != nil {
			return nil, err
		}
	}
	if err := json.Unmarshal(changes, &patch); err != nil {
		return nil, err
	}
	variant := original.Type
	if patch.Type != "" {
		variant = patch.Type
	}
	var allowed []string
	switch variant {
	case "social.craftsky.project.sewing#details":
		allowed = []string{"$type", "projectType", "projectSubtype", "sizeMade", "fitNotes"}
	case "social.craftsky.project.knitting#details":
		allowed = []string{"$type", "projectType", "projectSubtype", "yarnWeight", "needleSizeMm", "gauge", "finishedSize"}
	case "social.craftsky.project.crochet#details":
		allowed = []string{"$type", "projectType", "projectSubtype", "yarnWeight", "hookSizeMm", "gauge", "finishedSize"}
	case "social.craftsky.project.quilting#details":
		allowed = []string{"$type", "projectType", "projectSubtype", "size", "piecingTechnique", "quiltingMethod"}
	default:
		return nil, errors.New("unsupported project detail edit")
	}
	if variant != original.Type {
		base = nil
	}
	return mergeSupportedPostEditObject(base, changes, allowed, map[string]postEditObjectMerger{
		"gauge": func(base, changes json.RawMessage) (json.RawMessage, error) {
			return mergeSupportedPostEditObject(base, changes, []string{"stitches", "rows", "measurement", "unit"}, nil)
		},
	})
}

func mergePostEditEntries(allowed []string) postEditObjectMerger {
	return func(base, changes json.RawMessage) (json.RawMessage, error) {
		var originals, entries []json.RawMessage
		if len(base) > 0 && !bytes.Equal(bytes.TrimSpace(base), []byte("null")) {
			if err := json.Unmarshal(base, &originals); err != nil {
				return nil, err
			}
		}
		if err := json.Unmarshal(changes, &entries); err != nil {
			return nil, err
		}
		if entries == nil {
			return nil, errors.New("post edit entries require an array")
		}
		result := make([]json.RawMessage, 0, len(entries))
		seen := map[int]bool{}
		for _, entry := range entries {
			var patch map[string]json.RawMessage
			if err := json.Unmarshal(entry, &patch); err != nil {
				return nil, err
			}
			var original json.RawMessage
			if raw, present := patch["existingIndex"]; present {
				var index int
				if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &index) != nil || index < 0 || index >= len(originals) || seen[index] {
					return nil, errors.New("invalid retained post edit entry")
				}
				seen[index] = true
				original = originals[index]
				delete(patch, "existingIndex")
			}
			patchRaw, err := json.Marshal(patch)
			if err != nil {
				return nil, err
			}
			merged, err := mergeSupportedPostEditObject(original, patchRaw, allowed, map[string]postEditObjectMerger{"aspectRatio": mergePostEditAspectRatio})
			if err != nil {
				return nil, err
			}
			result = append(result, merged)
		}
		return json.Marshal(result)
	}
}
