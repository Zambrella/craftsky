package routes

import (
	"context"
	"slices"
	"testing"
)

// IT-002 / IT-016: both editor routes stay inside current-member, age and
// suspension controls; saving is a body-limited, rate-limited write.
func TestPostEditRoutesHaveCurrentMemberPolicies(t *testing.T) {
	policies := V1RoutePolicies(EnvProd, Config{})
	for _, want := range []struct {
		method, path string
		rate         RateClass
		body         BodyKind
	}{
		{"GET", "/v1/posts/{did}/{rkey}/edit", RateClassRead, BodyNoBody},
		{"PATCH", "/v1/posts/{did}/{rkey}", RateClassWrite, BodyDefaultJSON},
	} {
		found := false
		for _, policy := range policies {
			if policy.Method != want.method || policy.PathPattern != want.path {
				continue
			}
			found = true
			if policy.AccessClass != AccessCurrentMember || policy.RateClass != want.rate || policy.BodyKind != want.body || policy.EligibilityClass.AllowedWhenRestricted() || policy.SuspensionClass.AllowedWhenSuspended() {
				t.Fatalf("wrong editor policy: %+v", policy)
			}
		}
		if !found {
			t.Errorf("missing editor policy: %s %s", want.method, want.path)
		}
	}
}

func TestPostEditHandlersAreRegistered(t *testing.T) {
	registrar := &inventoryRegistrar{}
	AddRoutes(context.Background(), registrar, testDeps())
	for _, pattern := range []string{"GET /v1/posts/{did}/{rkey}/edit", "PATCH /v1/posts/{did}/{rkey}"} {
		if !slices.Contains(registrar.patterns, pattern) {
			t.Errorf("missing registered handler: %s", pattern)
		}
	}
}
