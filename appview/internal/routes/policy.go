package routes

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
)

func (c AccessClass) Valid() bool {
	switch c {
	case AccessAnonymous, AccessAuthenticatedRecovery, AccessCurrentMember:
		return true
	default:
		return false
	}
}

type RoutePolicy struct {
	Method      string
	PathPattern string
	RateClass   RateClass
	BodyKind    BodyKind
	AccessClass AccessClass
	DevOnly     bool
}

func V1RoutePolicies(env Environment, cfg Config) []RoutePolicy {
	policies := baseV1RoutePolicies()
	if env == EnvDev {
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: "/v1/dev/media/{name}", RateClass: RateClassDevOnly, BodyKind: BodyNoBody, AccessClass: AccessAnonymous, DevOnly: true})
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: "/v1/dev/panic", RateClass: RateClassDevOnly, BodyKind: BodyNoBody, AccessClass: AccessAnonymous, DevOnly: true})
		if cfg.EnableDevModeration && cfg.DevModerationToken != "" {
			policies = append(policies, RoutePolicy{Method: "POST", PathPattern: "/v1/dev/moderation/ozone-events", RateClass: RateClassDevOnly, BodyKind: BodyDefaultJSON, AccessClass: AccessAnonymous, DevOnly: true})
		}
	}
	return policies
}

func mustPolicy(method, pathPattern string) RoutePolicy {
	for _, policy := range baseV1RoutePolicies() {
		if policy.Method == method && policy.PathPattern == pathPattern {
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
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/reports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/posts", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/projects", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/comments", RateClass: RateClassRead, BodyKind: BodyNoBody, AccessClass: AccessCurrentMember},
	}
}
