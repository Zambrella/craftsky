package api_test

import (
	"slices"
	"testing"
)

func TestLanguageVisibilityCorpusTruthTable(t *testing.T) {
	tests := []struct {
		name     string
		viewer   string
		author   string
		selected []string
		post     []string
		want     bool
	}{
		{name: "exact match", viewer: "viewer", author: "alice", selected: []string{"en", "cy"}, post: []string{"en"}, want: true},
		{name: "any exact match", viewer: "viewer", author: "alice", selected: []string{"en", "cy"}, post: []string{"fr", "cy"}, want: true},
		{name: "mismatch", viewer: "viewer", author: "alice", selected: []string{"en", "cy"}, post: []string{"fr"}, want: false},
		{name: "active filter hides untagged", viewer: "viewer", author: "alice", selected: []string{"en"}, post: []string{}, want: false},
		{name: "empty filter shows tagged", viewer: "viewer", author: "alice", selected: []string{}, post: []string{"fr"}, want: true},
		{name: "empty filter shows untagged", viewer: "viewer", author: "alice", selected: []string{}, post: []string{}, want: true},
		{name: "viewer owns mismatched", viewer: "viewer", author: "viewer", selected: []string{"en"}, post: []string{"fr"}, want: true},
		{name: "viewer owns untagged", viewer: "viewer", author: "viewer", selected: []string{"en"}, post: []string{}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := languageEligibleForCorpus(tt.viewer, tt.author, tt.selected, tt.post)
			if got != tt.want {
				t.Fatalf("eligible = %v, want %v", got, tt.want)
			}
		})
	}
}

func languageEligibleForCorpus(viewer, author string, selected, post []string) bool {
	if viewer == author || len(selected) == 0 {
		return true
	}
	return slices.ContainsFunc(post, func(tag string) bool {
		return slices.Contains(selected, tag)
	})
}
