package api

import "encoding/json"

// Project extension fields are public authored data. Retain them losslessly;
// typed fields always win, including when a caller clears an optional value.
func decodeProjectObject(raw []byte, target any, keys []string) (map[string]json.RawMessage, error) {
	if err := json.Unmarshal(raw, target); err != nil {
		return nil, err
	}
	var extra map[string]json.RawMessage
	if err := json.Unmarshal(raw, &extra); err != nil {
		return nil, err
	}
	for _, key := range keys {
		delete(extra, key)
	}
	return extra, nil
}
func encodeProjectObject(value any, extra map[string]json.RawMessage, keys []string) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	known := map[string]bool{}
	for _, key := range keys {
		known[key] = true
	}
	for key, value := range extra {
		if !known[key] {
			fields[key] = value
		}
	}
	return json.Marshal(fields)
}

var projectKeys = []string{"common", "details"}

func (p *Project) UnmarshalJSON(raw []byte) error {
	type plain Project
	var decoded plain
	extra, err := decodeProjectObject(raw, &decoded, projectKeys)
	if err != nil {
		return err
	}
	*p = Project(decoded)
	p.ExtraFields = extra
	return nil
}
func (p Project) MarshalJSON() ([]byte, error) {
	type plain Project
	return encodeProjectObject(plain(p), p.ExtraFields, projectKeys)
}

var projectCommonKeys = []string{"craftType", "additionalCrafts", "status", "title", "duration", "pattern", "materials", "colors", "designTags", "tags"}

func (p *ProjectCommon) UnmarshalJSON(raw []byte) error {
	type plain ProjectCommon
	var decoded plain
	extra, err := decodeProjectObject(raw, &decoded, projectCommonKeys)
	if err != nil {
		return err
	}
	*p = ProjectCommon(decoded)
	p.ExtraFields = extra
	return nil
}
func (p ProjectCommon) MarshalJSON() ([]byte, error) {
	type plain ProjectCommon
	return encodeProjectObject(plain(p), p.ExtraFields, projectCommonKeys)
}

var projectCraftKeys = []string{"craftType"}

func (p *ProjectCraft) UnmarshalJSON(raw []byte) error {
	type plain ProjectCraft
	var decoded plain
	extra, err := decodeProjectObject(raw, &decoded, projectCraftKeys)
	if err != nil {
		return err
	}
	*p = ProjectCraft(decoded)
	p.ExtraFields = extra
	return nil
}
func (p ProjectCraft) MarshalJSON() ([]byte, error) {
	type plain ProjectCraft
	return encodeProjectObject(plain(p), p.ExtraFields, projectCraftKeys)
}

var projectMaterialKeys = []string{"text", "facets"}

func (p *ProjectMaterial) UnmarshalJSON(raw []byte) error {
	type plain ProjectMaterial
	var decoded plain
	extra, err := decodeProjectObject(raw, &decoded, projectMaterialKeys)
	if err != nil {
		return err
	}
	*p = ProjectMaterial(decoded)
	p.ExtraFields = extra
	return nil
}
func (p ProjectMaterial) MarshalJSON() ([]byte, error) {
	type plain ProjectMaterial
	return encodeProjectObject(plain(p), p.ExtraFields, projectMaterialKeys)
}

var projectPatternKeys = []string{"url", "name", "nameFacets", "difficulty", "designer", "designerFacets", "publisher", "publisherFacets", "selfDrafted"}

func (p *ProjectPattern) UnmarshalJSON(raw []byte) error {
	type plain ProjectPattern
	var decoded plain
	extra, err := decodeProjectObject(raw, &decoded, projectPatternKeys)
	if err != nil {
		return err
	}
	*p = ProjectPattern(decoded)
	p.ExtraFields = extra
	return nil
}
func (p ProjectPattern) MarshalJSON() ([]byte, error) {
	type plain ProjectPattern
	return encodeProjectObject(plain(p), p.ExtraFields, projectPatternKeys)
}
