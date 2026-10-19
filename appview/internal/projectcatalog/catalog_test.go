package projectcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCatalogCoversStableTokensAndValidAcyclicParents(t *testing.T) {
	ids := map[string]Entry{}
	for _, e := range Current.Entries {
		if _, ok := ids[e.ID]; ok {
			t.Fatalf("duplicate %s", e.ID)
		}
		ids[e.ID] = e
		if e.Labels["en"] == "" {
			t.Fatalf("missing label %s", e.ID)
		}
	}
	for id := range ids {
		var visit func(string, map[string]bool)
		visit = func(key string, ancestors map[string]bool) {
			if ancestors[key] {
				t.Fatalf("cycle at %s", key)
			}
			e, ok := ids[key]
			if !ok {
				t.Fatalf("missing parent %s", key)
			}
			ancestors[key] = true
			for _, parent := range e.Parents {
				visit(parent, ancestors)
			}
			delete(ancestors, key)
		}
		visit(id, map[string]bool{})
	}
	root := "../../../lexicon/social/craftsky"
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}
		var doc struct {
			ID   string `json:"id"`
			Defs map[string]struct {
				Type string `json:"type"`
			} `json:"defs"`
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &doc); err != nil {
			return err
		}
		for name, def := range doc.Defs {
			if def.Type == "token" {
				if _, ok := ids[doc.ID+"#"+name]; !ok {
					t.Fatalf("uncatalogued token %s#%s", doc.ID, name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSearchTermsIncludeAliasesAndDirectionalAncestors(t *testing.T) {
	terms := SearchTerms()
	cardigan := terms["social.craftsky.project.knitting.defs#cardigan"]
	if !slices.Contains(cardigan, "Sweater") || !slices.Contains(cardigan, "jumper") || !slices.Contains(cardigan, "Garment") {
		t.Fatalf("cardigan terms=%v", cardigan)
	}
	if slices.Contains(terms["social.craftsky.project.knitting.defs#sweater"], "Cardigan") {
		t.Fatal("broad category expanded to a specific child")
	}
}
