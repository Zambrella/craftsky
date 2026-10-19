package api

import (
	"social.craftsky/appview/internal/projectcatalog"
	"strings"
	"unicode"
	"unicode/utf8"
)

var searchEquivalenceGroups = projectcatalog.Current.EquivalenceGroups

type searchCorrectionWord struct {
	Concept  int    `json:"concept"`
	Position int    `json:"position"`
	Word     string `json:"word"`
	Before   string `json:"before"`
	After    string `json:"after"`
}

// buildSearchMatchingPlan recognizes approved phrases before removing filler.
// PostgreSQL remains responsible for field/query lexemes and English stemming.
func buildSearchMatchingPlan(query string) ([][]string, []searchCorrectionWord) {
	// Keep decimal measurements intact so query and field lexemes agree (for
	// example 4.5mm). Other punctuation continues to separate words.
	runes := []rune(strings.ToLower(query))
	for i, r := range runes {
		decimal := r == '.' && i > 0 && i+1 < len(runes) && unicode.IsNumber(runes[i-1]) && unicode.IsNumber(runes[i+1])
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-' && r != '_' && !decimal {
			runes[i] = ' '
		}
	}
	words := strings.Fields(string(runes))
	concepts := make([][]string, 0, len(words))
	seen := map[string]bool{}
	corrections := make([]searchCorrectionWord, 0)
	for i := 0; i < len(words); {
		alternatives := []string{words[i]}
		consumed := 1
		found := false
		for _, group := range searchEquivalenceGroups {
			for _, phrase := range group {
				parts := strings.Fields(phrase)
				if len(parts) <= len(words)-i && strings.Join(words[i:i+len(parts)], " ") == phrase {
					alternatives = group
					consumed = len(parts)
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if found || (words[i] != "how" && words[i] != "to") {
			key := strings.Join(alternatives, "|")
			if !seen[key] {
				original := words[i : i+consumed]
				for j, word := range original {
					if utf8.RuneCountInString(word) >= 4 {
						corrections = append(corrections, searchCorrectionWord{Concept: len(concepts), Position: j, Word: word, Before: strings.Join(original[:j], " "), After: strings.Join(original[j+1:], " ")})
					}
				}
				concepts = append(concepts, alternatives)
				seen[key] = true
			}
		}
		i += consumed
	}
	return concepts, corrections
}
