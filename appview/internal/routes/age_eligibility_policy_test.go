package routes

import "testing"

func TestAgeRestrictedRoutePolicyRetainsOnlySafetyAndAccountMaintenance(t *testing.T) {
	allowed := map[string]bool{
		"GET /v1/account/eligibility":             true,
		"GET /v1/moderation/standing":             true,
		"POST /v1/profiles/{handleOrDid}/reports": true,
		"POST /v1/profiles/{handleOrDid}/blocks":  true,
		"POST /v1/profiles/{handleOrDid}/mutes":   true,
		"POST /v1/account-deletion/intents":       true,
		"DELETE /v1/posts/{did}/{rkey}":           true,
	}
	denied := map[string]bool{
		"GET /v1/feed/timeline": true,
		"POST /v1/posts":        true,
		"POST /v1/events":       true,
	}
	for _, policy := range V1RoutePolicies(EnvProd, Config{}) {
		key := policyKey(policy.Method, policy.PathPattern)
		if allowed[key] && !policy.EligibilityClass.AllowedWhenRestricted() {
			t.Errorf("retained route %s denied", key)
		}
		if denied[key] && policy.EligibilityClass.AllowedWhenRestricted() {
			t.Errorf("ordinary route %s retained", key)
		}
	}
}
