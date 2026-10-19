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
	ID      string            `json:"id"`
	Family  string            `json:"family"`
	Labels  map[string]string `json:"labels"`
	Aliases []string          `json:"aliases"`
	Parents []string          `json:"parents"`
	Concept string            `json:"concept"`
}
type Catalog struct {
	Version           int        `json:"version"`
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
	for id := range entries {
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

// DetailCraft only recognizes variants authored by this application. Unknown
// open-union variants remain independent, preserved public extensions.
func DetailCraft(detailType string) (string, bool) {
	for _, craft := range []string{"knitting", "crochet", "sewing", "quilting"} {
		if detailType == "social.craftsky.project."+craft+"#details" {
			return "social.craftsky.feed.defs#" + craft, true
		}
	}
	return "", false
}
