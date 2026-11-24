package projectcatalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	for _, entry := range Current.Entries {
		if _, active := terms[strings.ToLower(entry.ID)]; entry.PreviewOnly && active {
			t.Errorf("preview token %s appeared in search vocabulary", entry.ID)
		}
	}
	cardigan := terms["social.craftsky.project.knitting.defs#cardigan"]
	if !slices.Contains(cardigan, "Sweater") || !slices.Contains(cardigan, "jumper") || !slices.Contains(cardigan, "Garment") {
		t.Fatalf("cardigan terms=%v", cardigan)
	}
	if slices.Contains(terms["social.craftsky.project.knitting.defs#sweater"], "Cardigan") {
		t.Fatal("broad category expanded to a specific child")
	}
	for _, craft := range []string{"knitting", "crochet"} {
		pullover := terms["social.craftsky.project."+craft+".defs#pullover"]
		if !slices.Contains(pullover, "Jumper") || !slices.Contains(pullover, "Pullover") ||
			!slices.Contains(pullover, "Jumper or pullover") {
			t.Errorf("%s jumper terms=%v, want current and previous names", craft, pullover)
		}
	}
	for _, craft := range []string{"sewing", "embroidery"} {
		repair := terms["social.craftsky.project."+craft+".defs#repair"]
		if !slices.Contains(repair, "Repair") || !slices.Contains(repair, "Mending") ||
			!slices.Contains(repair, "Repair or mending") {
			t.Errorf("%s repair terms=%v, want current and previous names", craft, repair)
		}
	}
	tunisian := terms["social.craftsky.project.crochet.defs#tunisiancrochet"]
	if !slices.Contains(tunisian, "Afghan crochet") {
		t.Fatalf("Tunisian crochet terms=%v, want Afghan crochet alias", tunisian)
	}
	fairIsle := terms["social.craftsky.project.knitting.defs#fairisle"]
	if !slices.Contains(fairIsle, "Colourwork") {
		t.Fatalf("Fair Isle terms=%v, want Colourwork ancestor", fairIsle)
	}
}

func TestPrimaryCraftMappingsAndScopedDesignSuggestions(t *testing.T) {
	counts := map[string]int{"knitting": 21, "crochet": 21, "sewing": 26, "embroidery": 17, "quilting": 19}
	crafts := primaryCraftsForPhase(2)
	if len(crafts) != 11 {
		t.Fatalf("primary crafts=%d, want 11", len(crafts))
	}
	for i, craft := range crafts {
		wantPhase := 1
		if i >= 5 {
			wantPhase = 2
		}
		if craft.ProfileID == "" || craft.Phase != wantPhase {
			t.Fatalf("incomplete craft mapping: %+v", craft)
		}
		if got, ok := DetailCraft(craft.DetailsType); !ok || got != craft.ID {
			t.Fatalf("detail %q maps to %q, want %q", craft.DetailsType, got, craft.ID)
		}
		wantCount := 26
		if count, ok := counts[craft.ProfileID]; ok {
			wantCount = count
		}
		got := 0
		for _, entry := range Current.Entries {
			if entry.Family == "designTag" && (len(entry.Crafts) == 0 || slices.Contains(entry.Crafts, craft.ID)) {
				got++
			}
		}
		if got != wantCount {
			t.Fatalf("%s has %d design suggestions, want %d", craft.ProfileID, got, wantCount)
		}
	}
	punchNeedle, _ := Craft("social.craftsky.feed.defs#punchNeedle")
	if punchNeedle.ProfileID != "punch-needle" || punchNeedle.DetailsType != "social.craftsky.project.punchNeedle#details" {
		t.Fatalf("incorrect Punch Needle identifier mapping: %+v", punchNeedle)
	}
}

func TestReleasePhaseFiltersOffersWithoutLosingKnownCrafts(t *testing.T) {
	first := primaryCraftsForPhase(1)
	want := []string{"knitting", "crochet", "sewing", "embroidery", "quilting"}
	if len(first) != len(want) {
		t.Fatalf("Phase 1 offers %d crafts, want %d", len(first), len(want))
	}
	for i, craft := range first {
		if craft.ProfileID != want[i] {
			t.Fatalf("Phase 1 craft %d=%s, want %s", i, craft.ProfileID, want[i])
		}
	}
	if craft, ok := DetailCraft("social.craftsky.project.weaving#details"); !ok || craft != "social.craftsky.feed.defs#weaving" {
		t.Fatalf("Weaving remains readable before its offer is enabled: %q", craft)
	}
}
