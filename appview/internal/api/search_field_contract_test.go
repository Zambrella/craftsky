package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Adding a public field requires an explicit search decision; silently leaving
// it out of both keyword search and the exclusion list fails this guard.
func TestSearchFieldContractCoversProjectSchemas(t *testing.T) {
	covered := map[string]bool{}
	for _, field := range submittedSearchFields {
		covered[field.Name] = true
	}
	aliases := map[string]string{
		"project.common.craftType":                  "project.common.craftType/additionalCrafts",
		"project.common.additionalCrafts.craftType": "project.common.craftType/additionalCrafts",
		"project.details.needleSizeMm":              "project.details.needleSizeMm/hookSizeMm",
		"project.details.hookSizeMm":                "project.details.needleSizeMm/hookSizeMm",
		"project.details.finishedSize":              "project.details.finishedSize/size/sizeMade",
		"project.details.size":                      "project.details.finishedSize/size/sizeMade",
		"project.details.sizeMade":                  "project.details.finishedSize/size/sizeMade",
	}
	for _, name := range []string{"stitches", "rows", "measurement", "unit"} {
		aliases["project.details.gauge."+name] = "project.details.gauge"
	}
	excluded := map[string]bool{
		"project.common.pattern.url":             true,
		"project.common.pattern.nameFacets":      true,
		"project.common.pattern.designerFacets":  true,
		"project.common.pattern.publisherFacets": true,
		"project.common.materials.facets":        true,
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
	for _, craft := range []string{"knitting", "crochet", "sewing", "quilting"} {
		nsid := "social.craftsky.project." + craft
		walk(nsid, "project.details", docs[nsid]["defs"].(map[string]any)["details"].(map[string]any))
	}
}
