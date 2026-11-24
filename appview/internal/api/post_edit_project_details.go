package api

import (
	"encoding/json"
	"strings"
)

// The editable inventory defines which child objects can be changed and which
// list entries carry a reference to the corresponding authoritative entry.
type projectDetailEdit struct {
	fields   string
	array    bool
	children map[string]projectDetailEdit
}

var projectDetailEdits = func() map[string]projectDetailEdit {
	technique := projectDetailEdit{fields: "technique", array: true}
	measurement := projectDetailEdit{fields: "amount unit"}
	gauge := projectDetailEdit{fields: "stitches rows measurement unit"}
	return map[string]projectDetailEdit{
		"social.craftsky.project.sewing#details": {
			fields:   "$type projectType projectSubtype sizeMade fitNotes techniques",
			children: map[string]projectDetailEdit{"techniques": technique},
		},
		"social.craftsky.project.knitting#details": {
			fields: "$type projectType projectSubtype yarnWeight needleSizeMm needleSizesMm gauge finishedSize methods machineModel machineGaugeMm techniques",
			children: map[string]projectDetailEdit{
				"gauge": gauge,
				"techniques": {
					fields: "technique subtypes", array: true,
					children: map[string]projectDetailEdit{"subtypes": {fields: "subtype", array: true}},
				},
			},
		},
		"social.craftsky.project.crochet#details": {
			fields:   "$type projectType projectSubtype yarnWeight hookSizeMm hookSizesMm gauge finishedSize methods techniques",
			children: map[string]projectDetailEdit{"gauge": gauge, "techniques": technique},
		},
		"social.craftsky.project.quilting#details": {
			fields: "$type projectType projectSubtype size piecingTechnique quiltingMethod",
		},
		"social.craftsky.project.embroidery#details": {
			fields: "$type projectType projectSubtype method style methods",
			children: map[string]projectDetailEdit{"methods": {
				fields: "method styles", array: true,
				children: map[string]projectDetailEdit{"styles": {fields: "style", array: true}},
			}},
		},
		"social.craftsky.project.spinning#details": {
			fields: "$type tool fibrePreparation draftingTechniques plyingTechniques plyCount finishedYarn",
			children: map[string]projectDetailEdit{
				"draftingTechniques": technique,
				"plyingTechniques":   technique,
				"finishedYarn": {
					fields:   "mass length wrapsPerInch",
					children: map[string]projectDetailEdit{"mass": measurement, "length": measurement},
				},
			},
		},
		"social.craftsky.project.dyeing#details": {
			fields: "$type projectType projectSubtype materialForm dyeType applicationMethod colorEffect recipeNotes",
		},
		"social.craftsky.project.punchNeedle#details": {
			fields:   "$type projectType projectSubtype foundationFabric needle loopHeight",
			children: map[string]projectDetailEdit{"loopHeight": measurement},
		},
		"social.craftsky.project.macrame#details": {
			fields: "$type projectType projectSubtype cords knots finishedSize",
			children: map[string]projectDetailEdit{
				"cords": {fields: "material construction diameterMm", array: true},
				"knots": {fields: "knot", array: true},
			},
		},
		"social.craftsky.project.weaving#details": {
			fields: "$type projectType projectSubtype loomType structure materials sett finishedSize",
			children: map[string]projectDetailEdit{
				"materials": {fields: "role text", array: true},
				"sett":      measurement,
			},
		},
		"social.craftsky.project.felting#details": {
			fields:   "$type projectType projectSubtype methods method form needleGauges core armature",
			children: map[string]projectDetailEdit{"needleGauges": {fields: "gauge", array: true}},
		},
	}
}()

func (spec projectDetailEdit) childMergers() map[string]postEditObjectMerger {
	if len(spec.children) == 0 {
		return nil
	}
	mergers := make(map[string]postEditObjectMerger, len(spec.children))
	for key, child := range spec.children {
		allowed := strings.Fields(child.fields)
		nested := child.childMergers()
		if child.array {
			mergers[key] = mergePostEditEntryObjects(allowed, nested)
		} else {
			mergers[key] = func(base, changes json.RawMessage) (json.RawMessage, error) {
				return mergeSupportedPostEditObject(base, changes, allowed, nested)
			}
		}
	}
	return mergers
}

func (spec projectDetailEdit) indexChildren(fields map[string]json.RawMessage) error {
	for key, child := range spec.children {
		raw, present := fields[key]
		if !present {
			continue
		}
		if child.array {
			var entries []map[string]json.RawMessage
			if err := json.Unmarshal(raw, &entries); err != nil {
				return err
			}
			for index, entry := range entries {
				ordinal, err := json.Marshal(index)
				if err != nil {
					return err
				}
				entry["existingIndex"] = ordinal
				if err := child.indexChildren(entry); err != nil {
					return err
				}
			}
			indexed, err := json.Marshal(entries)
			if err != nil {
				return err
			}
			fields[key] = indexed
		}
	}
	return nil
}
