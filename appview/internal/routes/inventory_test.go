package routes

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type inventoryRegistrar struct {
	patterns []string
}

func (r *inventoryRegistrar) Handle(pattern string, _ http.Handler) {
	r.patterns = append(r.patterns, pattern)
}

func TestRouteInventoryAndV1PoliciesStayExact(t *testing.T) {
	deps := testDeps()
	deps.Config.EnableDevModeration = true
	deps.Config.DevModerationToken = "inventory-token"
	registrar := &inventoryRegistrar{}

	AddRoutes(context.Background(), registrar, deps)

	want := []string{
		"GET /health",
		"GET /healthz",
		"GET /v1/dev/media/{name}",
		"GET /v1/dev/panic",
		"GET /oauth/client-metadata.json",
		"GET /oauth/jwks.json",
		"GET /oauth/callback",
		"POST /v1/auth/login",
		"POST /v1/auth/registrations",
		"POST /v1/auth/handoffs/exchange",
		"POST /v1/auth/handoffs/confirm",
		"POST /v1/blobs/videos/authorization",
		"GET /v1/blobs/videos/limits",
		"GET /v1/whoami",
		"GET /v1/facets/mentions",
		"GET /v1/facets/mentions/resolve",
		"GET /v1/facets/hashtags",
		"GET /v1/projects",
		"GET /v1/search/hashtags/{tag}/posts",
		"GET /v1/search/suggestions",
		"GET /v1/search/hashtags",
		"GET /v1/search/profiles",
		"GET /v1/search/posts",
		"GET /v1/search/projects",
		"GET /v1/search/hashtags/top",
		"GET /v1/search/recent",
		"POST /v1/search/recent",
		"DELETE /v1/search/recent/{id}",
		"POST /v1/auth/logout",
		"POST /v1/account-deletion/intents",
		"DELETE /v1/account-deletion/intents/{jobId}",
		"POST /v1/account-deletions/{jobId}",
		"POST /v1/migrations/instagram/verifications",
		"GET /v1/migrations/instagram/verifications/current",
		"GET /v1/migrations/instagram/verifications/{verificationId}",
		"DELETE /v1/migrations/instagram/verifications/{verificationId}",
		"POST /v1/migrations/instagram/verifications/{verificationId}/confirm",
		"GET /v1/migrations/instagram/account",
		"DELETE /v1/migrations/instagram/account",
		"PATCH /v1/migrations/instagram/settings",
		"POST /v1/migrations/instagram/imports",
		"GET /v1/migrations/instagram/imports",
		"GET /v1/migrations/instagram/imports/{importId}",
		"PATCH /v1/migrations/instagram/imports/{importId}",
		"DELETE /v1/migrations/instagram/imports/{importId}",
		"GET /v1/migrations/instagram/suggestions",
		"POST /v1/migrations/instagram/suggestions/{suggestionId}/accept",
		"DELETE /v1/migrations/instagram/suggestions/{suggestionId}",
		"GET /v1/onboarding/status",
		"POST /v1/onboarding/completion",
		"GET /v1/profiles/{handleOrDid}",
		"GET /v1/profiles/me",
		"GET /v1/profiles/me/follower-growth",
		"GET /v1/profiles/me/followers",
		"GET /v1/profiles/me/following",
		"PUT /v1/profiles/me",
		"PUT /v1/profiles/me/customisation",
		"GET /v1/profiles/{handleOrDid}/mutual-followers",
		"POST /v1/profiles/{handleOrDid}/follows",
		"DELETE /v1/profiles/{handleOrDid}/follows",
		"POST /v1/profiles/{handleOrDid}/mutes",
		"DELETE /v1/profiles/{handleOrDid}/mutes",
		"POST /v1/profiles/{handleOrDid}/blocks",
		"DELETE /v1/profiles/{handleOrDid}/blocks",
		"GET /v1/profiles/me/mutes",
		"GET /v1/profiles/me/blocks",
		"POST /v1/profiles/{handleOrDid}/reports",
		"PUT /v1/profiles/me/account-type",
		"PUT /v1/profiles/me/business",
		"DELETE /v1/profiles/me/business",
		"GET /v1/profiles/{handleOrDid}/events",
		"POST /v1/events",
		"GET /v1/events",
		"GET /v1/events/{did}/{rkey}",
		"PUT /v1/events/{did}/{rkey}",
		"DELETE /v1/events/{did}/{rkey}",
		"POST /v1/events/{did}/{rkey}/reports",
		"GET /v1/feed/timeline",
		"GET /v1/notifications",
		"GET /v1/notifications/new-count",
		"POST /v1/notifications/seen",
		"GET /v1/notifications/preferences",
		"PATCH /v1/notifications/preferences",
		"GET /v1/languages/preferences",
		"PUT /v1/languages/preferences",
		"POST /v1/languages/preferences/initialize",
		"POST /v1/notifications/devices",
		"DELETE /v1/notifications/devices/{accountSubscriptionId}",
		"POST /v1/blobs/images",
		"PUT /v1/scheduled-post-media/{mediaId}",
		"GET /v1/scheduled-post-media/{mediaId}",
		"DELETE /v1/scheduled-post-media/{mediaId}",
		"POST /v1/scheduled-posts",
		"GET /v1/scheduled-posts",
		"GET /v1/scheduled-posts/{id}",
		"PUT /v1/scheduled-posts/{id}",
		"DELETE /v1/scheduled-posts/{id}",
		"POST /v1/scheduled-posts/{id}/publication",
		"POST /v1/posts",
		"GET /v1/posts/{did}/{rkey}",
		"GET /v1/posts/{did}/{rkey}/video-captions/{captionCid}",
		"POST /v1/posts/{did}/{rkey}/saves",
		"DELETE /v1/posts/{did}/{rkey}/saves",
		"GET /v1/profiles/me/pins",
		"PUT /v1/posts/{did}/{rkey}/pin",
		"DELETE /v1/posts/{did}/{rkey}/pin",
		"GET /v1/saved-posts",
		"GET /v1/saved-post-folders",
		"POST /v1/saved-post-folders",
		"PATCH /v1/saved-post-folders/{folderId}",
		"DELETE /v1/saved-post-folders/{folderId}",
		"GET /v1/posts/{did}/{rkey}/replies",
		"GET /v1/posts/{did}/{rkey}/comments",
		"POST /v1/posts/{did}/{rkey}/likes",
		"DELETE /v1/posts/{did}/{rkey}/likes",
		"POST /v1/posts/{did}/{rkey}/reposts",
		"DELETE /v1/posts/{did}/{rkey}/reposts",
		"DELETE /v1/posts/{did}/{rkey}",
		"POST /v1/posts/{did}/{rkey}/reports",
		"POST /v1/dev/moderation/ozone-events",
		"GET /v1/profiles/{handleOrDid}/posts",
		"GET /v1/profiles/{handleOrDid}/projects",
		"GET /v1/profiles/{handleOrDid}/comments",
		"POST /v1/link-previews",
		"/",
	}
	if !reflect.DeepEqual(registrar.patterns, want) {
		t.Fatalf("route inventory changed\n got: %#v\nwant: %#v", registrar.patterns, want)
	}

	registeredV1 := make(map[string]int)
	for _, pattern := range registrar.patterns {
		parts := strings.SplitN(pattern, " ", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[1], "/v1/") {
			registeredV1[pattern]++
		}
	}
	policyV1 := make(map[string]int)
	for _, policy := range V1RoutePolicies(deps.Config.Env, deps.Config) {
		policyV1[policy.Method+" "+policy.PathPattern]++
	}
	if !reflect.DeepEqual(registeredV1, policyV1) {
		t.Fatalf("registered v1 inventory and central policies differ\nregistered: %#v\npolicies: %#v", registeredV1, policyV1)
	}
	for pattern, count := range registeredV1 {
		if count != 1 {
			t.Errorf("v1 route %s registered %d times", pattern, count)
		}
	}
	for pattern, count := range policyV1 {
		if count != 1 {
			t.Errorf("v1 route %s has %d policies", pattern, count)
		}
	}
}
