package routes

import (
	"context"
	"net/http"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/app"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

const defaultJSONBodyLimitBytes int64 = 1024 * 1024

type v1Middleware struct {
	authN     func(http.Handler) http.Handler
	deviceID  func(http.Handler) http.Handler
	bodyLimit middleware.BodyLimitConfig
	rateLimit map[RateClass]func(http.Handler) http.Handler
	observer  *observability.Observer
}

func (m v1Middleware) wrap(policy RoutePolicy, handler http.Handler) http.Handler {
	wrapped := handler
	if rl := m.rateLimit[policy.RateClass]; rl != nil {
		wrapped = rl(wrapped)
	}
	if policy.AuthRequired {
		wrapped = m.deviceID(wrapped)
		wrapped = m.authN(wrapped)
	} else if policy.RateClass == RateClassAuth {
		wrapped = m.deviceID(wrapped)
	}
	wrapped = middleware.BodyLimit(m.bodyLimit, middleware.BodyKind(policy.BodyKind), nil)(wrapped)
	return middleware.HTTPInFlight(m.observer)(wrapped)
}

// AddRoutes registers all App View routes on mux.
func AddRoutes(ctx context.Context, mux *http.ServeMux, deps *app.Deps) {
	observer := deps.Observability
	if observer == nil {
		observer = observability.New(observability.Config{Env: string(deps.Config.Env)})
	}
	inFlight := middleware.HTTPInFlight(observer)

	// Public ops.
	mux.Handle("GET /health", inFlight(api.HealthHandler(deps.DB, deps.Logger)))
	mux.Handle("GET /healthz", inFlight(api.NewHealthHandler(deps.DB, deps.Consumer)))
	if deps.Config.Env == app.EnvDev {
		mux.Handle("GET /v1/dev/media/{name}", inFlight(api.DevMediaHandler()))
		mux.Handle("GET /v1/dev/panic", inFlight(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("synthetic appview dev panic")
		})))
	}

	// OAuth discovery endpoints (contracts with the AS; not versioned).
	oauthHandlers := auth.NewHTTPHandlers(
		deps.OAuthApp,
		deps.CraftskySessionStore,
		deps.DB,
		deps.Logger,
		deps.Config.Env == app.EnvDev,
		deps.NewPDSClient,
		deps.IdentityCacheUpdater,
	)
	oauthHandlers.RepositoryTracker = deps.RepositoryTracker
	mux.Handle("GET /oauth/client-metadata.json", inFlight(oauthHandlers.ClientMetadataHandler()))
	mux.Handle("GET /oauth/jwks.json", inFlight(oauthHandlers.JWKSHandler()))
	mux.Handle("GET /oauth/callback", inFlight(oauthHandlers.CallbackHandler()))

	// Middleware stacks.
	authN := middleware.Authenticated(deps.AuthService, deps.Logger)
	deviceID := middleware.DeviceID(deps.CraftskySessionStore, deps.Logger)
	rateLimits := map[RateClass]func(http.Handler) http.Handler{}
	if deps.RateLimiter != nil {
		rateLimits[RateClassAuth] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassAuth, deps.Logger)
		rateLimits[RateClassRead] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassRead, deps.Logger)
		rateLimits[RateClassWrite] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassWrite, deps.Logger)
		rateLimits[RateClassSearch] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassSearch, deps.Logger)
		rateLimits[RateClassUpload] = middleware.RateLimit(deps.RateLimiter, middleware.RateClassUpload, deps.Logger)
	}
	bodyLimitCfg := middleware.BodyLimitConfig{
		DefaultJSONBytes: deps.Config.JSONBodyLimitBytes,
		UploadBytes:      deps.Config.MaxImageUploadBytes,
	}
	if bodyLimitCfg.DefaultJSONBytes == 0 {
		bodyLimitCfg.DefaultJSONBytes = defaultJSONBodyLimitBytes
	}
	v1mw := v1Middleware{authN: authN, deviceID: deviceID, bodyLimit: bodyLimitCfg, rateLimit: rateLimits, observer: observer}

	// v1 — unauthenticated but device-id required.
	mux.Handle("POST /v1/auth/login", v1mw.wrap(mustPolicy("POST", "/v1/auth/login"), oauthHandlers.LoginHandler()))

	// v1 — authenticated + device-id required.
	mediaLimits := api.MediaLimits{
		MaxPostImages:       deps.Config.MaxPostImages,
		MaxImageUploadBytes: deps.Config.MaxImageUploadBytes,
	}
	facetStore := api.NewFacetStore(deps.DB, deps.HandleResolver)
	searchStore := api.NewSearchStore(deps.DB, observer)
	mux.Handle("GET /v1/whoami", v1mw.wrap(mustPolicy("GET", "/v1/whoami"), api.WhoAmIHandler(deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/facets/mentions", v1mw.wrap(mustPolicy("GET", "/v1/facets/mentions"), api.ListFacetMentionSuggestionsHandler(facetStore, deps.Logger)))
	mux.Handle("GET /v1/facets/mentions/resolve", v1mw.wrap(mustPolicy("GET", "/v1/facets/mentions/resolve"), api.ResolveFacetMentionHandler(facetStore, deps.Logger)))
	mux.Handle("GET /v1/facets/hashtags", v1mw.wrap(mustPolicy("GET", "/v1/facets/hashtags"), api.ListFacetHashtagSuggestionsHandler(facetStore, deps.Logger)))
	mux.Handle("GET /v1/projects", v1mw.wrap(mustPolicy("GET", "/v1/projects"), api.ListProjectsHandler(searchStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/search/hashtags/{tag}/posts", v1mw.wrap(mustPolicy("GET", "/v1/search/hashtags/{tag}/posts"), api.SearchHashtagPostsHandler(searchStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/search/suggestions", v1mw.wrap(mustPolicy("GET", "/v1/search/suggestions"), api.SearchSuggestionsHandler(searchStore, deps.Logger)))
	mux.Handle("GET /v1/search/hashtags", v1mw.wrap(mustPolicy("GET", "/v1/search/hashtags"), api.SearchHashtagsHandler(searchStore, deps.Logger)))
	mux.Handle("GET /v1/search/profiles", v1mw.wrap(mustPolicy("GET", "/v1/search/profiles"), api.SearchProfilesHandler(searchStore, deps.Logger)))
	mux.Handle("GET /v1/search/posts", v1mw.wrap(mustPolicy("GET", "/v1/search/posts"), api.SearchPostsHandler(searchStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/search/projects", v1mw.wrap(mustPolicy("GET", "/v1/search/projects"), api.SearchProjectsHandler(searchStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/search/hashtags/top", v1mw.wrap(mustPolicy("GET", "/v1/search/hashtags/top"), api.TopHashtagsHandler(searchStore, deps.Logger)))
	mux.Handle("GET /v1/search/recent", v1mw.wrap(mustPolicy("GET", "/v1/search/recent"), api.ListRecentSearchesHandler(searchStore, deps.Logger)))
	mux.Handle("POST /v1/search/recent", v1mw.wrap(mustPolicy("POST", "/v1/search/recent"), api.SaveRecentSearchHandler(searchStore, deps.Logger)))
	mux.Handle("DELETE /v1/search/recent/{id}", v1mw.wrap(mustPolicy("DELETE", "/v1/search/recent/{id}"), api.DeleteRecentSearchHandler(searchStore, deps.Logger)))
	mux.Handle("POST /v1/auth/logout", v1mw.wrap(mustPolicy("POST", "/v1/auth/logout"), oauthHandlers.LogoutHandler()))
	mux.Handle("GET /v1/profiles/{handleOrDid}", v1mw.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}"), api.GetProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/me", v1mw.wrap(mustPolicy("GET", "/v1/profiles/me"), api.GetMeProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/me/followers", v1mw.wrap(mustPolicy("GET", "/v1/profiles/me/followers"), api.GetMeFollowersHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/me/following", v1mw.wrap(mustPolicy("GET", "/v1/profiles/me/following"), api.GetMeFollowingHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("PUT /v1/profiles/me", v1mw.wrap(mustPolicy("PUT", "/v1/profiles/me"), api.PutMeProfileHandler(deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, mediaLimits, deps.Logger)))
	mux.Handle("GET /v1/profiles/{handleOrDid}/mutual-followers", v1mw.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/mutual-followers"), api.GetMutualFollowersHandler(deps.ProfileStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("POST /v1/profiles/{handleOrDid}/follows", v1mw.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/follows"), api.FollowProfileHandler(deps.FollowStore, deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, deps.Logger)))
	mux.Handle("DELETE /v1/profiles/{handleOrDid}/follows", v1mw.wrap(mustPolicy("DELETE", "/v1/profiles/{handleOrDid}/follows"), api.UnfollowProfileHandler(deps.FollowStore, deps.ProfileStore, deps.HandleResolver, deps.NewPDSClient, deps.Logger)))
	mux.Handle("POST /v1/profiles/{handleOrDid}/mutes", v1mw.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/mutes"), api.MuteProfileHandler(deps.RelationshipMutations, deps.RelationshipStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("DELETE /v1/profiles/{handleOrDid}/mutes", v1mw.wrap(mustPolicy("DELETE", "/v1/profiles/{handleOrDid}/mutes"), api.UnmuteProfileHandler(deps.RelationshipMutations, deps.RelationshipStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("POST /v1/profiles/{handleOrDid}/blocks", v1mw.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/blocks"), api.BlockProfileHandler(deps.RelationshipMutations, deps.RelationshipStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("DELETE /v1/profiles/{handleOrDid}/blocks", v1mw.wrap(mustPolicy("DELETE", "/v1/profiles/{handleOrDid}/blocks"), api.UnblockProfileHandler(deps.RelationshipMutations, deps.RelationshipStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/me/mutes", v1mw.wrap(mustPolicy("GET", "/v1/profiles/me/mutes"), api.ListMutedProfilesHandler(deps.RelationshipStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/me/blocks", v1mw.wrap(mustPolicy("GET", "/v1/profiles/me/blocks"), api.ListBlockedProfilesHandler(deps.RelationshipStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("POST /v1/profiles/{handleOrDid}/reports", v1mw.wrap(mustPolicy("POST", "/v1/profiles/{handleOrDid}/reports"), api.ReportProfileHandler(api.NewProfileReportTargetResolver(deps.ProfileStore, deps.HandleResolver), deps.ReportStore, deps.ReportForwarder, deps.Logger)))

	// v1 — post handlers (authenticated + device-id required).
	postStore := api.NewPostStore(deps.DB, observer)
	oauthHandlers.NotificationSubscriptions = postStore
	mux.Handle("GET /v1/feed/timeline", v1mw.wrap(mustPolicy("GET", "/v1/feed/timeline"), api.ListTimelineHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/notifications", v1mw.wrap(mustPolicy("GET", "/v1/notifications"), api.ListNotificationsHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/notifications/new-count", v1mw.wrap(mustPolicy("GET", "/v1/notifications/new-count"), api.NotificationNewCountHandler(postStore, deps.Logger)))
	mux.Handle("POST /v1/notifications/seen", v1mw.wrap(mustPolicy("POST", "/v1/notifications/seen"), api.MarkNotificationsSeenHandler(postStore, deps.Logger)))
	mux.Handle("GET /v1/notifications/preferences", v1mw.wrap(mustPolicy("GET", "/v1/notifications/preferences"), api.GetNotificationPreferencesHandler(postStore, deps.Logger)))
	mux.Handle("PATCH /v1/notifications/preferences", v1mw.wrap(mustPolicy("PATCH", "/v1/notifications/preferences"), api.PatchNotificationPreferencesHandler(postStore, deps.Logger)))
	mux.Handle("POST /v1/notifications/devices", v1mw.wrap(mustPolicy("POST", "/v1/notifications/devices"), api.RegisterNotificationDeviceHandler(postStore, deps.Logger)))
	mux.Handle("DELETE /v1/notifications/devices/{accountSubscriptionId}", v1mw.wrap(mustPolicy("DELETE", "/v1/notifications/devices/{accountSubscriptionId}"), api.RemoveNotificationDeviceHandler(postStore, deps.Logger)))
	mux.Handle("POST /v1/blobs/images", v1mw.wrap(mustPolicy("POST", "/v1/blobs/images"), api.ImageBlobUploadHandler(deps.NewPDSClient, mediaLimits, deps.Logger)))
	mux.Handle("POST /v1/posts", v1mw.wrap(mustPolicy("POST", "/v1/posts"), api.CreatePostHandler(postStore, deps.NewPDSClient, deps.HandleResolver, mediaLimits, deps.Logger)))
	mux.Handle("GET /v1/posts/{did}/{rkey}", v1mw.wrap(mustPolicy("GET", "/v1/posts/{did}/{rkey}"), api.GetPostHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/posts/{did}/{rkey}/replies", v1mw.wrap(mustPolicy("GET", "/v1/posts/{did}/{rkey}/replies"), api.ListCommentRepliesHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/posts/{did}/{rkey}/comments", v1mw.wrap(mustPolicy("GET", "/v1/posts/{did}/{rkey}/comments"), api.GetPostCommentsHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("POST /v1/posts/{did}/{rkey}/likes", v1mw.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/likes"), api.LikePostHandler(postStore, deps.NewPDSClient, deps.Logger)))
	mux.Handle("DELETE /v1/posts/{did}/{rkey}/likes", v1mw.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}/likes"), api.UnlikePostHandler(postStore, deps.NewPDSClient, deps.Logger)))
	mux.Handle("POST /v1/posts/{did}/{rkey}/reposts", v1mw.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/reposts"), api.RepostPostHandler(postStore, deps.NewPDSClient, deps.Logger)))
	mux.Handle("DELETE /v1/posts/{did}/{rkey}/reposts", v1mw.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}/reposts"), api.UnrepostPostHandler(postStore, deps.NewPDSClient, deps.Logger)))
	mux.Handle("DELETE /v1/posts/{did}/{rkey}", v1mw.wrap(mustPolicy("DELETE", "/v1/posts/{did}/{rkey}"), api.DeletePostHandler(deps.NewPDSClient, deps.Logger)))
	mux.Handle("POST /v1/posts/{did}/{rkey}/reports", v1mw.wrap(mustPolicy("POST", "/v1/posts/{did}/{rkey}/reports"), api.ReportPostHandler(postStore, deps.ReportStore, deps.ReportForwarder, deps.Logger)))
	if deps.Config.Env == app.EnvDev && deps.Config.EnableDevModeration && deps.Config.DevModerationToken != "" {
		mux.Handle("POST /v1/dev/moderation/ozone-events",
			inFlight(api.DevModerationOzoneEventsHandler(
				deps.Config.DevModerationToken,
				api.ModerationRequestConfig{
					DefaultSourceDID:  deps.Config.DevLabelerDID,
					TrustedSourceDIDs: deps.Config.TrustedModerationSourceDIDs,
				},
				deps.ModerationStore,
				deps.Logger,
			)))
	}
	mux.Handle("GET /v1/profiles/{handleOrDid}/posts", v1mw.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/posts"), api.ListPostsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/{handleOrDid}/projects", v1mw.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/projects"), api.ListProjectsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger)))
	mux.Handle("GET /v1/profiles/{handleOrDid}/comments", v1mw.wrap(mustPolicy("GET", "/v1/profiles/{handleOrDid}/comments"), api.ListCommentsByAuthorHandler(postStore, deps.HandleResolver, deps.Logger)))

	// Fallthrough.
	mux.Handle("/", inFlight(http.NotFoundHandler()))
}
