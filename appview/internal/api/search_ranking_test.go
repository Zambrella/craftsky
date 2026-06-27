package api_test

import (
	"math"
	"reflect"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestPopularityScoreFormula(t *testing.T) {
	rankedAt := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	createdAt := rankedAt.Add(-72 * time.Hour)
	got := api.PopularityScore(3, 2, 1, createdAt, rankedAt)
	// weighted = 3 + (2 * 2) + (3 * 1) = 10; decay denominator = pow(2, 1.5)
	want := 10.0 / math.Pow(2, 1.5)
	if math.Abs(got-want) > 0.0000001 {
		t.Fatalf("PopularityScore = %.8f, want %.8f", got, want)
	}
}

func TestPopularityScoreClampsFutureAge(t *testing.T) {
	rankedAt := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	createdAt := rankedAt.Add(1 * time.Hour)
	if got := api.PopularityScore(1, 1, 1, createdAt, rankedAt); got != 6 {
		t.Fatalf("future score = %v, want weighted engagement without decay", got)
	}
}

func TestRankHashtagResultsNormalizesAggregatesAndRanks(t *testing.T) {
	rows := []api.HashtagSuggestionRow{
		{Tag: "Sock", PostsLast28Days: 1},
		{Tag: "#sock", PostsLast28Days: 2},
		{Tag: "sockkal", PostsLast28Days: 3},
		{Tag: "sockmending", PostsLast28Days: 7},
		{Tag: "mending-sock", PostsLast28Days: 99},
		{Tag: "sockbad", PostsLast28Days: -5},
		{Tag: "hat", PostsLast28Days: 100},
	}

	got := api.RankHashtagResults("#Sock", rows)
	want := []api.HashtagSuggestionRow{
		{Tag: "sock", PostsLast28Days: 3},
		{Tag: "sockmending", PostsLast28Days: 7},
		{Tag: "sockkal", PostsLast28Days: 3},
		{Tag: "sockbad", PostsLast28Days: 0},
		{Tag: "mending-sock", PostsLast28Days: 99},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("RankHashtagResults = %#v, want %#v", got, want)
	}
}
