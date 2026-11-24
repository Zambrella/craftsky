package api

import (
	"encoding/json"

	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
)

// postEditEditable uses generated read types to expose only supported values.
// The write path always merges against the authoritative raw server record.
func postEditEditable(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var post craftskylex.FeedPost
	if err := json.Unmarshal(raw, &post); err != nil {
		return nil, err
	}
	unsupportedDetails := post.Project != nil && post.Project.Details != nil && post.Project.Details.Unknown != nil
	unsupportedEmbed := post.Embed != nil && post.Embed.Unknown != nil
	if unsupportedDetails {
		post.Project.Details = nil
	}
	if unsupportedEmbed {
		post.Embed = nil
	}
	encoded, err := json.Marshal(post)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	for _, fixed := range []string{"$type", "createdAt", "editedAt", "reply", "externalImport", "crafts"} {
		delete(fields, fixed)
	}
	if unsupportedDetails {
		fields["unsupportedProjectDetails"] = json.RawMessage(`true`)
	}
	if unsupportedEmbed {
		fields["unsupportedEmbed"] = json.RawMessage(`true`)
	}
	if images, present := fields["images"]; present {
		fields["images"], err = postEditIndexedEntries(images)
		if err != nil {
			return nil, err
		}
	}
	if project, present := fields["project"]; present {
		var source struct{ Project *Project }
		if err := json.Unmarshal(raw, &source); err != nil {
			return nil, err
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(project, &values); err != nil {
			return nil, err
		}
		var common map[string]json.RawMessage
		if err := json.Unmarshal(values["common"], &common); err != nil {
			return nil, err
		}
		if list := source.Project.Common.Patterns; list != nil && len(list) == 0 {
			common["patterns"] = json.RawMessage("[]")
		}
		projectPatternFields.project(common)
		for _, name := range []string{"materials", "additionalCrafts", "patterns"} {
			if entries, present := common[name]; present {
				common[name], err = postEditIndexedEntries(entries)
				if err != nil {
					return nil, err
				}
			}
		}
		if details, present := values["details"]; present {
			var detailFields map[string]json.RawMessage
			if err := json.Unmarshal(details, &detailFields); err != nil {
				return nil, err
			}
			var variant string
			_ = json.Unmarshal(detailFields["$type"], &variant)
			if keys := projectToolSizeFields(variant); keys.list != "" {
				var rawDetails map[string]json.RawMessage
				if err := json.Unmarshal(source.Project.Details, &rawDetails); err != nil {
					return nil, err
				}
				if list, present := rawDetails[keys.list]; present {
					detailFields[keys.list] = list
				}
				keys.project(detailFields)
			}
			if err := projectDetailEdits[variant].indexChildren(detailFields); err != nil {
				return nil, err
			}
			values["details"], err = json.Marshal(detailFields)
			if err != nil {
				return nil, err
			}
		}
		values["common"], err = json.Marshal(common)
		if err != nil {
			return nil, err
		}
		fields["project"], err = json.Marshal(values)
		if err != nil {
			return nil, err
		}
	}
	return fields, nil
}

func postEditIndexedEntries(raw json.RawMessage) (json.RawMessage, error) {
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	for index, entry := range entries {
		value, err := json.Marshal(index)
		if err != nil {
			return nil, err
		}
		entry["existingIndex"] = value
	}
	return json.Marshal(entries)
}
