package api_test

import "testing"

func TestBusinessRankingNeutralityRegression(t *testing.T) {
	t.Run("feed", assertBusinessFeedNeutrality)
	t.Run("search", assertBusinessSearchNeutrality)
}

func TestBusinessAuthorizationNeutralityRegression(t *testing.T) {
	assertBusinessPolicyNeutrality(t)
}
