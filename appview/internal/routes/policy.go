package routes

import "social.craftsky/appview/internal/app"

type RateClass string

const (
	RateClassAuth    RateClass = "auth"
	RateClassRead    RateClass = "read"
	RateClassWrite   RateClass = "write"
	RateClassSearch  RateClass = "expensive_search"
	RateClassUpload  RateClass = "upload"
	RateClassExempt  RateClass = "exempt"
	RateClassDevOnly RateClass = "dev_only_relaxed"
)

func (c RateClass) Valid() bool {
	switch c {
	case RateClassAuth, RateClassRead, RateClassWrite, RateClassSearch, RateClassUpload, RateClassExempt, RateClassDevOnly:
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

type RoutePolicy struct {
	Method       string
	PathPattern  string
	RateClass    RateClass
	BodyKind     BodyKind
	AuthRequired bool
	DevOnly      bool
}

func V1RoutePolicies(env app.Env, cfg app.Config) []RoutePolicy {
	policies := baseV1RoutePolicies()
	if env == app.EnvDev {
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: "/v1/dev/media/{name}", RateClass: RateClassDevOnly, BodyKind: BodyNoBody, DevOnly: true})
		policies = append(policies, RoutePolicy{Method: "GET", PathPattern: "/v1/dev/panic", RateClass: RateClassDevOnly, BodyKind: BodyNoBody, DevOnly: true})
		if cfg.EnableDevModeration && cfg.DevModerationToken != "" {
			policies = append(policies, RoutePolicy{Method: "POST", PathPattern: "/v1/dev/moderation/ozone-events", RateClass: RateClassDevOnly, BodyKind: BodyDefaultJSON, DevOnly: true})
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

func baseV1RoutePolicies() []RoutePolicy {
	return []RoutePolicy{
		{Method: "POST", PathPattern: "/v1/auth/login", RateClass: RateClassAuth, BodyKind: BodyDefaultJSON},
		{Method: "GET", PathPattern: "/v1/whoami", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/facets/mentions", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/facets/mentions/resolve", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/facets/hashtags", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/projects", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/hashtags/{tag}/posts", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/suggestions", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/hashtags", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/profiles", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/posts", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/projects", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/hashtags/top", RateClass: RateClassSearch, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/search/recent", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/search/recent", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AuthRequired: true},
		{Method: "DELETE", PathPattern: "/v1/search/recent/{id}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/auth/logout", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/me", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/me/followers", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/me/following", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "PUT", PathPattern: "/v1/profiles/me", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/mutual-followers", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/profiles/{handleOrDid}/follows", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "DELETE", PathPattern: "/v1/profiles/{handleOrDid}/follows", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/profiles/{handleOrDid}/reports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/feed/timeline", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/notifications", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/blobs/images", RateClass: RateClassUpload, BodyKind: BodyUpload, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/posts", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/replies", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/posts/{did}/{rkey}/comments", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/likes", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}/reposts", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "DELETE", PathPattern: "/v1/posts/{did}/{rkey}", RateClass: RateClassWrite, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "POST", PathPattern: "/v1/posts/{did}/{rkey}/reports", RateClass: RateClassWrite, BodyKind: BodyDefaultJSON, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/posts", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/projects", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
		{Method: "GET", PathPattern: "/v1/profiles/{handleOrDid}/comments", RateClass: RateClassRead, BodyKind: BodyNoBody, AuthRequired: true},
	}
}
