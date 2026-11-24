package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"social.craftsky/appview/internal/projectcatalog"
)

// Adding a public field requires an explicit search decision; silently leaving
// it out of both keyword search and the exclusion list fails this guard.
func TestSearchFieldContractCoversProjectSchemas(t *testing.T) {
	covered := map[string]bool{}
	for _, field := range submittedSearchFields {
		covered[field.Name] = true
	}
	for _, field := range patternSearchFields {
		covered[field.Name] = true
	}
	aliases := map[string]string{
		"project.common.craftType":                  "project.common.craftType/additionalCrafts",
		"project.common.additionalCrafts.craftType": "project.common.craftType/additionalCrafts",
		"project.details.needleSizeMm":              "project.details.needleSizeMm/needleSizesMm",
		"project.details.needleSizesMm":             "project.details.needleSizeMm/needleSizesMm",
		"project.details.hookSizeMm":                "project.details.hookSizeMm/hookSizesMm",
		"project.details.hookSizesMm":               "project.details.hookSizeMm/hookSizesMm",
		"project.details.finishedSize":              "project.details.finishedSize/size/sizeMade",
		"project.details.size":                      "project.details.finishedSize/size/sizeMade",
		"project.details.sizeMade":                  "project.details.finishedSize/size/sizeMade",
		"project.details.materialForm":              "project.details.projectType",
		"project.details.methods.method":            "project.details.method",
		"project.details.methods.styles.style":      "project.details.style",
	}
	for _, name := range []string{"stitches", "rows", "measurement", "unit"} {
		aliases["project.details.gauge."+name] = "project.details.gauge"
	}
	for _, field := range []string{"name", "designer", "publisher", "difficulty", "selfDrafted"} {
		aliases["project.common.pattern."+field] = "project.common.patterns." + field
	}
	excluded := map[string]bool{
		"project.common.materials.facets":  true,
		"project.details.machineModel":     true,
		"project.details.machineGaugeMm":   true,
		"project.details.tool":             true,
		"project.details.fibrePreparation": true,
		"project.details.plyCount":         true,
		"project.details.finishedYarn":     true,
		"project.details.dyeType":          true,
		"project.details.colorEffect":      true,
		"project.details.recipeNotes":      true,
		"project.details.foundationFabric": true,
		"project.details.needle":           true,
		"project.details.loopHeight":       true,
		"project.details.cords":            true,
		"project.details.materials":        true,
		"project.details.sett":             true,
		"project.details.needleGauges":     true,
		"project.details.core":             true,
		"project.details.armature":         true,
	}
	for _, shape := range []string{"pattern", "patterns"} {
		for _, field := range []string{"url", "nameFacets", "designerFacets", "publisherFacets"} {
			excluded["project.common."+shape+"."+field] = true
		}
	}
	root := "../../../lexicon/social/craftsky"
	docs := map[string]map[string]any{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var doc map[string]any
		if err = json.Unmarshal(raw, &doc); err != nil {
			return err
		}
		docs[doc["id"].(string)] = doc
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var walk func(string, string, map[string]any)
	walk = func(nsid, path string, node map[string]any) {
		if excluded[path] {
			return
		}
		switch node["type"] {
		case "ref":
			ref := node["ref"].(string)
			parts := strings.SplitN(ref, "#", 2)
			target := parts[0]
			if target == "" {
				target = nsid
			}
			def := "main"
			if len(parts) == 2 {
				def = parts[1]
			}
			walk(target, path, docs[target]["defs"].(map[string]any)[def].(map[string]any))
			return
		case "object":
			for name, property := range node["properties"].(map[string]any) {
				walk(nsid, path+"."+name, property.(map[string]any))
			}
			return
		case "array":
			items := node["items"].(map[string]any)
			if items["type"] == "ref" {
				walk(nsid, path, items)
				return
			}
		}
		key := path
		if alias, ok := aliases[path]; ok {
			key = alias
		}
		if !covered[key] {
			t.Errorf("field %s requires a searchable-field policy", path)
		}
	}
	shared := docs["social.craftsky.project.defs"]["defs"].(map[string]any)
	walk("social.craftsky.project.defs", "project.common", shared["projectCommon"].(map[string]any))
	for _, craft := range projectcatalog.PrimaryCrafts() {
		nsid, definition, _ := strings.Cut(craft.DetailsType, "#")
		walk(nsid, "project.details", docs[nsid]["defs"].(map[string]any)[definition].(map[string]any))
	}
}
