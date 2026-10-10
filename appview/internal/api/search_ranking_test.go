package api_test

import (
	"math"
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
