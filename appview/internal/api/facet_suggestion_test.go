package api

import "testing"

func TestRankMentionSuggestionRowsFollowedPrefixThenHandle(t *testing.T) {
	t.Parallel()
	rows := []MentionSuggestionRow{
		{DID: "did:plc:4", Handle: "zali.craftsky.social", ViewerIsFollowing: true, IsCraftskyProfile: true},
		{DID: "did:plc:2", Handle: "alicia.craftsky.social", ViewerIsFollowing: false, IsCraftskyProfile: true},
		{DID: "did:plc:1", Handle: "alice.craftsky.social", ViewerIsFollowing: true, IsCraftskyProfile: true},
		{DID: "did:plc:3", Handle: "mallory-alice.craftsky.social", ViewerIsFollowing: false, IsCraftskyProfile: true},
	}

	RankMentionSuggestionRows(rows, "ali")

	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.Handle)
	}
	want := []string{
		"alice.craftsky.social",
		"zali.craftsky.social",
		"alicia.craftsky.social",
		"mallory-alice.craftsky.social",
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestRankMentionSuggestionRowsUsesSharedProfileRelevance(t *testing.T) {
	t.Parallel()
	displayName := "Alice Maker"
	description := "Sews with Alice"
	rows := []MentionSuggestionRow{
		{DID: "did:plc:description", Handle: "aaa.craftsky.social", DisplayName: nil, IsCraftskyProfile: true, ViewerIsFollowing: false},
		{DID: "did:plc:display", Handle: "zzz.craftsky.social", DisplayName: &displayName, IsCraftskyProfile: true, ViewerIsFollowing: false},
	}
	rows[0].Description = &description

	RankMentionSuggestionRows(rows, "alice")

	if rows[0].DID != "did:plc:display" {
		t.Fatalf("first ranked DID = %q, want display-name match before description match; rows=%#v", rows[0].DID, rows)
	}
}

func TestNormalizeHashtagSuggestionRowsLowercaseCountsAndSorts(t *testing.T) {
	t.Parallel()
	rows := []HashtagSuggestionRow{
		{Tag: "SockKAL", PostsLast28Days: 2},
		{Tag: "", PostsLast28Days: 99},
		{Tag: "sockkal", PostsLast28Days: 3},
		{Tag: "sockmending", PostsLast28Days: 3},
		{Tag: "sockbad", PostsLast28Days: -1},
	}

	got := NormalizeHashtagSuggestionRows(rows)
	want := []HashtagSuggestionRow{
		{Tag: "sockkal", PostsLast28Days: 5},
		{Tag: "sockmending", PostsLast28Days: 3},
		{Tag: "sockbad", PostsLast28Days: 0},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d = %#v, want %#v; all=%#v", i, got[i], want[i], got)
		}
	}
}

func TestEscapeFacetLikePatternTreatsWildcardCharactersLiterally(t *testing.T) {
	t.Parallel()

	got := EscapeFacetLikePattern(`50%_wool\craft`)
	want := `50\%\_wool\\craft`
	if got != want {
		t.Fatalf("escaped pattern = %q, want %q", got, want)
	}
}
