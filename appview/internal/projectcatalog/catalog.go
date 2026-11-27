// Package projectcatalog owns application taxonomy, independent of record schemas.
package projectcatalog

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed catalog-v1.json
var source []byte

type Entry struct {
	ID           string            `json:"id"`
	Family       string            `json:"family"`
	Labels       map[string]string `json:"labels"`
	Aliases      []string          `json:"aliases"`
	Parents      []string          `json:"parents"`
	Craft        string            `json:"craft,omitempty"`
	Crafts       []string          `json:"crafts,omitempty"`
	Group        string            `json:"group,omitempty"`
	PickerParent string            `json:"pickerParent,omitempty"`
	ProfileID    string            `json:"profileId,omitempty"`
	DetailsType  string            `json:"detailsType,omitempty"`
	Phase        int               `json:"phase,omitempty"`
	PreviewOnly  bool              `json:"previewOnly,omitempty"`
	Concept      string            `json:"concept"`
}
type Catalog struct {
	Version           int        `json:"version"`
	ReleasePhase      int        `json:"releasePhase"`
	EquivalenceGroups [][]string `json:"equivalenceGroups"`
	Entries           []Entry    `json:"entries"`
}

var Current = func() Catalog {
	var c Catalog
	if err := json.Unmarshal(source, &c); err != nil {
		panic(err)
	}
	return c
}()

// SearchTerms expands authored values toward their broader concepts. It never
// expands a broad category to its children or rewrites public record values.
func SearchTerms() map[string][]string {
	entries := map[string]Entry{}
	for _, e := range Current.Entries {
		entries[e.ID] = e
	}
	out := map[string][]string{}
	for id, entry := range entries {
		if entry.PreviewOnly {
			continue
		}
		seen := map[string]bool{}
		terms := []string{}
		var visit func(string)
		visit = func(key string) {
			if seen[key] {
				return
			}
			seen[key] = true
			e, ok := entries[key]
			if !ok {
				return
			}
			terms = append(terms, e.Labels["en"])
			terms = append(terms, e.Aliases...)
			for _, parent := range e.Parents {
				visit(parent)
			}
		}
		visit(id)
		out[strings.ToLower(id)] = terms
	}
	return out
}

var SearchTermsJSON = func() json.RawMessage {
	raw, err := json.Marshal(SearchTerms())
	if err != nil {
		panic(err)
	}
	return raw
}()

var browseItems = func() struct {
	roots    map[string]map[string][]string
	children map[string][]string
} {
	roots := map[string]map[string][]string{"projectType": {}, "projectSubtype": {}}
	children := map[string][]string{}
	for _, entry := range Current.Entries {
		if entry.PreviewOnly {
			continue
		}
		id := strings.ToLower(entry.ID)
		if family := roots[entry.Family]; family != nil {
			family[id] = []string{id}
			if _, fragment, ok := strings.Cut(id, "#"); ok {
				family[fragment] = append(family[fragment], id)
			}
		}
		switch entry.Family {
		case "projectType", "projectSubtype", "materialForm":
			for _, parent := range entry.Parents {
				parent = strings.ToLower(parent)
				children[parent] = append(children[parent], id)
			}
		}
	}
	return struct {
		roots    map[string]map[string][]string
		children map[string][]string
	}{roots, children}
}()

// BrowseItemIDs returns recognised item IDs and the narrower IDs they include.
// Keep custom text in keyword search and support older callers using token fragments.
func BrowseItemIDs(family string, values []string) []string {
	ids := []string{}
	seen := map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
		for _, child := range browseItems.children[id] {
			visit(child)
		}
	}
	for _, value := range values {
		for _, id := range browseItems.roots[family][strings.ToLower(value)] {
			visit(id)
		}
	}
	if len(values) > 0 && len(ids) == 0 {
		// An unrecognised selection must match nothing. An empty array would
		// remove the filter. The SQL excludes empty stored values from matching.
		return []string{""}
	}
	return ids
}

// DetailCraft only recognizes variants authored by this application. Unknown
// open-union variants remain independent, preserved public extensions.
func DetailCraft(detailType string) (string, bool) {
	for _, craft := range primaryCrafts {
		if craft.DetailsType == detailType {
			return craft.ID, true
		}
	}
	return "", false
}

var primaryCrafts = func() []Entry {
	var crafts []Entry
	for _, entry := range Current.Entries {
		if entry.Family == "craft" {
			crafts = append(crafts, entry)
		}
	}
	return crafts
}()

// PrimaryCrafts returns the ordered project inventory used by discovery.
func PrimaryCrafts() []Entry {
	return primaryCraftsForPhase(Current.ReleasePhase)
}

func primaryCraftsForPhase(phase int) []Entry {
	crafts := make([]Entry, 0, len(primaryCrafts))
	for _, craft := range primaryCrafts {
		if craft.Phase <= phase {
			crafts = append(crafts, craft)
		}
	}
	return crafts
}

func Craft(token string) (Entry, bool) {
	for _, craft := range primaryCrafts {
		if craft.ID == token {
			return craft, true
		}
	}
	return Entry{}, false
}
