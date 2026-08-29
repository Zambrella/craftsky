package api_test

import "testing"

func TestBusinessStateHasNoProductOrPermissionAdvantage(t *testing.T) {
	t.Run("chronological feed", assertBusinessFeedNeutrality)
	t.Run("search rank and cursor", assertBusinessSearchNeutrality)
	t.Run("relationship authorization and moderation", assertBusinessPolicyNeutrality)
}
