package routes

import "net/http"

type RateClass string

const (
	RateClassAuth        RateClass = "auth"
	RateClassRead        RateClass = "read"
	RateClassWrite       RateClass = "write"
	RateClassSearch      RateClass = "expensive_search"
	RateClassUpload      RateClass = "upload"
	RateClassLinkPreview RateClass = "link_preview"
	RateClassExempt      RateClass = "exempt"
	RateClassDevOnly     RateClass = "dev_only_relaxed"
)

func (c RateClass) Valid() bool {
	switch c {
	case RateClassAuth, RateClassRead, RateClassWrite, RateClassSearch, RateClassUpload, RateClassLinkPreview, RateClassExempt, RateClassDevOnly:
		return true
	default:
		return false
	}
}

type BodyKind string

const (
	BodyNoBody      BodyKind = "no_body"
	BodyDefaultJSON BodyKind = "default_json"
	BodyUpload      BodyKind = "upload"
	BodyExempt      BodyKind = "exempt"
)

func (k BodyKind) Valid() bool {
	switch k {
	case BodyNoBody, BodyDefaultJSON, BodyUpload, BodyExempt:
		return true
	default:
		return false
	}
}

// AccessClass defines the single authorization boundary for a v1 route.
// Zero is deliberately invalid so catalogue construction catches omissions.
type AccessClass uint8

const (
	accessUnspecified AccessClass = iota
	AccessAnonymous
	AccessAuthenticatedRecovery
	AccessCurrentMember
	AccessModerator
)

func (c AccessClass) Valid() bool {
	switch c {
	case AccessAnonymous, AccessAuthenticatedRecovery, AccessCurrentMember, AccessModerator:
		return true
	default:
		return false
	}
}

type RoutePolicy struct {
	Method          string
	PathPattern     string
	RateClass       RateClass
	BodyKind        BodyKind
	AccessClass     AccessClass
	SuspensionClass SuspensionClass
	DevOnly         bool
}

func V1RoutePolicies(env Environment, cfg Config) []RoutePolicy {
	policies := baseV1RoutePolicies()
	if cfg.ModerationAdminEnabled {
		policies = append(policies, adminModerationRoutePolicies()...)
	}
	if env == EnvDev {
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: "/v1/dev/media/{name}", RateClass: RateClassDevOnly, BodyKind: BodyNoBody, AccessClass: AccessAnonymous, DevOnly: true})
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: "/v1/dev/panic", RateClass: RateClassDevOnly, BodyKind: BodyNoBody, AccessClass: AccessAnonymous, DevOnly: true})
		if cfg.EnableDevModeration && cfg.DevModerationToken != "" {
			policies = append(policies, RoutePolicy{Method: "POST", PathPattern: "/v1/dev/moderation/ozone-events", RateClass: RateClassDevOnly, BodyKind: BodyDefaultJSON, AccessClass: AccessAnonymous, DevOnly: true})
		}
	}
	for index := range policies {
		policies[index].SuspensionClass = suspensionClassFor(policies[index])
	}
	return policies
}

type SuspensionClass uint8

const (
	SuspensionUnspecified SuspensionClass = iota
	SuspensionAllowed
	SuspensionDenied
)

func (class SuspensionClass) Valid() bool {
	return class == SuspensionAllowed || class == SuspensionDenied
}
func (class SuspensionClass) AllowedWhenSuspended() bool { return class == SuspensionAllowed }

func suspensionClassFor(policy RoutePolicy) SuspensionClass {
	if policy.Method == http.MethodGet || policy.AccessClass == AccessAnonymous || policy.AccessClass == AccessModerator {
		return SuspensionAllowed
	}
	if retainedSuspendedMutations[policyKey(policy.Method, policy.PathPattern)] {
		return SuspensionAllowed
	}
	if deniedSuspendedMutations[policyKey(policy.Method, policy.PathPattern)] {
		return SuspensionDenied
	}
	return SuspensionUnspecified
}

var retainedSuspendedMutations = map[string]bool{
	"POST /v1/auth/logout":                                           true,
	"POST /v1/account-deletion/intents":                              true,
	"DELETE /v1/account-deletion/intents/{jobId}":                    true,
	"POST /v1/account-deletions/{jobId}":                             true,
	"POST /v1/profiles/{handleOrDid}/mutes":                          true,
	"DELETE /v1/profiles/{handleOrDid}/mutes":                        true,
	"POST /v1/profiles/{handleOrDid}/blocks":                         true,
	"DELETE /v1/profiles/{handleOrDid}/blocks":                       true,
	"POST /v1/profiles/{handleOrDid}/reports":                        true,
	"POST /v1/events/{did}/{rkey}/reports":                           true,
	"POST /v1/posts/{did}/{rkey}/reports":                            true,
	"DELETE /v1/events/{did}/{rkey}":                                 true,
	"DELETE /v1/posts/{did}/{rkey}":                                  true,
	"DELETE /v1/profiles/me/business":                                true,
	"DELETE /v1/posts/{did}/{rkey}/pin":                              true,
	"POST /v1/notifications/seen":                                    true,
	"PATCH /v1/notifications/preferences":                            true,
	"PUT /v1/languages/preferences":                                  true,
	"POST /v1/languages/preferences/initialize":                      true,
	"POST /v1/notifications/devices":                                 true,
	"DELETE /v1/notifications/devices/{accountSubscriptionId}":       true,
	"POST /v1/search/recent":                                         true,
	"DELETE /v1/search/recent/{id}":                                  true,
	"POST /v1/posts/{did}/{rkey}/saves":                              true,
	"DELETE /v1/posts/{did}/{rkey}/saves":                            true,
	"POST /v1/saved-post-folders":                                    true,
	"PATCH /v1/saved-post-folders/{folderId}":                        true,
	"DELETE /v1/saved-post-folders/{folderId}":                       true,
	"DELETE /v1/scheduled-posts/{id}":                                true,
	"DELETE /v1/scheduled-post-media/{mediaId}":                      true,
	"DELETE /v1/migrations/instagram/verifications/{verificationId}": true,
	"DELETE /v1/migrations/instagram/account":                        true,
	"DELETE /v1/migrations/instagram/imports/{importId}":             true,
	"DELETE /v1/migrations/instagram/suggestions/{suggestionId}":     true,
}

// Denied mutations are explicit so a new authenticated mutation cannot silently
// inherit policy; an absent classification prevents catalogue construction.
var deniedSuspendedMutations = map[string]bool{
	"POST /v1/blobs/videos/authorization":                                  true,
	"POST /v1/migrations/instagram/verifications":                          true,
	"POST /v1/migrations/instagram/verifications/{verificationId}/confirm": true,
	"PATCH /v1/migrations/instagram/settings":                              true,
	"POST /v1/migrations/instagram/imports":                                true,
	"PATCH /v1/migrations/instagram/imports/{importId}":                    true,
	"POST /v1/migrations/instagram/suggestions/{suggestionId}/accept":      true,
	"POST /v1/onboarding/completion":                                       true,
	"PUT /v1/profiles/me":                                                  true,
	"PUT /v1/profiles/me/customisation":                                    true,
	"PUT /v1/profiles/me/account-type":                                     true,
	"PUT /v1/profiles/me/business":                                         true,
	"POST /v1/profiles/{handleOrDid}/follows":                              true,
	"DELETE /v1/profiles/{handleOrDid}/follows":                            true,
	"POST /v1/events":                                                      true,
	"PUT /v1/events/{did}/{rkey}":                                          true,
	"POST /v1/blobs/images":                                                true,
	"PUT /v1/scheduled-post-media/{mediaId}":                               true,
	"POST /v1/scheduled-posts":                                             true,
	"PUT /v1/scheduled-posts/{id}":                                         true,
	"POST /v1/scheduled-posts/{id}/publication":                            true,
	"POST /v1/posts":                                                       true,
	"POST /v1/link-previews":                                               true,
	"PUT /v1/posts/{did}/{rkey}/pin":                                       true,
	"POST /v1/posts/{did}/{rkey}/likes":                                    true,
	"DELETE /v1/posts/{did}/{rkey}/likes":                                  true,
	"POST /v1/posts/{did}/{rkey}/reposts":                                  true,
	"DELETE /v1/posts/{did}/{rkey}/reposts":                                true,
}

func mustPolicy(method, pathPattern string) RoutePolicy {
	for _, policy := range baseV1RoutePolicies() {
		if policy.Method == method && policy.PathPattern == pathPattern {
			policy.SuspensionClass = suspensionClassFor(policy)
			return policy
		}
	}
	panic("missing v1 route policy: " + method + " " + pathPattern)
}

func mustConfiguredPolicy(env Environment, cfg Config, method, pathPattern string) RoutePolicy {
	for _, policy := range V1RoutePolicies(env, cfg) {
		if policy.Method == method && policy.PathPattern == pathPattern {
			return policy
		}
	}
	panic("missing configured v1 route policy: " + method + " " + pathPattern)
}

func baseV1RoutePolicies() []RoutePolicy {
	return []RoutePolicy{
		{Method: "GET", PathPattern: "/v1/moderation/standing", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/moderation/history", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/moderation/history/{caseReference}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/auth/login", RateClass: RateClassAuth, BodyKind: BodyDefaultJSON, AccessClass: AccessAnonymous},
		{Method: "POST", PathPattern: "/v1/auth/registrations", RateClass: RateClassAuth, BodyKind: BodyDefaultJSON, AccessClass: AccessAnonymous},
		{Method: "POST", PathPattern: "/v1/auth/handoffs/exchange", RateClass: RateClassAuth, BodyKind: BodyDefaultJSON, AccessClass: AccessAnonymous},
		{Method: "POST", PathPattern: "/v1/auth/handoffs/confirm", RateClass: RateClassAuth, BodyKind: BodyDefaultJSON, AccessClass: AccessAnonymous},
		{Method: "GET", PathPattern: "/v1/whoami", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessAuthenticatedRecovery},
		{Method: "GET", PathPattern: "/v1/facets/mentions", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/facets/mentions/resolve", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/facets/hashtags", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/projects", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/hashtags/{tag}/posts", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/suggestions", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/hashtags", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/profiles", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/posts", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/projects", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/hashtags/top", RateClass: RateClassSearch, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/search/recent", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/search/recent", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/search/recent/{id}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/auth/logout", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessAuthenticatedRecovery},
		{Method: "POST", PathPattern: "/v1/blobs/videos/authorization", RateClass: RateClassUpload, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/blobs/videos/limits", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/account-deletion/intents", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/account-deletion/intents/{jobId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessAuthenticatedRecovery},
		{Method: "POST", PathPattern: "/v1/account-deletions/{jobId}", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessAuthenticatedRecovery},
		{Method: "POST", PathPattern: "/v1/migrations/instagram/verifications", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/migrations/instagram/verifications/current", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/migrations/instagram/verifications/{verificationId}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/migrations/instagram/verifications/{verificationId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/migrations/instagram/verifications/{verificationId}/confirm", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/migrations/instagram/account", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/migrations/instagram/account", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PATCH", PathPattern: "/v1/migrations/instagram/settings", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/migrations/instagram/imports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/migrations/instagram/imports", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/migrations/instagram/imports/{importId}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PATCH", PathPattern: "/v1/migrations/instagram/imports/{importId}", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/migrations/instagram/imports/{importId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/migrations/instagram/suggestions", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/migrations/instagram/suggestions/{suggestionId}/accept", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/migrations/instagram/suggestions/{suggestionId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/onboarding/status", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/onboarding/completion", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me/follower-growth", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me/pins", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me/followers", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me/following", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/profiles/me", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/profiles/me/customisation", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/profiles/me/account-type", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/profiles/me/business", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/profiles/me/business", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/mutual-followers", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/events", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/profiles/{handleOrDid}/follows", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/profiles/{handleOrDid}/follows", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/profiles/{handleOrDid}/mutes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/profiles/{handleOrDid}/mutes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/profiles/{handleOrDid}/blocks", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/profiles/{handleOrDid}/blocks", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me/mutes", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/me/blocks", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/profiles/{handleOrDid}/reports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/events", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/events", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/events/{did}/{rkey}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/events/{did}/{rkey}", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/events/{did}/{rkey}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/events/{did}/{rkey}/reports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/feed/timeline", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/notifications", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/notifications/new-count", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/notifications/seen", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/notifications/preferences", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PATCH", PathPattern: "/v1/notifications/preferences", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/languages/preferences", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/languages/preferences", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/languages/preferences/initialize", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/notifications/devices", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/notifications/devices/{accountSubscriptionId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/blobs/images", RateClass: RateClassUpload, BodyKind: BodyUpload, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/scheduled-post-media/{mediaId}", RateClass: RateClassUpload, BodyKind: BodyUpload, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/scheduled-post-media/{mediaId}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/scheduled-post-media/{mediaId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/scheduled-posts", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/scheduled-posts", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/scheduled-posts/{id}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/scheduled-posts/{id}", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/scheduled-posts/{id}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/scheduled-posts/{id}/publication", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/link-previews", RateClass: RateClassLinkPreview, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/video-captions/{captionCid}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/saves", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/saves", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "PUT", PathPattern: "/v1/posts/{did}/{rkey}/pin", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/pin", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/saved-posts", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/saved-post-folders", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/saved-post-folders", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "PATCH", PathPattern: "/v1/saved-post-folders/{folderId}", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/saved-post-folders/{folderId}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/replies", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/comments", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/quotes", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/reports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/posts", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/projects", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/comments", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
	}
}

func adminModerationRoutePolicies() []RoutePolicy {
	return []RoutePolicy{
		{Method: "GET", PathPattern: "/v1/admin/moderation/cases", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessModerator},
		{Method: "GET", PathPattern: "/v1/admin/moderation/cases/{caseReference}", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessModerator},
		{Method: "POST", PathPattern: "/v1/admin/moderation/cases/{caseReference}/decisions", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessModerator},
		{Method: "POST", PathPattern: "/v1/admin/moderation/cases/{caseReference}/appeal-confirmations", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessModerator},
		{Method: "POST", PathPattern: "/v1/admin/moderation/cases/{caseReference}/appeal-resolutions", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessModerator},
		{Method: "POST", PathPattern: "/v1/admin/moderation/cases/{caseReference}/effect-changes", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessModerator},
		{Method: "POST", PathPattern: "/v1/admin/moderation/cases/{caseReference}/restorations", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessModerator},
	}
}
