package api_test

import (
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestProfileRelevanceRank(t *testing.T) {
	for _, tc := range []struct {
		name        string
		query       string
		handle      string
		displayName string
		description string
		want        int
	}{
		{name: "exact handle", query: "ali", handle: "ali", want: 0},
		{name: "prefix handle", query: "ali", handle: "alice.craftsky.social", want: 1},
		{name: "substring handle", query: "ali", handle: "mali.craftsky.social", want: 2},
		{name: "display name", query: "ali", handle: "maker.example", displayName: "Alice Maker", want: 3},
		{name: "description", query: "ali", handle: "maker.example", description: "makes socks for Alice", want: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := api.ProfileRelevanceRank(tc.query, tc.handle, tc.displayName, tc.description)
			if !ok || got != tc.want {
				t.Fatalf("rank = %d/%v, want %d/true", got, ok, tc.want)
			}
		})
	}

	if _, ok := api.ProfileRelevanceRank("ali", "bob.example", "Bob", "crochet"); ok {
		t.Fatal("non-match ok = true, want false")
	}
}

func TestProfileSearchRankTupleFollowedFirst(t *testing.T) {
	followedWeak := api.ProfileSearchRankTuple(true, 4, "z.example", "did:plc:z")
	notFollowedStrong := api.ProfileSearchRankTuple(false, 0, "ali.example", "did:plc:a")
	if !followedWeak.Less(notFollowedStrong) {
		t.Fatalf("followed weak match must rank before non-followed strong match")
	}
}

func TestBuildProfileSearchSummaryIncludesCrafts(t *testing.T) {
	crafts := []string{"social.craftsky.feed.defs#knitting"}
	summary := api.BuildProfileSearchSummary(api.ProfileSearchRow{
		DID:               "did:plc:alice",
		Handle:            "alice.craftsky.social",
		HandleLower:       "alice.craftsky.social",
		IsCraftskyProfile: true,
		Crafts:            crafts,
	})
	if len(summary.Crafts) != 1 || summary.Crafts[0] != crafts[0] {
		t.Fatalf("summary crafts = %#v, want %#v", summary.Crafts, crafts)
	}
	crafts[0] = "mutated"
	if summary.Crafts[0] != "social.craftsky.feed.defs#knitting" {
		t.Fatalf("summary crafts aliases input slice: %#v", summary.Crafts)
	}
}
