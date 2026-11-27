package api

import (
	"encoding/json"
)

type projectListFields struct{ legacy, list string }

var projectPatternFields = projectListFields{"pattern", "patterns"}

func projectToolSizeFields(detailType string) projectListFields {
	switch detailType {
	case "social.craftsky.project.knitting#details":
		return projectListFields{"needleSizeMm", "needleSizesMm"}
	case "social.craftsky.project.crochet#details":
		return projectListFields{"hookSizeMm", "hookSizesMm"}
	default:
		return projectListFields{}
	}
}

// The raw field's presence matters, even when the array is empty.
func (keys projectListFields) effective(fields map[string]json.RawMessage) json.RawMessage {
	if list, ok := fields[keys.list]; ok {
		return list
	}
	if value := fields[keys.legacy]; len(value) != 0 && string(value) != "null" {
		return append(append(json.RawMessage{'['}, value...), ']')
	}
	return nil
}

func (keys projectListFields) project(fields map[string]json.RawMessage) {
	list := keys.effective(fields)
	if len(list) == 0 {
		return
	}
	var entries []json.RawMessage
	if json.Unmarshal(list, &entries) == nil {
		fields[keys.list] = list
		delete(fields, keys.legacy)
		if len(entries) == 1 {
			fields[keys.legacy] = entries[0]
		}
	}
}

func projectSizeDetails(raw json.RawMessage, compatibility bool) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return raw, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	var detailType string
	_ = json.Unmarshal(fields["$type"], &detailType)
	keys := projectToolSizeFields(detailType)
	if keys.list == "" || keys.effective(fields) == nil {
		return raw, nil
	}
	if compatibility {
		keys.project(fields)
	} else {
		fields[keys.list] = keys.effective(fields)
		delete(fields, keys.legacy)
	}
	return json.Marshal(fields)
}

func projectForPublication(project *Project) (*Project, error) {
	if project == nil {
		return nil, nil
	}
	out := *project
	out.Common.Patterns = project.Common.effectivePatterns()
	out.Common.Pattern = nil
	details, err := projectSizeDetails(out.Details, false)
	out.Details = details
	return &out, err
}

func projectForResponse(project *Project) *Project {
	if project == nil {
		return nil
	}
	out := *project
	out.Common.Patterns = project.Common.effectivePatterns()
	out.Common.Pattern = nil
	if len(out.Common.Patterns) == 1 {
		out.Common.Pattern = out.Common.Patterns[0]
	}
	if details, err := projectSizeDetails(out.Details, true); err == nil {
		out.Details = details
	}
	return &out
}

func (common ProjectCommon) effectivePatterns() []*ProjectPattern {
	if common.Patterns != nil {
		return common.Patterns
	}
	if common.Pattern != nil {
		return []*ProjectPattern{common.Pattern}
	}
	return nil
}
